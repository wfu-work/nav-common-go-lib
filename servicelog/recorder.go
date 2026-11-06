// Package servicelog records bounded, structured diagnostic events independently
// of browser connections. A Recorder owns its worker and must be closed.
package servicelog

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID           string         `json:"id"`
	Time         time.Time      `json:"time"`
	Level        string         `json:"level"`
	Source       string         `json:"source"`
	Stage        string         `json:"stage,omitempty"`
	Message      string         `json:"message"`
	TraceID      string         `json:"traceId,omitempty"`
	AttemptID    string         `json:"attemptId,omitempty"`
	ConnectionID string         `json:"connectionId,omitempty"`
	DeviceID     string         `json:"deviceId,omitempty"`
	ProjectID    string         `json:"projectId,omitempty"`
	ClientID     string         `json:"clientId,omitempty"`
	Topic        string         `json:"topic,omitempty"`
	DurationMS   float64        `json:"durationMs,omitempty"`
	Fields       map[string]any `json:"fields,omitempty"`
	Truncated    bool           `json:"truncated,omitempty"`
}

type Options struct {
	Console                                             io.Writer
	Directory                                           string
	MaxEvents, MaxMemoryBytes, MaxEventBytes, QueueSize int
	MaxFileBytes, MaxDiskBytes                          int64
	Retention                                           time.Duration
}

func (o *Options) defaults() {
	if o.MaxEvents <= 0 {
		o.MaxEvents = 10000
	}
	if o.MaxMemoryBytes <= 0 {
		o.MaxMemoryBytes = 16 << 20
	}
	if o.MaxEventBytes < 1024 {
		o.MaxEventBytes = 16 << 10
	}
	if o.QueueSize <= 0 {
		o.QueueSize = 4096
	}
	if o.MaxFileBytes <= 0 {
		o.MaxFileBytes = 20 << 20
	}
	if o.MaxDiskBytes <= 0 {
		o.MaxDiskBytes = 512 << 20
	}
	if o.MaxFileBytes > o.MaxDiskBytes {
		o.MaxFileBytes = o.MaxDiskBytes
	}
	if o.Retention <= 0 {
		o.Retention = 7 * 24 * time.Hour
	}
}

type Filter struct {
	Source    string    `json:"source"`
	Level     string    `json:"level"`
	DeviceID  string    `json:"deviceId"`
	ProjectID string    `json:"projectId"`
	ClientID  string    `json:"clientId"`
	Topic     string    `json:"topic"`
	TraceID   string    `json:"traceId"`
	Keyword   string    `json:"keyword"`
	From      time.Time `json:"from"`
	To        time.Time `json:"to"`
}

func (f Filter) Match(e Event) bool {
	if f.Source != "" && f.Source != e.Source || f.Level != "" && f.Level != e.Level || f.DeviceID != "" && f.DeviceID != e.DeviceID || f.ProjectID != "" && f.ProjectID != e.ProjectID || f.ClientID != "" && f.ClientID != e.ClientID || f.TraceID != "" && f.TraceID != e.TraceID {
		return false
	}
	if f.Topic != "" && !strings.Contains(e.Topic, f.Topic) || !f.From.IsZero() && e.Time.Before(f.From) || !f.To.IsZero() && e.Time.After(f.To) {
		return false
	}
	if f.Keyword != "" {
		b, _ := json.Marshal(e)
		return strings.Contains(strings.ToLower(string(b)), strings.ToLower(f.Keyword))
	}
	return true
}

type entry struct {
	event Event
	data  []byte
}
type Status struct {
	DiskDropped uint64 `json:"diskDropped"`
	DiskError   string `json:"diskError,omitempty"`
	Closed      bool   `json:"closed"`
}
type Batch struct {
	Events []Event `json:"events"`
	Cursor string  `json:"cursor"`
	Gap    bool    `json:"gap"`
	Status Status  `json:"status"`
}
type Recorder struct {
	mu        sync.Mutex
	opts      Options
	boot      string
	seq       uint64
	ring      []entry
	bytes     int
	wake      chan struct{}
	queue     chan entry
	done      chan struct{}
	status    Status
	closeOnce sync.Once
	debug     map[string]DebugSession
}

func New(opts Options) (*Recorder, error) {
	opts.defaults()
	r := &Recorder{opts: opts, boot: time.Now().UTC().Format("20060102T150405.000000000") + "-" + uuid.NewString(), wake: make(chan struct{}), queue: make(chan entry, opts.QueueSize), done: make(chan struct{}), debug: make(map[string]DebugSession)}
	if err := r.prepareDisk(); err != nil {
		return nil, err
	}
	// Replay retained history before accepting new producers.
	if err := r.readDisk(func(x entry) { r.appendRing(x) }); err != nil {
		return nil, err
	}
	go r.persist()
	return r, nil
}
func (r *Recorder) appendRing(x entry) {
	r.ring = append(r.ring, x)
	r.bytes += len(x.data)
	for len(r.ring) > 0 && (len(r.ring) > r.opts.MaxEvents || r.bytes > r.opts.MaxMemoryBytes) {
		r.bytes -= len(r.ring[0].data)
		r.ring[0] = entry{}
		r.ring = r.ring[1:]
	}
}
func (r *Recorder) Record(e Event) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.status.Closed {
		return ""
	}
	if e.Level == "debug" && !r.debugEnabled(e.DeviceID, e.ClientID, time.Now()) {
		return ""
	}
	r.seq++
	e.ID = fmt.Sprintf("%s-%020d", r.boot, r.seq)
	if e.Time.IsZero() {
		e.Time = time.Now().UTC()
	}
	switch strings.ToLower(e.Level) {
	case "debug", "info", "warn", "error":
		e.Level = strings.ToLower(e.Level)
	case "fatal", "panic", "dpanic":
		e.Level = "error"
	default:
		e.Level = "info"
	}
	if e.Source == "" {
		e.Source = "application"
	}
	// Roundtrip copies nested caller-owned fields before they can be mutated.
	data, err := json.Marshal(e)
	if err != nil {
		e.Fields = map[string]any{"encodingError": "unsupported log field"}
		data, _ = json.Marshal(e)
	}
	var clean any
	_ = json.Unmarshal(data, &clean)
	data, _ = json.Marshal(sanitize(clean, ""))
	var normalized Event
	_ = json.Unmarshal(data, &normalized)
	e = normalized
	if len(data) > r.opts.MaxEventBytes {
		e.Truncated = true
		e.Fields = map[string]any{"originalBytes": len(data)}
		e.Message = clip(e.Message, 2048)
		e.Topic = clip(e.Topic, 512)
		e.DeviceID = clip(e.DeviceID, 256)
		e.ClientID = clip(e.ClientID, 256)
		data, _ = json.Marshal(e)
	}
	if len(data) > r.opts.MaxEventBytes {
		e = Event{ID: e.ID, Time: e.Time, Source: "application", Level: e.Level, Message: "日志内容超过大小限制", Truncated: true}
		data, _ = json.Marshal(e)
	}
	x := entry{e, data}
	r.appendRing(x)
	select {
	case r.queue <- x:
	default:
		r.status.DiskDropped++
	}
	close(r.wake)
	r.wake = make(chan struct{})
	return e.ID
}
func (r *Recorder) Snapshot(f Filter, cursor string, limit int) Batch {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.snapshot(f, cursor, limit)
}
func (r *Recorder) snapshot(f Filter, cursor string, limit int) Batch {
	if limit <= 0 || limit > 2000 {
		limit = 200
	}
	b := Batch{Events: make([]Event, 0), Cursor: cursor, Status: r.status}
	if len(r.ring) == 0 {
		return b
	}
	if cursor != "" && (cursor < r.ring[0].event.ID || cursor > r.ring[len(r.ring)-1].event.ID) {
		b.Gap = true
		cursor = ""
	}
	// Advance over filtered events as well, preventing repeated empty scans.
	b.Cursor = r.ring[len(r.ring)-1].event.ID
	for _, x := range r.ring {
		if (cursor == "" || x.event.ID > cursor) && f.Match(x.event) {
			b.Events = append(b.Events, x.event)
		}
	}
	if len(b.Events) > limit {
		if cursor != "" {
			b.Cursor = b.Events[limit-1].ID
			b.Events = b.Events[:limit]
		} else {
			b.Events = b.Events[len(b.Events)-limit:]
		}
	}
	// Return deep copies so UI/API consumers cannot mutate the ring.
	data, _ := json.Marshal(b)
	var copy Batch
	_ = json.Unmarshal(data, &copy)
	return copy
}

// Watch atomically snapshots and obtains the next wakeup. Slow readers replay
// from the ring and receive Gap when their cursor falls outside its retention.
func (r *Recorder) Watch(f Filter, cursor string, limit int) (Batch, <-chan struct{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.snapshot(f, cursor, limit), r.wake
}
func (r *Recorder) History(f Filter, limit int) (Batch, error) {
	if limit <= 0 || limit > 2000 {
		limit = 200
	}
	r.mu.Lock()
	memory := append([]entry(nil), r.ring...)
	status := r.status
	r.mu.Unlock()
	found := map[string]Event{}
	add := func(x entry) {
		if f.Match(x.event) {
			found[x.event.ID] = x.event
		}
		if len(found) > limit*2 {
			trimEvents(found, limit)
		}
	}
	if err := r.readDisk(add); err != nil {
		return Batch{}, err
	}
	for _, x := range memory {
		add(x)
	}
	ids := trimEvents(found, limit)
	events := make([]Event, 0, len(ids))
	for _, id := range ids {
		events = append(events, found[id])
	}
	data, _ := json.Marshal(events)
	var cloned []Event
	_ = json.Unmarshal(data, &cloned)
	b := Batch{Events: cloned, Status: status}
	if len(memory) > 0 {
		b.Cursor = memory[len(memory)-1].event.ID
	}
	return b, nil
}
func trimEvents(m map[string]Event, n int) []string {
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if len(ids) > n {
		for _, id := range ids[:len(ids)-n] {
			delete(m, id)
		}
		ids = ids[len(ids)-n:]
	}
	return ids
}
func (r *Recorder) Close() error {
	r.closeOnce.Do(func() {
		r.mu.Lock()
		r.status.Closed = true
		close(r.wake)
		close(r.queue)
		r.mu.Unlock()
		<-r.done
	})
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.status.DiskError != "" {
		return fmt.Errorf("log persistence: %s", r.status.DiskError)
	}
	return nil
}
func (r *Recorder) diskError(err error) {
	if err != nil {
		r.mu.Lock()
		r.status.DiskError = Redact(err.Error())
		r.mu.Unlock()
	}
}

// Writer adapts standard log output without redirecting process descriptors.
func (r *Recorder) Writer(source string) io.Writer { return writer{r, source} }

type writer struct {
	r      *Recorder
	source string
}

func (w writer) Write(p []byte) (int, error) {
	for _, line := range strings.Split(strings.TrimSpace(string(p)), "\n") {
		if line != "" {
			w.r.Record(Event{Source: w.source, Message: line})
		}
	}
	return len(p), nil
}
