package global

import (
	"context"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/configs"
)

var configSnapshot atomic.Pointer[configs.Server]

type configContextKey struct{}

// Config returns a read-only snapshot. Use Clone before making changes.
// NAV_CONFIG is retained as a legacy startup input; runtime readers use Config.
func Config() *configs.Server {
	if snapshot := configSnapshot.Load(); snapshot != nil {
		return snapshot
	}
	Lock.RLock()
	defer Lock.RUnlock()
	copy := NAV_CONFIG.Clone()
	return &copy
}

// SetConfig atomically publishes an independent copy of the configuration.
func SetConfig(next configs.Server) {
	copy := next.Clone()
	configSnapshot.Store(&copy)
}

func ContextWithConfig(ctx context.Context, snapshot *configs.Server) context.Context {
	return context.WithValue(ctx, configContextKey{}, snapshot)
}

func RequestConfig(c *gin.Context) *configs.Server {
	if snapshot, ok := c.Request.Context().Value(configContextKey{}).(*configs.Server); ok {
		return snapshot
	}
	return Config()
}

// RegisterExtra publishes a new map; registered values must be treated as immutable.
func RegisterExtra(key string, value any) {
	for {
		previous := configSnapshot.Load()
		var next configs.Server
		if previous == nil {
			next = Config().Clone()
		} else {
			next = previous.Clone()
		}
		next.RegisterExtra(key, value)
		if configSnapshot.CompareAndSwap(previous, &next) {
			return
		}
	}
}
