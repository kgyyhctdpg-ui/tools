// Package jsonx provides convenient JSON helpers.
package jsonx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// MarshalString marshals value to a JSON string.
func MarshalString(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// MustMarshalString marshals value to a JSON string and returns empty string on
// error. It is useful for compatibility with older helper APIs.
func MustMarshalString(value any) string {
	text, err := MarshalString(value)
	if err != nil {
		return ""
	}
	return text
}

// UnmarshalString unmarshals a JSON string into out.
func UnmarshalString(text string, out any) error {
	return json.Unmarshal([]byte(text), out)
}

// Compact removes insignificant whitespace from a JSON string. Invalid JSON is
// returned unchanged.
func Compact(text string) string {
	var dst bytes.Buffer
	if err := json.Compact(&dst, []byte(text)); err != nil {
		return text
	}
	return dst.String()
}

// Indent formats JSON with prefix and indent. Invalid JSON returns an error.
func Indent(text, prefix, indent string) (string, error) {
	var dst bytes.Buffer
	if err := json.Indent(&dst, []byte(text), prefix, indent); err != nil {
		return "", err
	}
	return dst.String(), nil
}

// Pretty formats JSON with two-space indentation.
func Pretty(text string) (string, error) {
	return Indent(text, "", "  ")
}

// Valid reports whether text is valid JSON.
func Valid(text string) bool {
	return json.Valid([]byte(text))
}

// ToMap converts a JSON object or struct-like value into map[string]any.
func ToMap(value any) (map[string]any, error) {
	if value == nil {
		return nil, nil
	}
	if m, ok := value.(map[string]any); ok {
		return m, nil
	}
	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		var err error
		data, err = json.Marshal(value)
		if err != nil {
			return nil, err
		}
	}
	result := map[string]any{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Clone deep-copies JSON-marshalable data into out.
func Clone(in any, out any) error {
	data, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

// Get reads a value from JSON text using a dot path such as "user.name" or
// "items.0.id".
func Get(text, path string) (any, bool, error) {
	var value any
	if err := json.Unmarshal([]byte(text), &value); err != nil {
		return nil, false, err
	}
	got, ok := GetValue(value, path)
	return got, ok, nil
}

// GetString reads a string from JSON text using a dot path.
func GetString(text, path string) (string, bool, error) {
	value, ok, err := Get(text, path)
	if err != nil || !ok {
		return "", ok, err
	}
	if value == nil {
		return "", true, nil
	}
	return fmt.Sprint(value), true, nil
}

// GetValue reads a value from nested maps/slices using a dot path.
func GetValue(value any, path string) (any, bool) {
	if path == "" {
		return value, true
	}
	current := value
	for _, part := range strings.Split(path, ".") {
		if part == "" {
			return nil, false
		}
		switch typed := current.(type) {
		case map[string]any:
			next, ok := typed[part]
			if !ok {
				return nil, false
			}
			current = next
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(typed) {
				return nil, false
			}
			current = typed[index]
		default:
			return nil, false
		}
	}
	return current, true
}

// Set sets a value in a JSON object using a dot path and returns the updated
// compact JSON.
func Set(text, path string, value any) (string, error) {
	root, err := ToMap(text)
	if err != nil {
		return "", err
	}
	if err := SetValue(root, path, value); err != nil {
		return "", err
	}
	return MarshalString(root)
}

// SetValue sets a value in a nested map using a dot path. Missing maps are
// created automatically. Array mutation is intentionally not supported.
func SetValue(root map[string]any, path string, value any) error {
	if root == nil {
		return fmt.Errorf("jsonx: root map is nil")
	}
	parts := strings.Split(path, ".")
	if path == "" || len(parts) == 0 {
		return fmt.Errorf("jsonx: path is empty")
	}
	current := root
	for i, part := range parts {
		if part == "" {
			return fmt.Errorf("jsonx: path contains empty part")
		}
		if i == len(parts)-1 {
			current[part] = value
			return nil
		}
		next, ok := current[part].(map[string]any)
		if !ok {
			next = map[string]any{}
			current[part] = next
		}
		current = next
	}
	return nil
}

// Delete deletes a value from a JSON object using a dot path and returns the
// updated compact JSON.
func Delete(text, path string) (string, error) {
	root, err := ToMap(text)
	if err != nil {
		return "", err
	}
	DeleteValue(root, path)
	return MarshalString(root)
}

// DeleteValue deletes a value from a nested map using a dot path.
func DeleteValue(root map[string]any, path string) bool {
	if root == nil || path == "" {
		return false
	}
	parts := strings.Split(path, ".")
	current := root
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part].(map[string]any)
		if !ok {
			return false
		}
		current = next
	}
	last := parts[len(parts)-1]
	if _, ok := current[last]; !ok {
		return false
	}
	delete(current, last)
	return true
}
