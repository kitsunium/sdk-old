package parser

import (
	"os"
	"strings"

	"github.com/kitsunium/sdk/pkg/core/config/normalize"
)

// ENV is a parser for environment variables.
// It can optionally filter variables by prefix.
type ENV struct {
	Prefix string // Optional prefix to filter environment variables
}

// NewENV creates a new environment variable parser.
// If prefix is not empty, only variables starting with that prefix are included.
func NewENV(prefix string) *ENV {
	return &ENV{
		Prefix: prefix,
	}
}

func (e *ENV) Type() string {
	return "env"
}

func (e *ENV) Load() (map[string]string, error) {
	envVars := os.Environ()

	// Pre-count matching env vars for perfect allocation (like we did for ARGS)
	matchCount := 0
	prefixLen := len(e.Prefix)

	if prefixLen > 0 {
		// Count only matching vars
		for _, env := range envVars {
			if idx := strings.IndexByte(env, '='); idx >= prefixLen {
				if strings.HasPrefix(env[:idx], e.Prefix) {
					matchCount++
				}
			}
		}
	} else {
		// Count all valid env vars
		for _, env := range envVars {
			if strings.IndexByte(env, '=') != -1 {
				matchCount++
			}
		}
	}

	// Perfect size allocation to prevent rehashing
	config := make(map[string]string, matchCount)

	for _, env := range envVars {
		idx := strings.IndexByte(env, '=')
		if idx == -1 {
			continue
		}

		key := env[:idx]
		value := env[idx+1:]

		// Optimized prefix handling
		if prefixLen > 0 {
			if !strings.HasPrefix(key, e.Prefix) {
				continue
			}
			key = key[prefixLen:]
			// Strip optional underscore after prefix
			if len(key) > 0 && key[0] == '_' {
				key = key[1:]
			}
		}

		config[normalize.Key(key)] = normalize.Value(value)
	}

	return config, nil
}

// LoadFiltered parses environment variables with a custom filter function.
// The filter receives the full KEY=VALUE string and returns true to include it.
//
// Example:
//
//	parser := NewENV("")
//	config, _ := parser.LoadFiltered(func(env string) bool {
//	    return strings.HasPrefix(env, "DB_")
//	})
func (e *ENV) LoadFiltered(filter func(string) bool) (map[string]string, error) {
	envVars := os.Environ()
	config := make(map[string]string)

	for _, env := range envVars {
		idx := strings.IndexByte(env, '=')
		if idx == -1 {
			continue
		}

		key := env[:idx]
		if !filter(key) {
			continue
		}

		value := env[idx+1:]
		config[normalize.Key(key)] = normalize.Value(value)
	}

	return config, nil
}
