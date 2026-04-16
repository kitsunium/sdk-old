<!-- updated: 2026-04-16T00:00:00Z -->
# v1/internal/kernel/syscallx

Direct Linux syscall wrappers. **This is where the SDK earns its
Linux-only tax**: we bypass the portable `syscall` / `os` wrappers
when a vDSO path or a kernel-only feature gives us a real speedup.

## Planned API

```go
// Epoll: non-blocking fd multiplexing.
type Epoll struct{ /* ... */ }
func EpollNew() (*Epoll, error)
func (e *Epoll) Add(fd int, events uint32) error
func (e *Epoll) Wait(timeoutMs int) ([]Event, error)
func (e *Epoll) Close() error

// Futex: contention primitive.
func FutexWait(addr *uint32, want uint32, timeout time.Duration) error
func FutexWake(addr *uint32, n int) int

// Mmap: arena-backed allocation.
func Mmap(size int, flags int) ([]byte, error)
func Munmap(b []byte) error

// IoUring (roadmap): batched syscall submission.
// See https://kernel.dk/io_uring.pdf.
```

## Rules

1. **Linux only**, Debian-based preferred. No portability shims — if
   a helper has a non-Linux counterpart, it doesn't belong here.
2. Every wrapper returns a structured `errs.Instance`, never raw
   `errno` ints.
3. Benchmarks mandatory: `syscallx_bench_test.go`.
4. Prefer vDSO-backed calls when available (`time.Now`, `getcpu`).
5. `io_uring` integration comes after the initial `epoll` path is
   benched and stable.
