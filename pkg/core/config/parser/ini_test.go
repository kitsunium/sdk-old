package parser

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestINI_Type(t *testing.T) {
	ini := NewINI("test.ini")
	if ini.Type() != "ini" {
		t.Errorf("Type() = %q, want %q", ini.Type(), "ini")
	}
}

func TestINI_Load_ValidFile(t *testing.T) {
	// Create a temporary INI file
	tmpDir := t.TempDir()
	iniPath := filepath.Join(tmpDir, "test.ini")

	content := `
# This is a comment
[database]
host = localhost
port = 5432
name = testdb

[server]
host = 0.0.0.0
port = 8080
debug = true

[paths]
data = /var/data
logs = /var/logs
`

	if err := os.WriteFile(iniPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ini := NewINI(iniPath)
	result, err := ini.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	expected := map[string]string{
		"database.host": "localhost",
		"database.port": "5432",
		"database.name": "testdb",
		"server.host":   "0.0.0.0",
		"server.port":   "8080",
		"server.debug":  "true",
		"paths.data":    "/var/data",
		"paths.logs":    "/var/logs",
	}

	for key, expectedValue := range expected {
		if value, ok := result[key]; !ok || value != expectedValue {
			t.Errorf("key %q = %q, want %q", key, value, expectedValue)
		}
	}
}

func TestINI_Load_InvalidExtension(t *testing.T) {
	// Test with invalid extension
	ini := NewINI("test.txt")
	_, err := ini.Load()
	if err == nil {
		t.Error("Load() should error on invalid extension")
	}
	if !errors.Is(err, ErrInvalidExtension) {
		t.Errorf("Expected ErrInvalidExtension, got: %v", err)
	}
}

func TestINI_Load_NonExistentFile(t *testing.T) {
	ini := NewINI("/non/existent/file.ini")
	_, err := ini.Load()
	if err == nil {
		t.Error("Load() should error on non-existent file")
	}
	if !errors.Is(err, ErrFileNotFound) {
		t.Errorf("Expected ErrFileNotFound, got: %v", err)
	}
}

func TestINI_Load_ValidExtensions(t *testing.T) {
	tmpDir := t.TempDir()
	extensions := []string{".ini", ".cfg", ".conf"}

	for _, ext := range extensions {
		path := filepath.Join(tmpDir, "test"+ext)
		content := "[section]\nkey=value"

		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		ini := NewINI(path)
		result, err := ini.Load()
		if err != nil {
			t.Errorf("Load() error for %s = %v", ext, err)
		}

		if result["section.key"] != "value" {
			t.Errorf("section.key for %s = %q, want %q", ext, result["section.key"], "value")
		}
	}
}

func TestINI_LoadReader_BasicSyntax(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected map[string]string
	}{
		{
			name: "equals separator",
			content: `
[section]
key1=value1
key2 = value2
key3 =value3
`,
			expected: map[string]string{
				"section.key1": "value1",
				"section.key2": "value2",
				"section.key3": "value3",
			},
		},
		{
			name: "colon separator",
			content: `
[section]
key1:value1
key2 : value2
key3 :value3
`,
			expected: map[string]string{
				"section.key1": "value1",
				"section.key2": "value2",
				"section.key3": "value3",
			},
		},
		{
			name: "mixed separators",
			content: `
[section]
key1=value1
key2:value2
key3 = value3
key4 : value4
`,
			expected: map[string]string{
				"section.key1": "value1",
				"section.key2": "value2",
				"section.key3": "value3",
				"section.key4": "value4",
			},
		},
		{
			name: "quoted values",
			content: `
[section]
single='single quoted'
double="double quoted"
mixed="has 'quotes' inside"
unquoted=no quotes
`,
			expected: map[string]string{
				"section.single":   "single quoted",
				"section.double":   "double quoted",
				"section.mixed":    "has 'quotes' inside",
				"section.unquoted": "no quotes",
			},
		},
		{
			name: "no section",
			content: `
global1=value1
global2=value2
[section]
key=value
`,
			expected: map[string]string{
				"global1":     "value1",
				"global2":     "value2",
				"section.key": "value",
			},
		},
		{
			name: "multiple sections",
			content: `
[section1]
key1=value1
[section2]
key2=value2
[section3]
key3=value3
`,
			expected: map[string]string{
				"section1.key1": "value1",
				"section2.key2": "value2",
				"section3.key3": "value3",
			},
		},
		{
			name: "comments",
			content: `
# This is a comment
[section]
# Another comment
key1=value1
; Semicolon comment
key2=value2
  # Indented comment
key3=value3
`,
			expected: map[string]string{
				"section.key1": "value1",
				"section.key2": "value2",
				"section.key3": "value3",
			},
		},
		{
			name: "empty lines",
			content: `

[section]

key1=value1

key2=value2

`,
			expected: map[string]string{
				"section.key1": "value1",
				"section.key2": "value2",
			},
		},
		{
			name: "whitespace handling",
			content: `
[section]
  key1  =  value1  
	key2	=	value2	
key3 = " preserved spaces "
`,
			expected: map[string]string{
				"section.key1": "value1",
				"section.key2": "value2",
				"section.key3": "preserved spaces", // Quotes are stripped by the parser
			},
		},
		{
			name: "values with trailing spaces and tabs",
			content: `
[section]
trailing_space = value   
trailing_tab = value	
mixed = value 	 
normal = regular value
`,
			expected: map[string]string{
				"section.trailing.space": "value",
				"section.trailing.tab":   "value",
				"section.mixed":          "value",
				"section.normal":         "regular value",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ini := NewINI("")
			reader := strings.NewReader(tc.content)
			result, err := ini.LoadReader(reader)
			if err != nil {
				t.Fatalf("LoadReader() error = %v", err)
			}

			if len(result) != len(tc.expected) {
				t.Errorf("Result has %d items, want %d", len(result), len(tc.expected))
			}

			for key, expectedValue := range tc.expected {
				if value, ok := result[key]; !ok || value != expectedValue {
					t.Errorf("key %q = %q, want %q", key, value, expectedValue)
				}
			}
		})
	}
}

func TestINI_LoadReader_EdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected map[string]string
	}{
		{
			name:     "empty content",
			content:  "",
			expected: map[string]string{},
		},
		{
			name: "only comments",
			content: `
# Comment 1
; Comment 2
# Comment 3
`,
			expected: map[string]string{},
		},
		{
			name: "only sections",
			content: `
[section1]
[section2]
[section3]
`,
			expected: map[string]string{},
		},
		{
			name: "invalid lines",
			content: `
[section]
this is not a key value pair
neither is this
key=value
also not valid
`,
			expected: map[string]string{
				"section.key": "value",
			},
		},
		{
			name: "section normalization",
			content: `
[DATABASE_CONFIG]
host=localhost
[Server_Settings]
port=8080
[mixed.case.SECTION]
value=test
`,
			expected: map[string]string{
				"database.config.host":     "localhost",
				"server.settings.port":     "8080",
				"mixed.case.section.value": "test",
			},
		},
		{
			name: "key normalization",
			content: `
[section]
UPPER_KEY=value1
Mixed_Key=value2
lower_key=value3
already.dot.key=value4
`,
			expected: map[string]string{
				"section.upper.key":       "value1",
				"section.mixed.key":       "value2",
				"section.lower.key":       "value3",
				"section.already.dot.key": "value4",
			},
		},
		{
			name: "value normalization",
			content: `
[section]
trimmed=  value with spaces  
quoted="quoted value"
single='single quoted'
backtick=` + "`backtick value`" + `
`,
			expected: map[string]string{
				"section.trimmed":  "value with spaces",
				"section.quoted":   "quoted value",
				"section.single":   "single quoted",
				"section.backtick": "`backtick value`",
			},
		},
		{
			name: "empty values",
			content: `
[section]
empty1=
empty2=  
empty3=""
empty4=''
`,
			expected: map[string]string{
				"section.empty1": "",
				"section.empty2": "",
				"section.empty3": "",
				"section.empty4": "",
			},
		},
		{
			name: "special characters in values",
			content: `
[section]
url=https://example.com/path?query=value&other=123
path=/usr/local/bin
email=user@example.com
json={"key":"value","array":[1,2,3]}
`,
			expected: map[string]string{
				"section.url":   "https://example.com/path?query=value&other=123",
				"section.path":  "/usr/local/bin",
				"section.email": "user@example.com",
				"section.json":  `{"key":"value","array":[1,2,3]}`,
			},
		},
		{
			name: "equals in value",
			content: `
[section]
equation=a=b+c
config=key1=val1,key2=val2
`,
			expected: map[string]string{
				"section.equation": "a=b+c",
				"section.config":   "key1=val1,key2=val2",
			},
		},
		{
			name: "colon in value",
			content: `
[section]
time=12:30:45
ratio=16:9
`,
			expected: map[string]string{
				"section.time":  "12:30:45",
				"section.ratio": "16:9",
			},
		},
		{
			name: "section switching",
			content: `
key1=global1
[section1]
key2=value2
key3=value3
[section2]
key4=value4
key5=global5
`,
			expected: map[string]string{
				"key1":          "global1",
				"section1.key2": "value2",
				"section1.key3": "value3",
				"section2.key4": "value4",
				"section2.key5": "global5",
			},
		},
		{
			name: "malformed sections",
			content: `
[valid]
key1=value1
[incomplete
key2=value2
incomplete]
key3=value3
[]
key4=value4
[spaces]
key5=value5
`,
			expected: map[string]string{
				"valid.key1":  "value1",
				"valid.key2":  "value2",
				"valid.key3":  "value3",
				"key4":        "value4",
				"spaces.key5": "value5",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ini := NewINI("")
			reader := strings.NewReader(tc.content)
			result, err := ini.LoadReader(reader)
			if err != nil {
				t.Fatalf("LoadReader() error = %v", err)
			}

			if len(result) != len(tc.expected) {
				t.Errorf("Result has %d items, want %d", len(result), len(tc.expected))
				for k, v := range result {
					t.Logf("Got: %q = %q", k, v)
				}
				for k, v := range tc.expected {
					t.Logf("Want: %q = %q", k, v)
				}
			}

			for key, expectedValue := range tc.expected {
				if value, ok := result[key]; !ok || value != expectedValue {
					t.Errorf("key %q = %q (exists=%v), want %q", key, value, ok, expectedValue)
				}
			}
		})
	}
}

func TestINI_LoadBytes(t *testing.T) {
	content := []byte(`
[database]
host=localhost
port=5432

[server]
host=0.0.0.0
port=8080
`)

	ini := NewINI("")
	result, err := ini.LoadBytes(content)
	if err != nil {
		t.Fatalf("LoadBytes() error = %v", err)
	}

	expected := map[string]string{
		"database.host": "localhost",
		"database.port": "5432",
		"server.host":   "0.0.0.0",
		"server.port":   "8080",
	}

	for key, expectedValue := range expected {
		if value, ok := result[key]; !ok || value != expectedValue {
			t.Errorf("key %q = %q, want %q", key, value, expectedValue)
		}
	}
}

func TestINI_LoadBytes_Empty(t *testing.T) {
	ini := NewINI("")
	result, err := ini.LoadBytes([]byte{})
	if err != nil {
		t.Fatalf("LoadBytes() error = %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result for empty bytes, got %d items", len(result))
	}
}

func TestINI_LoadBytes_LargeContent(t *testing.T) {
	// Generate large content
	var buf bytes.Buffer
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&buf, "[section%d]\n", i)
		for j := 0; j < 10; j++ {
			fmt.Fprintf(&buf, "key%d=value%d_%d\n", j, i, j)
		}
	}

	ini := NewINI("")
	result, err := ini.LoadBytes(buf.Bytes())
	if err != nil {
		t.Fatalf("LoadBytes() error = %v", err)
	}

	// Check we have all expected entries
	expectedCount := 100 * 10 // 100 sections, 10 keys each
	if len(result) != expectedCount {
		t.Errorf("Expected %d entries, got %d", expectedCount, len(result))
	}

	// Spot check a few values
	if result["section0.key0"] != "value0_0" {
		t.Errorf("section0.key0 = %q, want %q", result["section0.key0"], "value0_0")
	}
	if result["section50.key5"] != "value50_5" {
		t.Errorf("section50.key5 = %q, want %q", result["section50.key5"], "value50_5")
	}
	if result["section99.key9"] != "value99_9" {
		t.Errorf("section99.key9 = %q, want %q", result["section99.key9"], "value99_9")
	}
}

func TestINI_ParserOptions(t *testing.T) {
	content := `
[section]
key1=value1
key2=value2
`

	// Test with custom buffer size
	ini := NewINI("", WithBufferSize(1024))
	if ini.options.bufferSize != 1024 {
		t.Errorf("bufferSize = %d, want %d", ini.options.bufferSize, 1024)
	}

	reader := strings.NewReader(content)
	result, err := ini.LoadReader(reader)
	if err != nil {
		t.Fatalf("LoadReader() error = %v", err)
	}

	if result["section.key1"] != "value1" {
		t.Errorf("section.key1 = %q, want %q", result["section.key1"], "value1")
	}

	// Test with pool disabled
	ini2 := NewINI("", WithPool(false))
	if ini2.options.usePool {
		t.Error("usePool should be false")
	}

	reader2 := strings.NewReader(content)
	result2, err := ini2.LoadReader(reader2)
	if err != nil {
		t.Fatalf("LoadReader() error = %v", err)
	}

	if result2["section.key2"] != "value2" {
		t.Errorf("section.key2 = %q, want %q", result2["section.key2"], "value2")
	}
}

func TestINI_LoadReader_ReaderError(t *testing.T) {
	// Create a reader that returns an error
	reader := &errorReader{err: io.ErrUnexpectedEOF}

	ini := NewINI("")
	_, err := ini.LoadReader(reader)
	if err == nil {
		t.Error("LoadReader() should return error from reader")
	}
	if !errors.Is(err, ErrReadFailed) {
		t.Errorf("Expected ErrReadFailed, got: %v", err)
	}
}

type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

func TestINI_RealWorldExample(t *testing.T) {
	content := `
# Application Configuration
# Generated: 2024-01-01

[database]
host = localhost
port = 5432
name = myapp_production
user = dbuser
password = "secret123"
pool_size = 10

[redis]
host = redis.local
port = 6379
db = 0
password = 
; Redis clustering is disabled
cluster_enabled = false

[logging]
level = info
format = json
output = stdout
file = /var/log/app.log
max_size = 100
max_backups = 3
max_age = 30

[features]
new_ui = true
beta_features = false
experimental = false
feature_flags = flag1,flag2,flag3

[server]
host = 0.0.0.0
port = 8080
read_timeout = 30
write_timeout = 30
idle_timeout = 120
max_header_bytes = 1048576

[security]
jwt_secret = "super-secret-key-do-not-share"
bcrypt_cost = 10
session_timeout = 3600
csrf_enabled = true
cors_origins = https://example.com,https://app.example.com
`

	ini := NewINI("")
	reader := strings.NewReader(content)
	result, err := ini.LoadReader(reader)
	if err != nil {
		t.Fatalf("LoadReader() error = %v", err)
	}

	// Check a selection of important values
	checks := map[string]string{
		"database.host":           "localhost",
		"database.password":       "secret123",
		"redis.password":          "",
		"redis.cluster.enabled":   "false",
		"logging.level":           "info",
		"logging.max.size":        "100",
		"features.new.ui":         "true",
		"server.max.header.bytes": "1048576",
		"security.jwt.secret":     "super-secret-key-do-not-share",
		"security.cors.origins":   "https://example.com,https://app.example.com",
	}

	for key, expectedValue := range checks {
		if value, ok := result[key]; !ok || value != expectedValue {
			t.Errorf("key %q = %q (exists=%v), want %q", key, value, ok, expectedValue)
		}
	}
}

func BenchmarkINI_LoadReader_Small(b *testing.B) {
	content := `
[section1]
key1=value1
key2=value2

[section2]
key3=value3
key4=value4
`

	ini := NewINI("")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(content)
		_, _ = ini.LoadReader(reader)
	}
}

func BenchmarkINI_LoadReader_Medium(b *testing.B) {
	var buf bytes.Buffer
	for i := 0; i < 10; i++ {
		fmt.Fprintf(&buf, "[section%d]\n", i)
		for j := 0; j < 10; j++ {
			fmt.Fprintf(&buf, "key%d=value%d\n", j, j)
		}
	}
	content := buf.String()

	ini := NewINI("")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(content)
		_, _ = ini.LoadReader(reader)
	}
}

func BenchmarkINI_LoadReader_Large(b *testing.B) {
	var buf bytes.Buffer
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&buf, "[section%d]\n", i)
		for j := 0; j < 20; j++ {
			fmt.Fprintf(&buf, "key%d=value%d_%d_with_some_longer_content\n", j, i, j)
		}
	}
	content := buf.String()

	ini := NewINI("")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(content)
		_, _ = ini.LoadReader(reader)
	}
}

func BenchmarkINI_LoadBytes(b *testing.B) {
	content := []byte(`
[database]
host=localhost
port=5432
user=admin
password=secret

[server]
host=0.0.0.0
port=8080
workers=4
`)

	ini := NewINI("")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = ini.LoadBytes(content)
	}
}
