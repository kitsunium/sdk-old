<!-- updated: 2026-04-16T00:00:00Z -->
# pkg/kernel/errs

Typed error catalog. Entries are registered once via `Define`, each carries a stable ID + (package, code) pair + default message. Catalog entries turn into concrete `Instance` objects at call sites with an optional cause, tags, and typed details.

## Files

| File | Role |
|---|---|
| `error.go` | `Config`, `Err`, `Define(Config) Err`, registry (sync.Map keyed on `(pkg, code)`), `clearRegistry()` test helper |
| `instance.go` | `Instance` with `New`/`Newf`/`Wrap`/`Wrapf`/`WithTag`/`WithDetail`/`Tag`/`Detail`/`Error`/`Unwrap`/`Is`/`As`/`Err`/`Package`/`Code`; generic `DetailAs[T]` |
| `context.go` | `ToContext`, `FromContext` (context-scoped instance propagation) |
| `error_test.go` | Unit tests for the full public surface |

## Typical use

```go
// Define once at init time
var ErrParse = errs.Define(errs.Config{
    Package: "parser", Code: 1, Message: "invalid input",
})

// Build an Instance at the call site
inst := ErrParse.Wrap(raw).WithTag("path", p).WithDetail("size", n)

// Propagate via context
ctx = errs.ToContext(ctx, inst)

// Extract downstream
if got, ok := errs.FromContext(ctx); ok {
    size, _ := errs.DetailAs[int](got, "size")
    _ = size
}
```

## Rules

1. **Register once per catalog entry**: one `errs.Define(...)` in an `init()` or package-var initializer. Duplicates panic — this signals a programmer error, not a runtime condition.
2. **No `fmt.Errorf` in new code** across the SDK — wrap via `errX.Wrap(err)` / `errX.Wrapf(err, ...)`.
3. `clearRegistry` is package-private (test-only). Never call from production.
4. `Instance` is not pooled — callers do not need to call any `Release()`. Keep it that way until a benchmark proves pooling is needed with documented lifecycle.
5. `ToContext(ctx, nil)` returns `ctx` unchanged; `FromContext(nil)` returns `(nil, false)`.
6. `DetailAs[T](i, key)` is the only type-safe accessor for `Detail` values — never cast `any` by hand.
7. Naming (roadmap §16): types are `Err` / `Config`. Do not reintroduce the former `KError` / `KConfig` names.

## Deliberately absent

Removed during the 2026-04 rebuild (see `/workspace/.claude/contexts/kernel-errs-audit.md`):
- `Result[T]`, `NewResult` — ambiguous wrapper, replaced by idiomatic `(T, error)`.
- `MetricsCollector` / `SetMetricsCollector` / `GetMetricsSnapshot` — belongs in `pkg/component/metrics`, not here.
- `GlobalConfig` / `Configure` / `GetConfig` — unused runtime knobs.
- `ExtractTraceID` / `ExtractSpanID` — stubs returning `""`.
- `WithContext` / `Context()` — context stored in a struct is a Go anti-pattern; use `ToContext`/`FromContext`.
- `Clone()`, JSON `MarshalJSON`/`UnmarshalJSON` on `Err`/`Instance`, `Batch*`/`Map*`/`Filter*`/`Merge*` tag helpers, `ListErrors`/`ListPackages`/`GetError`/`ValidatePackageCode` registry queries, per-error stack traces — zero production call-sites.

## Validation

```bash
bazel test //pkg/kernel/errs/...
go test -race ./pkg/kernel/errs
```
