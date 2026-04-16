<!-- updated: 2026-04-16T00:00:00Z -->
# v1/adapters

Concrete driver implementations of the port contracts. Adapters are
the **only** place in the SDK that talks to the outside world
(stdout, filesystem, syslog, cloud APIs).

## Contract

| | Rule |
|---|---|
| **Imports allowed** | `ports/*`, `internal/*`, stdlib, vetted 3rd-party |
| **Imports forbidden** | `components/*` |
| **Shape** | One subdirectory per backend: `logger/console/`, `logger/file/`, `logger/cloud/aws/s3/` |
| **Naming** | Constructor returns the port type: `console.Sink(w io.Writer) logger.Sink` |

## Subtrees

| Subtree | Purpose |
|---|---|
| [`logger/`](./logger) | Implementations of `ports/logger.Sink` |
| [`config/`](./config) | Implementations of `ports/config.Source` and `.Watcher` (future) |
| [`shared/`](./shared) | Cross-cutting backends (AWS client, HTTP client) reused by multiple adapter families |

## Cloud taxonomy

Cloud adapters are grouped by vendor, then by service:

```
adapters/logger/cloud/
├── aws/
│   ├── cloudwatch/
│   └── s3/
├── gcp/
│   └── logging/
└── grafana/
    └── loki/
```

A cloud adapter depends on `adapters/shared/<vendor>` for the HTTP
client, credentials chain, retry wiring — the `shared/` tree exists
precisely because CloudWatch, S3, and future AWS services share
authentication and transport.

## Rules

1. An adapter is **one backend**. Never branch on env vars to pick
   behaviour — that's two adapters wearing one hat.
2. Lifecycle (`Open/Close/Flush`) must be idempotent and context-
   aware.
3. External deps are pinned in the root `go.mod` (no per-adapter
   replace directives).
4. Adapters produce errors via their component's error catalog
   (`errs.Define` in the adapter package). Never expose vendor-
   specific error types across the port boundary.

## Validation

```bash
cd v1 && go test -race ./adapters/...
bazel test //v1/adapters/...
```
