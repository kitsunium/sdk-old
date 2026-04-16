<!-- updated: 2026-04-16T00:00:00Z -->
# v1/ports

Public interface contracts shared by components and adapters.

## Contract

| | Rule |
|---|---|
| **Imports allowed** | Go stdlib only |
| **Imports forbidden** | `internal/`, `components/`, `adapters/` |
| **Content** | Interfaces, value types, enums. **No logic.** |
| **Stability** | Changing a port is a wire-level breaking change — only across `vN` boundaries |

## Packages

| Package | Purpose |
|---|---|
| [`common`](./common) | Lifecycle triad: `Opener`, `Closer`, `Flusher` |
| [`logger`](./logger) | `Sink`, `EntryEvent`, `Severity`, `Format` |
| [`config`](./config) | `Accessor`, `Source`, `Watcher`, `SourceID` |

## Design principle

Ports are **small and orthogonal**. A single adapter often implements
`Sink` AND `Opener` AND `Closer` — each interface stays minimal so
composition is cheap.

Re-exports of `common.*` inside `logger/` and `config/` exist for
ergonomics (one import instead of two), NOT to break the layering.

## Validation

```bash
cd v1 && go test ./ports/...
bazel test //v1/ports/...
```
