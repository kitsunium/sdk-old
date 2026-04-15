// Package metrics - healthcheck implementation.
//
// A health check is a named wrapper around a user-supplied function
// `func(context.Context) Status`. The wrapper tolerates nil contexts
// and nil check functions so callers never need to add defensive
// nil-guards.
package metrics

import "context"

// healthcheck is the concrete [Healthcheck] implementation.
type healthcheck struct {
	desc  Descriptor
	check func(ctx context.Context) Status
}

// CheckFunc is the shape of a health-check evaluator.
type CheckFunc = func(ctx context.Context) Status

// Health returns a new health check named `name` that delegates to
// `check`. A nil check function is treated as an always-OK probe.
func Health(name string, check CheckFunc, opts ...Option) Healthcheck {
	cfg := defaultConfig()
	apply(cfg, opts)
	if check == nil {
		check = func(context.Context) Status { return OK("") }
	}
	return &healthcheck{
		desc: Descriptor{
			Name:   name,
			Help:   cfg.help,
			Labels: cloneLabels(cfg.labels),
		},
		check: check,
	}
}

// Name returns the health check name.
func (h *healthcheck) Name() string { return h.desc.Name }

// Describe returns a copy of the health check descriptor.
func (h *healthcheck) Describe() Descriptor {
	return Descriptor{
		Name:   h.desc.Name,
		Help:   h.desc.Help,
		Labels: cloneLabels(h.desc.Labels),
	}
}

// Check runs the underlying probe, substituting context.Background()
// for a nil context.
func (h *healthcheck) Check(ctx context.Context) Status {
	if ctx == nil {
		ctx = context.Background()
	}
	return h.check(ctx)
}
