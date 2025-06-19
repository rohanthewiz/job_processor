package util

// If implements a generic ternary operator, providing the functionality of a conditional operator
// found in many other languages (condition ? trueValue : falseValue).
// T can be any type - the values are returned based on the boolean condition.
// Returns ifTrue when true, otherwise returns ifFalse.
func If[T any](condition bool, ifTrue T, ifFalse T) T {
	if condition {
		return ifTrue
	}
	return ifFalse
}
