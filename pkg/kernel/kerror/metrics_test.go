package kerror

import (
	"reflect"
	"sync"
	"testing"
)

func TestNewSimpleMetrics(t *testing.T) {
	m := NewSimpleMetrics()
	
	if m == nil {
		t.Fatal("NewSimpleMetrics() returned nil")
	}
	
	if m.definitions == nil {
		t.Error("definitions map not initialized")
	}
	
	if m.instances == nil {
		t.Error("instances map not initialized")
	}
	
	if m.wrapped == nil {
		t.Error("wrapped map not initialized")
	}
}

func TestRecordErrorDefinition(t *testing.T) {
	m := NewSimpleMetrics()
	
	// Record multiple definitions
	m.RecordErrorDefinition("pkg1", 404)
	m.RecordErrorDefinition("pkg1", 404)
	m.RecordErrorDefinition("pkg1", 500)
	m.RecordErrorDefinition("pkg2", 404)
	
	// Verify counts
	if m.definitions["pkg1"][404] != 2 {
		t.Errorf("pkg1/404 count = %d, want 2", m.definitions["pkg1"][404])
	}
	
	if m.definitions["pkg1"][500] != 1 {
		t.Errorf("pkg1/500 count = %d, want 1", m.definitions["pkg1"][500])
	}
	
	if m.definitions["pkg2"][404] != 1 {
		t.Errorf("pkg2/404 count = %d, want 1", m.definitions["pkg2"][404])
	}
}

func TestRecordErrorInstance(t *testing.T) {
	m := NewSimpleMetrics()
	
	// Record instances
	m.RecordErrorInstance("pkg1", 404)
	m.RecordErrorInstance("pkg1", 404)
	m.RecordErrorInstance("pkg1", 404)
	m.RecordErrorInstance("pkg2", 500)
	
	// Verify counts
	if m.instances["pkg1"][404] != 3 {
		t.Errorf("pkg1/404 instances = %d, want 3", m.instances["pkg1"][404])
	}
	
	if m.instances["pkg2"][500] != 1 {
		t.Errorf("pkg2/500 instances = %d, want 1", m.instances["pkg2"][500])
	}
}

func TestRecordErrorWrapped(t *testing.T) {
	m := NewSimpleMetrics()
	
	// Record wrapped errors
	m.RecordErrorWrapped("pkg1", 500)
	m.RecordErrorWrapped("pkg1", 500)
	m.RecordErrorWrapped("pkg2", 503)
	
	// Verify counts
	if m.wrapped["pkg1"][500] != 2 {
		t.Errorf("pkg1/500 wrapped = %d, want 2", m.wrapped["pkg1"][500])
	}
	
	if m.wrapped["pkg2"][503] != 1 {
		t.Errorf("pkg2/503 wrapped = %d, want 1", m.wrapped["pkg2"][503])
	}
}

func TestGetMetrics(t *testing.T) {
	m := NewSimpleMetrics()
	
	// Record some metrics
	m.RecordErrorDefinition("pkg1", 404)
	m.RecordErrorInstance("pkg1", 404)
	m.RecordErrorWrapped("pkg1", 404)
	
	// Get snapshot
	snapshot := m.GetMetrics()
	
	if snapshot == nil {
		t.Fatal("GetMetrics() returned nil")
	}
	
	// Check structure
	if _, ok := snapshot["definitions"]; !ok {
		t.Error("Missing definitions in snapshot")
	}
	
	if _, ok := snapshot["instances"]; !ok {
		t.Error("Missing instances in snapshot")
	}
	
	if _, ok := snapshot["wrapped"]; !ok {
		t.Error("Missing wrapped in snapshot")
	}
	
	// Verify it's a copy
	defs := snapshot["definitions"].(map[string]map[int]uint64)
	defs["pkg1"][404] = 999
	
	if m.definitions["pkg1"][404] == 999 {
		t.Error("GetMetrics() should return a copy")
	}
}

func TestCopyNestedMap(t *testing.T) {
	src := map[string]map[int]uint64{
		"pkg1": {
			404: 1,
			500: 2,
		},
		"pkg2": {
			503: 3,
		},
	}
	
	dst := copyNestedMap(src)
	
	// Verify structure
	if !reflect.DeepEqual(src, dst) {
		t.Error("copyNestedMap() didn't copy correctly")
	}
	
	// Verify it's a deep copy
	dst["pkg1"][404] = 999
	if src["pkg1"][404] == 999 {
		t.Error("copyNestedMap() should create deep copy")
	}
	
	// Test empty map
	empty := copyNestedMap(nil)
	if empty == nil || len(empty) != 0 {
		t.Error("copyNestedMap(nil) should return empty map")
	}
}

func TestSetMetricsCollector(t *testing.T) {
	// Create custom collector
	custom := &customCollector{}
	SetMetricsCollector(custom)
	
	if metricsCollector != custom {
		t.Error("SetMetricsCollector() didn't set collector")
	}
	
	// Reset to default
	SetMetricsCollector(NewSimpleMetrics())
}

type customCollector struct {
	definitionsCalled bool
	instancesCalled   bool
	wrappedCalled     bool
}

func (c *customCollector) RecordErrorDefinition(pkg string, code int) {
	c.definitionsCalled = true
}

func (c *customCollector) RecordErrorInstance(pkg string, code int) {
	c.instancesCalled = true
}

func (c *customCollector) RecordErrorWrapped(pkg string, code int) {
	c.wrappedCalled = true
}

func TestRecordHelpers(t *testing.T) {
	// Test with metrics disabled
	Configure(GlobalConfig{EnableMetrics: false})
	custom := &customCollector{}
	SetMetricsCollector(custom)
	
	recordErrorDefinition("pkg", 404)
	recordErrorInstance("pkg", 404)
	recordErrorWrapped("pkg", 404)
	
	if custom.definitionsCalled || custom.instancesCalled || custom.wrappedCalled {
		t.Error("Helpers should not call collector when metrics disabled")
	}
	
	// Test with metrics enabled
	Configure(GlobalConfig{EnableMetrics: true})
	
	recordErrorDefinition("pkg", 404)
	if !custom.definitionsCalled {
		t.Error("recordErrorDefinition() should call collector")
	}
	
	recordErrorInstance("pkg", 404)
	if !custom.instancesCalled {
		t.Error("recordErrorInstance() should call collector")
	}
	
	recordErrorWrapped("pkg", 404)
	if !custom.wrappedCalled {
		t.Error("recordErrorWrapped() should call collector")
	}
	
	// Test with nil collector
	SetMetricsCollector(nil)
	// Should not panic
	recordErrorDefinition("pkg", 404)
	recordErrorInstance("pkg", 404)
	recordErrorWrapped("pkg", 404)
}

func TestGetMetricsSnapshot(t *testing.T) {
	// Test with SimpleMetrics
	simple := NewSimpleMetrics()
	SetMetricsCollector(simple)
	
	simple.RecordErrorDefinition("pkg", 404)
	
	snapshot := GetMetricsSnapshot()
	if snapshot == nil {
		t.Fatal("GetMetricsSnapshot() returned nil")
	}
	
	// Test with custom collector
	SetMetricsCollector(&customCollector{})
	snapshot = GetMetricsSnapshot()
	if snapshot != nil {
		t.Error("GetMetricsSnapshot() should return nil for non-SimpleMetrics")
	}
}

func TestMetricsConcurrency(t *testing.T) {
	m := NewSimpleMetrics()
	
	var wg sync.WaitGroup
	
	// Concurrent writes
	for i := 0; i < 10; i++ {
		wg.Add(3)
		
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				m.RecordErrorDefinition("pkg", n)
			}
		}(i)
		
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				m.RecordErrorInstance("pkg", n)
			}
		}(i)
		
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				m.RecordErrorWrapped("pkg", n)
			}
		}(i)
	}
	
	// Concurrent reads
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = m.GetMetrics()
			}
		}()
	}
	
	wg.Wait()
	
	// Verify data integrity
	snapshot := m.GetMetrics()
	defs := snapshot["definitions"].(map[string]map[int]uint64)
	insts := snapshot["instances"].(map[string]map[int]uint64)
	wraps := snapshot["wrapped"].(map[string]map[int]uint64)
	
	// Each metric should have been recorded 100 times for each of 10 codes
	for i := 0; i < 10; i++ {
		if defs["pkg"][i] != 100 {
			t.Errorf("definitions[pkg][%d] = %d, want 100", i, defs["pkg"][i])
		}
		if insts["pkg"][i] != 100 {
			t.Errorf("instances[pkg][%d] = %d, want 100", i, insts["pkg"][i])
		}
		if wraps["pkg"][i] != 100 {
			t.Errorf("wrapped[pkg][%d] = %d, want 100", i, wraps["pkg"][i])
		}
	}
}

func TestMetricsInitialization(t *testing.T) {
	// Test package-level initialization
	if metricsCollector == nil {
		t.Error("metricsCollector should be initialized")
	}
	
	// Reset to default SimpleMetrics for test
	SetMetricsCollector(NewSimpleMetrics())
	
	// Verify it's a SimpleMetrics by default
	if _, ok := metricsCollector.(*SimpleMetrics); !ok {
		t.Error("Default collector should be SimpleMetrics")
	}
}

func TestMetricsWithNilPackage(t *testing.T) {
	m := NewSimpleMetrics()
	
	// Should handle empty package name
	m.RecordErrorDefinition("", 404)
	m.RecordErrorInstance("", 404)
	m.RecordErrorWrapped("", 404)
	
	if m.definitions[""][404] != 1 {
		t.Error("Should handle empty package name")
	}
}

func TestMetricsEdgeCases(t *testing.T) {
	m := NewSimpleMetrics()
	
	// Test with negative codes
	m.RecordErrorDefinition("pkg", -1)
	if m.definitions["pkg"][-1] != 1 {
		t.Error("Should handle negative codes")
	}
	
	// Test with zero code
	m.RecordErrorDefinition("pkg", 0)
	if m.definitions["pkg"][0] != 1 {
		t.Error("Should handle zero code")
	}
	
	// Test very large code
	m.RecordErrorDefinition("pkg", 999999)
	if m.definitions["pkg"][999999] != 1 {
		t.Error("Should handle large codes")
	}
}

func TestMetricsAccumulation(t *testing.T) {
	m := NewSimpleMetrics()
	
	// Test accumulation
	for i := 0; i < 100; i++ {
		m.RecordErrorDefinition("pkg", 404)
	}
	
	if m.definitions["pkg"][404] != 100 {
		t.Errorf("Accumulation failed, got %d want 100", m.definitions["pkg"][404])
	}
	
	// Test multiple packages
	packages := []string{"pkg1", "pkg2", "pkg3"}
	for _, pkg := range packages {
		for i := 0; i < 10; i++ {
			m.RecordErrorInstance(pkg, 500)
		}
	}
	
	for _, pkg := range packages {
		if m.instances[pkg][500] != 10 {
			t.Errorf("Package %s instances = %d, want 10", pkg, m.instances[pkg][500])
		}
	}
}

func TestMetricsInterface(t *testing.T) {
	// Verify SimpleMetrics implements MetricsCollector
	var _ MetricsCollector = (*SimpleMetrics)(nil)
	var _ MetricsCollector = NewSimpleMetrics()
	
	// Verify customCollector implements MetricsCollector
	var _ MetricsCollector = (*customCollector)(nil)
}