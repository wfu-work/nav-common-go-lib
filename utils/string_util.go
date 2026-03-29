package utils

import (
	"crypto/rand"
	"encoding/base64"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

func RegexNumber(str string) float64 {
	re := regexp.MustCompile(`[\d.]+`)
	match := re.FindString(str)
	val, err := strconv.ParseFloat(match, 64)
	if err != nil {
		return 0
	}
	return val
}

func JoinField[T any](items []T, get func(T) string, sep string) string {
	var out []string
	for _, item := range items {
		out = append(out, get(item))
	}
	return strings.Join(out, sep)
}

// StringOrDefault 返回 str，如果 str 为空，则返回 def。
func StringOrDefault(str string, def string) string {
	if str != "" {
		return str
	}
	return def
}

// IntOrDefault 返回 val，如果 val 不为 0，则返回 val，否则返回 def。
func IntOrDefault(val int, def int) int {
	if val != 0 {
		return val
	}
	return def
}

// BoolOrDefault 返回 val，如果 val 为 true，则返回 true，否则返回 def。
func BoolOrDefault(val bool, def bool) bool {
	if val {
		return true
	}
	return def
}

// StringToInt64 将字符串转换为 int，如果失败返回默认值 0 和错误
func StringToInt64(s string) int64 {
	s = strings.TrimSpace(s)
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return i
}

// StringToFloat64 将字符串转换为 float64，如果失败返回默认值 0 和错误
func StringToFloat64(s string) (float64, error) {
	s = strings.TrimSpace(s)
	return strconv.ParseFloat(s, 64)
}

// MustStringToFloat64 转换失败时返回 0，不抛出错误（适合容错场景）
func MustStringToFloat64(s string) float64 {
	f, err := StringToFloat64(s)
	if err != nil {
		return 0
	}
	return f
}

// Str2Int 将字符串转换为 int，转换失败时返回 0
func Str2Int(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

// GenerateGuid 生成一个不带连字符的 UUID 字符串
func GenerateGuid() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}

// CamelToSnake 将驼峰命名法的字符串转换为下划线命名法
func CamelToSnake(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && (unicode.IsUpper(r) || unicode.IsDigit(r)) && ((i+1 < len(s) && unicode.IsLower(rune(s[i+1]))) || unicode.IsLower(rune(s[i-1]))) {
			result = append(result, '_')
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}

// GenerateRandCode 生成短而唯一的标识
func GenerateRandCode() string {
	b := make([]byte, 9)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return strings.ToLower(base64.RawURLEncoding.EncodeToString(b))
}
