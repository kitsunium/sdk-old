<!-- updated: 2026-04-16T00:00:00Z -->
# internal/core

Composition on top of `internal/kernel/*`. **Private to the SDK** — Go enforces external-import rejection.

## Dependency contract

- Imports allowed: `internal/kernel/*`, Go stdlib, vetted third-party deps (yaml.v3, go-toml).
- Imports forbidden: `components/`, `adapters/`, `ports/`.
- Enforced by `internal/core/contract/arch_test.go`.

## Packages

| Path | Purpose |
|---|---|
| [`config/parser`](./config/parser) | Multi-format parsers (JSON, YAML, TOML, INI, XML, ENV, ARGS) |
| [`config/normalize`](./config/normalize) | Key/value normalization, zero-alloc string↔bytes |

## Planned

- `fanout/` — ring buffer + worker pool, shared by logger/metrics (Phase 4)
- `lifecycle/` — Open/Close pipeline helpers
- `contract/` — architecture enforcement + Component conformance tests

## Validation

```bash
bazel build //internal/core/...
bazel test //internal/core/...
```
