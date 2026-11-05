package middlewares

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type auditEvent struct {
	operation *domains.SysLog
	login     *domains.LoginLog
}
type auditWriter struct {
	db     *gorm.DB
	logger *zap.Logger
	queue  chan auditEvent
	done   chan struct{}
	mu     sync.RWMutex
	closed bool
}

var activeAudit atomic.Pointer[auditWriter]

// StartAuditWriter starts a bounded queue. A full queue falls back to a
// synchronous write instead of silently discarding audit records.
func StartAuditWriter(db *gorm.DB, logger *zap.Logger, capacity int) func() {
	if db == nil {
		return func() {}
	}
	if capacity <= 0 {
		capacity = 1024
	}
	writer := &auditWriter{db: db, logger: logger, queue: make(chan auditEvent, capacity), done: make(chan struct{})}
	activeAudit.Store(writer)
	go func() {
		defer close(writer.done)
		for event := range writer.queue {
			writer.write(event)
		}
	}()
	return func() {
		writer.mu.Lock()
		if !writer.closed {
			writer.closed = true
			close(writer.queue)
		}
		writer.mu.Unlock()
		<-writer.done
		activeAudit.CompareAndSwap(writer, nil)
	}
}

func (w *auditWriter) write(event auditEvent) {
	if w.db == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var value any = event.operation
	if event.login != nil {
		value = event.login
	}
	if err := w.db.WithContext(ctx).Create(value).Error; err != nil && w.logger != nil {
		w.logger.Error("write audit log", zap.Error(err))
	}
}

func recordAudit(event auditEvent) {
	if writer := activeAudit.Load(); writer != nil {
		writer.mu.RLock()
		if !writer.closed {
			select {
			case writer.queue <- event:
				writer.mu.RUnlock()
				return
			default:
				// Keep the read lock until the fallback finishes so Close drains it too.
				writer.write(event)
				writer.mu.RUnlock()
				return
			}
		}
		writer.mu.RUnlock()
		return
	}
	(&auditWriter{db: global.NAV_DB, logger: global.NAV_LOG}).write(event)
}
