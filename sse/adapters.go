package sse

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type eventContextKey struct{}

func Context(ctx context.Context, e Event) context.Context {
	return context.WithValue(ctx, eventContextKey{}, e)
}
func FromContext(ctx context.Context) Event {
	if e, ok := ctx.Value(eventContextKey{}).(Event); ok {
		return e
	}
	return Event{}
}
func (r *Recorder) HTTPMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Log queries must not generate their own traffic or database traces.
		if len(c.FullPath()) >= 5 && containsLogRoute(c.FullPath()) {
			c.Request = c.Request.WithContext(Quiet(c.Request.Context()))
			c.Next()
			return
		}
		started := time.Now()
		e := Event{RequestID: uuid.NewString(), TraceID: uuid.NewString(), Source: "http", Stage: "request.completed"}
		c.Header("X-Request-ID", e.RequestID)
		c.Request = c.Request.WithContext(Context(c.Request.Context(), e))
		defer func() {
			if v := recover(); v != nil {
				e.Level = "error"
				e.Stage = "request.panic"
				e.Message = "HTTP 处理发生异常"
				e.DurationMS = float64(time.Since(started).Microseconds()) / 1000
				e.Fields = map[string]any{"method": c.Request.Method, "route": c.FullPath(), "panicType": fmt.Sprintf("%T", v)}
				r.Record(e)
				panic(v)
			}
		}()
		c.Next()
		elapsed := time.Since(started)
		status := c.Writer.Status()
		if status < 400 && len(c.Errors) == 0 {
			if elapsed < time.Second {
				return
			}
			// SSE duration measures the connection lifetime, not request latency.
			mediaType, _, err := mime.ParseMediaType(c.Writer.Header().Get("Content-Type"))
			if err == nil && mediaType == "text/event-stream" {
				return
			}
		}
		e.Level = "warn"
		if status >= 500 || len(c.Errors) > 0 {
			e.Level = "error"
		}
		e.Message = "HTTP 请求失败或耗时超过阈值"
		e.DurationMS = float64(elapsed.Microseconds()) / 1000
		e.Fields = map[string]any{"method": c.Request.Method, "route": c.FullPath(), "status": status, "errorCount": len(c.Errors)}
		r.Record(e)
	}
}
func containsLogRoute(path string) bool {
	for i := 0; i+6 <= len(path); i++ {
		if path[i:i+6] == "/logs/" {
			return true
		}
	}
	return false
}

type gormLogger struct {
	r     *Recorder
	level logger.LogLevel
}

func (r *Recorder) GormLogger() logger.Interface                    { return gormLogger{r, logger.Warn} }
func (l gormLogger) LogMode(level logger.LogLevel) logger.Interface { l.level = level; return l }
func (l gormLogger) Info(context.Context, string, ...interface{})   {}
func (l gormLogger) Warn(ctx context.Context, _ string, _ ...interface{}) {
	if l.level >= logger.Warn {
		e := FromContext(ctx)
		e.Source = "storage"
		e.Level = "warn"
		e.Message = "数据库警告"
		l.r.Record(e)
	}
}
func (l gormLogger) Error(ctx context.Context, _ string, _ ...interface{}) {
	if l.level >= logger.Error {
		e := FromContext(ctx)
		e.Source = "storage"
		e.Level = "error"
		e.Message = "数据库运行错误"
		l.r.Record(e)
	}
}
func (l gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level == logger.Silent || errors.Is(err, gorm.ErrRecordNotFound) || containsLogRouteContext(ctx) {
		return
	}
	elapsed := time.Since(begin)
	if err == nil && elapsed < 200*time.Millisecond {
		return
	}
	// Intentionally never invoke fc: it interpolates SQL parameters.
	e := FromContext(ctx)
	e.Source = "storage"
	e.Stage = "query.slow"
	e.Level = "warn"
	e.Message = "数据库查询耗时超过 200 ms"
	e.DurationMS = float64(elapsed.Microseconds()) / 1000
	if err != nil {
		e.Stage = "query.failed"
		e.Level = "error"
		e.Message = "数据库操作失败"
		e.Fields = map[string]any{"errorType": fmt.Sprintf("%T", err)}
	}
	l.r.Record(e)
}

type quietContextKey struct{}

func Quiet(ctx context.Context) context.Context {
	return context.WithValue(ctx, quietContextKey{}, true)
}
func containsLogRouteContext(ctx context.Context) bool {
	v, _ := ctx.Value(quietContextKey{}).(bool)
	return v
}
