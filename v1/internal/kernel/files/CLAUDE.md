<!-- updated: 2026-04-16T00:00:00Z -->
# internal/kernel/files

Filesystem abstractions: files, directories, stats, host/system. **POSIX-only** (`//go:build !windows`).

## Files

| File | Role | Build tag |
|---|---|---|
| `option.go` | `Path` named type + `Option` struct; `Path.Clean`, `Path.Parent`, `Option.Validate` | — |
| `file.go` | `File` interface + `NewFile(Option) (File, error)` | `!windows` |
| `directory.go` | `Directory` interface + `NewDirectory(Option) (Directory, error)` | `!windows` |
| `stats.go` | `Stats` interface + `UserInfo`, `GroupInfo`, `OtherInfo`, `Permissions`; `NewStats(path)` | `!windows` |
| `system.go` | `System` interface — filesystem-level introspection | — |
| `host.go` | Host-specific helpers (uid/gid package-level vars) | — |

## Interface surface (compact)

```go
type File interface {
    Path() string
    Parent() (Directory, error)
    Create() (File, error)
    Remove() error
    Write(data []byte) (int, error)
    Read() ([]byte, error)
    Copy(dst string) error
    Move(dst string) error
    Exists() bool
    Size() int64
    IsDotFile() bool
}

type Directory interface {
    Path() string
    Parent() (Directory, error)
    Create() (Directory, error)
    Remove() error
    Exists() bool
    Has(string) bool
    Size() int64
    List() ([]File, []Directory, error)
}
```

## Rules

1. **Always construct via `Option`**: `NewFile(Option{Path: Path(p)})` — raw string paths are not accepted by design.
2. `Option.Validate()` is the single gate for path-level invariants; extend it rather than adding ad-hoc checks.
3. `Path` is a `string` subtype — keep it distinct to catch unvalidated string paths at compile time.
4. Target naming migration (roadmap §16): `NewFile(Option{Path:p})` → `files.File(p, opts...)`, `NewStats(p)` → `files.Stats(p)`.

## Stdlib contract

- **Pure stdlib**: the package uses only `os`, `io`, `os/user`, `path/filepath`, `sync`, and `syscall` (for `syscall.Stat_t` / `syscall.EINVAL` behind the `!windows` build tag). The previous `golang.org/x/sys/unix` dependency has been removed from `go.mod`.
- **POSIX-only files** — `file.go`, `directory.go`, `stats.go` carry `//go:build !windows` because they rely on `syscall.Stat_t` fields (`Uid`, `Gid`, `Mode`) that exist on POSIX only. A Windows counterpart would live in `*_windows.go` files if/when needed.

## Deliberately absent

Removed during the 2026-04 rebuild (see `/workspace/.claude/contexts/kernel-files-audit.md`):
- `archive.go` / `archive_test.go` / `Archive` / `ArchiveOptions` / `NewArchive` — `Compress()` returned `nil` unconditionally (the function body was empty). Ten unused compression-type constants. Shipping was a correctness bug. Reintroduce only with a real implementation behind it.

## Validation

```bash
bazel test //internal/kernel/files/...
go test -race ./internal/kernel/files
```
