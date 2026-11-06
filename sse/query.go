package sse

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sort"
	"strings"
	"time"
)

var ErrBusy = errors.New("日志查询繁忙，请稍后重试")

// Sparse offsets are built once on startup and extended after successful writes.
// Queries inspect only blocks intersecting their ID/time window, with a scan budget.
type diskBlock struct {
	Path, First, Last string
	Offset, Bytes     int64
	Count             int
	MinTime, MaxTime  time.Time
}

func (r *Recorder) indexEntry(path string, offset int64, x entry) {
	r.diskMu.Lock()
	defer r.diskMu.Unlock()
	n := len(r.blocks)
	if n == 0 || r.blocks[n-1].Path != path || r.blocks[n-1].Count >= 128 {
		r.blocks = append(r.blocks, diskBlock{Path: path, Offset: offset, First: x.event.ID, MinTime: x.event.Time, MaxTime: x.event.Time})
		n++
	}
	b := &r.blocks[n-1]
	b.Last = x.event.ID
	b.Bytes = offset + int64(len(x.data)+1) - b.Offset
	b.Count++
	if x.event.Time.Before(b.MinTime) {
		b.MinTime = x.event.Time
	}
	if x.event.Time.After(b.MaxTime) {
		b.MaxTime = x.event.Time
	}
}
func (r *Recorder) Sync(ctx context.Context) error {
	barrier := make(chan error, 1)
	r.mu.Lock()
	if r.status.Closed {
		r.mu.Unlock()
		select {
		case <-r.done:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	select {
	case r.queue <- entry{barrier: barrier}:
	default:
		r.mu.Unlock()
		return ErrBusy
	}
	r.mu.Unlock()
	select {
	case err := <-barrier:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

type Page struct {
	After     string
	Before    string
	Watermark string
	Limit     int
	ScanBytes int64
}

// Query returns ascending events. After pages move forward (SSE/export); other
// pages move backwards (history). Watermark fixes a stable result across pages.
func (r *Recorder) Query(ctx context.Context, f Filter, p Page) (Batch, error) {
	select {
	case r.queries <- struct{}{}:
		defer func() { <-r.queries }()
	default:
		return Batch{}, ErrBusy
	}
	if p.Limit <= 0 || p.Limit > 2000 {
		p.Limit = 200
	}
	if p.ScanBytes <= 0 || p.ScanBytes > 64<<20 {
		p.ScanBytes = 16 << 20
	}
	// Capture the high water before flushing. New producers may continue throughout.
	r.mu.Lock()
	memory := append([]entry(nil), r.ring...)
	status := r.status
	watermark := p.Watermark
	if watermark == "" && len(memory) > 0 {
		watermark = memory[len(memory)-1].event.ID
	}
	r.mu.Unlock()
	if err := r.Sync(ctx); err != nil {
		return Batch{}, err
	}
	r.diskMu.RLock()
	blocks := append([]diskBlock(nil), r.blocks...)
	r.diskMu.RUnlock()
	result := Batch{Events: make([]Event, 0), Status: status, Watermark: watermark, InstanceID: r.boot, Cursor: p.After}
	forward := p.After != ""
	oldest := ""
	if len(blocks) > 0 {
		oldest = blocks[0].First
	}
	if len(memory) > 0 && (oldest == "" || memory[0].event.ID < oldest) {
		oldest = memory[0].event.ID
	}
	result.Gap = p.After != "" && oldest != "" && p.After < oldest
	found := map[string]Event{}
	eligible := func(e Event) bool {
		return e.ID <= watermark && (p.After == "" || e.ID > p.After) && (p.Before == "" || e.ID < p.Before)
	}
	add := func(e Event) {
		if eligible(e) && f.Match(e) {
			found[e.ID] = e
		}
	}
	for _, x := range memory {
		add(x.event)
	}
	boundary := ""
	stopped := false
	countThrough := func() int {
		n := 0
		for id := range found {
			if !forward || boundary != "" && id <= boundary {
				n++
			}
		}
		return n
	}
	visit := func(block diskBlock) error {
		if block.First > watermark || p.After != "" && block.Last <= p.After || p.Before != "" && block.First >= p.Before || !f.From.IsZero() && block.MaxTime.Before(f.From) || !f.To.IsZero() && block.MinTime.After(f.To) {
			return nil
		}
		if result.ScannedBytes > 0 && result.ScannedBytes+block.Bytes > p.ScanBytes {
			result.Limited = true
			stopped = true
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		file, err := os.Open(block.Path)
		if errors.Is(err, os.ErrNotExist) {
			result.Gap = true
			return nil
		}
		if err != nil {
			return err
		}
		defer file.Close()
		scanner := bufio.NewScanner(io.NewSectionReader(file, block.Offset, block.Bytes))
		scanner.Buffer(make([]byte, 4096), r.opts.MaxEventBytes+1024)
		for scanner.Scan() {
			var e Event
			if json.Unmarshal(scanner.Bytes(), &e) == nil {
				add(e)
			}
		}
		if err := scanner.Err(); err != nil {
			return err
		}
		result.ScannedBytes += block.Bytes
		if forward {
			boundary = block.Last
		} else {
			boundary = block.First
		}
		if countThrough() > p.Limit {
			stopped = true
		}
		return nil
	}
	if !forward && len(found) > p.Limit {
		stopped = true
	}
	if !stopped {
		for i := 0; i < len(blocks); i++ {
			index := i
			if !forward {
				index = len(blocks) - 1 - i
			}
			if err := visit(blocks[index]); err != nil {
				return result, err
			}
			if stopped {
				break
			}
		}
	}
	ids := make([]string, 0, len(found))
	for id := range found {
		// Forward pages must merge memory with the disk range actually scanned,
		// including records lost by the persistence queue inside that range.
		if !forward || !stopped || boundary != "" && id <= boundary {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	result.HasMore = stopped
	if len(ids) > p.Limit {
		result.HasMore = true
		if forward {
			ids = ids[:p.Limit]
		} else {
			ids = ids[len(ids)-p.Limit:]
		}
	}
	for _, id := range ids {
		result.Events = append(result.Events, found[id])
	}
	if result.HasMore {
		result.Next = boundary
		if len(ids) > 0 {
			if forward {
				result.Next = ids[len(ids)-1]
			} else {
				result.Next = ids[0]
			}
		}
		if result.Next == "" {
			result.HasMore = false
		}
	}
	result.Cursor = watermark
	if forward && result.HasMore {
		result.Cursor = result.Next
	}
	data, _ := json.Marshal(result)
	var clone Batch
	_ = json.Unmarshal(data, &clone)
	return clone, nil
}

// Export traverses a stable snapshot and includes a final manifest even when a
// limit or retention gap is reached. Readers can distinguish complete exports.
func (r *Recorder) Export(ctx context.Context, w io.Writer, f Filter, maxEvents int, maxBytes int64) (int, bool, error) {
	if maxEvents <= 0 || maxEvents > 100000 {
		maxEvents = 100000
	}
	if maxBytes <= 0 || maxBytes > 64<<20 {
		maxBytes = 64 << 20
	}
	cursor := "!"
	watermark := ""
	count := 0
	var bytes int64
	complete := true
	for {
		b, err := r.Query(ctx, f, Page{After: cursor, Watermark: watermark, Limit: 500})
		if err != nil {
			return count, false, err
		}
		watermark = b.Watermark
		// The synthetic first cursor is not an actual missing historical cursor.
		if cursor != "!" && b.Gap || b.Status.DiskDropped > 0 {
			complete = false
		}
		for _, e := range b.Events {
			data, _ := json.Marshal(e)
			if count >= maxEvents || bytes+int64(len(data)+1) > maxBytes {
				complete = false
				break
			}
			if _, err := w.Write(append(data, '\n')); err != nil {
				return count, false, err
			}
			bytes += int64(len(data) + 1)
			count++
		}
		if !complete || !b.HasMore {
			break
		}
		if b.Next == cursor {
			return count, false, errors.New("日志分页游标未前进")
		}
		cursor = b.Next
	}
	manifest := map[string]any{"type": "export_manifest", "events": count, "complete": complete, "watermark": watermark, "maxEvents": maxEvents, "maxBytes": maxBytes, "redacted": true, "exportedAt": time.Now().UTC()}
	data, _ := json.Marshal(manifest)
	_, err := w.Write(append(data, '\n'))
	return count, complete, err
}
func Instance(id string) string {
	if i := strings.LastIndex(id, "-"); i > 0 {
		return id[:i]
	}
	return ""
}
