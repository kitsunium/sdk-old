package kerror

// Result represents a value that may or may not be present
type Result[T any] struct {
	Value T
	Ok    bool
}

// NewResult creates a new Result
func NewResult[T any](value T, ok bool) Result[T] {
	return Result[T]{Value: value, Ok: ok}
}

// Unwrap returns the value or panics if not ok
func (r Result[T]) Unwrap() T {
	if !r.Ok {
		panic("attempted to unwrap failed Result")
	}
	return r.Value
}

// UnwrapOr returns the value or a default
func (r Result[T]) UnwrapOr(defaultValue T) T {
	if r.Ok {
		return r.Value
	}
	return defaultValue
}
