package errs

import (
	"context"
)

// contextKey is the type for context keys
type contextKey int

const (
	// errorContextKey is the key for storing error instances in context
	errorContextKey contextKey = iota
)

// FromContext extracts an error instance from context
func FromContext(ctx context.Context) (*Instance, bool) {
	if ctx == nil {
		return nil, false
	}

	val := ctx.Value(errorContextKey)
	if inst, ok := val.(*Instance); ok && inst != nil {
		return inst, true
	}
	return nil, false
}

// ToContext adds an error instance to context
func ToContext(ctx context.Context, inst *Instance) context.Context {
	return context.WithValue(ctx, errorContextKey, inst)
}

// Variables for trace/span extraction functions (allows mocking in tests)
var (
	extractTraceIDFunc = defaultExtractTraceID
	extractSpanIDFunc  = defaultExtractSpanID
)

// ExtractTraceID extracts a trace ID from context if available
func ExtractTraceID(ctx context.Context) string {
	return extractTraceIDFunc(ctx)
}

// ExtractSpanID extracts a span ID from context if available
func ExtractSpanID(ctx context.Context) string {
	return extractSpanIDFunc(ctx)
}

// defaultExtractTraceID is the default implementation
// This is a placeholder - implement based on your tracing library
func defaultExtractTraceID(ctx context.Context) string {
	// This would integrate with your tracing library
	// For example, with OpenTelemetry:
	// if span := trace.SpanFromContext(ctx); span.SpanContext().HasTraceID() {
	//     return span.SpanContext().TraceID().String()
	// }
	return ""
}

// defaultExtractSpanID is the default implementation
// This is a placeholder - implement based on your tracing library
func defaultExtractSpanID(ctx context.Context) string {
	// This would integrate with your tracing library
	// For example, with OpenTelemetry:
	// if span := trace.SpanFromContext(ctx); span.SpanContext().HasSpanID() {
	//     return span.SpanContext().SpanID().String()
	// }
	return ""
}
