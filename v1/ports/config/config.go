// Package config holds the port contracts the config component declares
// and adapters implement. Types here are configuration-domain specific
// (Accessor, Source, Watcher); cross-cutting lifecycle interfaces live
// in ports/common and compose orthogonally.
package config

import (
	"context"
	"time"

	"github.com/kitsunium/sdk/v1/ports/common"
)

// SourceID is a stable identifier for a configuration source (e.g.
// "file:/etc/app/config.yaml", "env:APP_", "args"). Repositories use
// it to dedup, to order, and to report provenance in diagnostics.
type SourceID string

// Accessor is the read-only contract consumers depend on. It exposes
// the flattened, normalized key-value namespace assembled from all
// configured sources. All lookups take a dot-separated path
// ("database.host", "server.read.timeout") and return a value plus an
// "ok" presence flag — zero values are indistinguishable from missing
// keys without the flag.
//
// Implementations MUST be safe for concurrent use by multiple
// goroutines. Accessor is the primary seam for mocking in tests; see
// components/config.FromStatic for an in-memory implementation.
type Accessor interface {
	// String returns the value at path as a string.
	//
	// Params:
	//   - path: dot-separated, normalized key
	//
	// Returns:
	//   - v: the stored value, or "" when absent
	//   - ok: true when the key was present
	String(path string) (v string, ok bool)

	// Int returns the value parsed as int.
	Int(path string) (v int, ok bool)

	// Int64 returns the value parsed as int64.
	Int64(path string) (v int64, ok bool)

	// Float returns the value parsed as float64.
	Float(path string) (v float64, ok bool)

	// Bool returns the value parsed as a Go boolean.
	Bool(path string) (v, ok bool)

	// Duration returns the value parsed via time.ParseDuration.
	Duration(path string) (v time.Duration, ok bool)

	// Strings returns the value split on comma with surrounding
	// whitespace trimmed. Useful for "allow_origins" style lists.
	Strings(path string) (v []string, ok bool)

	// Has reports whether path is present.
	Has(path string) bool

	// Walk iterates every key starting with prefix. The callback may
	// return false to stop iteration early. The iteration order is
	// unspecified but stable within a single snapshot.
	Walk(prefix string, fn func(k, v string) bool)

	// Decode unmarshals all keys under prefix into target using
	// `cfg:"path"` struct tags. Uses reflection; OFF the hot path.
	Decode(prefix string, target any) error
}

// Source is implemented by anything that can produce a flat
// map[string]string of normalized keys. The component assembles
// multiple Sources into a single Accessor.
//
// Sources are loaded in the order passed to the component, with LATER
// sources OVERRIDING earlier ones on key collision.
type Source interface {
	// ID identifies the source for diagnostics and deduplication.
	ID() (id SourceID)

	// Load returns the source's key/value payload. Implementations
	// MUST honour ctx.Err and return promptly on cancellation.
	Load(ctx context.Context) (values map[string]string, err error)
}

// Watcher is implemented by adapters that can signal when the
// underlying source has changed (inotify, SIGHUP, remote config
// revision bump). The component calls Reload on the signal.
//
// Watcher is optional — sources that cannot change at runtime need
// not implement it.
type Watcher interface {
	// Watch blocks until ctx is cancelled, invoking onChange every
	// time the source mutates. onChange MUST be safe to call from any
	// goroutine.
	Watch(ctx context.Context, onChange func()) (err error)
}

// Re-exported lifecycle interfaces for ergonomics.
type (
	// Opener is aliased from common.Opener.
	Opener = common.Opener
	// Closer is aliased from common.Closer.
	Closer = common.Closer
	// Flusher is aliased from common.Flusher.
	Flusher = common.Flusher
)
