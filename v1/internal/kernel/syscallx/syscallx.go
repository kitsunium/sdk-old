//go:build linux

// Package syscallx exposes direct Linux syscall wrappers the SDK
// relies on for high-performance I/O. The current focus is epoll for
// non-blocking socket I/O, futex for contention primitives, and
// mmap for arena-backed allocations.
//
// io_uring (kernel >= 5.1) is the future-looking target — see the
// roadmap in CLAUDE.md.
//
// Implementation pending — see CLAUDE.md for the spec.
package syscallx
