// Package logger - JSON implementation.
//
// This file implements the JSON-formatted logger returned by [JSON].
// Each log record is serialized as a single-line JSON object containing
// at minimum "time", "level", "msg" plus every accumulated Field.
package logger

import (
	"encoding/json"
	"sync"
)

// jsonLogger is the concrete JSON implementation of [Logger].
type jsonLogger struct {
	cfg    *config
	mu     *sync.Mutex
	fields []Field
}

// JSON builds a Logger that emits one JSON object per line.
func JSON(opts ...Option) Logger {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return &jsonLogger{
		cfg:    cfg,
		mu:     &sync.Mutex{},
		fields: cloneFields(cfg.base),
	}
}

// Debug logs at Debug level.
func (l *jsonLogger) Debug(msg string, fields ...Field) {
	l.emit(LevelDebug, nil, msg, fields)
}

// Info logs at Info level.
func (l *jsonLogger) Info(msg string, fields ...Field) {
	l.emit(LevelInfo, nil, msg, fields)
}

// Warn logs at Warn level.
func (l *jsonLogger) Warn(msg string, fields ...Field) {
	l.emit(LevelWarn, nil, msg, fields)
}

// Error logs at Error level along with err.
func (l *jsonLogger) Error(err error, msg string, fields ...Field) {
	l.emit(LevelError, err, msg, fields)
}

// With returns a derived logger that will prepend the given fields to
// every subsequent record.
func (l *jsonLogger) With(fields ...Field) Logger {
	merged := make([]Field, 0, len(l.fields)+len(fields))
	merged = append(merged, l.fields...)
	merged = append(merged, fields...)
	return &jsonLogger{
		cfg:    l.cfg,
		mu:     l.mu,
		fields: merged,
	}
}

// emit serializes and writes a single record if level passes the
// configured threshold.
func (l *jsonLogger) emit(level Level, err error, msg string, fields []Field) {
	if level < l.cfg.level || l.cfg.level == LevelOff {
		return
	}

	record := make(map[string]any, 4+len(l.fields)+len(fields))
	record["time"] = l.cfg.now().UTC().Format(l.cfg.timeFormat)
	record["level"] = level.String()
	record["msg"] = msg
	if err != nil {
		record["error"] = err.Error()
	}
	for _, f := range l.fields {
		encodeField(record, f)
	}
	for _, f := range fields {
		encodeField(record, f)
	}

	buf, jerr := json.Marshal(record)
	if jerr != nil {
		// Fall back to a best-effort error record so we never lose a
		// log line due to a malformed field.
		buf, _ = json.Marshal(map[string]string{
			"time":  l.cfg.now().UTC().Format(l.cfg.timeFormat),
			"level": LevelError.String(),
			"msg":   "logger: marshal failed",
			"error": jerr.Error(),
		})
	}
	buf = append(buf, '\n')

	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = l.cfg.out.Write(buf)
}

// encodeField writes a Field into the destination map using its kind.
func encodeField(dst map[string]any, f Field) {
	switch f.Kind {
	case KindString:
		dst[f.Key] = f.StrVal
	case KindInt:
		dst[f.Key] = f.IntVal
	case KindFloat:
		dst[f.Key] = f.FltVal
	case KindBool:
		dst[f.Key] = f.BoolVal
	case KindError:
		if f.ErrVal == nil {
			dst[f.Key] = nil
		} else {
			dst[f.Key] = f.ErrVal.Error()
		}
	case KindAny:
		dst[f.Key] = f.AnyVal
	default:
		dst[f.Key] = f.AnyVal
	}
}
