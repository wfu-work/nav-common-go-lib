package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Marshal 将任意 Go 值编码为紧凑 JSON 字节切片。
func Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// MarshalString 将任意 Go 值编码为紧凑 JSON 字符串。
func MarshalString(v any) (string, error) {
	b, err := Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// MarshalIndent 以缩进格式输出 JSON，便于调试或日志。
func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(v, prefix, indent)
}

// MustMarshal 是 Marshal 的 panic 版，适合初始化常量数据（慎用）。
func MustMarshal(v any) []byte {
	b, err := Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// MustMarshalString 是 MarshalString 的 panic 版。
func MustMarshalString(v any) string {
	s, err := MarshalString(v)
	if err != nil {
		panic(err)
	}
	return s
}

// ------------------------------
// 解码（JSON -> 对象）
// ------------------------------

type decodeOptions struct {
	disallowUnknown bool // 是否拒绝未知字段
	useNumber       bool // 将数字解码为 json.Number，避免 float64 精度丢失
	allowTrailing   bool // 是否允许多余数据（多个 JSON 值拼接）；默认 false
}

// Option 用于配置解码行为。
type Option func(*decodeOptions)

// WithDisallowUnknownFields 启用严格模式：输入 JSON 含未知字段时返回错误。
func WithDisallowUnknownFields() Option {
	return func(o *decodeOptions) { o.disallowUnknown = true }
}

// WithUseNumber 在解码时使用 json.Number 以保留原始数字字符串，便于后续精度处理。
func WithUseNumber() Option {
	return func(o *decodeOptions) { o.useNumber = true }
}

// WithAllowTrailingData 允许在一个字节流中存在多个 JSON 值；常见于日志聚合等场景。
func WithAllowTrailingData() Option {
	return func(o *decodeOptions) { o.allowTrailing = true }
}

// UnmarshalBytes 将 JSON 字节解码到给定指针 target（必须为指针）。
// 提供 Option 以控制解码行为。
func UnmarshalBytes(data []byte, target any, opts ...Option) error {
	return decode(bytes.NewReader(data), target, opts...)
}

// UnmarshalString 将 JSON 字符串解码到给定指针 target。
func UnmarshalString(s string, target any, opts ...Option) error {
	return decode(bytes.NewBufferString(s), target, opts...)
}

// Unmarshal 从 io.Reader 流式解码 JSON。
func Unmarshal(r io.Reader, target any, opts ...Option) error {
	return decode(r, target, opts...)
}

// decode 内部公共入口。
func decode(r io.Reader, target any, opts ...Option) error {
	decOpts := &decodeOptions{}
	for _, opt := range opts {
		opt(decOpts)
	}

	dec := json.NewDecoder(r)
	if decOpts.useNumber {
		dec.UseNumber()
	}
	if decOpts.disallowUnknown {
		dec.DisallowUnknownFields()
	}

	if err := dec.Decode(target); err != nil {
		return err
	}

	if !decOpts.allowTrailing {
		// 检查是否还有多余非空白数据
		if err := consumeAllWhitespace(dec); err != nil {
			return fmt.Errorf("gson: extra data after top-level value: %w", err)
		}
	}
	return nil
}

// consumeAllWhitespace 尝试继续解码第二个值；若存在有效 token，则视为多余数据。
func consumeAllWhitespace(dec *json.Decoder) error {
	// Peek 下一个 token；若 EOF 则正常。
	// json.Decoder 没有直接 Peek；我们尝试 decode 到 RawMessage。
	var m json.RawMessage
	if err := dec.Decode(&m); err == io.EOF {
		return nil
	} else if err != nil {
		// 如果是纯空白会得到 EOF，因此这里若出现除 EOF 外错误，也说明有残余垃圾。
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	// 读到了额外 JSON 值。
	return errors.New("non-whitespace trailing data present")
}

// ------------------------------
// 泛型便捷函数（Go 1.18+）
// ------------------------------

// FromJSONBytes FromJSONBytes[T] 将 JSON 字节解码为类型 T 并返回。
func FromJSONBytes[T any](data []byte, opts ...Option) (T, error) {
	var v T
	err := UnmarshalBytes(data, &v, opts...)
	return v, err
}

// FromJSONString FromJSONString[T] 将 JSON 字符串解码为类型 T 并返回。
func FromJSONString[T any](s string, opts ...Option) (T, error) {
	var v T
	err := UnmarshalString(s, &v, opts...)
	return v, err
}

// MustFromJSONBytes MustFromJSONBytes[T] panic 版本。
func MustFromJSONBytes[T any](data []byte, opts ...Option) T {
	v, err := FromJSONBytes[T](data, opts...)
	if err != nil {
		panic(err)
	}
	return v
}

// MustFromJSONString MustFromJSONString[T] panic 版本。
func MustFromJSONString[T any](s string, opts ...Option) T {
	v, err := FromJSONString[T](s, opts...)
	if err != nil {
		panic(err)
	}
	return v
}

// ------------------------------
// Map / Struct 互转
// ------------------------------

// ToMap 将任意结构转换为 map[string]any；内部先 JSON 再反序列化，字段 tag 生效。
// 注意：不适合对性能极端敏感路径。
func ToMap(v any) (map[string]any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// FromMap 将 map[string]any 转回指定结构体指针 target。
// 利用中转 JSON 保证 tag 规则一致。
func FromMap(m map[string]any, target any, opts ...Option) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return UnmarshalBytes(b, target, opts...)
}

// ConvertStructAtoB 通过 JSON 中转实现任意两个结构体之间字段兼容复制。
// 常用场景：API DTO <-> 内部模型。
func ConvertStructAtoB[A any, B any](src A, opts ...Option) (B, error) {
	var zero B
	b, err := json.Marshal(src)
	if err != nil {
		return zero, err
	}
	var dst B
	if err := UnmarshalBytes(b, &dst, opts...); err != nil {
		return zero, err
	}
	return dst, nil
}

// ------------------------------
// JSON 规范化 / Pretty / 压缩
// ------------------------------

// Pretty 对任意 JSON 文本进行缩进美化（prefix 通常为空字符串，indent 为 "  " 或 "\t"）。
func Pretty(data []byte, prefix, indent string) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, prefix, indent); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Compact 去除多余空白，使 JSON 更紧凑。
func Compact(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.Compact(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Normalize 先解码为 interface{} 再重新编码，获得字段按编码器排序（Go map 随机）后的稳定输出。
// 注意：map 的键顺序仍非强保证；如需真正确定性序列化，请使用结构体或额外排序逻辑。
func Normalize(data []byte) ([]byte, error) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

// ------------------------------
// RawMessage 辅助
// ------------------------------

// Raw 返回 json.RawMessage 包装（便于延迟解码）。
func Raw(data []byte) json.RawMessage { return json.RawMessage(data) }

// RawString 将字符串转换为 RawMessage。
func RawString(s string) json.RawMessage { return json.RawMessage([]byte(s)) }

// ------------------------------
// Decoder / Encoder 包装（Builder 风格）
// ------------------------------

// DecoderBuilder 允许按选项构造一个 json.Decoder。
type DecoderBuilder struct {
	r     io.Reader
	opts  []Option
	reuse *json.Decoder
}

func NewDecoderBuilder(r io.Reader) *DecoderBuilder { return &DecoderBuilder{r: r} }

func (b *DecoderBuilder) With(opts ...Option) *DecoderBuilder {
	b.opts = append(b.opts, opts...)
	return b
}

// Build 根据当前设置返回 *json.Decoder（每次新建）。
func (b *DecoderBuilder) Build() *json.Decoder {
	decOpts := &decodeOptions{}
	for _, opt := range b.opts {
		opt(decOpts)
	}
	dec := json.NewDecoder(b.r)
	if decOpts.useNumber {
		dec.UseNumber()
	}
	if decOpts.disallowUnknown {
		dec.DisallowUnknownFields()
	}
	return dec
}
