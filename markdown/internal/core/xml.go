package core

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strconv"
	"strings"
)

type XMLNode struct {
	Name     string
	Space    string
	Attrs    map[string]string
	NSAttrs  map[string]string
	Text     string
	Children []*XMLNode
}

func ParseXML(data []byte) (*XMLNode, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	root := &XMLNode{Name: "#document", Attrs: map[string]string{}, NSAttrs: map[string]string{}}
	stack := []*XMLNode{root}
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch value := token.(type) {
		case xml.StartElement:
			node := &XMLNode{Name: value.Name.Local, Space: value.Name.Space, Attrs: map[string]string{}, NSAttrs: map[string]string{}}
			for _, attribute := range value.Attr {
				node.Attrs[attribute.Name.Local] = attribute.Value
				node.NSAttrs[attribute.Name.Space+"|"+attribute.Name.Local] = attribute.Value
			}
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, node)
			stack = append(stack, node)
		case xml.EndElement:
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			stack[len(stack)-1].Text += string(value)
		}
	}
	return root, nil
}

func (node *XMLNode) Child(name string) *XMLNode {
	if node == nil {
		return nil
	}
	for _, child := range node.Children {
		if child.Name == name {
			return child
		}
	}
	return nil
}

func (node *XMLNode) ChildrenNamed(name string) []*XMLNode {
	if node == nil {
		return nil
	}
	result := make([]*XMLNode, 0)
	for _, child := range node.Children {
		if child.Name == name {
			result = append(result, child)
		}
	}
	return result
}

func (node *XMLNode) First(name string) *XMLNode {
	if node == nil {
		return nil
	}
	if node.Name == name {
		return node
	}
	for _, child := range node.Children {
		if result := child.First(name); result != nil {
			return result
		}
	}
	return nil
}

func (node *XMLNode) Descendants(name string) []*XMLNode {
	result := make([]*XMLNode, 0)
	var walk func(*XMLNode)
	walk = func(current *XMLNode) {
		if current == nil {
			return
		}
		if current.Name == name {
			result = append(result, current)
		}
		for _, child := range current.Children {
			walk(child)
		}
	}
	walk(node)
	return result
}

func (node *XMLNode) Attr(name string) string {
	if node == nil {
		return ""
	}
	return node.Attrs[name]
}

func (node *XMLNode) TextContent() string {
	if node == nil {
		return ""
	}
	var out strings.Builder
	var walk func(*XMLNode)
	walk = func(current *XMLNode) {
		out.WriteString(current.Text)
		for _, child := range current.Children {
			walk(child)
		}
	}
	walk(node)
	return out.String()
}

func OpenZip(data []byte) (*zip.Reader, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("markdown: open zip container: %w", err)
	}
	return reader, nil
}

func ReadZipEntry(file *zip.File, maxSize int64) ([]byte, error) {
	if maxSize > 0 && file.UncompressedSize64 > uint64(maxSize) {
		return nil, fmt.Errorf("%w: %s", ErrArchiveLimit, file.Name)
	}
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := ReadLimited(reader, maxSize)
	if errors.Is(err, ErrInputTooLarge) {
		return nil, fmt.Errorf("%w: %s", ErrArchiveLimit, file.Name)
	}
	return data, err
}

func OfficeParts(data []byte, wanted func(string) bool) (map[string][]byte, error) {
	reader, err := OpenZip(data)
	if err != nil {
		return nil, err
	}
	if len(reader.File) > 4096 {
		return nil, fmt.Errorf("%w: too many office parts", ErrArchiveLimit)
	}
	parts := make(map[string][]byte)
	for _, file := range reader.File {
		name := path.Clean(strings.TrimPrefix(file.Name, "/"))
		if file.FileInfo().IsDir() || !wanted(name) {
			continue
		}
		content, err := ReadZipEntry(file, 64<<20)
		if err != nil {
			return nil, err
		}
		parts[name] = content
	}
	return parts, nil
}

type Relationship struct {
	ID       string
	Target   string
	Type     string
	External bool
}

func ParseRelationships(data []byte) map[string]Relationship {
	result := make(map[string]Relationship)
	root, err := ParseXML(data)
	if err != nil {
		return result
	}
	for _, node := range root.Descendants("Relationship") {
		item := Relationship{ID: node.Attr("Id"), Target: node.Attr("Target"), Type: node.Attr("Type"), External: strings.EqualFold(node.Attr("TargetMode"), "External")}
		if item.ID != "" {
			result[item.ID] = item
		}
	}
	return result
}

func CoreProperties(data []byte) (string, map[string]string) {
	metadata := make(map[string]string)
	root, err := ParseXML(data)
	if err != nil {
		return "", metadata
	}
	for xmlName, key := range map[string]string{"title": "title", "creator": "author", "subject": "subject", "description": "description", "keywords": "keywords", "created": "created", "modified": "modified"} {
		if node := root.First(xmlName); node != nil {
			metadata[key] = strings.TrimSpace(node.TextContent())
		}
	}
	return metadata["title"], metadata
}

func NaturalXMLPartOrder(names []string) {
	sort.Slice(names, func(i, j int) bool {
		left, leftOK := TrailingNumber(names[i])
		right, rightOK := TrailingNumber(names[j])
		if leftOK && rightOK && left != right {
			return left < right
		}
		return names[i] < names[j]
	})
}

func TrailingNumber(name string) (int, bool) {
	base := strings.TrimSuffix(path.Base(name), path.Ext(name))
	index := len(base)
	for index > 0 && base[index-1] >= '0' && base[index-1] <= '9' {
		index--
	}
	if index == len(base) {
		return 0, false
	}
	value, err := strconv.Atoi(base[index:])
	return value, err == nil
}
