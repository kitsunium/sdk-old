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
	// Pre-size based on typical XML structure
	estimatedSize := len(data) / 50
	if estimatedSize < 32 {
		estimatedSize = 32
	}
	config := make(map[string]string, estimatedSize)

	// Use bytes.Reader to avoid string conversion
	decoder := xml.NewDecoder(bytes.NewReader(data))

	stack := make([]*xmlNode, 0, 8) // Most XML has shallow nesting
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
			node := &xmlNode{
				name:     normalize.Key(t.Name.Local),
				parent:   current,
				children: nil, // Allocate lazily only if needed
			}

			for _, attr := range t.Attr {
				attrKey := normalize.Key(attr.Name.Local)
				attrValue := normalize.Value(attr.Value)
				node.attrs = append(node.attrs, xmlAttr{key: attrKey, value: attrValue})
			}

			if current.children == nil {
				current.children = make(map[string][]*xmlNode)
			}
			current.children[node.name] = append(current.children[node.name], node)

			stack = append(stack, current)
			current = node

		case xml.EndElement:
			if len(stack) > 0 {
				current = stack[len(stack)-1]
				stack = stack[:len(stack)-1]
			}

		case xml.CharData:
			text := strings.TrimSpace(string(t))
			if text != "" && current != root {
				// Accumulate text fragments (XML can split text across multiple CharData tokens)
				if current.value != "" {
					current.value += " " + normalize.Value(text)
				} else {
					current.value = normalize.Value(text)
				}
			}
		}
	}

	x.flattenXMLNode(root, "", config)
	return config, nil
}

// flattenXMLNode recursively flattens an XML node tree into a key-value map.
// Handles repeated elements by adding numeric indices.
func (x *XML) flattenXMLNode(node *xmlNode, prefix string, output map[string]string) {
	for name, nodes := range node.children {
		for i, child := range nodes {
			var key string
			if len(nodes) > 1 {
				if prefix == "" {
					key = fmt.Sprintf("%s.%d", name, i)
				} else {
					key = fmt.Sprintf("%s.%s.%d", prefix, name, i)
				}
			} else {
				if prefix == "" {
					key = name
				} else {
					key = fmt.Sprintf("%s.%s", prefix, name)
				}
			}

			for _, attr := range child.attrs {
				attrKey := fmt.Sprintf("%s.%s", key, attr.key)
				output[attrKey] = attr.value
			}

			if child.value != "" {
				output[key] = child.value
			} else if len(child.children) == 0 {
				// Empty element or self-closing tag
				output[key] = ""
			}

			if len(child.children) > 0 {
				x.flattenXMLNode(child, key, output)
			}
		}
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
