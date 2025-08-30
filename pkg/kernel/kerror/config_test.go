package kerror

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
)

func TestConfigure(t *testing.T) {
	tests := []struct {
		name     string
		input    GlobalConfig
		expected GlobalConfig
	}{
		{
			name: "valid configuration",
			input: GlobalConfig{
				EnableStackTrace: true,
				EnableMetrics:    true,
				MaxInstancePool:  2000,
				DefaultPackage:   "custom",
				MaxTags:          100,
				MaxDetails:       200,
				MaxTagKeyLen:     200,
				MaxTagValueLen:   2000,
				StackTraceDepth:  64,
				EnableValidation: false,
			},
			expected: GlobalConfig{
				EnableStackTrace: true,
				EnableMetrics:    true,
				MaxInstancePool:  2000,
				DefaultPackage:   "custom",
				MaxTags:          100,
				MaxDetails:       200,
				MaxTagKeyLen:     200,
				MaxTagValueLen:   2000,
				StackTraceDepth:  64,
				EnableValidation: false,
			},
		},
		{
			name: "zero values replaced with defaults",
			input: GlobalConfig{
				MaxInstancePool: 0,
				DefaultPackage:  "",
				MaxTags:         0,
				MaxDetails:      0,
				MaxTagKeyLen:    0,
				MaxTagValueLen:  0,
				StackTraceDepth: 0,
			},
			expected: GlobalConfig{
				EnableStackTrace: false,
				EnableMetrics:    false,
				MaxInstancePool:  defaultConfig.MaxInstancePool,
				DefaultPackage:   defaultConfig.DefaultPackage,
				MaxTags:          defaultConfig.MaxTags,
				MaxDetails:       defaultConfig.MaxDetails,
				MaxTagKeyLen:     defaultConfig.MaxTagKeyLen,
				MaxTagValueLen:   defaultConfig.MaxTagValueLen,
				StackTraceDepth:  defaultConfig.StackTraceDepth,
				EnableValidation: false,
			},
		},
		{
			name: "negative values replaced with defaults",
			input: GlobalConfig{
				MaxInstancePool: -1,
				MaxTags:         -10,
				MaxDetails:      -20,
				MaxTagKeyLen:    -5,
				MaxTagValueLen:  -100,
				StackTraceDepth: -1,
				DefaultPackage:  "test",
			},
			expected: GlobalConfig{
				EnableStackTrace: false,
				EnableMetrics:    false,
				MaxInstancePool:  defaultConfig.MaxInstancePool,
				DefaultPackage:   "test",
				MaxTags:          defaultConfig.MaxTags,
				MaxDetails:       defaultConfig.MaxDetails,
				MaxTagKeyLen:     defaultConfig.MaxTagKeyLen,
				MaxTagValueLen:   defaultConfig.MaxTagValueLen,
				StackTraceDepth:  defaultConfig.StackTraceDepth,
				EnableValidation: false,
			},
		},
		{
			name: "partial configuration",
			input: GlobalConfig{
				EnableStackTrace: true,
				MaxInstancePool:  500,
				DefaultPackage:   "partial",
			},
			expected: GlobalConfig{
				EnableStackTrace: true,
				EnableMetrics:    false,
				MaxInstancePool:  500,
				DefaultPackage:   "partial",
				MaxTags:          defaultConfig.MaxTags,
				MaxDetails:       defaultConfig.MaxDetails,
				MaxTagKeyLen:     defaultConfig.MaxTagKeyLen,
				MaxTagValueLen:   defaultConfig.MaxTagValueLen,
				StackTraceDepth:  defaultConfig.StackTraceDepth,
				EnableValidation: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Configure(tt.input)
			got := GetConfig()
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Configure() = %+v, want %+v", got, tt.expected)
			}
		})
	}
}

func TestGetConfig(t *testing.T) {
	// Test default configuration
	config.Store(defaultConfig)
	got := GetConfig()
	if !reflect.DeepEqual(got, defaultConfig) {
		t.Errorf("GetConfig() = %+v, want %+v", got, defaultConfig)
	}

	// Test custom configuration
	customConfig := GlobalConfig{
		EnableStackTrace: true,
		EnableMetrics:    true,
		MaxInstancePool:  5000,
		DefaultPackage:   "test",
		MaxTags:          200,
		MaxDetails:       300,
		MaxTagKeyLen:     500,
		MaxTagValueLen:   5000,
		StackTraceDepth:  100,
		EnableValidation: false,
	}

	Configure(customConfig)
	got = GetConfig()
	if !reflect.DeepEqual(got, customConfig) {
		t.Errorf("GetConfig() after Configure = %+v, want %+v", got, customConfig)
	}
}

func TestConfigureConcurrency(t *testing.T) {
	// Test concurrent configuration updates
	var wg sync.WaitGroup
	configs := []GlobalConfig{
		{
			EnableStackTrace: true,
			MaxInstancePool:  100,
			DefaultPackage:   "concurrent1",
		},
		{
			EnableMetrics:   true,
			MaxInstancePool: 200,
			DefaultPackage:  "concurrent2",
		},
		{
			EnableValidation: true,
			MaxInstancePool:  300,
			DefaultPackage:   "concurrent3",
		},
	}

	for _, cfg := range configs {
		wg.Add(1)
		go func(c GlobalConfig) {
			defer wg.Done()
			Configure(c)
			_ = GetConfig()
		}(cfg)
	}

	wg.Wait()

	// Verify that configuration is still valid
	finalConfig := GetConfig()
	if finalConfig.MaxInstancePool <= 0 {
		t.Errorf("Invalid MaxInstancePool after concurrent updates: %d", finalConfig.MaxInstancePool)
	}
	if finalConfig.DefaultPackage == "" {
		t.Errorf("Empty DefaultPackage after concurrent updates")
	}
}

func TestInitializePools(t *testing.T) {
	// Test that configuration works correctly
	Configure(GlobalConfig{
		MaxInstancePool: 100,
		DefaultPackage:  "pools",
	})

	cfg := GetConfig()
	if cfg.MaxInstancePool != 100 {
		t.Error("MaxInstancePool should be 100")
	}
	if cfg.DefaultPackage != "pools" {
		t.Error("DefaultPackage should be 'pools'")
	}
}

func TestConfigureOnce(t *testing.T) {
	// Test that Configure updates config
	// (configOnce was removed, so Configure can be called multiple times)

	// First call should execute
	Configure(GlobalConfig{
		MaxInstancePool: 1000,
		DefaultPackage:  "once1",
	})

	// Subsequent calls should still update config
	Configure(GlobalConfig{
		MaxInstancePool: 2000,
		DefaultPackage:  "once2",
	})

	got := GetConfig()
	if got.DefaultPackage != "once2" {
		t.Errorf("Configure should update config, got package = %s, want once2", got.DefaultPackage)
	}
}

func TestDefaultConfigValues(t *testing.T) {
	// Verify default values are sensible
	if defaultConfig.MaxInstancePool != 1000 {
		t.Errorf("defaultConfig.MaxInstancePool = %d, want 1000", defaultConfig.MaxInstancePool)
	}
	if defaultConfig.DefaultPackage != "unknown" {
		t.Errorf("defaultConfig.DefaultPackage = %s, want unknown", defaultConfig.DefaultPackage)
	}
	if defaultConfig.MaxTags != 50 {
		t.Errorf("defaultConfig.MaxTags = %d, want 50", defaultConfig.MaxTags)
	}
	if defaultConfig.MaxDetails != 100 {
		t.Errorf("defaultConfig.MaxDetails = %d, want 100", defaultConfig.MaxDetails)
	}
	if defaultConfig.MaxTagKeyLen != 100 {
		t.Errorf("defaultConfig.MaxTagKeyLen = %d, want 100", defaultConfig.MaxTagKeyLen)
	}
	if defaultConfig.MaxTagValueLen != 1000 {
		t.Errorf("defaultConfig.MaxTagValueLen = %d, want 1000", defaultConfig.MaxTagValueLen)
	}
	if defaultConfig.StackTraceDepth != 32 {
		t.Errorf("defaultConfig.StackTraceDepth = %d, want 32", defaultConfig.StackTraceDepth)
	}
	if !defaultConfig.EnableValidation {
		t.Errorf("defaultConfig.EnableValidation = false, want true")
	}
	if defaultConfig.EnableStackTrace {
		t.Errorf("defaultConfig.EnableStackTrace = true, want false")
	}
	if defaultConfig.EnableMetrics {
		t.Errorf("defaultConfig.EnableMetrics = true, want false")
	}
}

// Additional tests for better coverage

func TestConfigurePanicRecovery(t *testing.T) {
	// Test that Configure doesn't panic with extreme values
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Configure panicked: %v", r)
		}
	}()

	// Test with very large values
	Configure(GlobalConfig{
		MaxInstancePool: 1000000,
		MaxTags:         10000,
		MaxDetails:      10000,
		MaxTagKeyLen:    100000,
		MaxTagValueLen:  1000000,
		StackTraceDepth: 1000,
	})

	// Test with zero/negative values (should be replaced with defaults)
	Configure(GlobalConfig{
		MaxInstancePool: -100,
		MaxTags:         -50,
		MaxDetails:      -25,
		MaxTagKeyLen:    -10,
		MaxTagValueLen:  -1000,
		StackTraceDepth: -32,
	})

	// Verify configuration is still valid
	cfg := GetConfig()
	if cfg.MaxInstancePool <= 0 {
		t.Error("MaxInstancePool should be positive after negative input")
	}
}

func TestConfigureConcurrentAccess(t *testing.T) {
	// Test massive concurrent access to Configure and GetConfig
	var wg sync.WaitGroup
	const numGoroutines = 100
	const operationsPerGoroutine = 100

	// Start concurrent configurers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(2)

		// Configuration updaters
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				Configure(GlobalConfig{
					EnableStackTrace: id%2 == 0,
					EnableMetrics:    id%3 == 0,
					MaxInstancePool:  100 + id,
					DefaultPackage:   fmt.Sprintf("concurrent%d", id),
					MaxTags:          50 + id,
					MaxDetails:       100 + id,
				})
			}
		}(i)

		// Configuration readers
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				cfg := GetConfig()
				// Basic validity checks
				if cfg.MaxInstancePool <= 0 {
					t.Errorf("Invalid MaxInstancePool: %d", cfg.MaxInstancePool)
				}
				if cfg.DefaultPackage == "" {
					t.Error("DefaultPackage should not be empty")
				}
			}
		}()
	}

	wg.Wait()

	// Verify final configuration is valid
	finalConfig := GetConfig()
	if finalConfig.MaxInstancePool <= 0 {
		t.Error("Final configuration invalid")
	}
}

func TestConfigureRapidUpdates(t *testing.T) {
	// Test rapid successive updates don't cause issues
	for i := 0; i < 1000; i++ {
		Configure(GlobalConfig{
			MaxInstancePool: 1000 + i,
			DefaultPackage:  fmt.Sprintf("rapid%d", i),
			MaxTags:         50 + (i % 100),
		})

		// Immediately read back
		cfg := GetConfig()
		if cfg.MaxInstancePool != 1000+i {
			t.Errorf("Configuration not updated correctly at iteration %d", i)
		}
	}
}

func TestConfigureMemoryStress(t *testing.T) {
	// Test configuration with many rapid updates to stress memory
	originalConfig := GetConfig()
	defer Configure(originalConfig) // Restore original

	for i := 0; i < 10000; i++ {
		Configure(GlobalConfig{
			DefaultPackage:  fmt.Sprintf("stress_test_package_with_very_long_name_%d", i),
			MaxInstancePool: 1000 + (i % 1000),
			MaxTags:         10 + (i % 90),
			MaxDetails:      20 + (i % 180),
		})

		if i%1000 == 0 {
			// Periodic verification
			cfg := GetConfig()
			if cfg.DefaultPackage == "" {
				t.Errorf("Configuration corrupted at iteration %d", i)
			}
		}
	}
}

// Benchmarks

func BenchmarkConfigureBasic(b *testing.B) {
	cfg := GlobalConfig{
		EnableStackTrace: true,
		EnableMetrics:    true,
		MaxInstancePool:  1000,
		DefaultPackage:   "benchmark",
		MaxTags:          50,
		MaxDetails:       100,
		MaxTagKeyLen:     100,
		MaxTagValueLen:   1000,
		StackTraceDepth:  32,
		EnableValidation: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Configure(cfg)
	}
}

func BenchmarkGetConfig(b *testing.B) {
	Configure(GlobalConfig{
		EnableStackTrace: true,
		MaxInstancePool:  1000,
		DefaultPackage:   "benchmark",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetConfig()
	}
}

func BenchmarkConfigureConcurrent(b *testing.B) {
	cfg := GlobalConfig{
		MaxInstancePool: 1000,
		DefaultPackage:  "concurrent",
		MaxTags:         50,
		MaxDetails:      100,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Configure(cfg)
		}
	})
}

func BenchmarkGetConfigConcurrent(b *testing.B) {
	Configure(GlobalConfig{
		MaxInstancePool: 1000,
		DefaultPackage:  "benchmark",
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = GetConfig()
		}
	})
}

func BenchmarkConfigureMixed(b *testing.B) {
	configs := []GlobalConfig{
		{EnableStackTrace: true, MaxInstancePool: 1000, DefaultPackage: "config1"},
		{EnableMetrics: true, MaxInstancePool: 2000, DefaultPackage: "config2"},
		{EnableValidation: false, MaxInstancePool: 500, DefaultPackage: "config3"},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			Configure(configs[i%len(configs)])
			i++
		}
	})
}

func TestInit(t *testing.T) {
	// Test that init properly sets default config
	// This should already be done by package initialization
	got := GetConfig()
	if got.DefaultPackage == "" {
		t.Errorf("Init should set DefaultPackage, got empty string")
	}
}

func TestConfigureAllFields(t *testing.T) {
	// Test setting all fields explicitly
	cfg := GlobalConfig{
		EnableStackTrace: true,
		EnableMetrics:    true,
		MaxInstancePool:  999,
		DefaultPackage:   "allfields",
		MaxTags:          99,
		MaxDetails:       199,
		MaxTagKeyLen:     299,
		MaxTagValueLen:   2999,
		StackTraceDepth:  99,
		EnableValidation: false,
	}

	Configure(cfg)
	got := GetConfig()

	if got.EnableStackTrace != cfg.EnableStackTrace {
		t.Errorf("EnableStackTrace = %v, want %v", got.EnableStackTrace, cfg.EnableStackTrace)
	}
	if got.EnableMetrics != cfg.EnableMetrics {
		t.Errorf("EnableMetrics = %v, want %v", got.EnableMetrics, cfg.EnableMetrics)
	}
	if got.MaxInstancePool != cfg.MaxInstancePool {
		t.Errorf("MaxInstancePool = %d, want %d", got.MaxInstancePool, cfg.MaxInstancePool)
	}
	if got.DefaultPackage != cfg.DefaultPackage {
		t.Errorf("DefaultPackage = %s, want %s", got.DefaultPackage, cfg.DefaultPackage)
	}
	if got.MaxTags != cfg.MaxTags {
		t.Errorf("MaxTags = %d, want %d", got.MaxTags, cfg.MaxTags)
	}
	if got.MaxDetails != cfg.MaxDetails {
		t.Errorf("MaxDetails = %d, want %d", got.MaxDetails, cfg.MaxDetails)
	}
	if got.MaxTagKeyLen != cfg.MaxTagKeyLen {
		t.Errorf("MaxTagKeyLen = %d, want %d", got.MaxTagKeyLen, cfg.MaxTagKeyLen)
	}
	if got.MaxTagValueLen != cfg.MaxTagValueLen {
		t.Errorf("MaxTagValueLen = %d, want %d", got.MaxTagValueLen, cfg.MaxTagValueLen)
	}
	if got.StackTraceDepth != cfg.StackTraceDepth {
		t.Errorf("StackTraceDepth = %d, want %d", got.StackTraceDepth, cfg.StackTraceDepth)
	}
	if got.EnableValidation != cfg.EnableValidation {
		t.Errorf("EnableValidation = %v, want %v", got.EnableValidation, cfg.EnableValidation)
	}
}
