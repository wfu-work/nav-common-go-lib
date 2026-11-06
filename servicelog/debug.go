package servicelog

import (
	"errors"
	"github.com/google/uuid"
	"time"
)

type DebugSession struct {
	ID        string    `json:"id"`
	DeviceID  string    `json:"deviceId,omitempty"`
	ClientID  string    `json:"clientId,omitempty"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func (r *Recorder) StartDebug(deviceID, clientID string) (DebugSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.debugEnabled("", "", time.Now())
	if deviceID == "" && clientID == "" {
		return DebugSession{}, errors.New("必须指定设备或 clientId")
	}
	if len(r.debug) >= 20 {
		return DebugSession{}, errors.New("临时调试会话已达到上限")
	}
	s := DebugSession{uuid.NewString(), deviceID, clientID, time.Now().Add(10 * time.Minute)}
	r.debug[s.ID] = s
	return s, nil
}
func (r *Recorder) StopDebug(id string) { r.mu.Lock(); defer r.mu.Unlock(); delete(r.debug, id) }
func (r *Recorder) DebugSessions() []DebugSession {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.debugEnabled("", "", time.Now())
	s := make([]DebugSession, 0, len(r.debug))
	for _, x := range r.debug {
		s = append(s, x)
	}
	return s
}
func (r *Recorder) DebugEnabled(deviceID, clientID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.debugEnabled(deviceID, clientID, time.Now())
}
func (r *Recorder) debugEnabled(deviceID, clientID string, now time.Time) bool {
	enabled := false
	for id, s := range r.debug {
		if !now.Before(s.ExpiresAt) {
			delete(r.debug, id)
			continue
		}
		if (s.DeviceID == "" || s.DeviceID == deviceID) && (s.ClientID == "" || s.ClientID == clientID) {
			enabled = true
		}
	}
	return enabled
}
