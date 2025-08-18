package kerror

import (
	"sync"
	"sync/atomic"
)

// MetricsCollector interface for pluggable metrics implementations
type MetricsCollector interface {
	RecordErrorDefinition(pkg string, code int)
	RecordErrorInstance(pkg string, code int)
	RecordErrorWrapped(pkg string, code int)
}

// SimpleMetrics is a basic in-memory metrics implementation
type SimpleMetrics struct {
	definitions map[string]map[int]uint64 // package -> code -> count
	instances   map[string]map[int]uint64
	wrapped     map[string]map[int]uint64
	mu          sync.RWMutex
}

// Global metrics instance
var (
	metricsCollector MetricsCollector = NewSimpleMetrics()
	metricsEnabled   atomic.Bool
)

// NewSimpleMetrics creates a new simple metrics collector
func NewSimpleMetrics() *SimpleMetrics {
	return &SimpleMetrics{
		definitions: make(map[string]map[int]uint64),
		instances:   make(map[string]map[int]uint64),
		wrapped:     make(map[string]map[int]uint64),
	}
}

// RecordErrorDefinition records a new error definition
func (m *SimpleMetrics) RecordErrorDefinition(pkg string, code int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.definitions[pkg] == nil {
		m.definitions[pkg] = make(map[int]uint64)
	}
	m.definitions[pkg][code]++
}

// RecordErrorInstance records a new error instance creation
func (m *SimpleMetrics) RecordErrorInstance(pkg string, code int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.instances[pkg] == nil {
		m.instances[pkg] = make(map[int]uint64)
	}
	m.instances[pkg][code]++
}

// RecordErrorWrapped records an error wrap operation
func (m *SimpleMetrics) RecordErrorWrapped(pkg string, code int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.wrapped[pkg] == nil {
		m.wrapped[pkg] = make(map[int]uint64)
	}
	m.wrapped[pkg][code]++
}

// GetMetrics returns current metrics snapshot
func (m *SimpleMetrics) GetMetrics() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]any{
		"definitions": copyNestedMap(m.definitions),
		"instances":   copyNestedMap(m.instances),
		"wrapped":     copyNestedMap(m.wrapped),
	}
}

func copyNestedMap(src map[string]map[int]uint64) map[string]map[int]uint64 {
	dst := make(map[string]map[int]uint64)
	for k, v := range src {
		inner := make(map[int]uint64)
		for ik, iv := range v {
			inner[ik] = iv
		}
		dst[k] = inner
	}
	return dst
}

// SetMetricsCollector sets a custom metrics collector
func SetMetricsCollector(collector MetricsCollector) {
	metricsCollector = collector
}

// Internal helpers for metrics recording
func recordErrorDefinition(pkg string, code int) {
	if metricsCollector != nil && GetConfig().EnableMetrics {
		metricsCollector.RecordErrorDefinition(pkg, code)
	}
}

func recordErrorInstance(pkg string, code int) {
	if metricsCollector != nil && GetConfig().EnableMetrics {
		metricsCollector.RecordErrorInstance(pkg, code)
	}
}

func recordErrorWrapped(pkg string, code int) {
	if metricsCollector != nil && GetConfig().EnableMetrics {
		metricsCollector.RecordErrorWrapped(pkg, code)
	}
}

// GetMetricsSnapshot returns current metrics if using SimpleMetrics
func GetMetricsSnapshot() map[string]any {
	if sm, ok := metricsCollector.(*SimpleMetrics); ok {
		return sm.GetMetrics()
	}
	return nil
}
