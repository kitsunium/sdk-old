// Package logger provides structured logging for the Kitsunium SDK.
//
// It exposes a small Logger interface with two interchangeable
// implementations: [JSON] for machine-readable output and [Text] for
// human-readable output. Both are safe for concurrent use and share the
// same [Field] value type, [Level] threshold, and functional [Option]
// configuration.
//
// The package follows the SDK naming convention `package.Thing(...)`:
// `logger.JSON(opts...)` and `logger.Text(opts...)` are the two products
// exposed. Both return a value that implements [Logger].
package logger

import (
	"io"
	"os"
	"time"
)

// Level represents a logging severity threshold.
type Level int8

// Logging levels in ascending severity. Messages at a level below the
// configured threshold are dropped.
const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelOff
)

// String returns the uppercase name of the level.
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelOff:
		return "OFF"
	default:
		return "UNKNOWN"
	}
}

// FieldKind identifies the concrete value carried by a [Field].
type FieldKind uint8

// Field kinds supported by the logger.
const (
	KindAny FieldKind = iota
	KindString
	KindInt
	KindFloat
	KindBool
	KindError
)

// Field is a single structured key/value pair attached to a log line.
//
// A Field carries a typed value via its [FieldKind] so that formatters
// can serialize the value efficiently without reflection.
type Field struct {
	Key     string
	Kind    FieldKind
	StrVal  string
	IntVal  int64
	FltVal  float64
	BoolVal bool
	ErrVal  error
	AnyVal  any
}

// String returns a string-valued Field.
func String(k, v string) Field {
	return Field{Key: k, Kind: KindString, StrVal: v}
}

// Int returns an int-valued Field.
func Int(k string, v int) Field {
	return Field{Key: k, Kind: KindInt, IntVal: int64(v)}
}

// Int64 returns an int64-valued Field.
func Int64(k string, v int64) Field {
	return Field{Key: k, Kind: KindInt, IntVal: v}
}

// Float returns a float-valued Field.
func Float(k string, v float64) Field {
	return Field{Key: k, Kind: KindFloat, FltVal: v}
}

// Bool returns a bool-valued Field.
func Bool(k string, v bool) Field {
	return Field{Key: k, Kind: KindBool, BoolVal: v}
}

// Err returns an error-valued Field keyed as "error".
func Err(err error) Field {
	return Field{Key: "error", Kind: KindError, ErrVal: err}
}

// NamedErr returns an error-valued Field with a custom key.
func NamedErr(k string, err error) Field {
	return Field{Key: k, Kind: KindError, ErrVal: err}
}

// Any returns a Field wrapping an arbitrary value.
func Any(k string, v any) Field {
	return Field{Key: k, Kind: KindAny, AnyVal: v}
}

// Logger is the structured logging interface implemented by every
// concrete logger in this package.
type Logger interface {
	// Debug logs a message at Debug level.
	Debug(msg string, fields ...Field)
	// Info logs a message at Info level.
	Info(msg string, fields ...Field)
	// Warn logs a message at Warn level.
	Warn(msg string, fields ...Field)
	// Error logs a message at Error level along with err.
	Error(err error, msg string, fields ...Field)
	// With returns a derived logger whose fields are prepended to every
	// emitted record.
	With(fields ...Field) Logger
}

// Option mutates the internal configuration of a logger constructor.
type Option func(*config)

// config is the shared configuration of every logger implementation.
type config struct {
	level      Level
	out        io.Writer
	timeFormat string
	now        func() time.Time
	base       []Field
}

// defaultConfig returns a config populated with sensible defaults.
func defaultConfig() *config {
	return &config{
		level:      LevelInfo,
		out:        os.Stdout,
		timeFormat: time.RFC3339,
		now:        time.Now,
	}
}

// WithLevel sets the minimum level emitted by the logger.
func WithLevel(l Level) Option {
	return func(c *config) { c.level = l }
}

// WithOutput redirects log output to w.
func WithOutput(w io.Writer) Option {
	return func(c *config) {
		if w != nil {
			c.out = w
		}
	}
}

// WithTimeFormat overrides the timestamp layout used by the logger.
func WithTimeFormat(layout string) Option {
	return func(c *config) {
		if layout != "" {
			c.timeFormat = layout
		}
	}
}

// withNow is used by tests to inject a deterministic clock. It is not
// exported because callers have no legitimate reason to freeze time.
func withNow(fn func() time.Time) Option {
	return func(c *config) {
		if fn != nil {
			c.now = fn
		}
	}
}

// cloneFields returns a defensive copy of the supplied field slice.
// It is used by With() to avoid aliasing a caller-owned slice.
func cloneFields(in []Field) []Field {
	if len(in) == 0 {
		return nil
	}
	out := make([]Field, len(in))
	copy(out, in)
	return out
}
