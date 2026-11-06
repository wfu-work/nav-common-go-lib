package sse

import (
	"encoding/json"
	"regexp"
	"strings"
	"unicode/utf8"
)

var sensitive = regexp.MustCompile(`(?i)(password|passwd|secret|token|authorization|device[_-]?key|api[_-]?key|credential|cookie)`)
var credentials = regexp.MustCompile(`(?i)([a-z][a-z0-9+.-]*://)[^\s/@]+(?::[^\s/@]*)?@`)
var bearer = regexp.MustCompile(`(?i)bearer\s+[^\s,"}]+`)
var assignments = regexp.MustCompile(`(?i)((?:password|passwd|secret|token|authorization|device[_-]?key|api[_-]?key)["']?\s*[=:]\s*)(?:"[^"\r\n]*"|'[^'\r\n]*'|[^\s,;&}\]]+)`)

func Redact(s string) string {
	// Embedded JSON (including error messages) is sanitized structurally too.
	var value any
	if json.Unmarshal([]byte(s), &value) == nil {
		if _, ok := value.(string); !ok {
			b, _ := json.Marshal(sanitize(value, ""))
			return string(b)
		}
	}
	s = credentials.ReplaceAllString(s, "${1}[REDACTED]@")
	s = bearer.ReplaceAllString(s, "Bearer [REDACTED]")
	return assignments.ReplaceAllString(s, "${1}[REDACTED]")
}
func sanitize(v any, key string) any {
	if sensitive.MatchString(key) {
		return "[REDACTED]"
	}
	switch x := v.(type) {
	case map[string]any:
		for k, y := range x {
			x[k] = sanitize(y, k)
		}
		return x
	case []any:
		for i, y := range x {
			x[i] = sanitize(y, "")
		}
		return x
	case string:
		return Redact(x)
	default:
		return v
	}
}
func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	s = s[:n]
	for !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s + "…"
}
func Preview(payload []byte) string {
	if len(payload) > 64<<10 {
		return "[载荷超过 64 KiB，仅记录字节数]"
	}
	s := Redact(string(payload))
	if !json.Valid(payload) {
		return "[非 JSON 数据，仅记录字节数]"
	}
	return clip(strings.TrimSpace(s), 1024)
}
