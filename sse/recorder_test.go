package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func newTestRecorder(t *testing.T, o Options) *Recorder {
	t.Helper()
	r, err := New(o)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Error(err)
		}
	})
	return r
}
func TestStartupHistoryRedactionAndRestart(t *testing.T) {
	dir := t.TempDir()
	r := newTestRecorder(t, Options{Directory: dir})
	logger := zap.New(r.ZapCore(zapcore.InfoLevel)).Named("worker").With(zap.String("job", "abc"))
	fields := map[string]any{"label": "unchanged", "password": "original-secret"}
	r.Record(Event{Message: "immutable", Fields: fields})
	if fields["password"] != "original-secret" {
		t.Fatal("recorder mutated caller fields")
	}
	logger.Warn("request failed", zap.String("password", "private-1"), zap.Any("nested", map[string]any{"deviceKey": "private-2", "url": "tcp://alice:private-3@localhost:1883/?token=private-4"}), zap.String("error", `error {"password":"private-5"}`))
	_, _ = fmt.Fprintln(r.Writer("standard"), "boot complete")
	b := r.Snapshot(Filter{}, "", 200)
	if len(b.Events) != 3 || b.Events[1].Level != "warn" || b.Events[1].Fields["job"] != "abc" || b.Events[1].Source != "worker" {
		t.Fatalf("bad adapter events: %+v", b)
	}
	// Snapshot callers must not be able to mutate stored fields.
	b.Events[1].Fields["job"] = "mutated"
	if got := r.Snapshot(Filter{}, "", 200).Events[1].Fields["job"]; got != "abc" {
		t.Fatal(got)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if len(files) == 0 {
		t.Fatal("no history persisted")
	}
	data, _ := os.ReadFile(files[0])
	for i := 1; i <= 5; i++ {
		if strings.Contains(string(data), fmt.Sprintf("private-%d", i)) {
			t.Fatalf("secret persisted: %s", data)
		}
	}
	r2 := newTestRecorder(t, Options{Directory: dir})
	h, err := r2.History(Filter{Source: "worker"}, 200)
	if err != nil || len(h.Events) != 1 {
		t.Fatalf("restart lost history: %+v %v", h, err)
	}
}
func TestCursorFilteringSlowReaderAndConcurrency(t *testing.T) {
	r := newTestRecorder(t, Options{MaxEvents: 10, MaxMemoryBytes: 16 << 10})
	first := r.Record(Event{DeviceID: "a", Message: "first"})
	b, wake := r.Watch(Filter{DeviceID: "a"}, first, 200)
	if len(b.Events) != 0 {
		t.Fatal(b)
	}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 30; j++ {
				r.Record(Event{DeviceID: "b", Message: "other"})
			}
		}()
	}
	wg.Wait()
	select {
	case <-wake:
	default:
		t.Fatal("watch missed write")
	}
	b = r.Snapshot(Filter{DeviceID: "a"}, first, 200)
	if !b.Gap || len(b.Events) != 0 || b.Cursor == first {
		t.Fatal(b)
	}
	r.Record(Event{DeviceID: "a", Message: "new"})
	next := r.Snapshot(Filter{DeviceID: "a"}, b.Cursor, 200)
	if len(next.Events) != 1 || next.Gap {
		t.Fatal(next)
	}
	for i := 0; i < 450; i++ {
		r.Record(Event{Message: "batch"})
	}
	b = r.Snapshot(Filter{}, "", 5)
	if len(b.Events) != 5 {
		t.Fatal(b)
	}
}
func TestDebugScopeExpiryAndEventBounds(t *testing.T) {
	r := newTestRecorder(t, Options{MaxEventBytes: 2048})
	if r.Record(Event{Level: "debug", DeviceID: "d"}) != "" {
		t.Fatal("unscoped debug recorded")
	}
	s, err := r.StartDebug("d", "c")
	if err != nil {
		t.Fatal(err)
	}
	if r.DebugEnabled("d", "other") || !r.DebugEnabled("d", "c") {
		t.Fatal("scope mismatch")
	}
	id := r.Record(Event{Level: "debug", DeviceID: "d", ClientID: "c", Message: strings.Repeat("x", 50000)})
	if id == "" {
		t.Fatal("debug not recorded")
	}
	b := r.Snapshot(Filter{}, "", 200)
	raw, _ := json.Marshal(b.Events[0])
	if len(raw) > 2048 || !b.Events[0].Truncated {
		t.Fatalf("unbounded event: %d", len(raw))
	}
	r.mu.Lock()
	s.ExpiresAt = time.Now().Add(-time.Second)
	r.debug[s.ID] = s
	r.mu.Unlock()
	if r.DebugEnabled("d", "c") || len(r.DebugSessions()) != 0 {
		t.Fatal("session did not expire")
	}
	if Preview([]byte(`{"password":"raw-value","value":1}`)) == "" || strings.Contains(Preview([]byte(`{"password":"raw-value"}`)), "raw-value") {
		t.Fatal("preview leaks secrets")
	}
}
func TestRotationQuotaAndClose(t *testing.T) {
	dir := t.TempDir()
	r := newTestRecorder(t, Options{Directory: dir, MaxFileBytes: 1024, MaxDiskBytes: 4096, QueueSize: 1000})
	for i := 0; i < 100; i++ {
		r.Record(Event{Message: strings.Repeat("x", 200)})
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	var total int64
	for _, p := range files {
		st, _ := os.Stat(p)
		if st.Size() > 1024 {
			t.Fatal("file quota exceeded")
		}
		total += st.Size()
	}
	if total > 4096 {
		t.Fatalf("disk quota exceeded: %d", total)
	}
	if r.Record(Event{Message: "after close"}) != "" {
		t.Fatal("closed recorder accepted event")
	}
}
func TestSSEGapAndCancellation(t *testing.T) {
	r := newTestRecorder(t, Options{MaxEvents: 2})
	id := r.Record(Event{Message: "old"})
	for i := 0; i < 3; i++ {
		r.Record(Event{Message: "new"})
	}
	ctx, cancel := context.WithCancel(context.Background())
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/events", nil).WithContext(ctx)
	cancel()
	r.Stream(w, req, Filter{}, id, func() bool { return true })
	body := w.Body.String()
	if !strings.Contains(body, "event: gap") || !strings.Contains(body, "event: log") || !strings.Contains(body, "id: ") {
		t.Fatal(body)
	}
	w = httptest.NewRecorder()
	r.Stream(w, req, Filter{}, "", func() bool { return false })
	if !strings.Contains(w.Body.String(), "event: auth_expired") {
		t.Fatal(w.Body.String())
	}
}

type blockedConsole struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (b *blockedConsole) Write(p []byte) (int, error) {
	b.once.Do(func() { close(b.entered) })
	<-b.release
	return len(p), nil
}
func TestSlowSinkDoesNotBlockProducersAndReportsDrops(t *testing.T) {
	sink := &blockedConsole{entered: make(chan struct{}), release: make(chan struct{})}
	r := newTestRecorder(t, Options{Console: sink, QueueSize: 2})
	r.Record(Event{Message: "first"})
	<-sink.entered
	for i := 0; i < 20; i++ {
		r.Record(Event{Message: "burst"})
	}
	batch := r.Snapshot(Filter{}, "", 200)
	if batch.Status.DiskDropped == 0 || len(batch.Events) != 21 {
		t.Fatalf("missing drop status or live events: %+v", batch)
	}
	close(sink.release)
}

func TestResumeDrainsAllBatchesAndWatermarkIncludesFilteredEvents(t *testing.T) {
	r := newTestRecorder(t, Options{})
	cursor := r.Record(Event{DeviceID: "d", Message: "baseline"})
	for i := 0; i < 550; i++ {
		r.Record(Event{DeviceID: "d", Message: fmt.Sprintf("event-%d", i)})
	}
	r.Record(Event{DeviceID: "unrelated", Message: "filtered tail"})
	total := 0
	seen := map[string]bool{}
	for {
		b := r.Snapshot(Filter{DeviceID: "d"}, cursor, 200)
		if b.Gap {
			t.Fatal("unexpected gap")
		}
		for _, e := range b.Events {
			if seen[e.ID] {
				t.Fatal("duplicate replay")
			}
			seen[e.ID] = true
		}
		total += len(b.Events)
		cursor = b.Cursor
		if len(b.Events) < 200 {
			break
		}
	}
	if total != 550 {
		t.Fatalf("incomplete replay: %d", total)
	}
	if b := r.Snapshot(Filter{}, cursor, 200); len(b.Events) != 0 {
		t.Fatal("watermark did not advance over filtered tail")
	}
}
