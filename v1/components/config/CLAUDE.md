<!-- updated: 2026-04-16T00:00:00Z -->
# components/config — Cahier des charges

Unified configuration component. Loads from **any supported source**
(args, env, file formats: JSON/YAML/TOML/INI/XML), normalizes into a
single flat key-value namespace with dot-separated keys, and exposes a
**string-path accessor API**: give it `"database.host"` and it returns
the value regardless of which source carried it.

This component was historically a private `internal/core/config/*`.
**Promoted** to a public component in v1 because configuration is a
product callers directly consume.

## 1.0 Scope & non-scope

**In scope**
- Multi-format parsers: JSON, YAML, TOML, INI, XML.
- Dynamic sources: OS environment, command-line args.
- Normalized key namespace: lowercase, dot-separated, underscore→dot.
- Merge of multiple sources with explicit precedence.
- String-path accessor: `cfg.String("database.host")`,
  `cfg.Int("server.port")`, `cfg.Bool("tls.enabled")`.
- Mockable for tests — the `ports/config.Accessor` interface is the
  only contract consumers depend on.
- (stretch) `Watcher` port for hot-reload via SIGHUP / file events.

**Out of scope**
- Schema validation — callers validate after `Accessor.Decode`.
- Secrets decryption — a future `adapters/config/sops` plugs in.
- Reflection-based struct decoding — off by default; opt-in only.

## 2.0 DDD structure

```
components/config/
├── domain/                      # value objects (no I/O)
│   ├── key.go                   # Key type — dot-path
│   ├── value.go                 # Value type — string + typed getters
│   └── source.go                # SourceID value object
├── application/                 # use cases
│   ├── registry.go              # Repository: merge sources by precedence
│   ├── accessor.go              # Accessor implementation
│   └── watcher.go               # optional Watcher for hot-reload
├── parser/                      # format parsers (already-migrated code)
│   ├── parser.go                # Parser + FileParser interfaces
│   ├── errors.go                # errs.Define catalog
│   ├── args.go                  # ARGS
│   ├── env.go                   # ENV
│   ├── json.go, yaml.go, toml.go, ini.go, xml.go
│   └── testdata/                # sample inputs for tests
├── ports/                       # re-exports from ports/config
│   └── ports.go
├── config.go                    # facade — `config.FromSources(...)`
└── CLAUDE.md                    # this file
```

## 3.0 Public API (target v1.0.0)

```go
// Facade — build a Repository from a list of sources.
// Earlier sources have LOWER precedence; later ones OVERRIDE.
cfg, err := config.FromSources(
    config.File("config.yaml"),           // base
    config.File("config.local.yaml"),     // local overrides
    config.Env("APP_"),                   // APP_DATABASE_URL → "database.url"
    config.Args(true),                    // --database.host=... wins
)

// String-path accessor — the star of the API.
host, ok := cfg.String("database.host")
port, ok := cfg.Int("server.port")         // parses, returns 0 + false on miss
tls, ok  := cfg.Bool("tls.enabled")
dur, ok  := cfg.Duration("timeout.read")   // via time.ParseDuration
strs, ok := cfg.Strings("allow.origins")   // comma-split

// Structured decode (opt-in, uses reflect — NOT on hot path).
var s struct {
    Host string `cfg:"database.host"`
    Port int    `cfg:"server.port"`
}
err := cfg.Decode("database", &s)

// Iterate every key under a prefix (no allocation beyond the callback).
cfg.Walk("feature.", func(k string, v string) bool { ... })

// Reload — rebuild the Repository from the same sources. Zero-downtime.
err := cfg.Reload(ctx)

// Optional watcher port — an adapter (e.g. adapters/config/fsnotify)
// calls Reload on file changes.
cfg.Watch(ctx, watcher)
```

## 4.0 Source precedence model

Precedence = **last-source-wins**. Each `config.FromSources(...)` call
pins the order at startup; later `Reload` rebuilds with the same list.

Canonical pattern:

```go
// 1. defaults baked in the binary (lowest)
// 2. /etc/app/config.yaml
// 3. $HOME/.config/app/config.local.yaml
// 4. APP_* environment
// 5. --flag=... command-line (highest)
```

This matches the XDG + 12-factor convention.

## 5.0 Key normalization

Authoritative rule (see `internal/core/normalize`):

- Lowercase every ASCII letter.
- Replace `_` and `-` with `.` (dot).
- No leading/trailing dots.
- Dot-separated segments only.

Examples:

| Input (source) | Normalized key |
|---|---|
| `DATABASE_URL` (env) | `database.url` |
| `--server-port=8080` (args) | `server.port` |
| `{"database": {"host": "x"}}` (JSON) | `database.host` |
| `[database]\nhost = x` (INI) | `database.host` |

Consumers always see the normalized form. **Never** pass raw env-style
keys at the accessor API.

## 6.0 Ports

```go
// ports/config/config.go
type Accessor interface {
    String(path string) (v string, ok bool)
    Int(path string) (v int, ok bool)
    Int64(path string) (v int64, ok bool)
    Float(path string) (v float64, ok bool)
    Bool(path string) (v bool, ok bool)
    Duration(path string) (v time.Duration, ok bool)
    Strings(path string) (v []string, ok bool)
    Has(path string) bool
    Walk(prefix string, fn func(k, v string) bool)
    Decode(prefix string, target any) error
}

type Source interface {
    ID() SourceID
    Load(ctx context.Context) (map[string]string, error)
}

type Watcher interface {
    Watch(ctx context.Context, onChange func()) error
}

type SourceID string
```

## 7.0 Dependency rules

- `components/config` imports `ports/config`, `ports/common`,
  `internal/kernel/errs`, `internal/core/normalize`.
- **Not** allowed: `adapters/*`, any other component.
- Parser package (third-party `yaml.v3`, `pelletier/go-toml/v2`) is
  allowed because `components/` may pull vetted 3rd-party deps.

## 8.0 Performance targets

| Operation | Target | Measurement |
|---|---|---|
| `Accessor.String(existing key)` | ≤ 15 ns, 0 allocs | map lookup cached |
| `Accessor.Int(existing key)` | ≤ 50 ns, 0 allocs | parsed at load, cached |
| `FromSources` 4 files + env + args | ≤ 5 ms | one-off startup cost |
| `Reload` same sources | ≤ 5 ms, no downtime | atomic pointer swap |

Hot-path accessors MUST NOT allocate. Integer/bool/duration values are
pre-parsed at load time into a typed cache.

## 9.0 Mocking

For tests that need a Repository without touching disk or env:

```go
cfg := config.FromStatic(map[string]string{
    "database.host": "localhost",
    "server.port":   "8080",
})
```

`FromStatic` is a first-class `Source` that returns its map verbatim.

## 10.0 Error handling

All errors go through `components/config/parser.ErrXxx` +
`errs.Define`. No raw `fmt.Errorf` in new code. Wrap IO errors with
`ErrReadFailed.Wrap(err).WithTag("path", p)`.

## 11.0 Test plan

1. Unit per parser: JSON/YAML/TOML/INI/XML/ENV/ARGS golden tests.
2. Normalize: table-driven on the canonical form rules.
3. Precedence: overlap across 4 sources, verify last-wins.
4. Accessor: typed getters, missing keys, Walk, Decode.
5. Reload: atomic swap, no reader sees a torn view.
6. Race: `-race` everywhere.
7. Bench: accessor latency and zero-alloc invariant.

## 12.0 Validation

```bash
cd v1 && go test -race ./components/config/...
cd v1 && go test -bench=. -benchmem ./components/config/...
bazel test //v1/components/config/...
```
