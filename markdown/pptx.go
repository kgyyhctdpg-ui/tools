package markdown

import (
	"context"
	"fmt"
	"html"
	"strconv"
	"strings"
)

func newPPTXConverter() Converter {
	return newExtensionConverter(
		[]string{".pptx", ".pptm"},
		[]string{"application/vnd.openxmlformats-officedocument.presentationml.presentation", "application/vnd.ms-powerpoint.presentation.macroenabled.12"},
		func(ctx context.Context, data []byte, _ StreamInfo) (*Result, error) {
			parts, err := officeParts(data, func(name string) bool {
				return strings.HasPrefix(name, "ppt/slides/slide") && strings.HasSuffix(name, ".xml") || name == "docProps/core.xml"
			})
			if err != nil {
				return nil, err
			}
			names := make([]string, 0)
			for name := range parts {
				if strings.HasPrefix(name, "ppt/slides/slide") {
					names = append(names, name)
				}
			}
			naturalXMLPartOrder(names)
			blocks := make([]string, 0, len(names))
			for index, name := range names {
				root, err := parseXML(parts[name])
				if err != nil {
					return nil, fmt.Errorf("markdown: parse slide %d: %w", index+1, err)
				}
				config, _ := ctx.Value(conversionConfigKey{}).(conversionConfig)
				content := renderPPTXNode(root, config.tableFormat)
				blocks = append(blocks, fmt.Sprintf("## Slide %d\n\n%s", index+1, content))
			}
			title, metadata := coreProperties(parts["docProps/core.xml"])
			return &Result{Title: title, Markdown: joinMarkdownBlocks(blocks), Metadata: metadata}, nil
		},
	)
}

func renderPPTXNode(node *xmlNode, tableFormat TableFormat) string {
	if node == nil {
		return ""
	}
	if node.Name == "tbl" {
		return renderPPTXTable(node, tableFormat)
	}
	if node.Name == "p" {
		return pptxText(node)
	}
	blocks := make([]string, 0)
	for _, child := range node.Children {
		if value := strings.TrimSpace(renderPPTXNode(child, tableFormat)); value != "" {
			blocks = append(blocks, value)
		}
	}
	return joinMarkdownBlocks(blocks)
}

func renderPPTXTable(table *xmlNode, tableFormat TableFormat) string {
	htmlRows := make([]docxTableRow, 0)
	plainRows := make([][]string, 0)
	hasSpans := false
	for _, row := range table.children("tr") {
		htmlRow := docxTableRow{}
		plainRow := make([]string, 0)
		for _, cell := range row.children("tc") {
			if drawingMLOnOff(cell.attr("hMerge")) || drawingMLOnOff(cell.attr("vMerge")) {
				hasSpans = true
				continue
			}
			colSpan := positiveInteger(cell.attr("gridSpan"))
			rowSpan := positiveInteger(cell.attr("rowSpan"))
			if colSpan == 0 {
				colSpan = 1
			}
			if rowSpan == 0 {
				rowSpan = 1
			}
			value := pptxText(cell)
			htmlRow.cells = append(htmlRow.cells, &docxTableCell{
				content: html.EscapeString(value),
				colSpan: colSpan,
				rowSpan: rowSpan,
			})
			plainRow = append(plainRow, value)
			if colSpan > 1 || rowSpan > 1 {
				hasSpans = true
			}
		}
		htmlRows = append(htmlRows, htmlRow)
		plainRows = append(plainRows, plainRow)
	}
	if tableFormat != TableFormatHTML && !hasSpans {
		return markdownTable(plainRows)
	}
	return renderDOCXHTMLTable(htmlRows)
}

func drawingMLOnOff(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "on", "yes":
		return true
	default:
		return false
	}
}

func positiveInteger(value string) int {
	result, err := strconv.Atoi(value)
	if err != nil || result < 1 {
		return 0
	}
	return result
}

func pptxText(node *xmlNode) string {
	texts := make([]string, 0)
	for _, text := range node.descendants("t") {
		texts = append(texts, text.Text)
	}
	return strings.TrimSpace(strings.Join(texts, ""))
}
