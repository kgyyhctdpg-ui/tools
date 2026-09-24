package text

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/scoming-dev/tools/markdown/internal/core"
	"io"
	"regexp"
	"strconv"
	"strings"

	nethtml "golang.org/x/net/html"
)

// NewTextConverter builds the plain-text converter.
func NewTextConverter() core.Converter {
	return core.NewExtensionConverter(
		[]string{".txt", ".md", ".markdown", ".rst", ".log"},
		[]string{"text/plain", "text/markdown"},
		func(_ context.Context, data []byte, _ core.StreamInfo) (*core.Result, error) {
			return &core.Result{Markdown: strings.TrimPrefix(string(data), "\ufeff")}, nil
		},
	)
}

// NewHTMLConverter builds the HTML converter.
func NewHTMLConverter() core.Converter {
	return core.NewExtensionConverter(
		[]string{".html", ".htm", ".xhtml"},
		[]string{"text/html", "application/xhtml+xml"},
		func(ctx context.Context, data []byte, _ core.StreamInfo) (*core.Result, error) {
			source := strings.TrimPrefix(string(data), "\ufeff")
			tableFormat := core.TableFormatFromContext(ctx)
			markdown, err := htmlToMarkdown(source, tableFormat)
			if err != nil {
				return nil, fmt.Errorf("markdown: convert html: %w", err)
			}
			return &core.Result{Title: htmlDocumentTitle(source), Markdown: markdown}, nil
		},
	)
}

// NewCSVConverter builds the CSV/TSV converter.
func NewCSVConverter() core.Converter {
	return core.NewExtensionConverter(
		[]string{".csv", ".tsv"},
		[]string{"text/csv", "text/tab-separated-values"},
		func(ctx context.Context, data []byte, info core.StreamInfo) (*core.Result, error) {
			reader := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))))
			if info.Extension == ".tsv" || info.MIMEType == "text/tab-separated-values" {
				reader.Comma = '\t'
			}
			reader.FieldsPerRecord = -1
			rows, err := reader.ReadAll()
			if err != nil {
				return nil, fmt.Errorf("markdown: parse %s: %w", info.Extension, err)
			}
			tableFormat := core.TableFormatFromContext(ctx)
			if tableFormat == core.TableFormatHTML {
				return &core.Result{Markdown: core.HTMLTable(rows)}, nil
			}
			return &core.Result{Markdown: core.MarkdownTable(rows)}, nil
		},
	)
}

// NewStructuredTextConverter builds the JSON/XML/YAML converter.
func NewStructuredTextConverter() core.Converter {
	return core.NewExtensionConverter(
		[]string{".json", ".xml", ".yaml", ".yml"},
		[]string{"application/json", "application/xml", "text/xml", "application/yaml", "text/yaml"},
		func(_ context.Context, data []byte, info core.StreamInfo) (*core.Result, error) {
			content := strings.TrimPrefix(string(data), "\ufeff")
			language := strings.TrimPrefix(info.Extension, ".")
			switch info.Extension {
			case ".json":
				var formatted bytes.Buffer
				if err := json.Indent(&formatted, data, "", "  "); err != nil {
					return nil, fmt.Errorf("markdown: parse json: %w", err)
				}
				content = formatted.String()
			case ".xml":
				formatted, err := indentXML(data)
				if err != nil {
					return nil, fmt.Errorf("markdown: parse xml: %w", err)
				}
				content = formatted
			case ".yml":
				language = "yaml"
			}
			return &core.Result{Markdown: core.FencedCode(language, content)}, nil
		},
	)
}

func indentXML(data []byte) (string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var out bytes.Buffer
	encoder := xml.NewEncoder(&out)
	encoder.Indent("", "  ")
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if err := encoder.EncodeToken(token); err != nil {
			return "", err
		}
	}
	if err := encoder.Flush(); err != nil {
		return "", err
	}
	return out.String(), nil
}

var htmlTitlePattern = regexp.MustCompile(`(?is)<title(?:\s[^>]*)?>(.*?)</title>`)

func htmlDocumentTitle(source string) string {
	match := htmlTitlePattern.FindStringSubmatch(source)
	if len(match) != 2 {
		return ""
	}
	value := regexp.MustCompile(`<[^>]+>`).ReplaceAllString(match[1], "")
	return strings.TrimSpace(value)
}

func htmlToMarkdown(source string, tableFormat core.TableFormat) (string, error) {
	root, err := nethtml.Parse(strings.NewReader(source))
	if err != nil {
		return "", err
	}
	markdown := renderHTMLNode(root, tableFormat)
	markdown = regexp.MustCompile(`[ \t]+\n`).ReplaceAllString(markdown, "\n")
	markdown = regexp.MustCompile(`\n{3,}`).ReplaceAllString(markdown, "\n\n")
	return strings.TrimSpace(markdown), nil
}

func renderHTMLNode(node *nethtml.Node, tableFormat core.TableFormat) string {
	if node == nil {
		return ""
	}
	if node.Type == nethtml.TextNode {
		return collapseHTMLWhitespace(node.Data)
	}
	if node.Type != nethtml.ElementNode && node.Type != nethtml.DocumentNode {
		return renderHTMLChildren(node, tableFormat)
	}

	tag := strings.ToLower(node.Data)
	switch tag {
	case "head", "script", "style", "noscript", "template", "title":
		return ""
	case "h1", "h2", "h3", "h4", "h5", "h6":
		level, _ := strconv.Atoi(strings.TrimPrefix(tag, "h"))
		return "\n\n" + strings.Repeat("#", level) + " " + strings.TrimSpace(renderHTMLChildren(node, tableFormat)) + "\n\n"
	case "p", "div", "section", "article", "main", "header", "footer", "aside", "figure", "figcaption":
		return "\n\n" + strings.TrimSpace(renderHTMLChildren(node, tableFormat)) + "\n\n"
	case "br":
		return "  \n"
	case "hr":
		return "\n\n---\n\n"
	case "strong", "b":
		return "**" + strings.TrimSpace(renderHTMLChildren(node, tableFormat)) + "**"
	case "em", "i":
		return "*" + strings.TrimSpace(renderHTMLChildren(node, tableFormat)) + "*"
	case "del", "s", "strike":
		return "~~" + strings.TrimSpace(renderHTMLChildren(node, tableFormat)) + "~~"
	case "sup":
		return "<sup>" + strings.TrimSpace(renderHTMLChildren(node, tableFormat)) + "</sup>"
	case "sub":
		return "<sub>" + strings.TrimSpace(renderHTMLChildren(node, tableFormat)) + "</sub>"
	case "code":
		if node.Parent != nil && strings.EqualFold(node.Parent.Data, "pre") {
			return htmlRawText(node)
		}
		content := strings.TrimSpace(renderHTMLChildren(node, tableFormat))
		fence := "`"
		for strings.Contains(content, fence) {
			fence += "`"
		}
		return fence + content + fence
	case "pre":
		return "\n\n" + core.FencedCode("", htmlRawText(node)) + "\n\n"
	case "a":
		content := strings.TrimSpace(renderHTMLChildren(node, tableFormat))
		href := htmlAttribute(node, "href")
		if href == "" || content == "" {
			return content
		}
		return "[" + content + "](" + href + ")"
	case "img":
		source := htmlAttribute(node, "src")
		if source == "" {
			return ""
		}
		return core.MarkdownImage(htmlAttribute(node, "alt"), source)
	case "ul":
		return renderHTMLList(node, false, tableFormat)
	case "ol":
		return renderHTMLList(node, true, tableFormat)
	case "li":
		return strings.TrimSpace(renderHTMLChildren(node, tableFormat))
	case "blockquote":
		content := strings.TrimSpace(renderHTMLChildren(node, tableFormat))
		return "\n\n> " + strings.ReplaceAll(content, "\n", "\n> ") + "\n\n"
	case "table":
		if tableFormat == core.TableFormatHTML || htmlTableHasSpans(node) {
			return "\n\n" + renderOriginalHTMLNode(node) + "\n\n"
		}
		return "\n\n" + renderHTMLTable(node, tableFormat) + "\n\n"
	}
	return renderHTMLChildren(node, tableFormat)
}

func renderHTMLChildren(node *nethtml.Node, tableFormat core.TableFormat) string {
	var out strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		out.WriteString(renderHTMLNode(child, tableFormat))
	}
	return out.String()
}

func renderHTMLList(list *nethtml.Node, ordered bool, tableFormat core.TableFormat) string {
	lines := make([]string, 0)
	index := 1
	for child := list.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != nethtml.ElementNode || !strings.EqualFold(child.Data, "li") {
			continue
		}
		marker := "- "
		if ordered {
			marker = strconv.Itoa(index) + ". "
		}
		content := strings.TrimSpace(renderHTMLChildren(child, tableFormat))
		content = strings.ReplaceAll(content, "\n", "\n  ")
		lines = append(lines, marker+content)
		index++
	}
	return "\n\n" + strings.Join(lines, "\n") + "\n\n"
}

func renderHTMLTable(table *nethtml.Node, tableFormat core.TableFormat) string {
	rows := make([][]string, 0)
	var walk func(*nethtml.Node)
	walk = func(node *nethtml.Node) {
		if node != table && node.Type == nethtml.ElementNode && strings.EqualFold(node.Data, "table") {
			return
		}
		if node.Type == nethtml.ElementNode && strings.EqualFold(node.Data, "tr") {
			row := make([]string, 0)
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				if child.Type == nethtml.ElementNode && (strings.EqualFold(child.Data, "th") || strings.EqualFold(child.Data, "td")) {
					row = append(row, strings.TrimSpace(renderHTMLChildren(child, tableFormat)))
				}
			}
			if len(row) > 0 {
				rows = append(rows, row)
			}
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(table)
	return core.MarkdownTable(rows)
}

func htmlTableHasSpans(table *nethtml.Node) bool {
	var walk func(*nethtml.Node) bool
	walk = func(node *nethtml.Node) bool {
		if node != table && node.Type == nethtml.ElementNode && strings.EqualFold(node.Data, "table") {
			return false
		}
		if node.Type == nethtml.ElementNode && (strings.EqualFold(node.Data, "th") || strings.EqualFold(node.Data, "td")) {
			for _, name := range []string{"colspan", "rowspan"} {
				if value, err := strconv.Atoi(htmlAttribute(node, name)); err == nil && value > 1 {
					return true
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if walk(child) {
				return true
			}
		}
		return false
	}
	return walk(table)
}

func renderOriginalHTMLNode(node *nethtml.Node) string {
	var out bytes.Buffer
	if err := nethtml.Render(&out, node); err != nil {
		return ""
	}
	return strings.TrimSpace(out.String())
}

func htmlAttribute(node *nethtml.Node, name string) string {
	for _, attribute := range node.Attr {
		if strings.EqualFold(attribute.Key, name) {
			return attribute.Val
		}
	}
	return ""
}

func htmlRawText(node *nethtml.Node) string {
	var out strings.Builder
	var walk func(*nethtml.Node)
	walk = func(current *nethtml.Node) {
		if current.Type == nethtml.TextNode {
			out.WriteString(current.Data)
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return strings.TrimSpace(out.String())
}

func collapseHTMLWhitespace(value string) string {
	if strings.TrimSpace(value) == "" {
		if strings.ContainsAny(value, " \t\r\n") {
			return " "
		}
		return ""
	}
	leading := value[0] == ' ' || value[0] == '\t' || value[0] == '\r' || value[0] == '\n'
	trailing := value[len(value)-1] == ' ' || value[len(value)-1] == '\t' || value[len(value)-1] == '\r' || value[len(value)-1] == '\n'
	value = strings.Join(strings.Fields(value), " ")
	if leading {
		value = " " + value
	}
	if trailing {
		value += " "
	}
	return value
}
