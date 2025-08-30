package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Comprehensive Parser Package Benchmarks
// This file contains cross-format performance comparisons and memory usage analysis

var (
	// Sample data structures for benchmarking
	smallConfigData = map[string]interface{}{
		"database": map[string]interface{}{
			"host":     "localhost",
			"port":     5432,
			"username": "admin",
			"password": "secret",
		},
		"server": map[string]interface{}{
			"host":    "0.0.0.0",
			"port":    8080,
			"workers": 4,
			"debug":   true,
		},
	}

	mediumConfigData = generateMediumConfigData()
	largeConfigData  = generateLargeConfigData()
)

// Data generators for different sizes
func generateMediumConfigData() map[string]interface{} {
	data := make(map[string]interface{})
	for i := 0; i < 50; i++ {
		section := make(map[string]interface{})
		for j := 0; j < 10; j++ {
			section[fmt.Sprintf("key_%d", j)] = fmt.Sprintf("value_%d_%d", i, j)
		}
		section["number"] = i * 10
		section["enabled"] = i%2 == 0
		data[fmt.Sprintf("section_%d", i)] = section
	}
	return data
}

func generateLargeConfigData() map[string]interface{} {
	data := make(map[string]interface{})
	for i := 0; i < 500; i++ {
		section := make(map[string]interface{})
		for j := 0; j < 20; j++ {
			section[fmt.Sprintf("key_%d", j)] = fmt.Sprintf("value_%d_%d", i, j)
		}

		// Add nested structures
		nested := make(map[string]interface{})
		for k := 0; k < 5; k++ {
			nested[fmt.Sprintf("nested_key_%d", k)] = fmt.Sprintf("nested_value_%d_%d", i, k)
		}
		section["nested"] = nested

		// Add arrays
		array := make([]interface{}, 10)
		for a := 0; a < 10; a++ {
			array[a] = fmt.Sprintf("item_%d_%d", i, a)
		}
		section["items"] = array

		section["number"] = i * 10
		section["enabled"] = i%2 == 0
		data[fmt.Sprintf("section_%d", i)] = section
	}
	return data
}

// Format-specific content generators
func generateJSONContent(data map[string]interface{}) ([]byte, error) {
	return json.MarshalIndent(data, "", "  ")
}

func generateYAMLContent(data map[string]interface{}) []byte {
	var buf bytes.Buffer
	writeYAMLMap(&buf, data, 0)
	return buf.Bytes()
}

func writeYAMLMap(buf *bytes.Buffer, data map[string]interface{}, indent int) {
	indentStr := strings.Repeat("  ", indent)
	for key, value := range data {
		buf.WriteString(fmt.Sprintf("%s%s:", indentStr, key))
		switch v := value.(type) {
		case map[string]interface{}:
			buf.WriteString("\n")
			writeYAMLMap(buf, v, indent+1)
		case []interface{}:
			buf.WriteString("\n")
			for _, item := range v {
				buf.WriteString(fmt.Sprintf("%s  - %v\n", indentStr, item))
			}
		default:
			buf.WriteString(fmt.Sprintf(" %v\n", v))
		}
	}
}

func generateTOMLContent(data map[string]interface{}) []byte {
	var buf bytes.Buffer
	writeTOMLMap(&buf, data, "")
	return buf.Bytes()
}

func writeTOMLMap(buf *bytes.Buffer, data map[string]interface{}, prefix string) {
	for key, value := range data {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			buf.WriteString(fmt.Sprintf("[%s]\n", fullKey))
			for subKey, subValue := range v {
				if _, ok := subValue.(map[string]interface{}); ok {
					writeTOMLMap(buf, map[string]interface{}{subKey: subValue}, fullKey)
				} else {
					buf.WriteString(fmt.Sprintf("%s = %v\n", subKey, formatTOMLValue(subValue)))
				}
			}
			buf.WriteString("\n")
		default:
			buf.WriteString(fmt.Sprintf("%s = %v\n", key, formatTOMLValue(v)))
		}
	}
}

func formatTOMLValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case []interface{}:
		var items []string
		for _, item := range v {
			items = append(items, fmt.Sprintf("\"%v\"", item))
		}
		return fmt.Sprintf("[%s]", strings.Join(items, ", "))
	default:
		return fmt.Sprintf("%v", v)
	}
}

func generateXMLContent(data map[string]interface{}) []byte {
	var buf bytes.Buffer
	buf.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<root>\n")
	writeXMLMap(&buf, data, 1)
	buf.WriteString("</root>")
	return buf.Bytes()
}

func writeXMLMap(buf *bytes.Buffer, data map[string]interface{}, indent int) {
	indentStr := strings.Repeat("  ", indent)
	for key, value := range data {
		switch v := value.(type) {
		case map[string]interface{}:
			buf.WriteString(fmt.Sprintf("%s<%s>\n", indentStr, key))
			writeXMLMap(buf, v, indent+1)
			buf.WriteString(fmt.Sprintf("%s</%s>\n", indentStr, key))
		case []interface{}:
			for _, item := range v {
				buf.WriteString(fmt.Sprintf("%s<%s>%v</%s>\n", indentStr, key, item, key))
			}
		default:
			buf.WriteString(fmt.Sprintf("%s<%s>%v</%s>\n", indentStr, key, v, key))
		}
	}
}

func generateINIContent(data map[string]interface{}) []byte {
	var buf bytes.Buffer
	for key, value := range data {
		if sectionMap, ok := value.(map[string]interface{}); ok {
			buf.WriteString(fmt.Sprintf("[%s]\n", key))
			for subKey, subValue := range sectionMap {
				if _, ok := subValue.(map[string]interface{}); !ok {
					buf.WriteString(fmt.Sprintf("%s = %v\n", subKey, subValue))
				}
			}
			buf.WriteString("\n")
		}
	}
	return buf.Bytes()
}

// Cross-Format Parsing Benchmarks

func BenchmarkParser_Small_JSON_vs_YAML_vs_TOML_vs_XML_vs_INI(b *testing.B) {
	jsonContent, _ := generateJSONContent(smallConfigData)
	yamlContent := generateYAMLContent(smallConfigData)
	tomlContent := generateTOMLContent(smallConfigData)
	xmlContent := generateXMLContent(smallConfigData)
	iniContent := generateINIContent(smallConfigData)

	b.Run("JSON", func(b *testing.B) {
		j := NewJSON("")
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = j.LoadBytes(jsonContent)
		}
	})

	b.Run("YAML", func(b *testing.B) {
		y := NewYAML("")
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = y.LoadBytes(yamlContent)
		}
	})

	b.Run("TOML", func(b *testing.B) {
		t := NewTOML("")
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = t.LoadBytes(tomlContent)
		}
	})

	b.Run("XML", func(b *testing.B) {
		x := NewXML("")
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = x.LoadBytes(xmlContent)
		}
	})

	b.Run("INI", func(b *testing.B) {
		ini := NewINI("")
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = ini.LoadBytes(iniContent)
		}
	})
}

func BenchmarkParser_Medium_JSON_vs_YAML_vs_TOML_vs_XML(b *testing.B) {
	jsonContent, _ := generateJSONContent(mediumConfigData)
	yamlContent := generateYAMLContent(mediumConfigData)
	tomlContent := generateTOMLContent(mediumConfigData)
	xmlContent := generateXMLContent(mediumConfigData)

	b.Run("JSON", func(b *testing.B) {
		j := NewJSON("")
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = j.LoadBytes(jsonContent)
		}
	})

	b.Run("YAML", func(b *testing.B) {
		y := NewYAML("")
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = y.LoadBytes(yamlContent)
		}
	})

	b.Run("TOML", func(b *testing.B) {
		t := NewTOML("")
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = t.LoadBytes(tomlContent)
		}
	})

	b.Run("XML", func(b *testing.B) {
		x := NewXML("")
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = x.LoadBytes(xmlContent)
		}
	})
}

func BenchmarkParser_Large_JSON_vs_YAML_vs_XML(b *testing.B) {
	jsonContent, _ := generateJSONContent(largeConfigData)
	yamlContent := generateYAMLContent(largeConfigData)
	xmlContent := generateXMLContent(largeConfigData)

	b.Run("JSON", func(b *testing.B) {
		j := NewJSON("")
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = j.LoadBytes(jsonContent)
		}
	})

	b.Run("YAML", func(b *testing.B) {
		y := NewYAML("")
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = y.LoadBytes(yamlContent)
		}
	})

	b.Run("XML", func(b *testing.B) {
		x := NewXML("")
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = x.LoadBytes(xmlContent)
		}
	})
}

// Concurrent parsing benchmarks across formats
func BenchmarkParser_Concurrent_All_Formats(b *testing.B) {
	jsonContent, _ := generateJSONContent(mediumConfigData)
	yamlContent := generateYAMLContent(mediumConfigData)
	tomlContent := generateTOMLContent(mediumConfigData)
	xmlContent := generateXMLContent(mediumConfigData)
	iniContent := generateINIContent(mediumConfigData)

	b.Run("JSON_Concurrent", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				j := NewJSON("")
				_, _ = j.LoadBytes(jsonContent)
			}
		})
	})

	b.Run("YAML_Concurrent", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				y := NewYAML("")
				_, _ = y.LoadBytes(yamlContent)
			}
		})
	})

	b.Run("TOML_Concurrent", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				t := NewTOML("")
				_, _ = t.LoadBytes(tomlContent)
			}
		})
	})

	b.Run("XML_Concurrent", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				x := NewXML("")
				_, _ = x.LoadBytes(xmlContent)
			}
		})
	})

	b.Run("INI_Concurrent", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ini := NewINI("")
				_, _ = ini.LoadBytes(iniContent)
			}
		})
	})
}

// Memory usage analysis benchmarks
func BenchmarkParser_Memory_Usage_Analysis(b *testing.B) {
	jsonContent, _ := generateJSONContent(largeConfigData)
	yamlContent := generateYAMLContent(largeConfigData)
	xmlContent := generateXMLContent(largeConfigData)

	b.Run("JSON_Memory", func(b *testing.B) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		j := NewJSON("")
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = j.LoadBytes(jsonContent)
		}

		runtime.GC()
		runtime.ReadMemStats(&m2)
		b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
	})

	b.Run("YAML_Memory", func(b *testing.B) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		y := NewYAML("")
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = y.LoadBytes(yamlContent)
		}

		runtime.GC()
		runtime.ReadMemStats(&m2)
		b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
	})

	b.Run("XML_Memory", func(b *testing.B) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		x := NewXML("")
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = x.LoadBytes(xmlContent)
		}

		runtime.GC()
		runtime.ReadMemStats(&m2)
		b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
	})
}

// File-based loading benchmarks
func BenchmarkParser_File_Loading_All_Formats(b *testing.B) {
	tmpDir := b.TempDir()

	// Create test files
	jsonContent, _ := generateJSONContent(mediumConfigData)
	yamlContent := generateYAMLContent(mediumConfigData)
	tomlContent := generateTOMLContent(mediumConfigData)
	xmlContent := generateXMLContent(mediumConfigData)
	iniContent := generateINIContent(mediumConfigData)

	jsonPath := filepath.Join(tmpDir, "config.json")
	yamlPath := filepath.Join(tmpDir, "config.yaml")
	tomlPath := filepath.Join(tmpDir, "config.toml")
	xmlPath := filepath.Join(tmpDir, "config.xml")
	iniPath := filepath.Join(tmpDir, "config.ini")

	os.WriteFile(jsonPath, jsonContent, 0644)
	os.WriteFile(yamlPath, yamlContent, 0644)
	os.WriteFile(tomlPath, tomlContent, 0644)
	os.WriteFile(xmlPath, xmlContent, 0644)
	os.WriteFile(iniPath, iniContent, 0644)

	b.Run("JSON_File_Load", func(b *testing.B) {
		j := NewJSON(jsonPath)
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = j.Load()
		}
	})

	b.Run("YAML_File_Load", func(b *testing.B) {
		y := NewYAML(yamlPath)
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = y.Load()
		}
	})

	b.Run("TOML_File_Load", func(b *testing.B) {
		t := NewTOML(tomlPath)
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = t.Load()
		}
	})

	b.Run("XML_File_Load", func(b *testing.B) {
		x := NewXML(xmlPath)
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = x.Load()
		}
	})

	b.Run("INI_File_Load", func(b *testing.B) {
		ini := NewINI(iniPath)
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = ini.Load()
		}
	})
}

// Pool vs Non-Pool performance comparison for formats that support it
func BenchmarkParser_Pool_vs_NonPool(b *testing.B) {
	yamlContent := generateYAMLContent(mediumConfigData)
	tomlContent := generateTOMLContent(mediumConfigData)

	b.Run("YAML_WithPool", func(b *testing.B) {
		y := NewYAML("", WithPool(true))
		reader := strings.NewReader(string(yamlContent))
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			reader.Reset(string(yamlContent))
			_, _ = y.LoadReader(reader)
		}
	})

	b.Run("YAML_WithoutPool", func(b *testing.B) {
		y := NewYAML("", WithPool(false))
		reader := strings.NewReader(string(yamlContent))
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			reader.Reset(string(yamlContent))
			_, _ = y.LoadReader(reader)
		}
	})

	b.Run("TOML_WithPool", func(b *testing.B) {
		t := NewTOML("", WithPool(true))
		reader := strings.NewReader(string(tomlContent))
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			reader.Reset(string(tomlContent))
			_, _ = t.LoadReader(reader)
		}
	})

	b.Run("TOML_WithoutPool", func(b *testing.B) {
		t := NewTOML("", WithPool(false))
		reader := strings.NewReader(string(tomlContent))
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			reader.Reset(string(tomlContent))
			_, _ = t.LoadReader(reader)
		}
	})
}

// ENV parser specific benchmarks
func BenchmarkParser_ENV_Various_Sizes(b *testing.B) {
	// Setup different environment variable scenarios
	b.Run("ENV_Small", func(b *testing.B) {
		// Setup small environment
		for i := 0; i < 10; i++ {
			os.Setenv(fmt.Sprintf("BENCH_SMALL_VAR_%d", i), fmt.Sprintf("value_%d", i))
			defer os.Unsetenv(fmt.Sprintf("BENCH_SMALL_VAR_%d", i))
		}

		env := NewENV("BENCH_SMALL_")
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = env.Load()
		}
	})

	b.Run("ENV_Medium", func(b *testing.B) {
		// Setup medium environment
		for i := 0; i < 100; i++ {
			os.Setenv(fmt.Sprintf("BENCH_MEDIUM_VAR_%d", i), fmt.Sprintf("value_%d", i))
			defer os.Unsetenv(fmt.Sprintf("BENCH_MEDIUM_VAR_%d", i))
		}

		env := NewENV("BENCH_MEDIUM_")
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = env.Load()
		}
	})

	b.Run("ENV_Large", func(b *testing.B) {
		// Setup large environment
		for i := 0; i < 1000; i++ {
			os.Setenv(fmt.Sprintf("BENCH_LARGE_VAR_%d", i), fmt.Sprintf("value_%d", i))
			defer os.Unsetenv(fmt.Sprintf("BENCH_LARGE_VAR_%d", i))
		}

		env := NewENV("BENCH_LARGE_")
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = env.Load()
		}
	})
}

// ARGS parser specific benchmarks
func BenchmarkParser_ARGS_Various_Sizes(b *testing.B) {
	smallArgs := []string{"--host=localhost", "--port=8080", "--debug"}
	mediumArgs := make([]string, 50)
	largeArgs := make([]string, 500)

	for i := 0; i < 50; i++ {
		mediumArgs[i] = fmt.Sprintf("--key%d=value%d", i, i)
	}
	for i := 0; i < 500; i++ {
		largeArgs[i] = fmt.Sprintf("--key%d=value%d", i, i)
	}

	b.Run("ARGS_Small", func(b *testing.B) {
		parser := NewARGS(false)
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = parser.ParseArgs(smallArgs)
		}
	})

	b.Run("ARGS_Medium", func(b *testing.B) {
		parser := NewARGS(false)
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = parser.ParseArgs(mediumArgs)
		}
	})

	b.Run("ARGS_Large", func(b *testing.B) {
		parser := NewARGS(false)
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = parser.ParseArgs(largeArgs)
		}
	})

	b.Run("ARGS_Strict_vs_Normal", func(b *testing.B) {
		parser := NewARGS(false)
		validArgs := []string{"--valid=value", "--another=test"}

		b.Run("Normal", func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = parser.ParseArgs(validArgs)
			}
		})

		b.Run("Strict", func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = parser.ParseArgsStrict(validArgs)
			}
		})
	})
}

// Real-world scenario benchmarks
func BenchmarkParser_RealWorld_ConfigFiles(b *testing.B) {
	// Database configuration scenario
	dbConfig := map[string]interface{}{
		"database": map[string]interface{}{
			"host":               "localhost",
			"port":               5432,
			"username":           "admin",
			"password":           "secret123",
			"database":           "production_db",
			"pool_size":          20,
			"connection_timeout": 30,
			"idle_timeout":       300,
			"max_lifetime":       3600,
			"ssl_mode":           "require",
			"auto_migrate":       true,
		},
		"redis": map[string]interface{}{
			"host":           "redis.example.com",
			"port":           6379,
			"password":       "redis_secret",
			"db":             0,
			"pool_size":      10,
			"min_idle_conns": 5,
			"dial_timeout":   5,
			"read_timeout":   3,
			"write_timeout":  3,
		},
		"server": map[string]interface{}{
			"host":               "0.0.0.0",
			"port":               8080,
			"read_timeout":       30,
			"write_timeout":      30,
			"idle_timeout":       120,
			"max_header_bytes":   1048576,
			"enable_compression": true,
			"enable_cors":        true,
			"cors_origins":       []interface{}{"*"},
		},
		"logging": map[string]interface{}{
			"level":       "info",
			"format":      "json",
			"output":      "stdout",
			"enable_file": true,
			"file_path":   "/var/log/app.log",
			"max_size":    100,
			"max_backups": 5,
			"max_age":     30,
		},
	}

	jsonContent, _ := generateJSONContent(dbConfig)
	yamlContent := generateYAMLContent(dbConfig)

	b.Run("DatabaseConfig_JSON", func(b *testing.B) {
		j := NewJSON("")
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = j.LoadBytes(jsonContent)
		}
	})

	b.Run("DatabaseConfig_YAML", func(b *testing.B) {
		y := NewYAML("")
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = y.LoadBytes(yamlContent)
		}
	})
}
