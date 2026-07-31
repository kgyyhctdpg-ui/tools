package markdown

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

type xmlNode struct {
	Name     string
	Space    string
	Attrs    map[string]string
	NSAttrs  map[string]string
	Text     string
	Children []*xmlNode
}

func parseXML(data []byte) (*xmlNode, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	root := &xmlNode{Name: "#document", Attrs: map[string]string{}, NSAttrs: map[string]string{}}
	stack := []*xmlNode{root}
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
			node := &xmlNode{Name: value.Name.Local, Space: value.Name.Space, Attrs: map[string]string{}, NSAttrs: map[string]string{}}
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

func (node *xmlNode) child(name string) *xmlNode {
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

func (node *xmlNode) children(name string) []*xmlNode {
	if node == nil {
		return nil
	}
	result := make([]*xmlNode, 0)
	for _, child := range node.Children {
		if child.Name == name {
			result = append(result, child)
		}
	}
	return result
}

func (node *xmlNode) first(name string) *xmlNode {
	if node == nil {
		return nil
	}
	if node.Name == name {
		return node
	}
	for _, child := range node.Children {
		if result := child.first(name); result != nil {
			return result
		}
	}
	return nil
}

func (node *xmlNode) descendants(name string) []*xmlNode {
	result := make([]*xmlNode, 0)
	var walk func(*xmlNode)
	walk = func(current *xmlNode) {
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

func (node *xmlNode) attr(name string) string {
	if node == nil {
		return ""
	}
	return node.Attrs[name]
}

func (node *xmlNode) textContent() string {
	if node == nil {
		return ""
	}
	var out strings.Builder
	var walk func(*xmlNode)
	walk = func(current *xmlNode) {
		out.WriteString(current.Text)
		for _, child := range current.Children {
			walk(child)
		}
	}
	walk(node)
	return out.String()
}

func openZip(data []byte) (*zip.Reader, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("markdown: open zip container: %w", err)
	}
	return reader, nil
}

func readZipEntry(file *zip.File, maxSize int64) ([]byte, error) {
	if maxSize > 0 && file.UncompressedSize64 > uint64(maxSize) {
		return nil, fmt.Errorf("%w: %s", ErrArchiveLimit, file.Name)
	}
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := readLimited(reader, maxSize)
	if errors.Is(err, ErrInputTooLarge) {
		return nil, fmt.Errorf("%w: %s", ErrArchiveLimit, file.Name)
	}
	return data, err
}

func officeParts(data []byte, wanted func(string) bool) (map[string][]byte, error) {
	reader, err := openZip(data)
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
		content, err := readZipEntry(file, 64<<20)
		if err != nil {
			return nil, err
		}
		parts[name] = content
	}
	return parts, nil
}

type relationship struct {
	ID       string
	Target   string
	Type     string
	External bool
}

func parseRelationships(data []byte) map[string]relationship {
	result := make(map[string]relationship)
	root, err := parseXML(data)
	if err != nil {
		return result
	}
	for _, node := range root.descendants("Relationship") {
		item := relationship{ID: node.attr("Id"), Target: node.attr("Target"), Type: node.attr("Type"), External: strings.EqualFold(node.attr("TargetMode"), "External")}
		if item.ID != "" {
			result[item.ID] = item
		}
	}
	return result
}

func coreProperties(data []byte) (string, map[string]string) {
	metadata := make(map[string]string)
	root, err := parseXML(data)
	if err != nil {
		return "", metadata
	}
	for xmlName, key := range map[string]string{"title": "title", "creator": "author", "subject": "subject", "description": "description", "keywords": "keywords", "created": "created", "modified": "modified"} {
		if node := root.first(xmlName); node != nil {
			metadata[key] = strings.TrimSpace(node.textContent())
		}
	}
	return metadata["title"], metadata
}

func naturalXMLPartOrder(names []string) {
	sort.Slice(names, func(i, j int) bool {
		left, leftOK := trailingNumber(names[i])
		right, rightOK := trailingNumber(names[j])
		if leftOK && rightOK && left != right {
			return left < right
		}
		return names[i] < names[j]
	})
}

func trailingNumber(name string) (int, bool) {
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
