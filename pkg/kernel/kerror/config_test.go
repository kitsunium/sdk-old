package kerror

import (
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
				EnableStackTrace:  true,
				EnableMetrics:     true,
				MaxInstancePool:   2000,
				DefaultPackage:    "custom",
				MaxTags:           100,
				MaxDetails:        200,
				MaxTagKeyLen:      200,
				MaxTagValueLen:    2000,
				StackTraceDepth:   64,
				EnableValidation:  false,
			},
			expected: GlobalConfig{
				EnableStackTrace:  true,
				EnableMetrics:     true,
				MaxInstancePool:   2000,
				DefaultPackage:    "custom",
				MaxTags:           100,
				MaxDetails:        200,
				MaxTagKeyLen:      200,
				MaxTagValueLen:    2000,
				StackTraceDepth:   64,
				EnableValidation:  false,
			},
		},
		{
			name: "zero values replaced with defaults",
			input: GlobalConfig{
				MaxInstancePool:   0,
				DefaultPackage:    "",
				MaxTags:           0,
				MaxDetails:        0,
				MaxTagKeyLen:      0,
				MaxTagValueLen:    0,
				StackTraceDepth:   0,
			},
			expected: GlobalConfig{
				EnableStackTrace:  false,
				EnableMetrics:     false,
				MaxInstancePool:   defaultConfig.MaxInstancePool,
				DefaultPackage:    defaultConfig.DefaultPackage,
				MaxTags:           defaultConfig.MaxTags,
				MaxDetails:        defaultConfig.MaxDetails,
				MaxTagKeyLen:      defaultConfig.MaxTagKeyLen,
				MaxTagValueLen:    defaultConfig.MaxTagValueLen,
				StackTraceDepth:   defaultConfig.StackTraceDepth,
				EnableValidation:  false,
			},
		},
		{
			name: "negative values replaced with defaults",
			input: GlobalConfig{
				MaxInstancePool:   -1,
				MaxTags:           -10,
				MaxDetails:        -20,
				MaxTagKeyLen:      -5,
				MaxTagValueLen:    -100,
				StackTraceDepth:   -1,
				DefaultPackage:    "test",
			},
			expected: GlobalConfig{
				EnableStackTrace:  false,
				EnableMetrics:     false,
				MaxInstancePool:   defaultConfig.MaxInstancePool,
				DefaultPackage:    "test",
				MaxTags:           defaultConfig.MaxTags,
				MaxDetails:        defaultConfig.MaxDetails,
				MaxTagKeyLen:      defaultConfig.MaxTagKeyLen,
				MaxTagValueLen:    defaultConfig.MaxTagValueLen,
				StackTraceDepth:   defaultConfig.StackTraceDepth,
				EnableValidation:  false,
			},
		},
		{
			name: "partial configuration",
			input: GlobalConfig{
				EnableStackTrace:  true,
				MaxInstancePool:   500,
				DefaultPackage:    "partial",
			},
			expected: GlobalConfig{
				EnableStackTrace:  true,
				EnableMetrics:     false,
				MaxInstancePool:   500,
				DefaultPackage:    "partial",
				MaxTags:           defaultConfig.MaxTags,
				MaxDetails:        defaultConfig.MaxDetails,
				MaxTagKeyLen:      defaultConfig.MaxTagKeyLen,
				MaxTagValueLen:    defaultConfig.MaxTagValueLen,
				StackTraceDepth:   defaultConfig.StackTraceDepth,
				EnableValidation:  false,
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
		EnableStackTrace:  true,
		EnableMetrics:     true,
		MaxInstancePool:   5000,
		DefaultPackage:    "test",
		MaxTags:           200,
		MaxDetails:        300,
		MaxTagKeyLen:      500,
		MaxTagValueLen:    5000,
		StackTraceDepth:   100,
		EnableValidation:  false,
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
	// Reset configOnce for testing
	configOnce = sync.Once{}
	
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
		EnableStackTrace:  true,
		EnableMetrics:     true,
		MaxInstancePool:   999,
		DefaultPackage:    "allfields",
		MaxTags:           99,
		MaxDetails:        199,
		MaxTagKeyLen:      299,
		MaxTagValueLen:    2999,
		StackTraceDepth:   99,
		EnableValidation:  false,
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