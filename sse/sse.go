package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Stream replays memory atomically, then follows new events. valid is rechecked
// on every batch/heartbeat; the caller authenticates before writing headers.
func (r *Recorder) Stream(w http.ResponseWriter, req *http.Request, f Filter, cursor string, valid func() bool) {
	if _, ok := w.(http.Flusher); !ok {
		http.Error(w, "streaming unsupported", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("X-Accel-Buffering", "no")
	rc := http.NewResponseController(w)
	send := func(kind, id string, v any) bool {
		_ = rc.SetWriteDeadline(time.Now().Add(10 * time.Second))
		data, _ := json.Marshal(v)
		if id != "" {
			if _, e := fmt.Fprintf(w, "id: %s\n", id); e != nil {
				return false
			}
		}
		if _, e := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", kind, data); e != nil {
			return false
		}
		return rc.Flush() == nil
	}
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		if valid != nil && !valid() {
			send("auth_expired", "", map[string]string{"message": "登录已失效，请重新登录"})
			return
		}
		batch, wake := r.Watch(f, cursor, 200)
		backlog := false
		if cursor != "" && batch.Gap && r.opts.Directory != "" {
			recovered, err := r.Query(req.Context(), f, Page{After: cursor, Limit: 200})
			if err != nil {
				if !send("retry", "", map[string]string{"message": "历史续传正在等待日志落盘"}) {
					return
				}
				select {
				case <-req.Context().Done():
					return
				case <-time.After(time.Second):
					continue
				}
			}
			batch = recovered
			backlog = batch.HasMore
		}
		if batch.Status.Closed {
			return
		}
		if batch.Gap {
			if !send("gap", "", map[string]string{"message": "部分日志已超过磁盘保留范围或读取期间被清理，当前续传结果存在缺口"}) {
				return
			}
		}
		for _, e := range batch.Events {
			if !send("log", e.ID, e) {
				return
			}
		}
		cursor = batch.Cursor
		ready := map[string]any{"status": batch.Status, "diskDropped": batch.Status.DiskDropped, "diskError": batch.Status.DiskError, "closed": batch.Status.Closed, "instanceId": r.boot, "cursor": cursor, "syncedAt": time.Now().UTC(), "replaying": backlog}
		if !send("ready", cursor, ready) {
			return
		}
		if backlog {
			continue
		}
		// A full batch may have more retained events; drain before waiting.
		if len(batch.Events) == 200 && len(batch.Events) > 0 && cursor == batch.Events[len(batch.Events)-1].ID {
			next := r.Snapshot(f, cursor, 1)
			if len(next.Events) > 0 {
				continue
			}
		}
		select {
		case <-req.Context().Done():
			return
		case <-wake:
		case <-heartbeat.C:
			if !send("heartbeat", "", time.Now().UTC()) {
				return
			}
		}
	}
}
