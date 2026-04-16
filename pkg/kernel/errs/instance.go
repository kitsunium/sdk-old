// Package errs instance types for the Kitsunium error catalog.
//
// See error.go for the full package documentation.
package errs

import (
	"errors"
	"fmt"
	"strings"
)

// Instance is a concrete occurrence of an Err.
//
// It may wrap a cause, carry string tags, and typed details.
// Create instances via Err.New, Err.Newf, Err.Wrap, or Err.Wrapf.
type Instance struct {
	err     Err
	cause   error
	message string
	tags    map[string]string
	details map[string]any
}

// New returns a fresh Instance using the Err default message.
//
// Returns:
//   - inst: the new Instance; never nil
func (e Err) New() (inst *Instance) {
	//: allocate a new occurrence with the catalog default message
	return &Instance{err: e, message: e.message}
}

// Newf returns a fresh Instance with a formatted message.
//
// Params:
//   - format: printf-style format string
//   - args: format arguments
//
// Returns:
//   - inst: the new Instance; never nil
func (e Err) Newf(format string, args ...any) (inst *Instance) {
	//: allocate with a caller-provided formatted message
	return &Instance{err: e, message: fmt.Sprintf(format, args...)}
}

// Wrap returns an Instance that wraps cause.
//
// Params:
//   - cause: the underlying error to wrap; nil returns nil
//
// Returns:
//   - inst: the new Instance, or nil when cause is nil
func (e Err) Wrap(cause error) (inst *Instance) {
	//: nil cause means "no error to wrap" — propagate nil
	if cause == nil {
		//: caller can test the result for nil safely
		return nil
	}
	//: embed cause for errors.Is/As chain traversal
	return &Instance{err: e, cause: cause, message: e.message}
}

// Wrapf returns an Instance that wraps cause with a formatted message.
//
// Params:
//   - cause: the underlying error to wrap; nil returns nil
//   - format: printf-style format string
//   - args: format arguments
//
// Returns:
//   - inst: the new Instance, or nil when cause is nil
func (e Err) Wrapf(cause error, format string, args ...any) (inst *Instance) {
	//: nil cause means "no error to wrap" — propagate nil
	if cause == nil {
		//: caller can test the result for nil safely
		return nil
	}
	//: embed cause and use the caller-provided message
	return &Instance{err: e, cause: cause, message: fmt.Sprintf(format, args...)}
}

// Error implements the error interface. Format: "[pkg:code] message: cause".
//
// Returns:
//   - s: the fully formatted error string
func (i *Instance) Error() (s string) {
	var b strings.Builder
	b.WriteByte('[')
	b.WriteString(i.err.pkg)
	b.WriteByte(':')
	fmt.Fprintf(&b, "%d", i.err.code)
	b.WriteByte(']')
	b.WriteByte(' ')
	b.WriteString(i.message)
	//: append wrapped cause only when present
	if i.cause != nil {
		//: separator matches stdlib convention
		b.WriteString(": ")
		b.WriteString(i.cause.Error())
	}
	return b.String()
}

// Cause returns the wrapped cause, or nil.
//
// Returns:
//   - err: the wrapped error, or nil when none was provided
func (i *Instance) Cause() (err error) {
	//: expose wrapped cause for errors.Unwrap chain
	return i.cause
}

// Unwrap returns the wrapped cause for compatibility with errors.Unwrap.
//
// Returns:
//   - err: the wrapped error, or nil when none was provided
func (i *Instance) Unwrap() (err error) {
	//: delegate to Cause to satisfy the errors.Unwrap interface
	return i.cause
}

// Is reports whether this Instance matches the given target.
//
// An Instance matches another Instance or an Err by catalog ID.
//
// Params:
//   - target: the error to compare against
//
// Returns:
//   - match: true when target refers to the same catalog entry
func (i *Instance) Is(target error) (match bool) {
	//: compare by catalog ID, accepting Instance, Err, and pointer forms
	switch t := target.(type) {
	//: *Instance match — compare catalog IDs (nil pointer never matches)
	case *Instance:
		return t != nil && i.err.id == t.err.id
	//: Err value match
	case Err:
		return i.err.id == t.id
	//: *Err pointer match — nil pointer never matches
	case *Err:
		return t != nil && i.err.id == t.id
	}
	//: unrecognised type — no match
	return false
}

// As delegates to errors.As on the wrapped cause.
//
// Params:
//   - target: pointer to the target type (same semantics as errors.As)
//
// Returns:
//   - ok: true when the cause chain contains a value assignable to target
func (i *Instance) As(target any) (ok bool) {
	//: no cause means nothing to unwrap into
	if i.cause == nil {
		//: report no match immediately
		return false
	}
	//: delegate chain traversal to the standard library
	return errors.As(i.cause, target)
}

// Err returns the catalog entry this Instance was built from.
//
// Returns:
//   - entry: the originating Err catalog entry
func (i *Instance) Err() (entry Err) {
	//: expose the originating catalog entry for comparison
	return i.err
}

// Package returns the catalog package.
//
// Returns:
//   - pkg: the package name from the catalog entry
func (i *Instance) Package() (pkg string) {
	//: delegate to the embedded catalog entry
	return i.err.pkg
}

// Code returns the catalog code.
//
// Returns:
//   - code: the integer error code from the catalog entry
func (i *Instance) Code() (code int) {
	//: delegate to the embedded catalog entry
	return i.err.code
}

// WithTag attaches a key/value tag and returns i for chaining.
//
// Params:
//   - key: the tag name
//   - value: the tag value
//
// Returns:
//   - inst: i itself for method chaining
func (i *Instance) WithTag(key, value string) (inst *Instance) {
	//: lazy-allocate to avoid map overhead on Instances that never use tags
	if i.tags == nil {
		//: preallocate with a small capacity to reduce rehashing
		i.tags = make(map[string]string, initialMapCapacity)
	}
	i.tags[key] = value
	return i
}

// WithDetail attaches a typed detail and returns i for chaining.
//
// Params:
//   - key: the detail name
//   - value: the detail value (any type)
//
// Returns:
//   - inst: i itself for method chaining
func (i *Instance) WithDetail(key string, value any) (inst *Instance) {
	//: lazy-allocate to avoid map overhead on Instances that never use details
	if i.details == nil {
		//: preallocate with a small capacity to reduce rehashing
		i.details = make(map[string]any, initialMapCapacity)
	}
	i.details[key] = value
	return i
}

// Tag returns the value of a tag and whether it was present.
//
// Params:
//   - key: the tag name to look up
//
// Returns:
//   - value: the tag value, or empty string when absent
//   - ok: true when the tag was found
func (i *Instance) Tag(key string) (value string, ok bool) {
	//: nil-safe map lookup
	v, found := i.tags[key]
	return v, found
}

// Detail returns the value of a detail and whether it was present.
//
// Params:
//   - key: the detail name to look up
//
// Returns:
//   - value: the stored value, or nil when absent
//   - ok: true when the detail was found
func (i *Instance) Detail(key string) (value any, ok bool) {
	//: nil-safe map lookup
	v, found := i.details[key]
	return v, found
}

// DetailAs returns a typed view over a detail value.
//
// It returns the zero value of T and false when the key is absent or
// the stored value has a different type.
//
// Params:
//   - i: the Instance to inspect; nil returns zero, false
//   - key: the detail name to look up
//
// Returns:
//   - value: the typed value, or zero when absent or wrong type
//   - ok: true when the detail was found and successfully typed
func DetailAs[T any](i *Instance, key string) (value T, ok bool) {
	var zero T
	//: guard against nil receiver and missing map
	if i == nil || i.details == nil {
		//: nothing to look up — return zero
		return zero, false
	}
	v, found := i.details[key]
	//: key absent
	if !found {
		//: key not present — return zero
		return zero, false
	}
	out, typed := v.(T)
	//: stored type does not match requested type
	if !typed {
		//: type mismatch — return zero
		return zero, false
	}
	return out, true
}
