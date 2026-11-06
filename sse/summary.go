package sse

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"time"
)

type faultTick struct {
	Second           int64
	Warnings, Errors int
}

type faultGroup struct {
	First, Last       time.Time
	Count, Suppressed uint64
	Example           Event
}

// suppress bounds repeated failures while preserving first/last timestamps and
// exact occurrence counts in summaries. One representative is persisted / 30s.
func (r *Recorder) suppress(e *Event) bool {
	key := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s|%s", e.Level, e.Source, e.Stage, e.CompanyID, e.ProjectID, e.DeviceID, e.TaskID, e.ClientID, e.Topic, e.Message))))
	now := time.Now()
	tick := &r.faultTicks[now.Unix()%3600]
	if tick.Second != now.Unix() {
		*tick = faultTick{Second: now.Unix()}
	}
	if e.Level == "error" {
		tick.Errors++
	} else {
		tick.Warnings++
	}
	g := r.faults[key]
	g.Count++
	g.Last = now
	if g.First.IsZero() {
		g.First = now
	}
	suppressed := g.Count > 1 && !g.Example.Time.IsZero() && now.Sub(g.Example.Time) < 30*time.Second
	if suppressed {
		g.Suppressed++
		r.status.Suppressed++
	} else {
		e.Fields = cloneFields(e.Fields)
		e.Fields["occurrences"] = g.Count
		e.Fields["firstOccurrence"] = g.First
		e.Fields["lastOccurrence"] = g.Last
		e.Fields["suppressed"] = g.Suppressed
		g.Example = Event{Source: clip(e.Source, 256), Stage: clip(e.Stage, 256), Message: clip(Redact(e.Message), 2048), DeviceID: clip(Redact(e.DeviceID), 256), TraceID: e.TraceID, Level: e.Level, Time: now}
	}
	if _, ok := r.faults[key]; !ok {
		r.faultOrder = append(r.faultOrder, key)
	}
	r.faults[key] = g
	if len(r.faultOrder) > 1024 {
		delete(r.faults, r.faultOrder[0])
		r.faultOrder = r.faultOrder[1:]
	}
	return suppressed
}
func cloneFields(fields map[string]any) map[string]any {
	out := make(map[string]any, len(fields)+4)
	for k, v := range fields {
		out[k] = v
	}
	return out
}

type Fault struct {
	Source   string    `json:"source"`
	Stage    string    `json:"stage"`
	Message  string    `json:"message"`
	DeviceID string    `json:"deviceId,omitempty"`
	TraceID  string    `json:"traceId,omitempty"`
	Count    uint64    `json:"count"`
	First    time.Time `json:"first"`
	Last     time.Time `json:"last"`
}
type Summary struct {
	InstanceID string    `json:"instanceId"`
	Cursor     string    `json:"cursor"`
	Warnings   int       `json:"warnings"`
	Errors     int       `json:"errors"`
	Faults     []Fault   `json:"faults"`
	Status     Status    `json:"status"`
	CheckedAt  time.Time `json:"checkedAt"`
	Limited    bool      `json:"limited"`
	From       time.Time `json:"from"`
	Scope      string    `json:"scope"`
}

func (r *Recorder) Summary(ctx context.Context, from time.Time) (Summary, error) {
	s := Summary{InstanceID: r.boot, Faults: make([]Fault, 0), CheckedAt: time.Now().UTC()}
	// Summary is deliberately cheap: recent memory + bounded in-process counters;
	// historical searches and exports use the indexed Query API.
	r.mu.Lock()
	defer r.mu.Unlock()
	s.Status = r.status
	s.From = from
	s.Scope = "本次运行，最近最多一小时，按秒聚合；异常组次数为本次运行累计"
	if s.From.Before(r.started) {
		s.From = r.started
		s.Limited = true
	}
	if earliest := time.Now().Add(-3599 * time.Second); s.From.Before(earliest) {
		s.From = earliest
		s.Limited = true
	}
	for _, tick := range r.faultTicks {
		if tick.Second >= s.From.Unix() {
			s.Warnings += tick.Warnings
			s.Errors += tick.Errors
		}
	}
	if len(r.ring) > 0 {
		s.Cursor = r.ring[len(r.ring)-1].event.ID
	}
	for _, g := range r.faults {
		if g.Last.Before(from) {
			continue
		}
		e := g.Example
		s.Faults = append(s.Faults, Fault{Source: Redact(e.Source), Stage: Redact(e.Stage), Message: Redact(e.Message), DeviceID: Redact(e.DeviceID), TraceID: e.TraceID, Count: g.Count, First: g.First, Last: g.Last})
	}
	sort.Slice(s.Faults, func(i, j int) bool { return s.Faults[i].Last.After(s.Faults[j].Last) })
	if len(s.Faults) > 20 {
		s.Faults = s.Faults[:20]
		s.Limited = true
	}
	return s, nil
}
