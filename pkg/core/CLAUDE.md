<!-- updated: 2026-04-16T00:00:00Z -->
# pkg/core

Composable bricks built on top of `pkg/kernel/*`. Prefer kernel wrappers over stdlib where one exists (e.g. use `pool` for buffers, `errs` for error construction).

## Dependency contract

- Imports allowed: `pkg/kernel/*`, Go stdlib.
- Imports forbidden: `pkg/component/*`, non-stdlib deps without explicit PR approval.
- Reason for using a stdlib primitive directly instead of a kernel wrapper must be called out in the PR.

## Packages

| Path | Package | Purpose | Public entry points |
|---|---|---|---|
| [`config/parser`](./config/parser) | `parser` | Multi-format configuration parsers | `JSON`, `YAML`, `TOML`, `INI`, `XML`, `ENV`, `ARGS`, `Parser`, `FileParser`, `ParserOption`, `WithBufferSize`, `WithPool` |
| [`config/normalize`](./config/normalize) | `normalize` | Key/value normalization, zero-alloc string↔bytes conversion | `Key`, `Value`, `Map`, `StringToBytesSafe`, `BytesToStringSafe` |

### `config/parser` — file format matrix

| Format | Type | Constructor (legacy) | Target API |
|---|---|---|---|
| JSON | `*JSON` | `parser.NewJSON(path, opts...)` | `parser.JSON.Load(path, opts...)` |
| YAML | `*YAML` | `parser.NewYAML(path, opts...)` | `parser.YAML.Load(path, opts...)` |
| TOML | `*TOML` | `parser.NewTOML(path, opts...)` | `parser.TOML.Load(path, opts...)` |
| INI | `*INI` | `parser.NewINI(path, opts...)` | `parser.INI.Load(path, opts...)` |
| XML | `*XML` | `parser.NewXML(path, opts...)` | `parser.XML.Load(path, opts...)` |
| ENV | `*ENV` | `parser.NewENV(prefix)` | `parser.ENV(prefix)` |
| ARGS | `*ARGS` | `parser.NewARGS(skipFirst)` | `parser.ARGS(...)` |

All file parsers share a common `baseParser` wiring through functional options (`WithBufferSize`, `WithPool`). Pooling uses `pkg/kernel/pool`.

## Conventions

1. **Error construction**: use `errs.Define(...)` at package init, then `Instance(...)` at call sites. Don't introduce new `fmt.Errorf` chains in new code.
2. **I/O buffering**: always go through `pool.GetBuffer` / `pool.PutBuffer` on the hot path; no raw `bytes.Buffer` for streaming reads.
3. **No component imports**: `pkg/core/*` must never import `pkg/component/*`. If you need a logger here, pass it as an interface parameter.
4. **Naming**: the `parser` package uses uppercase type names (`JSON`, `YAML`) as exposed types — keep this style; don't rename to `Json`/`Yaml`.

## Planned — not yet implemented

The following subpackages are referenced in older task metadata but were **removed** during the Bazel 9 / kernel-rename refactor:
- `core/observability/*` — moved to `pkg/component/{logger,metrics}`
- `core/storage/*` — deleted (stub)
- `core/serveur/*` — pending, will land as `pkg/component/server` instead

If any stale tooling still references these paths, update it to the component layer targets.

## Validation

```bash
bazel build //pkg/core/...
bazel test //pkg/core/...
```
