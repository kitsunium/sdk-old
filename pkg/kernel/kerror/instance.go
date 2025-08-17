package kerror

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
)

// Instance represents an instance of an error with context
type Instance struct {
	err      KError
	cause    error
	message  string
	tags     map[string]string
	details  map[string]any
	stack    []uintptr // Stack trace
	context  context.Context
	mu       sync.RWMutex
}

// Pool for reusing Instance objects to reduce allocations
var instancePool = &sync.Pool{
	New: func() any {
		return &Instance{
			tags:    make(map[string]string, 4),
			details: make(map[string]any, 4),
		}
	},
}

// Pool for string builders
var builderPool = &sync.Pool{
	New: func() any {
		return &strings.Builder{}
	},
}

// New creates a new error instance from a KError
func (e KError) New() *Instance {
	inst := instancePool.Get().(*Instance)
	inst.err = e
	inst.message = e.message
	inst.cause = nil
	inst.context = nil
	inst.stack = inst.stack[:0] // Reset slice but keep capacity
	
	// Capture stack trace if enabled
	if GetConfig().EnableStackTrace {
		inst.CaptureStack(3) // Skip New, CaptureStack, and runtime.Callers
	}
	
	// Record metrics if enabled
	if GetConfig().EnableMetrics {
		recordErrorInstance(e.pkg, e.code)
	}
	
	return inst
}

// Newf creates a new error instance with formatted message
func (e KError) Newf(format string, args ...any) *Instance {
	inst := instancePool.Get().(*Instance)
	inst.err = e
	inst.message = fmt.Sprintf(format, args...)
	inst.cause = nil
	inst.context = nil
	inst.stack = inst.stack[:0]
	
	if GetConfig().EnableStackTrace {
		inst.CaptureStack(3)
	}
	
	if GetConfig().EnableMetrics {
		recordErrorInstance(e.pkg, e.code)
	}
	
	return inst
}

// Wrap wraps an existing error
func (e KError) Wrap(cause error) *Instance {
	if cause == nil {
		return nil
	}
	inst := instancePool.Get().(*Instance)
	inst.err = e
	inst.cause = cause
	inst.message = e.message
	inst.context = nil
	inst.stack = inst.stack[:0]
	
	if GetConfig().EnableStackTrace {
		inst.CaptureStack(3)
	}
	
	if GetConfig().EnableMetrics {
		recordErrorWrapped(e.pkg, e.code)
	}
	
	return inst
}

// Wrapf wraps an existing error with formatted message
func (e KError) Wrapf(cause error, format string, args ...any) *Instance {
	if cause == nil {
		return nil
	}
	inst := instancePool.Get().(*Instance)
	inst.err = e
	inst.cause = cause
	inst.message = fmt.Sprintf(format, args...)
	inst.context = nil
	inst.stack = inst.stack[:0]
	
	if GetConfig().EnableStackTrace {
		inst.CaptureStack(3)
	}
	
	if GetConfig().EnableMetrics {
		recordErrorWrapped(e.pkg, e.code)
	}
	
	return inst
}

// CaptureStack captures the current stack trace
func (i *Instance) CaptureStack(skip int) *Instance {
	cfg := GetConfig()
	if cap(i.stack) < cfg.StackTraceDepth {
		i.stack = make([]uintptr, cfg.StackTraceDepth)
	} else {
		i.stack = i.stack[:cfg.StackTraceDepth]
	}
	n := runtime.Callers(skip, i.stack)
	i.stack = i.stack[:n]
	return i
}

// StackTrace returns the stack trace as a string
func (i *Instance) StackTrace() string {
	if len(i.stack) == 0 {
		return ""
	}
	
	sb := builderPool.Get().(*strings.Builder)
	defer func() {
		sb.Reset()
		builderPool.Put(sb)
	}()
	
	frames := runtime.CallersFrames(i.stack)
	for {
		frame, more := frames.Next()
		fmt.Fprintf(sb, "%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line)
		if !more {
			break
		}
	}
	
	return sb.String()
}

// WithContext attaches a context to the error instance
func (i *Instance) WithContext(ctx context.Context) *Instance {
	i.context = ctx
	
	// Extract trace ID if available
	if traceID := ExtractTraceID(ctx); traceID != "" {
		i.WithTag("trace_id", traceID)
	}
	
	// Extract span ID if available  
	if spanID := ExtractSpanID(ctx); spanID != "" {
		i.WithTag("span_id", spanID)
	}
	
	return i
}

// Context returns the attached context
func (i *Instance) Context() context.Context {
	return i.context
}

// Error implements the error interface
func (i *Instance) Error() string {
	if i.cause == nil {
		return i.message
	}
	
	// Use string builder from pool
	sb := builderPool.Get().(*strings.Builder)
	defer func() {
		sb.Reset()
		builderPool.Put(sb)
	}()
	
	sb.Grow(len(i.message) + 2 + 50) // Pre-allocate
	sb.WriteString(i.message)
	sb.WriteString(": ")
	sb.WriteString(i.cause.Error())
	return sb.String()
}

// Unwrap implements errors.Unwrap
func (i *Instance) Unwrap() error {
	return i.cause
}

// Is implements errors.Is
func (i *Instance) Is(target error) bool {
	// Check if target is the same KError
	if t, ok := target.(KError); ok {
		return i.err.Is(t)
	}
	if t, ok := target.(*KError); ok {
		return i.err.Is(t)
	}
	// Check if target is the same Instance
	if t, ok := target.(*Instance); ok {
		return i.err.Is(t.err)
	}
	return false
}

// KError returns the underlying KError
func (i *Instance) KError() KError {
	return i.err
}

// Package returns the package name
func (i *Instance) Package() string {
	return i.err.pkg
}

// Code returns the error code
func (i *Instance) Code() int {
	return i.err.code
}

// WithTag adds a tag (returns same instance for chaining)
func (i *Instance) WithTag(key, value string) *Instance {
	cfg := GetConfig()
	
	// Validate if enabled
	if cfg.EnableValidation {
		if len(key) > cfg.MaxTagKeyLen || len(value) > cfg.MaxTagValueLen {
			return i // Silently ignore
		}
		
		i.mu.Lock()
		defer i.mu.Unlock()
		
		if len(i.tags) >= cfg.MaxTags {
			return i // Maximum tags reached
		}
	} else {
		i.mu.Lock()
		defer i.mu.Unlock()
	}
	
	i.tags[key] = value
	return i
}

// WithTags adds multiple tags
func (i *Instance) WithTags(tags map[string]string) *Instance {
	if len(tags) == 0 {
		return i
	}
	
	cfg := GetConfig()
	i.mu.Lock()
	defer i.mu.Unlock()
	
	for k, v := range tags {
		// Validate if enabled
		if cfg.EnableValidation {
			if len(i.tags) >= cfg.MaxTags {
				break
			}
			if len(k) > cfg.MaxTagKeyLen || len(v) > cfg.MaxTagValueLen {
				continue
			}
		}
		i.tags[k] = v
	}
	
	return i
}

// Tag returns a tag value
func (i *Instance) Tag(key string) (string, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	val, ok := i.tags[key]
	return val, ok
}

// Tags returns all tags (safe copy)
func (i *Instance) Tags() map[string]string {
	i.mu.RLock()
	defer i.mu.RUnlock()
	result := make(map[string]string, len(i.tags))
	for k, v := range i.tags {
		result[k] = v
	}
	return result
}

// WithDetail adds a detail (returns same instance for chaining)
func (i *Instance) WithDetail(key string, value any) *Instance {
	cfg := GetConfig()
	
	i.mu.Lock()
	defer i.mu.Unlock()
	
	if cfg.EnableValidation && len(i.details) >= cfg.MaxDetails {
		return i // Maximum details reached
	}
	
	i.details[key] = value
	return i
}

// WithDetails adds multiple details
func (i *Instance) WithDetails(details map[string]any) *Instance {
	if len(details) == 0 {
		return i
	}
	
	cfg := GetConfig()
	i.mu.Lock()
	defer i.mu.Unlock()
	
	for k, v := range details {
		if cfg.EnableValidation && len(i.details) >= cfg.MaxDetails {
			break
		}
		i.details[k] = v
	}
	
	return i
}

// Detail returns a detail value
func (i *Instance) Detail(key string) (any, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	val, ok := i.details[key]
	return val, ok
}

// Details returns all details (safe copy)
func (i *Instance) Details() map[string]any {
	i.mu.RLock()
	defer i.mu.RUnlock()
	result := make(map[string]any, len(i.details))
	for k, v := range i.details {
		result[k] = v
	}
	return result
}

// Release returns the instance to the pool for reuse
func (i *Instance) Release() {
	i.mu.Lock()
	defer i.mu.Unlock()
	
	// Clear maps without reallocating
	for k := range i.tags {
		delete(i.tags, k)
	}
	for k := range i.details {
		delete(i.details, k)
	}
	
	i.cause = nil
	i.message = ""
	i.context = nil
	i.stack = i.stack[:0] // Keep capacity
	
	instancePool.Put(i)
}

// OTelAttributes returns OpenTelemetry compatible attributes
func (i *Instance) OTelAttributes() map[string]any {
	i.mu.RLock()
	
	// Calculate exact size needed
	size := 4 // Standard attributes
	size += len(i.tags)
	size += len(i.details)
	if i.cause != nil {
		size++
	}
	if len(i.stack) > 0 {
		size++
	}
	
	attrs := make(map[string]any, size)
	
	// Standard attributes
	attrs["error.id"] = i.err.id
	attrs["error.package"] = i.err.pkg
	attrs["error.code"] = i.err.code
	attrs["error.message"] = i.message
	
	// Use string builder from pool
	sb := builderPool.Get().(*strings.Builder)
	defer func() {
		sb.Reset()
		builderPool.Put(sb)
	}()
	
	sb.Grow(32) // Pre-allocate
	
	for k, v := range i.tags {
		sb.Reset()
		sb.WriteString("error.tag.")
		sb.WriteString(k)
		attrs[sb.String()] = v
	}
	
	for k, v := range i.details {
		sb.Reset()
		sb.WriteString("error.detail.")
		sb.WriteString(k)
		attrs[sb.String()] = v
	}
	
	if i.cause != nil {
		attrs["error.cause"] = i.cause.Error()
	}
	
	if len(i.stack) > 0 {
		attrs["error.stack_trace"] = i.StackTrace()
	}
	
	i.mu.RUnlock()
	return attrs
}

// MarshalJSON implements json.Marshaler
func (i *Instance) MarshalJSON() ([]byte, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	
	data := map[string]any{
		"error": i.err,
		"message": i.message,
		"tags": i.tags,
		"details": i.details,
	}
	
	if i.cause != nil {
		data["cause"] = i.cause.Error()
	}
	
	if len(i.stack) > 0 {
		data["stack_trace"] = i.StackTrace()
	}
	
	return json.Marshal(data)
}

// As implements errors.As
func (i *Instance) As(target any) bool {
	switch t := target.(type) {
	case **Instance:
		*t = i
		return true
	case *KError:
		*t = i.err
		return true
	default:
		return errors.As(i.cause, target)
	}
}