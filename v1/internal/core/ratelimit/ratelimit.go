// Package ratelimit provides rate-control primitives used by
// components to throttle or batch I/O. The first consumer is the
// logger, which uses a token bucket to switch between synchronous
// one-record-per-Write and batched delivery when the emit rate
// crosses a configured threshold.
//
// Two algorithms are planned: a standard token bucket for threshold
// decisions, and a leaky bucket for smoothed outbound traffic.
//
// Implementation pending — see CLAUDE.md for the spec.
package ratelimit
