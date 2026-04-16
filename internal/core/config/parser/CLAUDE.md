<!-- updated: 2026-04-16T00:00:00Z -->
# pkg/core/config/parser

Multi-format configuration parsers sharing a common `baseParser` (via `ParserOption`).

## Files

| File | Role |
|---|---|
| `interface.go` | `Parser`, `FileParser` interfaces; `ParserOption`, `WithBufferSize(int)`, `WithPool(bool)`; private `baseParser` |
| `errors.go` | Package-local `errs.Define(...)` registrations — single source of parser error identity |
| `json.go` | `type JSON`, `NewJSON(path, opts...) *JSON` |
| `yaml.go` | `type YAML`, `NewYAML(path, opts...) *YAML` |
| `toml.go` | `type TOML`, `NewTOML(path, opts...) *TOML` |
| `ini.go` | `type INI`, `NewINI(path, opts...) *INI` |
| `xml.go` | `type XML`, `NewXML(path, opts...) *XML` |
| `env.go` | `type ENV`, `NewENV(prefix) *ENV` — reads `os.Environ()` filtered by prefix |
| `args.go` | `type ARGS`, `NewARGS(skipFirst bool) *ARGS` — parses `os.Args` |

## Interface surface

```go
type Parser interface {
    Type() string
    Load() (map[string]string, error)
}

type FileParser interface {
    Parser
    LoadReader(r io.Reader) (map[string]string, error)
}
```

All file-format types (JSON/YAML/TOML/INI/XML) implement `FileParser`. `ENV` and `ARGS` implement `Parser` only.

## Rules

1. **Uppercase type names** (`JSON`, `YAML`, not `Json`, `Yaml`) — matches the target API `parser.YAML.Load(...)` in roadmap §16.
2. **Errors**: define once in `errors.go` with `errs.Define(KConfig{Package:"parser", ...})`. Never inline `fmt.Errorf` at call sites.
3. **Buffering**: the `WithPool(true)` option routes reads through `pkg/kernel/pool` — keep parity across all file parsers.
4. **Map output**: `Load` always returns `map[string]string`. Nested structures are flattened with dot-notation keys by the parser; normalization is the `normalize` package's job, not this one.
5. A new format = new file + new type following the template exactly (constructor signature, `Type()`, `Load()`, `LoadReader(io.Reader)`).
6. Target naming migration (roadmap §16): `NewJSON(path)` → `parser.JSON.Load(path)`. Pending the refactor, prefer the existing form and do not introduce a third spelling.

## Validation

```bash
bazel test //pkg/core/config/parser/...
go test -race ./pkg/core/config/parser
```
