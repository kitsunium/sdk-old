<!-- updated: 2026-04-16T00:00:00Z -->
# v1/ports/config

Port contract for the `components/config` component.

## API

```go
type Accessor interface {
    String(path string) (v string, ok bool)
    Int(path string) (v int, ok bool)
    Int64(path string) (v int64, ok bool)
    Float(path string) (v float64, ok bool)
    Bool(path string) (v, ok bool)
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

## Semantics

- **Path**: dot-separated, lowercase, pre-normalized. Consumers always
  see the normalized form — not raw `APP_DATABASE_URL`.
- **`ok` flag**: distinguishes missing keys from zero values. A
  lookup that returns `0, true` means "present, value is 0".
- **`Walk`**: iteration order is stable within a snapshot but
  unspecified across snapshots.
- **`Decode`**: opt-in, reflection-based, NOT on hot paths.
- **Source precedence**: LAST source wins on key collision. The
  component documents the canonical order in
  `components/config/CLAUDE.md`.

## Rules

1. This file MUST NOT import anything beyond Go stdlib and
   `ports/common`.
2. Changing a method signature breaks `vN` — only across major
   boundaries.
3. Adding a method is backwards-compatible for consumers but breaks
   every adapter — prefer narrow extension interfaces over bloat.

## Validation

```bash
cd v1 && go test ./ports/config/...
```
