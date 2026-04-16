<!-- updated: 2026-04-16T00:00:00Z -->
# pkg/core/config

Configuration primitives: file/env/args parsers + key/value normalization. All children stay within the `core` layer dependency contract (kernel + stdlib only).

## Subpackages

| Path | Package | Role |
|---|---|---|
| [`parser`](./parser) | `parser` | JSON / YAML / TOML / INI / XML / ENV / ARGS loaders, shared `baseParser` via `ParserOption` |
| [`normalize`](./normalize) | `normalize` | Canonical key/value formatting, flattening, zero-alloc string↔bytes |

## Typical flow

```go
p := parser.NewYAML("config.yaml", parser.WithPool(true))
raw, err := p.Load()                    // map[string]string
clean := normalize.Map(rawAny)          // normalized keys + stringified values
```

## Rules

1. **Buffering** on file parsers goes through `pkg/kernel/pool` when `WithPool(true)`.
2. **Errors** constructed here must use `pkg/kernel/errs` — no raw `errors.New` / `fmt.Errorf` in new code.
3. Do not import `pkg/component/*` from anywhere under `config/`.
4. A new file format = new file in `parser/` following the existing `JSON/YAML/TOML/INI/XML` template (type struct, `NewXxx`, `Type()`, `Load()`, `LoadReader(io.Reader)`).

## Validation

```bash
bazel test //pkg/core/config/...
go test -race ./pkg/core/config/...
```
