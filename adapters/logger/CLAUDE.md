<!-- updated: 2026-04-16T00:00:00Z -->
# adapters/logger

Implementations of `ports/logger.Sink`. One subdirectory per backend.

## Planned first landings (phase 5 of the roadmap)

| Backend | Package path | Depends on |
|---|---|---|
| stdout / stderr (text + JSON) | `adapters/logger/console` | stdlib only |
| `io.Writer` pass-through | `adapters/logger/stream` | stdlib only |
| local + remote RFC 5424 syslog | `adapters/logger/syslog` | `log/syslog` + net |
| rotating file | `adapters/logger/file` | stdlib + `internal/kernel/files` |
| AWS CloudWatch Logs | `adapters/logger/cloud/aws/cloudwatch` | `aws-sdk-go-v2` + `adapters/shared/aws` |
| GCP Cloud Logging | `adapters/logger/cloud/gcp/cloudlogging` | `cloud.google.com/go/logging` |

## Contract

Every sub-package exposes a constructor returning an implementation of `ports/logger.Sink` (+ optionally `common.Opener`/`Closer`/`Flusher`). Config is a single struct with functional options.
