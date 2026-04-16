<!-- updated: 2026-04-16T00:00:00Z -->
# internal/core/config/normalize

Canonical key/value formatting for configuration maps, plus zero-allocation `string↔[]byte` conversions used on the hot path.

## Files

| File | Role |
|---|---|
| `normalize.go` | `Key(string) string`, `Value(string) string`, `Map(map[string]any) map[string]string`, `StringToBytesSafe(string) []byte`, `BytesToStringSafe([]byte) string` |

## Public API

| Symbol | Purpose |
|---|---|
| `Key(k)` | Canonicalize a configuration key (lowercase, dot-separated, trimmed) |
| `Value(v)` | Canonicalize a value (trim, expand env refs when applicable) |
| `Map(m)` | Normalize and flatten a heterogeneous `map[string]any` into `map[string]string` |
| `StringToBytesSafe(s)` | Zero-alloc view over a string's bytes — **read-only** |
| `BytesToStringSafe(b)` | Zero-alloc view over a byte slice as string — **read-only** |

## Rules

1. **`StringToBytesSafe` / `BytesToStringSafe` return aliases into the same memory**. Never mutate the result, never retain it past the lifetime of the source. If you need to mutate, `append([]byte(nil), s...)` first.
2. The `Key` canonical form is **lowercase**, segments joined with `.`, no leading/trailing separators. Keep this invariant when extending.
3. `Map` is the sole entry point for flattening — do not hand-roll traversal at call sites.
4. No external deps, no `reflect` on the hot path (the zero-alloc converters use `unsafe` through the `runtime` headers pattern; any rewrite must be benchmarked).
5. Changes to the zero-alloc converters must stay safe under `-race`.

## Validation

```bash
bazel test //internal/core/normalize/...
go test -race -bench=. -benchmem ./internal/core/config/normalize
```
