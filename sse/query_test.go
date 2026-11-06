package sse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestIndexedPagingAcrossRestartAndStableWatermark(t *testing.T) {
	dir := t.TempDir()
	r := newTestRecorder(t, Options{Directory: dir, MaxEvents: 10, QueueSize: 2000})
	ids := []string{}
	for i := 0; i < 620; i++ {
		ids = append(ids, r.Record(Event{Message: fmt.Sprintf("event-%d", i)}))
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	r = newTestRecorder(t, Options{Directory: dir, MaxEvents: 10, QueueSize: 2000})
	r.Record(Event{Message: "new instance"})
	ctx := context.Background()
	before, watermark := "", ""
	seen := map[string]bool{}
	for page := 0; page < 20; page++ {
		b, err := r.Query(ctx, Filter{}, Page{Before: before, Watermark: watermark, Limit: 37})
		if err != nil {
			t.Fatal(err)
		}
		if watermark == "" {
			watermark = b.Watermark
			r.Record(Event{Message: "after fixed water"})
		}
		for _, e := range b.Events {
			if seen[e.ID] || e.ID > watermark {
				t.Fatalf("unstable or duplicate: %s", e.ID)
			}
			seen[e.ID] = true
		}
		if !b.HasMore {
			break
		}
		if before == b.Next {
			t.Fatal("stuck")
		}
		before = b.Next
	}
	if len(seen) != 621 {
		t.Fatalf("history lost events: %d", len(seen))
	}
	cursor := ids[0]
	seen = map[string]bool{}
	for page := 0; page < 30; page++ {
		b, err := r.Query(ctx, Filter{}, Page{After: cursor, Limit: 31, Watermark: watermark})
		if err != nil {
			t.Fatal(err)
		}
		if b.Gap {
			t.Fatal("false retention gap")
		}
		for _, e := range b.Events {
			if seen[e.ID] || e.ID <= cursor {
				t.Fatal("duplicate/invalid order")
			}
			seen[e.ID] = true
		}
		if !b.HasMore {
			break
		}
		if b.Next <= cursor {
			t.Fatal("no progress")
		}
		cursor = b.Next
	}
	if len(seen) != 620 {
		t.Fatalf("forward replay incomplete %d", len(seen))
	}
}
func TestBudgetPaginationCanProgressWithoutMatches(t *testing.T) {
	r := newTestRecorder(t, Options{Directory: t.TempDir(), MaxEvents: 2, QueueSize: 2000})
	for i := 0; i < 1000; i++ {
		r.Record(Event{Message: strings.Repeat("x", 100), DeviceID: fmt.Sprint(i)})
	}
	cursor := "!"
	pages := 0
	for ; pages < 30; pages++ {
		b, err := r.Query(context.Background(), Filter{DeviceID: "absent"}, Page{After: cursor, Limit: 20, ScanBytes: 1000})
		if err != nil {
			t.Fatal(err)
		}
		if len(b.Events) != 0 {
			t.Fatal("filter ignored")
		}
		if !b.HasMore {
			break
		}
		if !b.Limited || b.Next <= cursor {
			t.Fatalf("budget cannot advance %+v", b)
		}
		cursor = b.Next
	}
	if pages < 2 || pages >= 30 {
		t.Fatalf("unexpected budget pages %d", pages)
	}
}
func TestExportCompletenessRedactionAndLimit(t *testing.T) {
	r := newTestRecorder(t, Options{Directory: t.TempDir(), MaxEvents: 2, QueueSize: 1000})
	for i := 0; i < 230; i++ {
		r.Record(Event{Message: fmt.Sprintf("event-%d password=private-value", i)})
	}
	for _, limit := range []int{230, 20} {
		var out bytes.Buffer
		n, complete, err := r.Export(context.Background(), &out, Filter{}, limit, 1<<20)
		if err != nil || n != limit || complete != (limit == 230) {
			t.Fatalf("bad export %d %t %v", n, complete, err)
		}
		if strings.Contains(out.String(), "private-value") {
			t.Fatal("export leaks")
		}
		lines := strings.Split(strings.TrimSpace(out.String()), "\n")
		var manifest map[string]any
		if json.Unmarshal([]byte(lines[len(lines)-1]), &manifest) != nil || manifest["type"] != "export_manifest" || manifest["complete"] != complete {
			t.Fatal("missing manifest")
		}
	}
}
func TestSuppressionCountsAndCriticalReserve(t *testing.T) {
	sink := &blockedConsole{entered: make(chan struct{}), release: make(chan struct{})}
	r := newTestRecorder(t, Options{Console: sink, QueueSize: 2})
	r.Record(Event{Message: "block"})
	<-sink.entered
	for i := 0; i < 50; i++ {
		r.Record(Event{Message: "normal"})
	}
	for i := 0; i < 25; i++ {
		r.Record(Event{Source: "test", Level: "error", Message: "same failure"})
	}
	b := r.Snapshot(Filter{}, "", 200)
	if b.Status.CriticalDropped != 0 || b.Status.DiskDropped == 0 || b.Status.Suppressed != 24 {
		t.Fatalf("wrong reserve %+v", b.Status)
	}
	s, _ := r.Summary(context.Background(), time.Now().Add(-time.Minute))
	if s.Errors != 25 || len(s.Faults) != 1 || s.Faults[0].Count != 25 {
		t.Fatalf("wrong summary %+v", s)
	}
	close(sink.release)
}
func TestSSEDiskReplayAndConcurrentQueries(t *testing.T) {
	r := newTestRecorder(t, Options{Directory: t.TempDir(), MaxEvents: 2, QueueSize: 2000})
	cursor := r.Record(Event{Message: "cursor"})
	for i := 0; i < 520; i++ {
		r.Record(Event{Message: fmt.Sprint(i)})
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	writer := &cancellingResponse{ResponseRecorder: httptest.NewRecorder(), cancel: cancel}
	r.Stream(writer, httptest.NewRequest("GET", "/events", nil).WithContext(ctx), Filter{}, cursor, func() bool { return true })
	if strings.Contains(writer.Body.String(), "event: gap") || strings.Count(writer.Body.String(), "event: log") != 520 {
		t.Fatalf("disk replay incomplete: %d", strings.Count(writer.Body.String(), "event: log"))
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_, err := r.Query(context.Background(), Filter{}, Page{Limit: 20})
				if err != nil && err != ErrBusy {
					t.Error(err)
				}
			}
		}()
	}
	for i := 0; i < 200; i++ {
		r.Record(Event{Message: "during queries"})
	}
	wg.Wait()
}

type cancellingResponse struct {
	*httptest.ResponseRecorder
	cancel context.CancelFunc
}

func (w *cancellingResponse) Flush() {
	w.ResponseRecorder.Flush()
	if strings.Contains(w.Body.String(), `"replaying":false`) {
		w.cancel()
	}
}
func BenchmarkRecordWithQuery(b *testing.B) {
	r, err := New(Options{Directory: b.TempDir(), QueueSize: 20000})
	if err != nil {
		b.Fatal(err)
	}
	defer r.Close()
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for ctx.Err() == nil {
			_, _ = r.Query(ctx, Filter{}, Page{Limit: 200})
		}
	}()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Record(Event{Source: "mqtt.server", Message: "throughput summary", Fields: map[string]any{"bytes": 1000}})
	}
	b.StopTimer()
	cancel()
	wg.Wait()
}
