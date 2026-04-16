package parser

import (
	"os"
	"strings"

	"github.com/kitsunium/sdk/internal/core/config/normalize"
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
	matchCount := e.countMatchingVars(envVars)
	config := make(map[string]string, matchCount)

	for _, env := range envVars {
		key, value, ok := e.parseEnvVar(env)
		if !ok {
			continue
		}
		config[normalize.Key(key)] = normalize.Value(value)
	}

	return config, nil
}

func (e *ENV) countMatchingVars(envVars []string) int {
	count := 0
	prefixLen := len(e.Prefix)

	for _, env := range envVars {
		idx := strings.IndexByte(env, '=')
		if idx == -1 {
			continue
		}
		if prefixLen > 0 && !strings.HasPrefix(env[:idx], e.Prefix) {
			continue
		}
		count++
	}
	return count
}

func (e *ENV) parseEnvVar(env string) (string, string, bool) {
	idx := strings.IndexByte(env, '=')
	if idx == -1 {
		return "", "", false
	}

	key := env[:idx]
	value := env[idx+1:]

	if len(e.Prefix) > 0 {
		if !strings.HasPrefix(key, e.Prefix) {
			return "", "", false
		}
		key = e.processPrefix(key)
	}

	return key, value, true
}

func (e *ENV) processPrefix(key string) string {
	key = key[len(e.Prefix):]
	if len(key) > 0 && key[0] == '_' {
		key = key[1:]
	}
	return key
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
		if !filter(env) {
			continue
		}

		value := env[idx+1:]
		config[normalize.Key(key)] = normalize.Value(value)
	}

	return config, nil
}
