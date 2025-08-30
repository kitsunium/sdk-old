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

func TestINI_WhitespaceFunctions(t *testing.T) {
	i := &INI{}

	t.Run("isTrimWhitespace", func(t *testing.T) {
		tests := []struct {
			ch   byte
			want bool
		}{
			{' ', true},
			{'\t', true},
			{'\r', true},
			{'\n', false},
			{'a', false},
		}

		for _, tt := range tests {
			got := i.isTrimWhitespace(tt.ch)
			if got != tt.want {
				t.Errorf("isTrimWhitespace(%q) = %v, want %v", tt.ch, got, tt.want)
			}
		}
	})

	t.Run("isLineEndWhitespace", func(t *testing.T) {
		tests := []struct {
			ch   byte
			want bool
		}{
			{' ', true},
			{'\t', true},
			{'\r', false},
			{'\n', false},
			{'a', false},
		}

		for _, tt := range tests {
			got := i.isLineEndWhitespace(tt.ch)
			if got != tt.want {
				t.Errorf("isLineEndWhitespace(%q) = %v, want %v", tt.ch, got, tt.want)
			}
		}
	})

	t.Run("trimRight functionality", func(t *testing.T) {
		// trimRight only removes space and tab from the end (not \r)
		tests := []struct {
			input    []byte
			expected []byte
			desc     string
		}{
			{
				input:    []byte("test  \t"),
				expected: []byte("test"),
				desc:     "removes trailing spaces and tabs",
			},
			{
				input:    []byte("test"),
				expected: []byte("test"),
				desc:     "no trailing whitespace",
			},
			{
				input:    []byte("test\r"),
				expected: []byte("test\r"),
				desc:     "preserves \\r",
			},
			{
				input:    []byte("test\r  "),
				expected: []byte("test\r"),
				desc:     "removes spaces after \\r",
			},
		}

		for _, tt := range tests {
			result := i.trimRight(tt.input)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("%s: trimRight(%q) = %q, want %q", tt.desc, tt.input, result, tt.expected)
			}
		}
	})
}

func TestINI_Load_ErrorCases(t *testing.T) {
	t.Run("invalid file path", func(t *testing.T) {
		ini := NewINI("/nonexistent/path/file.txt") // Not .ini extension
		_, err := ini.Load()
		if err == nil {
			t.Error("Load() should return error for invalid extension")
		}
	})

	t.Run("cfg extension", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "test.cfg")

		content := "[section]\nkey=value"
		if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		ini := NewINI(cfgPath)
		result, err := ini.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if result["section.key"] != "value" {
			t.Errorf("Expected section.key=value, got %v", result["section.key"])
		}
	})

	t.Run("conf extension", func(t *testing.T) {
		tmpDir := t.TempDir()
		confPath := filepath.Join(tmpDir, "test.conf")

		content := "[section]\nkey=value"
		if err := os.WriteFile(confPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		ini := NewINI(confPath)
		result, err := ini.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if result["section.key"] != "value" {
			t.Errorf("Expected section.key=value, got %v", result["section.key"])
		}
	})
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

// Tests for missing coverage in INI parser - error paths and edge cases

func TestINI_Load_ReadError(t *testing.T) {
	// Test general read error handling (not file not found)
	tmpDir := t.TempDir()
	iniPath := filepath.Join(tmpDir, "unreadable.ini")

	// Write file first
	if err := os.WriteFile(iniPath, []byte(`[section]\nkey=value`), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Make it unreadable by changing permissions
	if err := os.Chmod(iniPath, 0000); err != nil {
		t.Fatalf("Failed to change file permissions: %v", err)
	}

	// Restore permissions after test
	defer os.Chmod(iniPath, 0644)

	ini := NewINI(iniPath)
	_, err := ini.Load()
	if err == nil {
		t.Error("Load() should error on unreadable file")
	}
	if !errors.Is(err, ErrReadFailed) {
		t.Errorf("Expected ErrReadFailed for read error, got: %v", err)
	}
}

func TestINI_LoadBytes_LastLineWithoutNewline(t *testing.T) {
	// Test processing a file that doesn't end with newline
	content := []byte(`[section1]
key1=value1

[section2]
key2=value2`) // No trailing newline

	ini := NewINI("")
	result, err := ini.LoadBytes(content)
	if err != nil {
		t.Fatalf("LoadBytes() error = %v", err)
	}

	expected := map[string]string{
		"section1.key1": "value1",
		"section2.key2": "value2",
	}

	for key, expectedValue := range expected {
		if value, ok := result[key]; !ok || value != expectedValue {
			t.Errorf("key %q = %q (exists=%v), want %q", key, value, ok, expectedValue)
		}
	}
}

func TestINI_LoadBytes_SingleLineNoNewline(t *testing.T) {
	// Test single line without newline
	content := []byte(`key=value`) // No newline at all

	ini := NewINI("")
	result, err := ini.LoadBytes(content)
	if err != nil {
		t.Fatalf("LoadBytes() error = %v", err)
	}

	if result["key"] != "value" {
		t.Errorf("key = %q, want %q", result["key"], "value")
	}
}

func TestINI_ProcessLine_EdgeCases(t *testing.T) {
	// Test edge cases in line processing
	ini := NewINI("")

	testCases := []struct {
		name            string
		content         string
		expectedResults map[string]string
	}{
		{
			name:    "carriage return handling",
			content: "[section]\r\nkey=value\r\n",
			expectedResults: map[string]string{
				"section.key": "value",
			},
		},
		{
			name:    "mixed line endings",
			content: "[section1]\nkey1=value1\r\n[section2]\r\nkey2=value2\n",
			expectedResults: map[string]string{
				"section1.key1": "value1",
				"section2.key2": "value2",
			},
		},
		{
			name:    "only carriage returns",
			content: "[section]\rkey=value\r",
			expectedResults: map[string]string{
				"section.key": "value",
			},
		},
		{
			name:    "section with spaces inside brackets",
			content: "[ section with spaces ]\nkey=value",
			expectedResults: map[string]string{
				"section.with.spaces.key": "value",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ini.LoadBytes([]byte(tc.content))
			if err != nil {
				t.Fatalf("LoadBytes() error = %v", err)
			}

			for key, expectedValue := range tc.expectedResults {
				if value, ok := result[key]; !ok || value != expectedValue {
					t.Errorf("key %q = %q (exists=%v), want %q", key, value, ok, expectedValue)
				}
			}
		})
	}
}

func TestINI_FindSeparator_EdgeCases(t *testing.T) {
	// Test findSeparator with various scenarios
	ini := &INI{}

	testCases := []struct {
		name     string
		line     string
		expected int
	}{
		{
			name:     "equals first",
			line:     "key=value:extra",
			expected: 3, // Position of first '='
		},
		{
			name:     "colon first",
			line:     "key:value=extra",
			expected: 3, // Position of first ':'
		},
		{
			name:     "no separator",
			line:     "just text",
			expected: -1,
		},
		{
			name:     "separator at start",
			line:     "=value",
			expected: 0,
		},
		{
			name:     "separator at end",
			line:     "key=",
			expected: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ini.findSeparator([]byte(tc.line))
			if result != tc.expected {
				t.Errorf("findSeparator(%q) = %d, want %d", tc.line, result, tc.expected)
			}
		})
	}
}

func TestINI_ProcessValue_QuotingEdgeCases(t *testing.T) {
	// Test processValue with various quoting scenarios
	ini := &INI{}

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single quote incomplete",
			input:    "'incomplete",
			expected: "'incomplete",
		},
		{
			name:     "double quote incomplete",
			input:    "\"incomplete",
			expected: "\"incomplete",
		},
		{
			name:     "mismatched quotes",
			input:    "'double\"",
			expected: "'double\"",
		},
		{
			name:     "empty quotes",
			input:    "\"\"",
			expected: "",
		},
		{
			name:     "single char in quotes",
			input:    "'a'",
			expected: "a",
		},
		{
			name:     "quotes with spaces",
			input:    "  \"  spaced  \"  ",
			expected: "  spaced  ",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ini.processValue([]byte(tc.input))
			resultStr := string(result)
			if resultStr != tc.expected {
				t.Errorf("processValue(%q) = %q, want %q", tc.input, resultStr, tc.expected)
			}
		})
	}
}

func TestINI_TrimBytes_EdgeCases(t *testing.T) {
	// Test trimBytes with various whitespace scenarios
	ini := &INI{}

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "all whitespace",
			input:    "   \t\t\r\r   ",
			expected: "",
		},
		{
			name:     "only tabs",
			input:    "\t\t\t",
			expected: "",
		},
		{
			name:     "only spaces",
			input:    "    ",
			expected: "",
		},
		{
			name:     "only carriage returns",
			input:    "\r\r\r",
			expected: "",
		},
		{
			name:     "mixed whitespace around text",
			input:    " \t\r text \t\r ",
			expected: "text",
		},
		{
			name:     "preserve newlines",
			input:    " \n text \n ",
			expected: "\n text \n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ini.trimBytes([]byte(tc.input))
			resultStr := string(result)
			if resultStr != tc.expected {
				t.Errorf("trimBytes(%q) = %q, want %q", tc.input, resultStr, tc.expected)
			}
		})
	}
}

func TestINI_TrimRight_vs_TrimRightBytes(t *testing.T) {
	// Test that trimRight (line-end whitespace) differs from trimRightBytes (all whitespace)
	ini := &INI{}

	testInput := "text\r  \t" // Text followed by \r, spaces, and tabs

	// trimRight should only remove spaces and tabs at end (not \r)
	trimRightResult := ini.trimRight([]byte(testInput))
	expectedTrimRight := "text\r"
	if string(trimRightResult) != expectedTrimRight {
		t.Errorf("trimRight(%q) = %q, want %q", testInput, string(trimRightResult), expectedTrimRight)
	}

	// trimRightBytes should remove all whitespace including \r
	trimRightBytesResult := ini.trimRightBytes([]byte(testInput))
	expectedTrimRightBytes := "text"
	if string(trimRightBytesResult) != expectedTrimRightBytes {
		t.Errorf("trimRightBytes(%q) = %q, want %q", testInput, string(trimRightBytesResult), expectedTrimRightBytes)
	}
}

func TestINI_WhitespaceValidation(t *testing.T) {
	// Test whitespace detection functions
	ini := &INI{}

	// Test isTrimWhitespace
	trimWhitespaceTests := []struct {
		char     byte
		expected bool
	}{
		{' ', true},
		{'\t', true},
		{'\r', true},
		{'\n', false},
		{'a', false},
		{'0', false},
	}

	for _, test := range trimWhitespaceTests {
		result := ini.isTrimWhitespace(test.char)
		if result != test.expected {
			t.Errorf("isTrimWhitespace(%q) = %v, want %v", test.char, result, test.expected)
		}
	}

	// Test isLineEndWhitespace
	lineEndTests := []struct {
		char     byte
		expected bool
	}{
		{' ', true},
		{'\t', true},
		{'\r', false}, // This is the key difference
		{'\n', false},
		{'a', false},
	}

	for _, test := range lineEndTests {
		result := ini.isLineEndWhitespace(test.char)
		if result != test.expected {
			t.Errorf("isLineEndWhitespace(%q) = %v, want %v", test.char, result, test.expected)
		}
	}
}

func TestINI_BytesToString_Coverage(t *testing.T) {
	// Test iniBytesToString function
	testBytes := []byte("test string")
	result := iniBytesToString(testBytes)
	expected := "test string"

	if result != expected {
		t.Errorf("iniBytesToString() = %q, want %q", result, expected)
	}

	// Test with empty bytes
	emptyResult := iniBytesToString([]byte{})
	if emptyResult != "" {
		t.Errorf("iniBytesToString([]) = %q, want empty string", emptyResult)
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

// Concurrent Tests

func TestINI_LoadReader_Concurrent(t *testing.T) {
	content := `
[database]
host = localhost
port = 5432
name = testdb

[server]
host = 0.0.0.0
port = 8080
debug = true
`

	const numGoroutines = 100
	results := make(chan map[string]string, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			ini := NewINI("")
			reader := strings.NewReader(content)
			result, err := ini.LoadReader(reader)
			if err != nil {
				errors <- err
				return
			}
			results <- result
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		select {
		case result := <-results:
			// Verify result consistency
			if result["database.host"] != "localhost" {
				t.Errorf("Concurrent test %d: database.host = %q, want %q", i, result["database.host"], "localhost")
			}
			if result["server.debug"] != "true" {
				t.Errorf("Concurrent test %d: server.debug = %q, want %q", i, result["server.debug"], "true")
			}
		case err := <-errors:
			t.Errorf("Concurrent test error: %v", err)
		}
	}
}

func TestINI_LoadBytes_Concurrent(t *testing.T) {
	content := []byte(`
[section1]
key1=value1
key2=value2

[section2]
key3=value3
key4=value4
`)

	const numGoroutines = 50
	results := make(chan map[string]string, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			ini := NewINI("")
			result, err := ini.LoadBytes(content)
			if err != nil {
				errors <- err
				return
			}
			results <- result
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		select {
		case result := <-results:
			if result["section1.key1"] != "value1" {
				t.Errorf("Concurrent bytes test %d: section1.key1 = %q, want %q", i, result["section1.key1"], "value1")
			}
			if result["section2.key3"] != "value3" {
				t.Errorf("Concurrent bytes test %d: section2.key3 = %q, want %q", i, result["section2.key3"], "value3")
			}
		case err := <-errors:
			t.Errorf("Concurrent bytes test error: %v", err)
		}
	}
}

func TestINI_Load_Concurrent(t *testing.T) {
	// Create temporary files concurrently
	tmpDir := t.TempDir()

	const numGoroutines = 20
	results := make(chan map[string]string, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			// Create unique file for each goroutine
			iniPath := filepath.Join(tmpDir, fmt.Sprintf("test_%d.ini", id))
			content := fmt.Sprintf(`
[section]
worker_id=%d
host=localhost
port=%d
`, id, 8080+id)

			if err := os.WriteFile(iniPath, []byte(content), 0644); err != nil {
				errors <- err
				return
			}

			ini := NewINI(iniPath)
			result, err := ini.Load()
			if err != nil {
				errors <- err
				return
			}
			results <- result
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		select {
		case result := <-results:
			// Each result should have unique worker_id
			if result["section.host"] != "localhost" {
				t.Errorf("Concurrent load test: section.host = %q, want %q", result["section.host"], "localhost")
			}
		case err := <-errors:
			t.Errorf("Concurrent load test error: %v", err)
		}
	}
}

// Panic Recovery Tests

func TestINI_LoadReader_PanicRecovery(t *testing.T) {
	// Test with various malformed inputs that could cause panics
	malformedContents := []string{
		"",                              // empty
		string([]byte{0, 1, 2, 3, 255}), // binary data
		strings.Repeat("[section]\nkey=value\n", 10000),        // very large content
		"[" + strings.Repeat("section", 1000) + "]\nkey=value", // very long section name
		"[section]\n" + strings.Repeat("key", 1000) + "=value", // very long key
		"[section]\nkey=" + strings.Repeat("value", 1000),      // very long value
		"[section]\nkey=value\x00with\x00nulls",                // null bytes
		"[测试]\n键=值",                                            // unicode
		"[section]\nkey=value\n\x1b[31mANSI codes\x1b[0m",      // ANSI escape codes
		"\xff\xfe[section]\nkey=value",                         // BOM and unusual encoding
	}

	for i, content := range malformedContents {
		t.Run(fmt.Sprintf("malformed_input_%d", i), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("LoadReader panicked with input %d: %v", i, r)
				}
			}()

			ini := NewINI("")
			reader := strings.NewReader(content)
			_, _ = ini.LoadReader(reader)
		})
	}
}

func TestINI_LoadBytes_PanicRecovery(t *testing.T) {
	// Test with potentially panic-inducing byte sequences
	panicInputs := [][]byte{
		nil,                   // nil slice
		{},                    // empty slice
		{0},                   // single null byte
		make([]byte, 1000000), // very large empty content
		bytes.Repeat([]byte("[section]\nkey=value\n"), 50000),         // extremely large valid content
		[]byte("[section\nkey=value"),                                 // malformed section
		[]byte("[section]\nkey"),                                      // incomplete key-value
		[]byte(strings.Repeat("=", 10000)),                            // just equals signs
		[]byte(strings.Repeat("[", 1000) + strings.Repeat("]", 1000)), // unbalanced brackets
	}

	for i, content := range panicInputs {
		t.Run(fmt.Sprintf("panic_input_%d", i), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("LoadBytes panicked with input %d: %v", i, r)
				}
			}()

			ini := NewINI("")
			_, _ = ini.LoadBytes(content)
		})
	}
}

func TestINI_Load_PanicRecovery(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with various file scenarios that could cause panics
	testCases := []struct {
		name    string
		setup   func() string
		cleanup func(string)
	}{
		{
			name: "empty_file",
			setup: func() string {
				path := filepath.Join(tmpDir, "empty.ini")
				os.WriteFile(path, []byte{}, 0644)
				return path
			},
			cleanup: func(path string) { os.Remove(path) },
		},
		{
			name: "binary_file",
			setup: func() string {
				path := filepath.Join(tmpDir, "binary.ini")
				os.WriteFile(path, make([]byte, 1000), 0644)
				return path
			},
			cleanup: func(path string) { os.Remove(path) },
		},
		{
			name: "very_large_file",
			setup: func() string {
				path := filepath.Join(tmpDir, "large.ini")
				content := strings.Repeat("[section]\nkey=value\n", 100000)
				os.WriteFile(path, []byte(content), 0644)
				return path
			},
			cleanup: func(path string) { os.Remove(path) },
		},
		{
			name: "unicode_file",
			setup: func() string {
				path := filepath.Join(tmpDir, "unicode.ini")
				content := "[测试节]\n键=值\n中文=内容"
				os.WriteFile(path, []byte(content), 0644)
				return path
			},
			cleanup: func(path string) { os.Remove(path) },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.setup()
			defer tc.cleanup(path)

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Load panicked with %s: %v", tc.name, r)
				}
			}()

			ini := NewINI(path)
			_, _ = ini.Load()
		})
	}
}

// Multi-threaded Benchmarks

func BenchmarkINI_LoadReader_Concurrent_Small(b *testing.B) {
	content := `
[section1]
key1=value1
key2=value2

[section2]
key3=value3
key4=value4
`

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ini := NewINI("")
			reader := strings.NewReader(content)
			_, _ = ini.LoadReader(reader)
		}
	})
}

func BenchmarkINI_LoadReader_Concurrent_Large(b *testing.B) {
	var buf bytes.Buffer
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&buf, "[section%d]\n", i)
		for j := 0; j < 20; j++ {
			fmt.Fprintf(&buf, "key%d=value_%d_%d\n", j, i, j)
		}
	}
	content := buf.String()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ini := NewINI("")
			reader := strings.NewReader(content)
			_, _ = ini.LoadReader(reader)
		}
	})
}

func BenchmarkINI_LoadBytes_Concurrent(b *testing.B) {
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

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ini := NewINI("")
			_, _ = ini.LoadBytes(content)
		}
	})
}
