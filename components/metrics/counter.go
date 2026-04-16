// Package metrics - counter implementation.
//
// The counter stores its float64 value in the bits of a [sync/atomic.Uint64]
// so increments are lock-free under contention. Negative deltas are
// rejected because they violate the monotonic contract of a counter.
package metrics

import (
	"math"
	"sync/atomic"
)

// counter is the concrete [Counter] implementation.
type counter struct {
	desc Descriptor
	bits atomic.Uint64
}

// Counter returns a new thread-safe counter named `name`.
func Counter(name string, opts ...Option) *counter {
	cfg := defaultConfig()
	apply(cfg, opts)
	return &counter{
		desc: Descriptor{
			Name:   name,
			Help:   cfg.help,
			Labels: cloneLabels(cfg.labels),
		},
	}
}

// Name returns the counter's name.
func (c *counter) Name() string { return c.desc.Name }

// Describe returns a copy of the counter's descriptor.
func (c *counter) Describe() Descriptor {
	return Descriptor{
		Name:   c.desc.Name,
		Help:   c.desc.Help,
		Labels: cloneLabels(c.desc.Labels),
	}
}

// Inc adds 1 to the counter atomically.
func (c *counter) Inc() { c.Add(1) }

// Add atomically adds delta to the counter. Negative deltas are
// silently dropped to preserve counter monotonicity.
func (c *counter) Add(delta float64) {
	if delta < 0 || math.IsNaN(delta) {
		return
	}
	for {
		old := c.bits.Load()
		next := math.Float64bits(math.Float64frombits(old) + delta)
		if c.bits.CompareAndSwap(old, next) {
			return
		}
	}
}

// Value returns the current counter value.
func (c *counter) Value() float64 {
	return math.Float64frombits(c.bits.Load())
}
