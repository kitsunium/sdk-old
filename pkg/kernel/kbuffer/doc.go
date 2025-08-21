// Package kbuffer provides high-performance, zero-allocation buffer management
// with advanced pooling mechanisms for memory reuse.
//
// Features:
//   - Zero-allocation patterns in hot paths
//   - Lock-free buffer pools with power-of-2 size classes
//   - CPU cache-aligned data structures
//   - Atomic statistics tracking
//   - Security-hardened bounds checking
//   - Unsafe optimizations for maximum performance
package kbuffer
