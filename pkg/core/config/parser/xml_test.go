package parser

import (
	"bytes"
	"encoding/xml"
	"errors"
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
		"configuration.database.host":    "localhost",
		"configuration.database.port":    "5432",
		"configuration.database.name":    "testdb",
		"configuration.database.enabled": "true",
		"configuration.server.host":      "0.0.0.0",
		"configuration.server.port":      "8080",
		"configuration.server.timeout":   "30.5",
		"configuration.paths.data":       "/var/data",
		"configuration.paths.logs":       "/var/logs",
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
		"root.section1.key1":         "value1",
		"root.section1.key2":         "42",
		"root.section1.key3":         "true",
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
		"root.element":                          "text content",
		"root.element.attr1":                    "value1",
		"root.element.attr2":                    "value2",
		"root.parent.child.0":                   "first",
		"root.parent.child.0.index":             "0",
		"root.parent.child.1":                   "second",
		"root.parent.child.2":                   "third",
		"root.whitespace":                       "trimmed",
		"root.cdata":                            "Some <special> content & symbols",
		"root.unicode":                          "Hello 世界 🌍",
		"root.nested.level1.level2.level3.deep": "value",
		"root.repeated.item.0":                  "1",
		"root.repeated.item.1":                  "2",
		"root.repeated.item.2":                  "3",
		"root.repeated.item.3":                  "4",
		"root.repeated.other":                   "value",
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
		"root.empty":         "",
		"root.selfclosing":   "",
		"root.withattr":      "",
		"root.withattr.attr": "val",
		"root.nested.empty":  "",
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

func TestXML_BuildIndexedKey(t *testing.T) {
	x := &XML{}

	tests := []struct {
		name     string
		nodeName string
		index    int
		prefix   string
		want     string
	}{
		{
			name:     "with prefix",
			nodeName: "item",
			index:    0,
			prefix:   "root",
			want:     "root.item.0",
		},
		{
			name:     "without prefix",
			nodeName: "item",
			index:    1,
			prefix:   "",
			want:     "item.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := x.buildIndexedKey(tt.nodeName, tt.index, tt.prefix)
			if got != tt.want {
				t.Errorf("buildIndexedKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestXML_EstimateSize(t *testing.T) {
	x := &XML{}

	tests := []struct {
		name string
		data []byte
		want int
	}{
		{
			name: "small data returns minimum",
			data: []byte("small"),
			want: 32,
		},
		{
			name: "large data calculates size",
			data: make([]byte, 5000),
			want: 100, // 5000 / 50
		},
		{
			name: "empty data returns minimum",
			data: []byte{},
			want: 32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := x.estimateSize(tt.data)
			if got != tt.want {
				t.Errorf("estimateSize() = %d, want %d", got, tt.want)
			}
		})
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
		"root.string":          "Hello World",
		"root.number":          "42",
		"root.float":           "3.14159",
		"root.boolean":         "true",
		"root.empty":           "",
		"root.selfclosing":     "",
		"root.element.0":       "Content",
		"root.element.0.id":    "123",
		"root.element.0.class": "test",
		"root.element.1":       "Namespaced",
		"root.deeply.nested.structure.with.many.levels": "value",
		"root.list.item.0":        "First",
		"root.list.item.1":        "Second",
		"root.list.item.2":        "Third",
		"root.special":            "<>&\"'",
		"root.unicode":            "你好世界 🌍 مرحبا",
		"root.items.item.0.name":  "Item 1",
		"root.items.item.0.value": "100",
		"root.items.item.0.type":  "A",
		"root.items.item.1.name":  "Item 2",
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

// Tests for missing coverage in XML parser - error paths and edge cases

func TestXML_Load_ReadError(t *testing.T) {
	// Test general read error handling (not file not found)
	tmpDir := t.TempDir()
	xmlPath := filepath.Join(tmpDir, "unreadable.xml")

	// Write file first
	if err := os.WriteFile(xmlPath, []byte(`<root><key>value</key></root>`), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Make it unreadable by changing permissions
	if err := os.Chmod(xmlPath, 0000); err != nil {
		t.Fatalf("Failed to change file permissions: %v", err)
	}

	// Restore permissions after test
	defer os.Chmod(xmlPath, 0644)

	xml := NewXML(xmlPath)
	_, err := xml.Load()
	if err == nil {
		t.Error("Load() should error on unreadable file")
	}
	if !errors.Is(err, ErrReadFailed) {
		t.Errorf("Expected ErrReadFailed for read error, got: %v", err)
	}
}

func TestXML_LoadReader_ReaderError(t *testing.T) {
	// Test LoadReader with reader that fails
	xml := NewXML("")
	reader := &xmlErrorReader{err: io.ErrUnexpectedEOF}

	_, err := xml.LoadReader(reader)
	if err == nil {
		t.Error("LoadReader() should return error from reader")
	}
	if !errors.Is(err, ErrReadFailed) {
		t.Errorf("Expected ErrReadFailed for reader error, got: %v", err)
	}
}

func TestXML_LoadBytes_MalformedXML(t *testing.T) {
	// Test various malformed XML scenarios
	testCases := []struct {
		name    string
		content string
	}{
		{
			name:    "unclosed tag",
			content: "<root><unclosed>value</root>",
		},
		{
			name:    "invalid characters in tag name",
			content: "<root><123invalid>value</123invalid></root>",
		},
		{
			name:    "mismatched tags",
			content: "<root><tag>value</different></root>",
		},
		{
			name:    "unclosed XML declaration",
			content: `<?xml version="1.0"<root>value</root>`,
		},
		{
			name:    "invalid attribute syntax",
			content: `<root invalid-attr=>value</root>`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			xml := NewXML("")
			_, err := xml.LoadBytes([]byte(tc.content))
			if err == nil {
				t.Error("LoadBytes() should error on malformed XML")
			}
			if !errors.Is(err, ErrXMLParse) {
				t.Errorf("Expected ErrXMLParse for malformed XML, got: %v", err)
			}
		})
	}
}

func TestXML_LoadBytes_SmallData(t *testing.T) {
	// Test estimateSize with small data (should return minimum 32)
	xml := NewXML("")

	smallContent := []byte("<root><key>value</key></root>")

	result, err := xml.LoadBytes(smallContent)
	if err != nil {
		t.Fatalf("LoadBytes() error = %v", err)
	}

	if result["root.key"] != "value" {
		t.Errorf("root.key = %q, want %q", result["root.key"], "value")
	}
}

func TestXML_BuildKeys_EdgeCases(t *testing.T) {
	// Test key building edge cases
	xml := &XML{}

	// Test buildSimpleKey with empty prefix
	simpleKey := xml.buildSimpleKey("element", "")
	if simpleKey != "element" {
		t.Errorf("buildSimpleKey('element', '') = %q, want %q", simpleKey, "element")
	}

	// Test buildSimpleKey with prefix
	keyWithPrefix := xml.buildSimpleKey("element", "root")
	if keyWithPrefix != "root.element" {
		t.Errorf("buildSimpleKey('element', 'root') = %q, want %q", keyWithPrefix, "root.element")
	}

	// Test buildIndexedKey with empty prefix
	indexedKey := xml.buildIndexedKey("element", 0, "")
	if indexedKey != "element.0" {
		t.Errorf("buildIndexedKey('element', 0, '') = %q, want %q", indexedKey, "element.0")
	}

	// Test buildIndexedKey with prefix
	indexedKeyWithPrefix := xml.buildIndexedKey("element", 1, "root")
	if indexedKeyWithPrefix != "root.element.1" {
		t.Errorf("buildIndexedKey('element', 1, 'root') = %q, want %q", indexedKeyWithPrefix, "root.element.1")
	}
}

func TestXML_EstimateSize_Coverage(t *testing.T) {
	// Test estimateSize function with various data sizes
	xml := &XML{}

	testCases := []struct {
		name     string
		dataSize int
		expected int
	}{
		{
			name:     "very small data",
			dataSize: 10,
			expected: 32, // minimum size
		},
		{
			name:     "small data",
			dataSize: 100,
			expected: 32, // 100/50 = 2, but minimum is 32
		},
		{
			name:     "medium data",
			dataSize: 5000,
			expected: 100, // 5000/50 = 100
		},
		{
			name:     "large data",
			dataSize: 100000,
			expected: 2000, // 100000/50 = 2000
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := make([]byte, tc.dataSize)
			result := xml.estimateSize(data)
			if result != tc.expected {
				t.Errorf("estimateSize(%d bytes) = %d, want %d", tc.dataSize, result, tc.expected)
			}
		})
	}
}

func TestXML_ProcessNodeMethods_Coverage(t *testing.T) {
	// Test individual node processing methods
	xml := &XML{}
	output := make(map[string]string)

	// Create a test node with attributes
	node := &xmlNode{
		name:  "test",
		value: "test value",
		attrs: []xmlAttr{
			{key: "attr1", value: "value1"},
			{key: "attr2", value: "value2"},
		},
	}

	key := "test.node"

	// Test processNodeAttributes
	xml.processNodeAttributes(node, key, output)
	if output["test.node.attr1"] != "value1" {
		t.Errorf("processNodeAttributes didn't set attr1 correctly")
	}
	if output["test.node.attr2"] != "value2" {
		t.Errorf("processNodeAttributes didn't set attr2 correctly")
	}

	// Test processNodeValue with value
	xml.processNodeValue(node, key, output)
	if output["test.node"] != "test value" {
		t.Errorf("processNodeValue didn't set value correctly")
	}

	// Test processNodeValue with empty element
	emptyNode := &xmlNode{
		name:     "empty",
		value:    "",
		children: nil,
	}
	xml.processNodeValue(emptyNode, "empty.node", output)
	if output["empty.node"] != "" {
		t.Errorf("processNodeValue didn't handle empty element correctly")
	}

	// Test processChildNodes
	childNode := &xmlNode{
		name: "child",
		children: map[string][]*xmlNode{
			"subchild": {{name: "subchild", value: "child value"}},
		},
	}
	xml.processChildNodes(childNode, "parent", output)
	if output["parent.subchild"] != "child value" {
		t.Errorf("processChildNodes didn't process children correctly")
	}
}

func TestXML_HandleCharData_EdgeCases(t *testing.T) {
	// Test handleCharData with various scenarios
	xml := &XML{}

	// Create root node
	root := &xmlNode{children: make(map[string][]*xmlNode)}

	// Create current node
	current := &xmlNode{
		name:   "element",
		value:  "",
		parent: root,
	}

	// Test with whitespace-only content (should be ignored)
	xml.handleCharData([]byte("   \t\n   "), current, root)
	if current.value != "" {
		t.Errorf("handleCharData should ignore whitespace-only content, got %q", current.value)
	}

	// Test with actual content
	xml.handleCharData([]byte("  actual content  "), current, root)
	if current.value != "actual content" {
		t.Errorf("handleCharData should trim and set content, got %q", current.value)
	}

	// Test appending to existing value
	xml.handleCharData([]byte(" more content"), current, root)
	if current.value != "actual content more content" {
		t.Errorf("handleCharData should append to existing value, got %q", current.value)
	}

	// Test with root node (should be ignored)
	xml.handleCharData([]byte("root content"), root, root)
	if root.value != "" {
		t.Errorf("handleCharData should ignore content for root node, got %q", root.value)
	}
}

func TestXML_HandleStartElement_Coverage(t *testing.T) {
	// Test handleStartElement method
	xmlParser := &XML{}

	// Create parent node
	parent := &xmlNode{children: make(map[string][]*xmlNode)}
	stack := []*xmlNode{}

	// Create start element with attributes (using encoding/xml types)
	startElement := xml.StartElement{
		Name: xml.Name{Local: "TestElement"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "TestAttr"}, Value: "TestValue"},
		},
	}

	// Test handleStartElement
	newCurrent := xmlParser.handleStartElement(startElement, parent, &stack)

	// Check that new node was created and added to parent
	if len(parent.children["testelement"]) != 1 {
		t.Errorf("handleStartElement should add child to parent")
	}

	// Check node properties
	childNode := parent.children["testelement"][0]
	if childNode.name != "testelement" {
		t.Errorf("child name = %q, want %q", childNode.name, "testelement")
	}

	if len(childNode.attrs) != 1 {
		t.Errorf("child should have 1 attribute, got %d", len(childNode.attrs))
	}

	if childNode.attrs[0].key != "testattr" || childNode.attrs[0].value != "TestValue" {
		t.Errorf("attribute not set correctly")
	}

	// Check that stack was updated
	if len(stack) != 1 || stack[0] != parent {
		t.Errorf("handleStartElement should update stack")
	}

	// Check return value
	if newCurrent != childNode {
		t.Errorf("handleStartElement should return new current node")
	}
}

func TestXML_HandleEndElement_Coverage(t *testing.T) {
	// Test handleEndElement method
	xmlParser := &XML{}

	// Create nodes
	root := &xmlNode{}
	child := &xmlNode{parent: root}

	// Create stack
	stack := []*xmlNode{root}

	// Test handleEndElement
	newCurrent := xmlParser.handleEndElement(child, &stack)

	// Check that current returned to parent
	if newCurrent != root {
		t.Errorf("handleEndElement should return parent node")
	}

	// Check that stack was popped
	if len(stack) != 0 {
		t.Errorf("handleEndElement should pop stack, got length %d", len(stack))
	}

	// Test with empty stack
	emptyStack := []*xmlNode{}
	newCurrent2 := xmlParser.handleEndElement(child, &emptyStack)
	if newCurrent2 != child {
		t.Errorf("handleEndElement with empty stack should return same node")
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

// Concurrent Tests

func TestXML_LoadReader_Concurrent(t *testing.T) {
	content := `<?xml version="1.0"?>
<root>
	<database>
		<host>localhost</host>
		<port>5432</port>
		<name>testdb</name>
	</database>
	<server>
		<host>0.0.0.0</host>
		<port>8080</port>
		<debug>true</debug>
	</server>
</root>`

	const numGoroutines = 100
	results := make(chan map[string]string, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			xml := NewXML("")
			reader := strings.NewReader(content)
			result, err := xml.LoadReader(reader)
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
			if result["root.database.host"] != "localhost" {
				t.Errorf("Concurrent test %d: root.database.host = %q, want %q", i, result["root.database.host"], "localhost")
			}
			if result["root.server.debug"] != "true" {
				t.Errorf("Concurrent test %d: root.server.debug = %q, want %q", i, result["root.server.debug"], "true")
			}
		case err := <-errors:
			t.Errorf("Concurrent test error: %v", err)
		}
	}
}

func TestXML_LoadBytes_Concurrent(t *testing.T) {
	content := []byte(`<root>
		<items>
			<item>first</item>
			<item>second</item>
		</items>
		<config enabled="true">
			<timeout>30</timeout>
		</config>
	</root>`)

	const numGoroutines = 50
	results := make(chan map[string]string, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			xml := NewXML("")
			result, err := xml.LoadBytes(content)
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
			if result["root.items.item.0"] != "first" {
				t.Errorf("Concurrent bytes test %d: root.items.item.0 = %q, want %q", i, result["root.items.item.0"], "first")
			}
			if result["root.config.enabled"] != "true" {
				t.Errorf("Concurrent bytes test %d: root.config.enabled = %q, want %q", i, result["root.config.enabled"], "true")
			}
		case err := <-errors:
			t.Errorf("Concurrent bytes test error: %v", err)
		}
	}
}

// Panic Recovery Tests

func TestXML_LoadReader_PanicRecovery(t *testing.T) {
	malformedContents := []string{
		"",                              // empty
		"<",                             // incomplete
		">",                             // incomplete
		"<root>",                        // unclosed
		"<root><unclosed></root>",       // mismatched tags
		string([]byte{0, 1, 2, 3, 255}), // binary data
		strings.Repeat("<element>content</element>", 10000),                                          // very large
		"<" + strings.Repeat("element", 1000) + ">content</" + strings.Repeat("element", 1000) + ">", // very long tag
		"<element>" + strings.Repeat("content", 1000) + "</element>",                                 // very long content
		"<element>content\x00with\x00nulls</element>",                                                // null bytes
		"<测试>值</测试>", // unicode
	}

	for i, content := range malformedContents {
		t.Run(fmt.Sprintf("malformed_input_%d", i), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("LoadReader panicked with input %d: %v", i, r)
				}
			}()

			xml := NewXML("")
			reader := strings.NewReader(content)
			_, _ = xml.LoadReader(reader)
		})
	}
}

func TestXML_LoadBytes_PanicRecovery(t *testing.T) {
	panicInputs := [][]byte{
		nil,                   // nil slice
		{},                    // empty slice
		{0},                   // single null byte
		make([]byte, 1000000), // very large empty content
		bytes.Repeat([]byte("<element>content</element>"), 50000), // extremely large
		[]byte("<incomplete"),             // incomplete tag
		[]byte("<>invalid</invalid>"),     // invalid tag name
		[]byte(strings.Repeat("<", 1000)), // many opening brackets
	}

	for i, content := range panicInputs {
		t.Run(fmt.Sprintf("panic_input_%d", i), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("LoadBytes panicked with input %d: %v", i, r)
				}
			}()

			xml := NewXML("")
			_, _ = xml.LoadBytes(content)
		})
	}
}

// Multi-threaded Benchmarks

func BenchmarkXML_LoadReader_Concurrent(b *testing.B) {
	content := `<root>
		<database>
			<host>localhost</host>
			<port>5432</port>
		</database>
		<server>
			<host>0.0.0.0</host>
			<port>8080</port>
		</server>
	</root>`

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			xml := NewXML("")
			reader := strings.NewReader(content)
			_, _ = xml.LoadReader(reader)
		}
	})
}

func BenchmarkXML_LoadBytes_Concurrent(b *testing.B) {
	content := []byte(`<root>
		<key1>value1</key1>
		<key2>42</key2>
		<key3>true</key3>
	</root>`)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			xml := NewXML("")
			_, _ = xml.LoadBytes(content)
		}
	})
}
