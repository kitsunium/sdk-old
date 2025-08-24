// Package parser provides configuration parsing for XML files.
package parser

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/kitsunium/sdk/pkg/core/config/normalize"
)

// XML is a parser for XML configuration files.
// It flattens nested XML structures into a dot-separated key-value map.
//
// The parser handles:
//   - Elements and nested elements
//   - Attributes (stored as element.attribute)
//   - Text content
//   - CDATA sections
//   - Empty and self-closing tags
//   - Repeated elements (indexed with .0, .1, etc.)
type XML struct {
	Path    string
	options baseParser
}

// NewXML creates a new XML parser instance.
//
// Parameters:
//   - path: Path to the XML file to parse
//   - opts: Optional parser configuration options
//
// Example:
//
//	parser := NewXML("config.xml")
//	config, err := parser.Load()
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewXML(path string, opts ...ParserOption) *XML {
	x := &XML{
		Path: path,
		options: baseParser{
			bufferSize: 8192,
			usePool:    false,
		},
	}

	for _, opt := range opts {
		opt(&x.options)
	}

	return x
}

// Type returns the parser type identifier "xml".
func (x *XML) Type() string {
	return "xml"
}

// Load reads and parses an XML file from disk.
// Validates that the file has a .xml extension.
//
// Returns a flattened map where:
//   - Nested elements become dot-separated keys
//   - Attributes are suffixed to element keys
//   - Repeated elements are indexed
//
// Example XML:
//
//	<config>
//	  <database host="localhost" port="5432"/>
//	</config>
//
// Becomes:
//
//	{"config.database.host": "localhost", "config.database.port": "5432"}
//
// Returns an error if:
//   - The file extension is not .xml
//   - The file cannot be read
//   - The XML is malformed
func (x *XML) Load() (map[string]string, error) {
	if ext := path.Ext(x.Path); ext != ".xml" {
		return nil, ErrInvalidExtension.Newf("expected .xml, got %s", ext)
	}

	file, err := os.Open(x.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound.Wrap(err).WithTag("path", x.Path)
		}
		return nil, ErrReadFailed.Wrap(err).WithTag("path", x.Path)
	}
	defer file.Close()

	return x.LoadReader(file)
}

// LoadReader parses XML from an io.Reader.
// This method reads all data into memory before parsing.
func (x *XML) LoadReader(r io.Reader) (map[string]string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, ErrReadFailed.Wrap(err).WithTag("parser", "xml")
	}

	return x.LoadBytes(data)
}

// LoadBytes parses XML from a byte slice.
// Uses a stack-based approach to handle nested elements.
func (x *XML) LoadBytes(data []byte) (map[string]string, error) {
	config := make(map[string]string, x.estimateSize(data))
	root, err := x.parseXML(data)
	if err != nil {
		return nil, err
	}
	x.flattenXMLNode(root, "", config)
	return config, nil
}

// estimateSize estimates the initial map size based on data length.
func (x *XML) estimateSize(data []byte) int {
	estimatedSize := len(data) / 50
	if estimatedSize < 32 {
		return 32
	}
	return estimatedSize
}

// parseXML parses XML data into a tree structure.
func (x *XML) parseXML(data []byte) (*xmlNode, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	stack := make([]*xmlNode, 0, 8)
	root := &xmlNode{children: make(map[string][]*xmlNode, 4)}
	current := root

	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, ErrXMLParse.Wrap(err).WithDetail("size", len(data))
		}

		switch t := token.(type) {
		case xml.StartElement:
			current = x.handleStartElement(t, current, &stack)
		case xml.EndElement:
			current = x.handleEndElement(current, &stack)
		case xml.CharData:
			x.handleCharData(t, current, root)
		}
	}
	return root, nil
}

// handleStartElement processes an XML start element.
func (x *XML) handleStartElement(t xml.StartElement, current *xmlNode, stack *[]*xmlNode) *xmlNode {
	node := &xmlNode{
		name:     normalize.Key(t.Name.Local),
		parent:   current,
		children: nil,
	}

	for _, attr := range t.Attr {
		node.attrs = append(node.attrs, xmlAttr{
			key:   normalize.Key(attr.Name.Local),
			value: normalize.Value(attr.Value),
		})
	}

	if current.children == nil {
		current.children = make(map[string][]*xmlNode)
	}
	current.children[node.name] = append(current.children[node.name], node)

	*stack = append(*stack, current)
	return node
}

// handleEndElement processes an XML end element.
func (x *XML) handleEndElement(current *xmlNode, stack *[]*xmlNode) *xmlNode {
	if len(*stack) > 0 {
		current = (*stack)[len(*stack)-1]
		*stack = (*stack)[:len(*stack)-1]
	}
	return current
}

// handleCharData processes XML character data.
func (x *XML) handleCharData(t xml.CharData, current, root *xmlNode) {
	text := strings.TrimSpace(string(t))
	if text != "" && current != root {
		if current.value != "" {
			current.value += " " + normalize.Value(text)
		} else {
			current.value = normalize.Value(text)
		}
	}
}

// flattenXMLNode recursively flattens an XML node tree into a key-value map.
// Handles repeated elements by adding numeric indices.
func (x *XML) flattenXMLNode(node *xmlNode, prefix string, output map[string]string) {
	for name, nodes := range node.children {
		for i, child := range nodes {
			key := x.buildNodeKey(name, i, len(nodes), prefix)
			x.processNodeAttributes(child, key, output)
			x.processNodeValue(child, key, output)
			x.processChildNodes(child, key, output)
		}
	}
}

func (x *XML) buildNodeKey(name string, index, nodeCount int, prefix string) string {
	if nodeCount > 1 {
		return x.buildIndexedKey(name, index, prefix)
	}
	return x.buildSimpleKey(name, prefix)
}

func (x *XML) buildIndexedKey(name string, index int, prefix string) string {
	if prefix == "" {
		return fmt.Sprintf("%s.%d", name, index)
	}
	return fmt.Sprintf("%s.%s.%d", prefix, name, index)
}

func (x *XML) buildSimpleKey(name, prefix string) string {
	if prefix == "" {
		return name
	}
	return fmt.Sprintf("%s.%s", prefix, name)
}

func (x *XML) processNodeAttributes(child *xmlNode, key string, output map[string]string) {
	for _, attr := range child.attrs {
		attrKey := fmt.Sprintf("%s.%s", key, attr.key)
		output[attrKey] = attr.value
	}
}

func (x *XML) processNodeValue(child *xmlNode, key string, output map[string]string) {
	if child.value != "" {
		output[key] = child.value
	} else if len(child.children) == 0 {
		// Empty element or self-closing tag
		output[key] = ""
	}
}

func (x *XML) processChildNodes(child *xmlNode, key string, output map[string]string) {
	if len(child.children) > 0 {
		x.flattenXMLNode(child, key, output)
	}
}

// xmlNode represents a node in the XML tree structure.
type xmlNode struct {
	name     string                // Element name (normalized)
	value    string                // Text content
	attrs    []xmlAttr             // Attributes
	parent   *xmlNode              // Parent node reference
	children map[string][]*xmlNode // Child nodes grouped by name
}

// xmlAttr represents an XML attribute with normalized key.
type xmlAttr struct {
	key   string // Attribute name (normalized)
	value string // Attribute value
}
