// Package metrics - gauge implementation.
//
// The gauge stores its float64 value in the bits of a [sync/atomic.Uint64]
// which allows lock-free reads, atomic replacement via Set, and CAS-based
// arithmetic via Add.
package metrics

import (
	"math"
	"sync/atomic"
)

// gauge is the concrete [Gauge] implementation.
type gauge struct {
	desc Descriptor
	bits atomic.Uint64
}

// Gauge returns a new thread-safe gauge named `name`.
func Gauge(name string, opts ...Option) *gauge {
	cfg := defaultConfig()
	apply(cfg, opts)
	return &gauge{
		desc: Descriptor{
			Name:   name,
			Help:   cfg.help,
			Labels: cloneLabels(cfg.labels),
		},
	}
}

// Name returns the gauge's name.
func (g *gauge) Name() string { return g.desc.Name }

// Describe returns a copy of the gauge's descriptor.
func (g *gauge) Describe() Descriptor {
	return Descriptor{
		Name:   g.desc.Name,
		Help:   g.desc.Help,
		Labels: cloneLabels(g.desc.Labels),
	}
}

// Set replaces the gauge value atomically.
func (g *gauge) Set(v float64) {
	g.bits.Store(math.Float64bits(v))
}

// Add atomically adds delta to the gauge value via CAS.
func (g *gauge) Add(delta float64) {
	if math.IsNaN(delta) || delta == 0 {
		return
	}
	for {
		old := g.bits.Load()
		next := math.Float64bits(math.Float64frombits(old) + delta)
		if g.bits.CompareAndSwap(old, next) {
			return
		}
	}
}

// Value returns the current gauge value.
func (g *gauge) Value() float64 {
	return math.Float64frombits(g.bits.Load())
}
