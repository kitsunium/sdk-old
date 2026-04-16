<!-- updated: 2026-04-16T00:00:00Z -->
# adapters

**Public implementations of the `ports/*` interfaces.** Each adapter is grouped by the component it serves so the tree scales as new backends arrive.

## Layout

```
adapters/
├── logger/          implementations of ports/logger.Sink
│   ├── console/     stdout/stderr text/JSON
│   ├── syslog/      local + remote RFC 5424
│   ├── file/        rotation, compression
│   ├── stream/      io.Writer generic (tests, bridging)
│   └── cloud/       aws/cloudwatch, gcp/cloudlogging, …
├── metrics/         implementations of ports/metrics.Exporter
│   ├── console/
│   ├── prometheus/
│   ├── statsd/
│   └── cloud/       aws/cloudwatch, gcp/monitoring, …
└── shared/          cross-cutting backends (auth, clients) reused by
                     multiple adapter families
    └── aws/         single AWS SDK client consumed by both
                     adapters/logger/cloud/aws and
                     adapters/metrics/cloud/aws
```

## Dependency contract

- Imports allowed: `ports/*`, `internal/*`, Go stdlib, vetted 3rd-party.
- Imports forbidden: `components/*`. This is the inversion-of-control seam — components consume ports, adapters implement them, neither knows the other.
- Enforced by `internal/core/contract/arch_external_test.go`.

## Rules

1. **Granularity**: one adapter = one backend. Console is `adapters/logger/console/`, not `adapters/logger/console_json/` + `adapters/logger/console_text/`. Format variants are functional options on the adapter's config.
2. **Naming**: adapter package name = rightmost path segment (`console`, `cloudwatch`). No `New*` prefix on constructors.
3. **Lifecycle**: adapters that open I/O MUST implement `common.Opener` + `common.Closer`; buffered adapters MUST implement `common.Flusher`.
4. **External deps**: allowed per adapter. Keep each cloud/provider adapter isolated so users who don't import it don't pay the dependency cost.
5. **Cross-component backends**: if an adapter serves multiple components (logger + metrics from the same AWS client), put the shared machinery under `adapters/shared/` and have each component-facing adapter depend on it.

## Validation

```bash
bazel build //adapters/...
bazel test //adapters/...
```
