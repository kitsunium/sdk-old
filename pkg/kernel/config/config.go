package config

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/kitsunium/sdk/pkg/kernel/kcache"
)

type Parser interface {
	Load() (map[string]string, error)
	Type() string
}

type Config struct {
	parsers []Parser
	data    map[string]any
	mu      sync.RWMutex
	cache   kcache.Cache[string, any]
}

// New creates a new Config instance.
func New(parsers ...Parser) *Config {
	return NewWithCache(parsers, 1000)
}

// NewWithCache creates a new Config instance with a custom cache size.
func NewWithCache(parsers []Parser, cacheSize int) *Config {
	return &Config{
		parsers: parsers,
		data:    make(map[string]any),
		cache:   kcache.NewLRU[string, any](cacheSize),
	}
}

// Load loads configuration from a list of sources with a priority order
//
// Parameters:
// - sources: ...Source - A list of sources to load configuration from
//
// Returns:
// - err: error - Error if any issue occurs during loading.
func (c *Config) Load() error {
	for _, source := range c.parsers {
		data, err := source.Load()
		if err != nil {
			return fmt.Errorf("failed to load config from %s parser: %w", source.Type(), err)
		}

		for key, value := range data {
			c.Set(key, value)
		}
	}

	return nil
}

// Set sets a value in the configuration
//
// Parameters:
// - key: string - The configuration key to set
// - value: any - The configuration value to set.
func (c *Config) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = value
}

// Get retrieves a value from the configuration
//
// Parameters:
// - key: string - The configuration key to retrieve
//
// Returns:
// - value: any - The configuration value.
func (c *Config) Get(key string, defaultValue any) any {
	if c.cache != nil {
		if val, ok := c.cache.Get(key); ok {
			return val
		}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if value, ok := c.data[key]; ok {
		if c.cache != nil {
			c.cache.Set(key, value)
		}
		return value
	}

	return defaultValue
}

// GetString retrieves a string value from the configuration.
func (c *Config) GetString(key string, defaultValue string) string {
	val := c.Get(key, defaultValue)
	if s, ok := val.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", val)
}

// GetInt retrieves an integer value from the configuration.
func (c *Config) GetInt(key string, defaultValue int) int {
	val := c.Get(key, defaultValue)
	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultValue
}

// GetBool retrieves a boolean value from the configuration.
func (c *Config) GetBool(key string, defaultValue bool) bool {
	val := c.Get(key, defaultValue)
	switch v := val.(type) {
	case bool:
		return v
	case string:
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return defaultValue
}

// GetFloat64 retrieves a float64 value from the configuration.
func (c *Config) GetFloat64(key string, defaultValue float64) float64 {
	val := c.Get(key, defaultValue)
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return defaultValue
}

// GetDuration retrieves a duration value from the configuration.
func (c *Config) GetDuration(key string, defaultValue time.Duration) time.Duration {
	val := c.Get(key, defaultValue)
	switch v := val.(type) {
	case time.Duration:
		return v
	case string:
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultValue
}

// Has checks if a key exists in the configuration.
func (c *Config) Has(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.data[key]
	return ok
}

// Keys returns all configuration keys.
func (c *Config) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.data))
	for k := range c.data {
		keys = append(keys, k)
	}
	return keys
}

// Clear clears all configuration data and cache.
func (c *Config) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]any)
	if c.cache != nil {
		c.cache.Clear()
	}
}

// Size returns the number of configuration entries.
func (c *Config) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}
