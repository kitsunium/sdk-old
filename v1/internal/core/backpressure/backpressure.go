// Package backpressure implements flow-control primitives used by
// adapters that must respect a downstream throughput cap: credit-
// based windows, token replenishment on ACK, and AIMD-style capacity
// adjustment.
//
// Implementation pending — see CLAUDE.md for the spec.
package backpressure
