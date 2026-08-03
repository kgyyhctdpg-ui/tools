package authx

import (
	"path"
	"strings"
)

type permissionCode struct {
	resource string
	action   string
}

func hasPermission(grants []string, required string) bool {
	for _, grant := range grants {
		if matchPermission(grant, required) {
			return true
		}
	}
	return false
}

func matchPermission(pattern, code string) bool {
	pattern = strings.TrimSpace(pattern)
	code = strings.TrimSpace(code)
	if pattern == "" || code == "" {
		return false
	}
	if pattern == "*" || pattern == "*:*" || pattern == code {
		return true
	}
	patternPermission, patternOK := parsePermission(pattern)
	codePermission, codeOK := parsePermission(code)
	if !patternOK || !codeOK {
		return wildcard(pattern, code)
	}
	return wildcard(patternPermission.resource, codePermission.resource) &&
		wildcard(patternPermission.action, codePermission.action)
}

func parsePermission(code string) (permissionCode, bool) {
	code = strings.TrimSpace(code)
	index := strings.LastIndex(code, ":")
	if index <= 0 || index == len(code)-1 {
		return permissionCode{}, false
	}
	return permissionCode{
		resource: strings.TrimSpace(code[:index]),
		action:   strings.TrimSpace(code[index+1:]),
	}, true
}

func wildcard(pattern, value string) bool {
	if pattern == "*" {
		return true
	}
	ok, err := path.Match(pattern, value)
	return err == nil && ok
}
