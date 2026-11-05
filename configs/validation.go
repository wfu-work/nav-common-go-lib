package configs

import (
	"errors"
	"fmt"
	"maps"
	"net"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
)

func (s Server) Clone() Server {
	s.DiskList = slices.Clone(s.DiskList)
	s.System.TrustedProxies = slices.Clone(s.System.TrustedProxies)
	s.Redis.ClusterAddrs = slices.Clone(s.Redis.ClusterAddrs)
	s.Cors.Whitelist = slices.Clone(s.Cors.Whitelist)
	s.Extras = maps.Clone(s.Extras)
	return s
}

// ParseDuration supports Go durations and a leading number of days (7d, 1d2h).
func ParseDuration(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if duration, err := time.ParseDuration(value); err == nil {
		return duration, nil
	}
	if days, rest, ok := strings.Cut(value, "d"); ok {
		n, err := strconv.ParseUint(days, 10, 32)
		if err != nil || n > uint64((1<<63-1)/int64(24*time.Hour)) {
			return 0, fmt.Errorf("invalid duration: %q", value)
		}
		base := time.Duration(n) * 24 * time.Hour
		if rest == "" {
			return base, nil
		}
		tail, err := time.ParseDuration(rest)
		if err != nil || tail < 0 || tail > time.Duration(1<<63-1)-base {
			return 0, fmt.Errorf("invalid duration: %q", value)
		}
		return base + tail, nil
	}
	n, err := strconv.ParseInt(value, 10, 64)
	return time.Duration(n), err
}

func (s Server) Validate() error {
	if strings.TrimSpace(s.JWT.SigningKey) == "" {
		return errors.New("jwt.signing-key 未配置")
	}
	expires, err := ParseDuration(s.JWT.ExpiresTime)
	if err != nil || expires <= 0 {
		return errors.New("jwt.expires-time 必须是正时长")
	}
	buffer, err := ParseDuration(s.JWT.BufferTime)
	if err != nil || buffer < 0 || buffer > expires {
		return errors.New("jwt.buffer-time 必须位于 0 与 expires-time 之间")
	}
	if s.System.Addr < 0 || s.System.Addr > 65535 {
		return errors.New("system.addr 无效")
	}
	if s.System.MaxRequestBytes < 0 || s.System.MaxUploadBytes < 0 {
		return errors.New("请求体大小上限不能为负数")
	}
	for _, proxy := range s.System.TrustedProxies {
		if net.ParseIP(proxy) == nil {
			if _, _, err := net.ParseCIDR(proxy); err != nil {
				return fmt.Errorf("invalid trusted proxy: %s", proxy)
			}
		}
	}
	if s.System.UseRedis && ((!s.Redis.UseCluster && s.Redis.Addr == "") || (s.Redis.UseCluster && len(s.Redis.ClusterAddrs) == 0)) {
		return errors.New("启用 Redis 时必须配置对应连接地址")
	}
	switch s.System.DbType {
	case "", "sqlite", "mysql", "pgsql":
	default:
		return errors.New("不支持的数据库类型")
	}
	switch s.Cors.Mode {
	case "", "allow-all", "whitelist", "strict-whitelist":
	default:
		return errors.New("cors.mode 无效")
	}
	switch s.JWT.CookieSameSite {
	case "", "lax", "strict", "none":
	default:
		return errors.New("jwt.cookie-same-site 无效")
	}
	if s.JWT.CookieSameSite == "none" && !s.JWT.CookieSecure {
		return errors.New("SameSite=None 需要 cookie-secure=true")
	}
	return nil
}

// ValidateReload keeps resources and routes consistent until the next restart.
func (s Server) ValidateReload(previous Server) error {
	if err := s.Validate(); err != nil {
		return err
	}
	next, old := s.Clone(), previous.Clone()
	next.JWT.ExpiresTime, next.JWT.BufferTime = "", ""
	old.JWT.ExpiresTime, old.JWT.BufferTime = "", ""
	next.Cors, old.Cors = CORS{}, CORS{}
	next.Captcha, old.Captcha = Captcha{}, Captcha{}
	next.Extras, old.Extras = nil, nil
	if !reflect.DeepEqual(next, old) {
		return errors.New("此配置修改需要重启服务，当前运行配置保持不变")
	}
	return nil
}
