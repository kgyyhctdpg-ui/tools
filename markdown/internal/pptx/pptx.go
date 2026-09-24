package pptx

import (
	"context"
	"fmt"
	"github.com/scoming-dev/tools/markdown/internal/core"
	"github.com/scoming-dev/tools/markdown/internal/ooxml"
	"html"
	"strconv"
	"strings"
)

// NewConverter builds the PPTX converter.
func NewConverter() core.Converter {
	return core.NewExtensionConverter(
		[]string{".pptx", ".pptm"},
		[]string{"application/vnd.openxmlformats-officedocument.presentationml.presentation", "application/vnd.ms-powerpoint.presentation.macroenabled.12"},
		func(ctx context.Context, data []byte, _ core.StreamInfo) (*core.Result, error) {
			parts, err := core.OfficeParts(data, func(name string) bool {
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
			core.NaturalXMLPartOrder(names)
			blocks := make([]string, 0, len(names))
			for index, name := range names {
				root, err := core.ParseXML(parts[name])
				if err != nil {
					return nil, fmt.Errorf("markdown: parse slide %d: %w", index+1, err)
				}
				tableFormat := core.TableFormatFromContext(ctx)
				content := renderPPTXNode(root, tableFormat)
				blocks = append(blocks, fmt.Sprintf("## Slide %d\n\n%s", index+1, content))
			}
			title, metadata := core.CoreProperties(parts["docProps/core.xml"])
			return &core.Result{Title: title, Markdown: core.JoinBlocks(blocks), Metadata: metadata}, nil
		},
	)
}

func renderPPTXNode(node *core.XMLNode, tableFormat core.TableFormat) string {
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
	return core.JoinBlocks(blocks)
}

func renderPPTXTable(table *core.XMLNode, tableFormat core.TableFormat) string {
	htmlRows := make([]ooxml.TableRow, 0)
	plainRows := make([][]string, 0)
	hasSpans := false
	for _, row := range table.ChildrenNamed("tr") {
		htmlRow := ooxml.TableRow{}
		plainRow := make([]string, 0)
		for _, cell := range row.ChildrenNamed("tc") {
			if drawingMLOnOff(cell.Attr("hMerge")) || drawingMLOnOff(cell.Attr("vMerge")) {
				hasSpans = true
				continue
			}
			colSpan := positiveInteger(cell.Attr("gridSpan"))
			rowSpan := positiveInteger(cell.Attr("rowSpan"))
			if colSpan == 0 {
				colSpan = 1
			}
			if rowSpan == 0 {
				rowSpan = 1
			}
			value := pptxText(cell)
			htmlRow.Cells = append(htmlRow.Cells, ooxml.TableCell{
				Content: html.EscapeString(value),
				ColSpan: colSpan,
				RowSpan: rowSpan,
			})
			plainRow = append(plainRow, value)
			if colSpan > 1 || rowSpan > 1 {
				hasSpans = true
			}
		}
		htmlRows = append(htmlRows, htmlRow)
		plainRows = append(plainRows, plainRow)
	}
	if tableFormat != core.TableFormatHTML && !hasSpans {
		return core.MarkdownTable(plainRows)
	}
	return ooxml.RenderHTMLTable(htmlRows)
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

func pptxText(node *core.XMLNode) string {
	texts := make([]string, 0)
	for _, text := range node.Descendants("t") {
		texts = append(texts, text.Text)
	}
	return strings.TrimSpace(strings.Join(texts, ""))
}
