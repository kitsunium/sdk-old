package parser

import (
	"errors"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestXML_Type(t *testing.T) {
	xml := NewXML("test.xml")
	if xml.Type() != "xml" {
		t.Errorf("Type() = %q, want %q", xml.Type(), "xml")
	}
}

func TestXML_NewXML(t *testing.T) {
	// Test without options
	x1 := NewXML("test.xml")
	if x1.Path != "test.xml" {
		t.Errorf("Path = %q, want %q", x1.Path, "test.xml")
	}
	if x1.options.bufferSize != 8192 {
		t.Errorf("bufferSize = %d, want %d", x1.options.bufferSize, 8192)
	}
	if x1.options.usePool != false {
		t.Errorf("usePool = %v, want %v", x1.options.usePool, false)
	}
	
	// Test with options
	x2 := NewXML("test.xml", WithBufferSize(4096), WithPool(false))
	if x2.options.bufferSize != 4096 {
		t.Errorf("bufferSize = %d, want %d", x2.options.bufferSize, 4096)
	}
	if x2.options.usePool != false {
		t.Errorf("usePool = %v, want %v", x2.options.usePool, false)
	}
}

func TestXML_Load_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	xmlPath := filepath.Join(tmpDir, "test.xml")
	
	content := `<?xml version="1.0" encoding="UTF-8"?>
<configuration>
	<database>
		<host>localhost</host>
		<port>5432</port>
		<name>testdb</name>
		<enabled>true</enabled>
	</database>
	<server>
		<host>0.0.0.0</host>
		<port>8080</port>
		<timeout>30.5</timeout>
	</server>
	<paths>
		<data>/var/data</data>
		<logs>/var/logs</logs>
	</paths>
</configuration>`
	
	if err := os.WriteFile(xmlPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	xml := NewXML(xmlPath)
	result, err := xml.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	
	expected := map[string]string{
		"configuration.database.host":   "localhost",
		"configuration.database.port":   "5432",
		"configuration.database.name":   "testdb",
		"configuration.database.enabled": "true",
		"configuration.server.host":     "0.0.0.0",
		"configuration.server.port":     "8080",
		"configuration.server.timeout":  "30.5",
		"configuration.paths.data":      "/var/data",
		"configuration.paths.logs":      "/var/logs",
	}
	
	for key, expectedValue := range expected {
		if value, ok := result[key]; !ok || value != expectedValue {
			t.Errorf("key %q = %q (exists=%v), want %q", key, value, ok, expectedValue)
		}
	}
}

func TestXML_Load_InvalidExtension(t *testing.T) {
	xml := NewXML("test.txt")
	_, err := xml.Load()
	if err == nil {
		t.Error("Load() should error on invalid extension")
	}
	if !errors.Is(err, ErrInvalidExtension) {
		t.Errorf("Expected error about invalid extension, got: %v", err)
	}
}

func TestXML_Load_NonExistentFile(t *testing.T) {
	xml := NewXML("/non/existent/file.xml")
	_, err := xml.Load()
	if err == nil {
		t.Error("Load() should error on non-existent file")
	}
	if !errors.Is(err, ErrFileNotFound) {
		t.Errorf("Expected error about opening file, got: %v", err)
	}
}

func TestXML_LoadReader(t *testing.T) {
	content := `<?xml version="1.0"?>
<root>
	<section1>
		<key1>value1</key1>
		<key2>42</key2>
		<key3>true</key3>
	</section1>
	<section2>
		<nested>
			<inner>value</inner>
		</nested>
		<array>
			<item>item1</item>
			<item>item2</item>
			<item>item3</item>
		</array>
	</section2>
</root>`
	
	xml := NewXML("")
	reader := strings.NewReader(content)
	result, err := xml.LoadReader(reader)
	if err != nil {
		t.Fatalf("LoadReader() error = %v", err)
	}
	
	expected := map[string]string{
		"root.section1.key1":        "value1",
		"root.section1.key2":        "42",
		"root.section1.key3":        "true",
		"root.section2.nested.inner": "value",
		"root.section2.array.item.0": "item1",
		"root.section2.array.item.1": "item2",
		"root.section2.array.item.2": "item3",
	}
	
	for key, expectedValue := range expected {
		if value, ok := result[key]; !ok || value != expectedValue {
			t.Errorf("key %q = %q (exists=%v), want %q", key, value, ok, expectedValue)
		}
	}
}

func TestXML_LoadReader_InvalidXML(t *testing.T) {
	testCases := []struct {
		name    string
		content string
	}{
		{"unclosed tag", "<root><tag>value</root>"},
		{"invalid syntax", "<root><>value</root>"},
		{"invalid characters", "<root><<value>></root>"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			xml := NewXML("")
			reader := strings.NewReader(tc.content)
			
			_, err := xml.LoadReader(reader)
			if err == nil {
				t.Error("LoadReader() should error on invalid XML")
			}
			if !errors.Is(err, ErrXMLParse) {
				t.Errorf("Expected error about parsing XML, got: %v", err)
			}
		})
	}
}

func TestXML_LoadReader_ComplexStructure(t *testing.T) {
	content := `<?xml version="1.0"?>
<root>
	<!-- Comment should be ignored -->
	<element attr1="value1" attr2="value2">text content</element>
	<parent>
		<child index="0">first</child>
		<child index="1">second</child>
		<child index="2">third</child>
	</parent>
	<empty/>
	<whitespace>  trimmed  </whitespace>
	<cdata><![CDATA[Some <special> content & symbols]]></cdata>
	<unicode>Hello 世界 🌍</unicode>
	<nested>
		<level1>
			<level2>
				<level3>
					<deep>value</deep>
				</level3>
			</level2>
		</level1>
	</nested>
	<repeated>
		<item>1</item>
		<item>2</item>
		<item>3</item>
		<other>value</other>
		<item>4</item>
	</repeated>
</root>`
	
	xml := NewXML("")
	reader := strings.NewReader(content)
	result, err := xml.LoadReader(reader)
	if err != nil {
		t.Fatalf("LoadReader() error = %v", err)
	}
	
	// Check various elements
	expected := map[string]string{
		"root.element":       "text content",
		"root.element.attr1": "value1",
		"root.element.attr2": "value2",
		"root.parent.child.0": "first",
		"root.parent.child.0.index": "0",
		"root.parent.child.1": "second",
		"root.parent.child.2": "third",
		"root.whitespace":    "trimmed",
		"root.cdata":         "Some <special> content & symbols",
		"root.unicode":       "Hello 世界 🌍",
		"root.nested.level1.level2.level3.deep": "value",
		"root.repeated.item.0": "1",
		"root.repeated.item.1": "2",
		"root.repeated.item.2": "3",
		"root.repeated.item.3": "4",
		"root.repeated.other": "value",
	}
	
	for key, expectedValue := range expected {
		if value, ok := result[key]; !ok || value != expectedValue {
			t.Errorf("key %q = %q (exists=%v), want %q", key, value, ok, expectedValue)
		}
	}
}

func TestXML_LoadReader_EmptyAndSelfClosing(t *testing.T) {
	content := `<?xml version="1.0"?>
<root>
	<empty></empty>
	<selfclosing/>
	<withattr attr="val"/>
	<nested>
		<empty></empty>
	</nested>
</root>`
	
	xml := NewXML("")
	reader := strings.NewReader(content)
	result, err := xml.LoadReader(reader)
	if err != nil {
		t.Fatalf("LoadReader() error = %v", err)
	}
	
	// Empty and self-closing tags should create empty values
	expected := map[string]string{
		"root.empty": "",
		"root.selfclosing": "",
		"root.withattr": "",
		"root.withattr.attr": "val",
		"root.nested.empty": "",
	}
	
	for key, expectedValue := range expected {
		if value, ok := result[key]; !ok || value != expectedValue {
			t.Errorf("key %q = %q (exists=%v), want %q", key, value, ok, expectedValue)
		}
	}
}

func TestXML_LoadReader_ErrorReading(t *testing.T) {
	xml := NewXML("")
	reader := &xmlErrorReader{err: io.ErrUnexpectedEOF}
	
	_, err := xml.LoadReader(reader)
	if err == nil {
		t.Error("LoadReader() should return error from reader")
	}
}

func TestXML_AllTypes(t *testing.T) {
	// Test ALL possible XML structures
	content := `<?xml version="1.0" encoding="UTF-8"?>
<root xmlns="http://example.com" xmlns:custom="http://custom.com">
	<!-- Various text content -->
	<string>Hello World</string>
	<number>42</number>
	<float>3.14159</float>
	<boolean>true</boolean>
	<empty></empty>
	<selfclosing/>
	
	<!-- Attributes -->
	<element id="123" class="test" data-value="custom">Content</element>
	
	<!-- CDATA -->
	<cdata><![CDATA[
		<html>
			<body>Raw content & symbols</body>
		</html>
	]]></cdata>
	
	<!-- Nested structures -->
	<deeply>
		<nested>
			<structure>
				<with>
					<many>
						<levels>value</levels>
					</many>
				</with>
			</structure>
		</nested>
	</deeply>
	
	<!-- Arrays/Repeated elements -->
	<list>
		<item>First</item>
		<item>Second</item>
		<item>Third</item>
	</list>
	
	<!-- Mixed content -->
	<mixed>
		Text before
		<child>element</child>
		Text after
	</mixed>
	
	<!-- Namespaced elements -->
	<custom:element>Namespaced</custom:element>
	
	<!-- Special characters -->
	<special>&lt;&gt;&amp;&quot;&apos;</special>
	<unicode>你好世界 🌍 مرحبا</unicode>
	
	<!-- Complex repeated structure -->
	<items>
		<item type="A" priority="1">
			<name>Item 1</name>
			<value>100</value>
		</item>
		<item type="B" priority="2">
			<name>Item 2</name>
			<value>200</value>
		</item>
	</items>
</root>`
	
	xml := NewXML("")
	reader := strings.NewReader(content)
	result, err := xml.LoadReader(reader)
	if err != nil {
		t.Fatalf("LoadReader() error = %v", err)
	}
	
	// Verify key types are handled
	checks := map[string]string{
		"root.string": "Hello World",
		"root.number": "42",
		"root.float": "3.14159",
		"root.boolean": "true",
		"root.empty": "",
		"root.selfclosing": "",
		"root.element.0": "Content",
		"root.element.0.id": "123",
		"root.element.0.class": "test",
		"root.element.1": "Namespaced",
		"root.deeply.nested.structure.with.many.levels": "value",
		"root.list.item.0": "First",
		"root.list.item.1": "Second",
		"root.list.item.2": "Third",
		"root.special": "<>&\"'",
		"root.unicode": "你好世界 🌍 مرحبا",
		"root.items.item.0.name": "Item 1",
		"root.items.item.0.value": "100",
		"root.items.item.0.type": "A",
		"root.items.item.1.name": "Item 2",
		"root.items.item.1.value": "200",
	}
	
	for key, expectedValue := range checks {
		if value, ok := result[key]; !ok {
			t.Errorf("Key %q not found in result", key)
		} else if value != expectedValue {
			t.Errorf("Key %q = %q, want %q", key, value, expectedValue)
		}
	}
}

// Helper type for testing reader errors
type xmlErrorReader struct {
	err error
}

func (r *xmlErrorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

func BenchmarkXML_LoadReader_Small(b *testing.B) {
	content := `<root>
		<key1>value1</key1>
		<key2>42</key2>
		<key3>true</key3>
	</root>`
	
	xml := NewXML("")
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(content)
		_, _ = xml.LoadReader(reader)
	}
}

func BenchmarkXML_LoadReader_Large(b *testing.B) {
	var buf bytes.Buffer
	buf.WriteString("<root>")
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&buf, "<section%d>", i)
		for j := 0; j < 20; j++ {
			fmt.Fprintf(&buf, "<key%d>value_%d_%d</key%d>", j, i, j, j)
		}
		fmt.Fprintf(&buf, "</section%d>", i)
	}
	buf.WriteString("</root>")
	content := buf.String()
	
	xml := NewXML("")
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(content)
		_, _ = xml.LoadReader(reader)
	}
}