package common

import (
	"regexp"
	"strings"
)

// FormatContent applies shared Markdown cleanup rules before rendering.
func FormatContent(content string) string {
	content = FormatTag(content)
	content = RemoveSpacesBeforeMathBlockAndLineBreak(content)
	content = RemoveEmptyLines(content)
	content = RemoveFirstLineDash(content)
	return content
}

// FormatTag normalizes custom s-tag blocks so Markdown parses them as HTML blocks.
func FormatTag(content string) string {
	content = strings.ReplaceAll(content, "<s-tag", "\n\n<s-tag")
	content = strings.ReplaceAll(content, "</s-tag>", "</s-tag>\n\n")
	content = strings.ReplaceAll(content, "$\\", "$")
	return content
}

// RemoveEmptyLines collapses repeated blank lines to a single blank line.
func RemoveEmptyLines(content string) string {
	re := regexp.MustCompile(`\n{2,}`)
	return re.ReplaceAllString(content, "\n\n")
}

// RemoveFirstLineDash removes a leading Markdown horizontal rule.
func RemoveFirstLineDash(content string) string {
	re := regexp.MustCompile(`^(\s*\n)*[-]{3,}\s*\n`)
	return re.ReplaceAllString(content, "")
}
