// Package permissionx provides small helpers for resource:action permission
// codes.
package permissionx

import (
	"errors"
	"path"
	"strings"
)

var ErrInvalidPermission = errors.New("permissionx: invalid permission")

// Permission is a resource/action permission pair.
type Permission struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// Code joins resource and action as "resource:action".
func Code(resource, action string) string {
	resource = strings.TrimSpace(resource)
	action = strings.TrimSpace(action)
	if resource == "" || action == "" {
		return ""
	}
	return resource + ":" + action
}

// String returns the "resource:action" permission code.
func (permission Permission) String() string {
	return Code(permission.Resource, permission.Action)
}

// Parse splits a permission code into resource and action.
func Parse(code string) (Permission, error) {
	code = Normalize(code)
	index := strings.LastIndex(code, ":")
	if index <= 0 || index == len(code)-1 {
		return Permission{}, ErrInvalidPermission
	}
	return Permission{
		Resource: strings.TrimSpace(code[:index]),
		Action:   strings.TrimSpace(code[index+1:]),
	}, nil
}

// Normalize trims a permission code.
func Normalize(code string) string {
	return strings.TrimSpace(code)
}

// Match reports whether pattern grants code. "*" and "*:*" grant everything.
// The resource and action parts support path.Match wildcards.
func Match(pattern, code string) bool {
	pattern = Normalize(pattern)
	code = Normalize(code)
	if pattern == "" || code == "" {
		return false
	}
	if pattern == "*" || pattern == "*:*" || pattern == code {
		return true
	}
	patternPermission, patternErr := Parse(pattern)
	codePermission, codeErr := Parse(code)
	if patternErr != nil || codeErr != nil {
		return wildcard(pattern, code)
	}
	return wildcard(patternPermission.Resource, codePermission.Resource) &&
		wildcard(patternPermission.Action, codePermission.Action)
}

// Has reports whether grants contain required.
func Has(grants []string, required string) bool {
	for _, grant := range grants {
		if Match(grant, required) {
			return true
		}
	}
	return false
}

// HasAny reports whether grants contain at least one required permission.
func HasAny(grants []string, required ...string) bool {
	if len(required) == 0 {
		return true
	}
	for _, item := range required {
		if Has(grants, item) {
			return true
		}
	}
	return false
}

// HasAll reports whether grants contain all required permissions.
func HasAll(grants []string, required ...string) bool {
	for _, item := range required {
		if !Has(grants, item) {
			return false
		}
	}
	return true
}

// Unique normalizes permissions, removes empty values, and preserves order.
func Unique(codes []string) []string {
	seen := make(map[string]struct{}, len(codes))
	result := make([]string, 0, len(codes))
	for _, code := range codes {
		code = Normalize(code)
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		result = append(result, code)
	}
	return result
}

func wildcard(pattern, value string) bool {
	if pattern == "*" {
		return true
	}
	ok, err := path.Match(pattern, value)
	return err == nil && ok
}
