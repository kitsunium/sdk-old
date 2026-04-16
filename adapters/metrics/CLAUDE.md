<!-- updated: 2026-04-16T00:00:00Z -->
# adapters/metrics

Implementations of `ports/metrics.Exporter`. One subdirectory per backend.

## Planned landings (phases 8-9 of the roadmap)

| Backend | Package path | Depends on |
|---|---|---|
| stdout pretty-print | `adapters/metrics/console` | stdlib only |
| Prometheus text exposition | `adapters/metrics/prometheus` | stdlib + net/http |
| StatsD UDP line protocol | `adapters/metrics/statsd` | stdlib only |
| AWS CloudWatch metrics | `adapters/metrics/cloud/aws/cloudwatch` | `aws-sdk-go-v2` + `adapters/shared/aws` |
| GCP Monitoring | `adapters/metrics/cloud/gcp/monitoring` | `cloud.google.com/go/monitoring` |

## Contract

Every sub-package exposes a constructor returning an implementation of `ports/metrics.Exporter` (+ optionally `common.Opener`/`Closer`/`Flusher`). Exporters are batch-oriented — implementations MUST respect the caller's ctx deadline and return `context.DeadlineExceeded` when a batch cannot be flushed in time.
