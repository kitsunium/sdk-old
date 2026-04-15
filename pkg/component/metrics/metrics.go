// Package metrics provides ready-to-use primitives for instrumenting
// Kitsunium SDK programs: counters, gauges, and health checks.
//
// Every product is exposed through the SDK naming convention
// `metrics.Thing(...)`:
//
//   - [Counter] builds a monotonically increasing counter.
//   - [Gauge]   builds a value that can move up and down.
//   - [Health]  builds a named health check.
//
// All implementations use stdlib only and are safe for concurrent use.
package metrics

import (
	"context"
)

// StatusCode classifies the outcome of a health check.
type StatusCode uint8

// Status codes, in increasing severity order.
const (
	StatusOK StatusCode = iota
	StatusDegraded
	StatusDown
)

// String returns a human-readable name for the status code.
func (s StatusCode) String() string {
	switch s {
	case StatusOK:
		return "ok"
	case StatusDegraded:
		return "degraded"
	case StatusDown:
		return "down"
	default:
		return "unknown"
	}
}

// Status is the result of a health check.
type Status struct {
	Code   StatusCode
	Reason string
}

// OK builds an OK status (optionally with a reason).
func OK(reason string) Status { return Status{Code: StatusOK, Reason: reason} }

// Degraded builds a Degraded status with the given reason.
func Degraded(reason string) Status { return Status{Code: StatusDegraded, Reason: reason} }

// Down builds a Down status with the given reason.
func Down(reason string) Status { return Status{Code: StatusDown, Reason: reason} }

// Descriptor describes a metric's identity and metadata.
type Descriptor struct {
	Name   string
	Help   string
	Labels map[string]string
}

// Metric is the common interface implemented by every metric.
type Metric interface {
	// Name returns the metric's canonical name.
	Name() string
	// Describe returns the metric's full descriptor.
	Describe() Descriptor
}

// Healthcheck evaluates application liveness/readiness on demand.
type Healthcheck interface {
	Metric
	// Check runs the check and returns a [Status]. A nil context is
	// tolerated and treated as context.Background().
	Check(ctx context.Context) Status
}

// Option mutates the shared configuration used by metric constructors.
type Option func(*config)

// config holds the settings shared by every metric implementation.
type config struct {
	help   string
	labels map[string]string
}

// defaultConfig returns a pre-populated config with safe defaults.
func defaultConfig() *config {
	return &config{labels: map[string]string{}}
}

// apply runs opts against the given config.
func apply(cfg *config, opts []Option) {
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}
}

// WithHelp sets the descriptor help text.
func WithHelp(help string) Option {
	return func(c *config) { c.help = help }
}

// WithLabel adds a single key/value label to the descriptor.
func WithLabel(key, value string) Option {
	return func(c *config) {
		if key == "" {
			return
		}
		if c.labels == nil {
			c.labels = map[string]string{}
		}
		c.labels[key] = value
	}
}

// WithLabels merges the given map into the descriptor labels.
func WithLabels(labels map[string]string) Option {
	return func(c *config) {
		if c.labels == nil {
			c.labels = map[string]string{}
		}
		for k, v := range labels {
			c.labels[k] = v
		}
	}
}

// cloneLabels returns a defensive copy of m to avoid aliasing caller data.
func cloneLabels(m map[string]string) map[string]string {
	if len(m) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
