// Package logger - unit tests.
package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// fixedClock returns a deterministic time for reproducible tests.
func fixedClock() func() time.Time {
	t := time.Date(2026, 4, 15, 12, 34, 56, 0, time.UTC)
	return func() time.Time { return t }
}

// assertContains fails the test when haystack does not contain needle.
func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("expected output to contain %q, got %q", needle, haystack)
	}
}

func TestLogger_InterfaceConformance(t *testing.T) {
	t.Run("JSON implements Logger", func(t *testing.T) {
		var _ Logger = JSON()
	})
	t.Run("Text implements Logger", func(t *testing.T) {
		var _ Logger = Text()
	})
}

func TestJSON_BasicRecord(t *testing.T) {
	var buf bytes.Buffer
	log := JSON(
		WithOutput(&buf),
		WithLevel(LevelDebug),
		withNow(fixedClock()),
	)
	log.Info("hello", String("user", "alice"), Int("n", 7), Bool("ok", true))

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("invalid JSON: %v (raw=%q)", err, buf.String())
	}
	if rec["level"] != "INFO" {
		t.Fatalf("want level INFO, got %v", rec["level"])
	}
	if rec["msg"] != "hello" {
		t.Fatalf("want msg hello, got %v", rec["msg"])
	}
	if rec["user"] != "alice" {
		t.Fatalf("want user alice, got %v", rec["user"])
	}
	if rec["ok"] != true {
		t.Fatalf("want ok true, got %v", rec["ok"])
	}
}

func TestText_BasicRecord(t *testing.T) {
	var buf bytes.Buffer
	log := Text(
		WithOutput(&buf),
		WithLevel(LevelDebug),
		withNow(fixedClock()),
	)
	log.Warn("disk low", String("path", "/var"), Int("free", 12))

	out := buf.String()
	assertContains(t, out, "2026-04-15T12:34:56Z")
	assertContains(t, out, "WARN")
	assertContains(t, out, "disk low")
	assertContains(t, out, "path=/var")
	assertContains(t, out, "free=12")
}

func TestLogger_LevelThreshold(t *testing.T) {
	cases := []struct {
		name       string
		threshold  Level
		shouldEmit map[string]bool
	}{
		{
			name:      "threshold=Info filters Debug",
			threshold: LevelInfo,
			shouldEmit: map[string]bool{
				"debug": false,
				"info":  true,
				"warn":  true,
				"error": true,
			},
		},
		{
			name:      "threshold=Error filters everything below",
			threshold: LevelError,
			shouldEmit: map[string]bool{
				"debug": false,
				"info":  false,
				"warn":  false,
				"error": true,
			},
		},
		{
			name:      "threshold=Off silences everything",
			threshold: LevelOff,
			shouldEmit: map[string]bool{
				"debug": false,
				"info":  false,
				"warn":  false,
				"error": false,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			log := JSON(WithOutput(&buf), WithLevel(tc.threshold), withNow(fixedClock()))
			log.Debug("debug")
			log.Info("info")
			log.Warn("warn")
			log.Error(errors.New("boom"), "error")

			out := buf.String()
			for keyword, want := range tc.shouldEmit {
				got := strings.Contains(out, `"msg":"`+keyword+`"`)
				if got != want {
					t.Fatalf("%s: emit=%v want=%v (out=%q)", keyword, got, want, out)
				}
			}
		})
	}
}

func TestLogger_WithInheritance(t *testing.T) {
	var buf bytes.Buffer
	base := JSON(WithOutput(&buf), WithLevel(LevelDebug), withNow(fixedClock()))
	child := base.With(String("service", "api"), Int("shard", 3))
	child.Info("hit")

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if rec["service"] != "api" {
		t.Fatalf("want service api, got %v", rec["service"])
	}
	if rec["shard"].(float64) != 3 {
		t.Fatalf("want shard 3, got %v", rec["shard"])
	}

	// Parent should NOT carry child fields.
	buf.Reset()
	base.Info("parent")
	var parent map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parent); err != nil {
		t.Fatalf("invalid parent JSON: %v", err)
	}
	if _, exists := parent["service"]; exists {
		t.Fatalf("parent leaked child field: %v", parent)
	}
}

func TestLogger_FieldKinds(t *testing.T) {
	var buf bytes.Buffer
	log := JSON(WithOutput(&buf), WithLevel(LevelDebug), withNow(fixedClock()))
	log.Info("all",
		String("s", "hi"),
		Int("i", 42),
		Int64("i64", 9000),
		Float("f", 3.14),
		Bool("b", false),
		Err(errors.New("boom")),
		NamedErr("cause", errors.New("root")),
		Any("any", map[string]int{"x": 1}),
	)

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if rec["s"] != "hi" || rec["i"].(float64) != 42 || rec["i64"].(float64) != 9000 {
		t.Fatalf("bad primitive fields: %v", rec)
	}
	if rec["f"].(float64) != 3.14 || rec["b"] != false {
		t.Fatalf("bad numeric/bool: %v", rec)
	}
	if rec["error"] != "boom" || rec["cause"] != "root" {
		t.Fatalf("bad error fields: %v", rec)
	}
	m, ok := rec["any"].(map[string]any)
	if !ok || m["x"].(float64) != 1 {
		t.Fatalf("bad Any field: %v", rec["any"])
	}
}

func TestLogger_ErrorWithNil(t *testing.T) {
	var buf bytes.Buffer
	log := Text(WithOutput(&buf), WithLevel(LevelDebug), withNow(fixedClock()))
	log.Error(nil, "no err")
	// Should not panic and must still emit.
	if buf.Len() == 0 {
		t.Fatalf("expected output when err is nil")
	}
}

func TestLevel_String(t *testing.T) {
	cases := map[Level]string{
		LevelDebug: "DEBUG",
		LevelInfo:  "INFO",
		LevelWarn:  "WARN",
		LevelError: "ERROR",
		LevelOff:   "OFF",
	}
	for l, want := range cases {
		if l.String() != want {
			t.Fatalf("Level(%d).String() = %q, want %q", l, l.String(), want)
		}
	}
}

func TestLogger_ConcurrentWrite(t *testing.T) {
	var buf bytes.Buffer
	log := JSON(WithOutput(&buf), WithLevel(LevelDebug), withNow(fixedClock()))

	const workers, per = 8, 50
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < per; j++ {
				log.Info("msg", Int("w", id), Int("j", j))
			}
		}(i)
	}
	wg.Wait()

	// Every line must be independently parseable JSON (no interleaved writes).
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != workers*per {
		t.Fatalf("want %d lines, got %d", workers*per, len(lines))
	}
	for i, line := range lines {
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("line %d not valid JSON: %v (%q)", i, err, line)
		}
	}
}

func TestText_QuotesValuesWithSpaces(t *testing.T) {
	var buf bytes.Buffer
	log := Text(WithOutput(&buf), WithLevel(LevelDebug), withNow(fixedClock()))
	log.Info("msg", String("path", "/tmp/with space"))
	assertContains(t, buf.String(), `path="/tmp/with space"`)
}
