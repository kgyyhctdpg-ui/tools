package common

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/88250/lute/html"
)

// GetNode returns the first HTML element matching selector.
func GetNode(content string, selector string) (nodes *html.Node, err error) {
	err = nil
	htmlTree, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return nil, err
	}
	var findNodes func(*html.Node) error
	findNodes = func(n *html.Node) error {
		if n.Type == html.ElementNode && n.Data == selector {
			nodes = n
			return nil
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if err = findNodes(c); err == nil {
				return nil
			}
		}
		return errors.New("search nodes error")
	}
	err = findNodes(htmlTree)
	if err != nil {
		return nil, err
	}
	if nodes == nil {
		return nil, errors.New("not found nodes")
	}
	return nodes, err
}

// GetNodeValue returns an element attribute value, or text content when attrKey is empty.
func GetNodeValue(content string, selector string, attrKey string) string {
	node, err := GetNode(content, selector)
	if err != nil {
		return ""
	}
	if attrKey == "" {
		if node.FirstChild == nil {
			return ""
		}
		return strings.TrimSpace(node.FirstChild.Data)
	}
	for _, attr := range node.Attr {
		if attr.Key == attrKey {
			return attr.Val
		}
	}
	return ""
}

// EditAttribute updates or adds one attribute on matching markdown HTML tags.
func EditAttribute(content string, selector string, filter map[string]string, key, val string) (string, error) {
	re := regexp.MustCompile(fmt.Sprintf(`(?i)<(%s)(\s[^>]*)?(/?)>`, regexp.QuoteMeta(selector)))
	result := re.ReplaceAllStringFunc(content, func(match string) string {
		matches := re.FindStringSubmatch(match)
		if len(matches) < 4 {
			return match
		}

		tagName := matches[1]
		attrStr := matches[2]
		closing := matches[3]

		attrRe1 := regexp.MustCompile(`([a-zA-Z0-9_\-:]+)\s*=\s*'([^']*)'`)
		attrRe2 := regexp.MustCompile(`([a-zA-Z0-9_\-:]+)\s*=\s*"([^"]*)"`)

		attrs := make(map[string]string)
		for _, m := range attrRe1.FindAllStringSubmatch(attrStr, -1) {
			if len(m) == 3 {
				attrs[m[1]] = m[2]
			}
		}
		for _, m := range attrRe2.FindAllStringSubmatch(attrStr, -1) {
			if len(m) == 3 {
				attrs[m[1]] = m[2]
			}
		}

		for k, v := range filter {
			if curVal, ok := attrs[k]; !ok || curVal != v {
				return match
			}
		}

		if _, exists := attrs[key]; exists {
			keyRe1 := regexp.MustCompile(fmt.Sprintf(`(?i)(\s+)%s\s*=\s*'[^']*'`, regexp.QuoteMeta(key)))
			keyRe2 := regexp.MustCompile(fmt.Sprintf(`(?i)(\s+)%s\s*=\s*"[^"]*"`, regexp.QuoteMeta(key)))

			if keyRe1.MatchString(attrStr) {
				attrStr = keyRe1.ReplaceAllString(attrStr, fmt.Sprintf(`${1}%s='%s'`, key, val))
			} else if keyRe2.MatchString(attrStr) {
				attrStr = keyRe2.ReplaceAllString(attrStr, fmt.Sprintf(`${1}%s="%s"`, key, val))
			}
		} else {
			attrStr += fmt.Sprintf(` %s="%s"`, key, val)
		}

		return fmt.Sprintf("<%s%s%s>", tagName, attrStr, closing)
	})

	return result, nil
}
