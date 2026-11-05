package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/vos"
)

var cacheFileMu sync.Mutex
var rtLoggingMu sync.Mutex

func InitCache() { InitCacheContext(context.Background()) }

// InitCacheContext returns an idempotent stop function that drains persistence.
func InitCacheContext(parent context.Context) func() {
	ctx, cancel := context.WithCancel(parent)
	c := cache.New(5*time.Minute, 0)
	global.LOCAL_CACHE = c
	path := global.Config().Local.CachePath
	loadCache(c, path)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.DeleteExpired()
				if err := saveItems[*vos.RtLoggingVo](c, path, "rtlogging_"); err != nil {
					fmt.Printf("persist cache: %v\n", err)
				}
			}
		}
	}()
	return func() { cancel(); <-done }
}

func loadCache(c *cache.Cache, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var items map[string]json.RawMessage
	if json.Unmarshal(data, &items) != nil {
		return
	}
	for key, raw := range items {
		if strings.HasPrefix(key, "rtlogging_") {
			var value vos.RtLoggingVo
			if json.Unmarshal(raw, &value) == nil {
				c.Set(key, &value, cache.NoExpiration)
			}
		}
	}
}

func SaveItems[T any](prefix string) error {
	return saveItems[T](global.LOCAL_CACHE, global.Config().Local.CachePath, prefix)
}

func saveItems[T any](c *cache.Cache, path, prefix string) error {
	if c == nil || path == "" {
		return nil
	}
	cacheFileMu.Lock()
	defer cacheFileMu.Unlock()
	existing := make(map[string]any)
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &existing)
	}
	for key := range existing {
		if strings.HasPrefix(key, prefix) {
			delete(existing, key)
		}
	}
	for key, value := range c.Items() {
		if strings.HasPrefix(key, prefix) {
			if object, ok := value.Object.(T); ok {
				existing[key] = object
			}
		}
	}
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".nav-cache-*")
	if err != nil {
		return err
	}
	defer file.Close()
	defer os.Remove(file.Name())
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func SaveRtLogging() error {
	return SaveItems[*vos.RtLoggingVo]("rtlogging_")
}

func PersistCache() {
	_ = SaveRtLogging()
}

func Set(key string, value interface{}) {
	global.LOCAL_CACHE.Set(key, value, cache.NoExpiration)
}

func Get[T any](key string) (T, bool) {
	var zero T
	v, ok := global.LOCAL_CACHE.Get(key)
	if !ok {
		return zero, false
	}
	obj, ok := v.(T)
	return obj, ok
}

func SetRtlogging(sncode string, size int64) {
	rtLoggingMu.Lock()
	defer rtLoggingMu.Unlock()
	v, _ := global.LOCAL_CACHE.Get("rtlogging_" + sncode)
	if v != nil {
		if rtVo, ok := v.(*vos.RtLoggingVo); ok {
			size += InterfaceToInt(rtVo.Size)
		}
	}
	rtloggingVo := &vos.RtLoggingVo{
		Sncode:   sncode,
		Size:     size,
		LastTime: time.Now().UnixMilli(),
	}
	global.LOCAL_CACHE.Set("rtlogging_"+sncode, rtloggingVo, cache.NoExpiration)
}

func GetRtlogging(sncode string) *vos.RtLoggingVo {
	if rt, ok := Get[*vos.RtLoggingVo]("rtlogging_" + sncode); ok {
		return rt
	}
	return nil
}

func InterfaceToInt(v interface{}) (i int64) {
	switch v := v.(type) {
	case int:
		i = int64(v)
		break
	case int64:
		i = v
		break
	default:
		i = 0
	}
	return
}
