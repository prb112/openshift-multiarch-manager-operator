package utils

func NewPtr[T any](a T) *T {
	return &a
}
