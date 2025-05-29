package util

// Tern implements a generic ternary operator
// T is constrained to any type, making this function work with any comparable type
// The function evaluates the condition and returns trueReturn if true, falseReturn if false
func Tern[T any](condition bool, trueReturn T, falseReturn T) T {
	if condition {
		return trueReturn
	}
	return falseReturn
}
