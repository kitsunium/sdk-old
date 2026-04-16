// Package logger - text implementation.
//
// This file implements the human-readable logger returned by [Text].
// Records are rendered as: "<time> <LEVEL> <msg> key=value key=value".
package logger

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
)

// textLogger is the concrete text implementation of [Logger].
type textLogger struct {
	cfg    *config
	mu     *sync.Mutex
	fields []Field
}

// Text builds a Logger that emits one human-readable line per record.
func Text(opts ...Option) Logger {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return &textLogger{
		cfg:    cfg,
		mu:     &sync.Mutex{},
		fields: cloneFields(cfg.base),
	}
}

// Debug logs at Debug level.
func (l *textLogger) Debug(msg string, fields ...Field) {
	l.emit(LevelDebug, nil, msg, fields)
}

// Info logs at Info level.
func (l *textLogger) Info(msg string, fields ...Field) {
	l.emit(LevelInfo, nil, msg, fields)
}

// Warn logs at Warn level.
func (l *textLogger) Warn(msg string, fields ...Field) {
	l.emit(LevelWarn, nil, msg, fields)
}

// Error logs at Error level along with err.
func (l *textLogger) Error(err error, msg string, fields ...Field) {
	l.emit(LevelError, err, msg, fields)
}

// With returns a derived logger that prepends the given fields.
func (l *textLogger) With(fields ...Field) Logger {
	merged := make([]Field, 0, len(l.fields)+len(fields))
	merged = append(merged, l.fields...)
	merged = append(merged, fields...)
	return &textLogger{
		cfg:    l.cfg,
		mu:     l.mu,
		fields: merged,
	}
}

// emit formats and writes a single record if level passes the threshold.
func (l *textLogger) emit(level Level, err error, msg string, fields []Field) {
	if level < l.cfg.level || l.cfg.level == LevelOff {
		return
	}

	var sb strings.Builder
	sb.Grow(64 + len(msg))
	sb.WriteString(l.cfg.now().UTC().Format(l.cfg.timeFormat))
	sb.WriteByte(' ')
	sb.WriteString(level.String())
	sb.WriteByte(' ')
	sb.WriteString(msg)

	if err != nil {
		sb.WriteByte(' ')
		writeKV(&sb, "error", err.Error())
	}
	for _, f := range l.fields {
		sb.WriteByte(' ')
		writeField(&sb, f)
	}
	for _, f := range fields {
		sb.WriteByte(' ')
		writeField(&sb, f)
	}
	sb.WriteByte('\n')

	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = l.cfg.out.Write([]byte(sb.String()))
}

// writeField renders a single Field into sb using its kind.
func writeField(sb *strings.Builder, f Field) {
	switch f.Kind {
	case KindString:
		writeKV(sb, f.Key, f.StrVal)
	case KindInt:
		sb.WriteString(f.Key)
		sb.WriteByte('=')
		sb.WriteString(strconv.FormatInt(f.IntVal, 10))
	case KindFloat:
		sb.WriteString(f.Key)
		sb.WriteByte('=')
		sb.WriteString(strconv.FormatFloat(f.FltVal, 'g', -1, 64))
	case KindBool:
		sb.WriteString(f.Key)
		sb.WriteByte('=')
		sb.WriteString(strconv.FormatBool(f.BoolVal))
	case KindError:
		if f.ErrVal == nil {
			writeKV(sb, f.Key, "<nil>")
		} else {
			writeKV(sb, f.Key, f.ErrVal.Error())
		}
	case KindAny:
		writeKV(sb, f.Key, anyToString(f.AnyVal))
	default:
		writeKV(sb, f.Key, anyToString(f.AnyVal))
	}
}

// writeKV writes "key=value", quoting value when it contains whitespace
// or a double quote. This mirrors the conventional logfmt style.
func writeKV(sb *strings.Builder, key, value string) {
	sb.WriteString(key)
	sb.WriteByte('=')
	if needsQuote(value) {
		sb.WriteString(strconv.Quote(value))
	} else {
		sb.WriteString(value)
	}
}

// needsQuote returns true when value must be quoted to preserve parsing.
func needsQuote(s string) bool {
	if s == "" {
		return true
	}
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '"' || r == '=' {
			return true
		}
	}
	return false
}

// anyToString renders an arbitrary value using strconv where possible and
// falls back to Go's default formatting otherwise.
func anyToString(v any) string {
	switch x := v.(type) {
	case nil:
		return "<nil>"
	case string:
		return x
	case bool:
		return strconv.FormatBool(x)
	case int:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64)
	case error:
		return x.Error()
	default:
		// Use Sprint-like formatting without importing fmt for hot path.
		// strconv handles the common cases above; fall back to generic.
		return genericString(v)
	}
}

// genericString produces a portable string rendering of v for the few
// cases not handled by [anyToString]. It defers to encoding/json to
// keep the dependency surface to stdlib only.
func genericString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "<unprintable>"
	}
	return string(b)
}
