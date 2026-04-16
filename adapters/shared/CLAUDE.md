<!-- updated: 2026-04-16T00:00:00Z -->
# adapters/shared

Cross-cutting backend helpers reused by multiple adapter families. Typical case: a single authenticated SDK client that feeds both a `logger.Sink` and a `metrics.Exporter` for the same cloud provider.

## Rule of thumb

If a piece of code is specific to one backend AND to one component (logger XOR metrics), it lives under `adapters/<component>/<backend>/`. If it is specific to one backend BUT shared across components, it lives here under `adapters/shared/<backend>/`.

## Planned landings

| Shared | Package path | Serves |
|---|---|---|
| AWS SDK client bootstrap + credentials | `adapters/shared/aws` | `adapters/logger/cloud/aws/*`, `adapters/metrics/cloud/aws/*` |
| GCP client bootstrap | `adapters/shared/gcp` | `adapters/logger/cloud/gcp/*`, `adapters/metrics/cloud/gcp/*` |
| OTEL trace-id / span-id plumbing | `adapters/shared/otel` | future tracing + logger/metrics correlation |

## Dependency contract

Same as `adapters/` at large: MAY use `ports/*`, `internal/*`, stdlib, 3rd-party; MUST NOT import `components/*`.
