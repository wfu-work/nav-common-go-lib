package servicelog

import (
	"go.uber.org/zap/zapcore"
)

type Core struct {
	recorder *Recorder
	fields   []zapcore.Field
	level    zapcore.Level
}

func (r *Recorder) ZapCore(level zapcore.Level) zapcore.Core { return &Core{recorder: r, level: level} }
func (c *Core) Enabled(l zapcore.Level) bool                 { return l >= c.level }
func (c *Core) With(f []zapcore.Field) zapcore.Core {
	n := *c
	n.fields = append(append([]zapcore.Field(nil), c.fields...), f...)
	return &n
}
func (c *Core) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(e.Level) {
		return ce.AddCore(e, c)
	}
	return ce
}
func (c *Core) Write(e zapcore.Entry, f []zapcore.Field) error {
	enc := zapcore.NewMapObjectEncoder()
	for _, x := range c.fields {
		x.AddTo(enc)
	}
	for _, x := range f {
		x.AddTo(enc)
	}
	if e.Caller.Defined {
		enc.Fields["caller"] = e.Caller.TrimmedPath()
	}
	if e.Stack != "" {
		enc.Fields["stack"] = e.Stack
	}
	source := "application"
	if e.LoggerName != "" {
		source = e.LoggerName
	}
	c.recorder.Record(Event{Time: e.Time, Level: e.Level.String(), Source: source, Message: e.Message, Fields: enc.Fields})
	if e.Level >= zapcore.FatalLevel {
		return c.recorder.Close()
	}
	return nil
}
func (c *Core) Sync() error { return nil }
