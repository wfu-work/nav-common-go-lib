package utils

// Stream 支持链式操作
type Stream[T any] struct {
	data []T
}

// NewStream 创建 Stream
func NewStream[T any](data []T) *Stream[T] {
	return &Stream[T]{data: data}
}

// Map 映射到 R 类型
func Map[T any, R any](stream *Stream[T], fn func(T) R) *Stream[R] {
	var result []R
	for _, v := range stream.data {
		result = append(result, fn(v))
	}
	return &Stream[R]{data: result}
}

// Filter 过滤
func (s *Stream[T]) Filter(fn func(T) bool) *Stream[T] {
	var filtered []T
	for _, v := range s.data {
		if fn(v) {
			filtered = append(filtered, v)
		}
	}
	s.data = filtered
	return s
}

// Distinct 去重，元素需要可比较，Go 1.21 开始支持 comparable 泛型约束
func (s *Stream[T]) Distinct(equal func(a, b T) bool) *Stream[T] {
	var result []T
	for _, v := range s.data {
		found := false
		for _, r := range result {
			if equal(v, r) {
				found = true
				break
			}
		}
		if !found {
			result = append(result, v)
		}
	}
	s.data = result
	return s
}

// Collect 返回结果切片
func (s *Stream[T]) Collect() []T {
	return s.data
}

func Contains[T comparable](slice []T, item T) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}
