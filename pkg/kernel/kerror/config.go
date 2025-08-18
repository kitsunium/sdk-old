package kerror

import (
	"sync"
	"sync/atomic"
)

// GlobalConfig holds the global configuration for the kerror package
type GlobalConfig struct {
	EnableStackTrace bool   // Capture stack traces for errors
	EnableMetrics    bool   // Enable Prometheus metrics
	MaxInstancePool  int    // Maximum number of instances in pool
	DefaultPackage   string // Default package name when auto-detection fails
	MaxTags          int    // Maximum number of tags per instance
	MaxDetails       int    // Maximum number of details per instance
	MaxTagKeyLen     int    // Maximum length of tag keys
	MaxTagValueLen   int    // Maximum length of tag values
	StackTraceDepth  int    // Maximum depth of stack trace
	EnableValidation bool   // Enable validation of inputs
}

// Default configuration values
var defaultConfig = GlobalConfig{
	EnableStackTrace: false,
	EnableMetrics:    false,
	MaxInstancePool:  1000,
	DefaultPackage:   "unknown",
	MaxTags:          50,
	MaxDetails:       100,
	MaxTagKeyLen:     100,
	MaxTagValueLen:   1000,
	StackTraceDepth:  32,
	EnableValidation: true,
}

var (
	config     atomic.Value // GlobalConfig
	configOnce sync.Once
)

func init() {
	config.Store(defaultConfig)
}

// Configure sets the global configuration
func Configure(cfg GlobalConfig) {
	// Validate configuration
	if cfg.MaxInstancePool <= 0 {
		cfg.MaxInstancePool = defaultConfig.MaxInstancePool
	}
	if cfg.MaxTags <= 0 {
		cfg.MaxTags = defaultConfig.MaxTags
	}
	if cfg.MaxDetails <= 0 {
		cfg.MaxDetails = defaultConfig.MaxDetails
	}
	if cfg.MaxTagKeyLen <= 0 {
		cfg.MaxTagKeyLen = defaultConfig.MaxTagKeyLen
	}
	if cfg.MaxTagValueLen <= 0 {
		cfg.MaxTagValueLen = defaultConfig.MaxTagValueLen
	}
	if cfg.StackTraceDepth <= 0 {
		cfg.StackTraceDepth = defaultConfig.StackTraceDepth
	}
	if cfg.DefaultPackage == "" {
		cfg.DefaultPackage = defaultConfig.DefaultPackage
	}

	config.Store(cfg)
}

// GetConfig returns the current global configuration
func GetConfig() GlobalConfig {
	return config.Load().(GlobalConfig)
}
