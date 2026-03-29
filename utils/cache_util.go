package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/vos"
)

func InitCache() {
	c := cache.New(5*time.Minute, 10*time.Minute)
	global.LOCAL_CACHE = c

	load()

	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			PersistCache()
		}
	}()
}

func load() {
	data, err := os.ReadFile(global.NAV_CONFIG.Local.CachePath)
	if err != nil {
		fmt.Printf("❌初始化缓存读取文件异常：%+v\n", err)
		return
	}
	var items map[string]json.RawMessage
	if err := json.Unmarshal(data, &items); err != nil {
		return
	}
	for k, v := range items {
		if strings.HasPrefix(k, "rtlogging_") {
			var rt vos.RtLoggingVo
			objBytes, _ := json.Marshal(v)
			if err := json.Unmarshal(objBytes, &rt); err == nil {
				global.LOCAL_CACHE.Set(k, &rt, cache.NoExpiration)
			}
		} else {
			global.LOCAL_CACHE.Set(k, v, cache.NoExpiration)
		}
	}
}

func SaveItems[T any](prefix string) error {
	items := make(map[string]T)
	for k, v := range global.LOCAL_CACHE.Items() {
		if prefix == "" || (len(k) >= len(prefix) && k[:len(prefix)] == prefix) {
			if obj, ok := v.Object.(T); ok {
				items[k] = obj
			}
		}
	}
	if len(items) == 0 {
		return nil
	}
	existing := make(map[string]interface{})
	if data, err := os.ReadFile(global.NAV_CONFIG.Local.CachePath); err == nil {
		_ = json.Unmarshal(data, &existing)
	}
	for k, v := range items {
		existing[k] = v
	}
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(global.NAV_CONFIG.Local.CachePath, data, 0644)
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
