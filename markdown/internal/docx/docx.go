package docx

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/scoming-dev/tools/markdown/internal/core"
	"github.com/scoming-dev/tools/markdown/internal/ooxml"
	"html"
	"net/url"
	"path"
	"strconv"
	"strings"
)

type docxConverter struct {
	settings *core.Settings
}

// NewConverter builds the DOCX converter.
func NewConverter(settings *core.Settings) core.Converter {
	converter := &docxConverter{settings: settings}
	return core.NewExtensionConverter(
		[]string{".docx"},
		[]string{"application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		converter.convert,
	)
}

func (converter *docxConverter) convert(ctx context.Context, data []byte, _ core.StreamInfo) (*core.Result, error) {
	parts, err := core.OfficeParts(data, func(name string) bool {
		return name == "word/document.xml" || name == "word/numbering.xml" ||
			name == "word/styles.xml" || name == "word/footnotes.xml" ||
			name == "word/endnotes.xml" || name == "docProps/core.xml" ||
			strings.HasPrefix(name, "word/_rels/") || strings.HasPrefix(name, "word/media/")
	})
	if err != nil {
		return nil, err
	}
	documentXML, ok := parts["word/document.xml"]
	if !ok {
		return nil, fmt.Errorf("markdown: docx has no word/document.xml")
	}
	root, err := core.ParseXML(documentXML)
	if err != nil {
		return nil, fmt.Errorf("markdown: parse docx document: %w", err)
	}

	renderer := &docxRenderer{
		ctx:       ctx,
		settings:  converter.settings,
		parts:     parts,
		relations: make(map[string]map[string]core.Relationship),
		numbering: parseDOCXNumbering(parts["word/numbering.xml"]),
	}
	body := root.First("body")
	if body == nil {
		return nil, fmt.Errorf("markdown: docx document has no body")
	}
	blocks := renderer.renderBlocks(body, "word/document.xml", false)
	blocks = append(blocks, renderer.renderNotes("word/footnotes.xml", "footnote")...)
	blocks = append(blocks, renderer.renderNotes("word/endnotes.xml", "endnote")...)
	if renderer.err != nil {
		return nil, renderer.err
	}

	title, metadata := core.CoreProperties(parts["docProps/core.xml"])
	return &core.Result{Title: title, Markdown: core.JoinBlocks(blocks), Metadata: metadata}, nil
}

type docxRenderer struct {
	ctx       context.Context
	settings  *core.Settings
	parts     map[string][]byte
	relations map[string]map[string]core.Relationship
	numbering map[string]map[int]string
	err       error
}

func (renderer *docxRenderer) renderBlocks(parent *core.XMLNode, partName string, inTable bool) []string {
	if parent == nil || renderer.err != nil {
		return nil
	}
	blocks := make([]string, 0)
	for _, child := range parent.Children {
		if renderer.skipNode(child) {
			continue
		}
		switch child.Name {
		case "p":
			if paragraph := renderer.renderParagraph(child, partName, inTable); paragraph != "" {
				blocks = append(blocks, paragraph)
			}
		case "tbl":
			if table := renderer.renderTable(child, partName); table != "" {
				blocks = append(blocks, table)
			}
		case "AlternateContent":
			if selected := selectAlternateContent(child); selected != nil {
				blocks = append(blocks, renderer.renderBlocks(selected, partName, inTable)...)
			}
		case "sdt", "sdtContent", "customXml", "ins", "moveTo", "smartTag":
			blocks = append(blocks, renderer.renderBlocks(child, partName, inTable)...)
		}
	}
	return blocks
}

func (renderer *docxRenderer) renderParagraph(paragraph *core.XMLNode, partName string, inTable bool) string {
	var out strings.Builder
	for _, child := range paragraph.Children {
		if child.Name == "pPr" || renderer.skipNode(child) {
			continue
		}
		out.WriteString(renderer.renderInline(child, partName, inTable))
	}
	content := strings.TrimSpace(out.String())
	if content == "" || inTable || strings.HasPrefix(content, "$$") {
		return content
	}

	properties := paragraph.Child("pPr")
	if level := docxHeadingLevel(properties); level > 0 {
		return strings.Repeat("#", level) + " " + content
	}
	if marker, indent := renderer.listMarker(properties); marker != "" {
		return strings.Repeat("  ", indent) + marker + " " + content
	}
	style := ""
	if properties != nil && properties.Child("pStyle") != nil {
		style = strings.ToLower(properties.Child("pStyle").Attr("val"))
	}
	if strings.Contains(style, "quote") {
		return "> " + strings.ReplaceAll(content, "\n", "\n> ")
	}
	return content
}

func (renderer *docxRenderer) renderInline(node *core.XMLNode, partName string, inTable bool) string {
	if node == nil || renderer.err != nil || renderer.skipNode(node) {
		return ""
	}
	switch node.Name {
	case "r":
		return renderer.renderRun(node, partName, inTable)
	case "hyperlink":
		return renderer.renderHyperlink(node, partName, inTable)
	case "oMath":
		latex := strings.TrimSpace(ommlToLatex(node))
		if latex != "" {
			return "$" + latex + "$"
		}
		return ""
	case "oMathPara":
		latex := strings.TrimSpace(ommlToLatex(node))
		if latex != "" {
			return "$$\n" + latex + "\n$$"
		}
		return ""
	case "drawing", "pict", "object":
		return renderer.renderImages(node, partName, inTable)
	case "AlternateContent":
		return renderer.renderInline(selectAlternateContent(node), partName, inTable)
	case "t", "delText":
		if inTable {
			return core.HTMLEscapeText(node.Text)
		}
		return escapeDOCXMarkdownText(node.Text)
	case "tab":
		return "\t"
	case "br", "cr":
		if inTable {
			return "<br>"
		}
		return "  \n"
	case "noBreakHyphen":
		return "-"
	case "softHyphen":
		return "\u00ad"
	case "sym":
		return renderDOCXSymbol(node)
	case "footnoteReference":
		return "[^" + node.Attr("id") + "]"
	case "endnoteReference":
		return "[^endnote-" + node.Attr("id") + "]"
	case "sdt", "sdtContent", "customXml", "ins", "moveTo", "smartTag", "fldSimple":
		return renderer.renderInlineChildren(node, partName, inTable)
	case "instrText", "pPr", "rPr", "bookmarkStart", "bookmarkEnd", "proofErr", "lastRenderedPageBreak":
		return ""
	default:
		return renderer.renderInlineChildren(node, partName, inTable)
	}
}

func (renderer *docxRenderer) renderInlineChildren(node *core.XMLNode, partName string, inTable bool) string {
	if node == nil {
		return ""
	}
	var out strings.Builder
	for _, child := range node.Children {
		out.WriteString(renderer.renderInline(child, partName, inTable))
	}
	return out.String()
}

func (renderer *docxRenderer) renderRun(run *core.XMLNode, partName string, inTable bool) string {
	properties := run.Child("rPr")
	hasNativeMath := len(run.Descendants("oMath")) > 0 || len(run.Descendants("oMathPara")) > 0
	var out strings.Builder
	for _, child := range run.Children {
		if child.Name == "rPr" {
			continue
		}
		if hasNativeMath && (child.Name == "drawing" || child.Name == "pict" || child.Name == "object") {
			continue
		}
		out.WriteString(renderer.renderInline(child, partName, inTable))
	}
	content := out.String()
	if content == "" || strings.HasPrefix(strings.TrimSpace(content), "$$") {
		return content
	}

	if properties != nil {
		if docxOnOff(properties.Child("vertAlign")) || properties.Child("vertAlign") != nil {
			vertical := properties.Child("vertAlign").Attr("val")
			if vertical == "superscript" {
				content = "<sup>" + content + "</sup>"
			} else if vertical == "subscript" {
				content = "<sub>" + content + "</sub>"
			}
		}
		if docxOnOff(properties.Child("strike")) || docxOnOff(properties.Child("dstrike")) {
			if inTable {
				content = "<del>" + content + "</del>"
			} else {
				content = "~~" + content + "~~"
			}
		}
		if docxOnOff(properties.Child("i")) || docxOnOff(properties.Child("iCs")) {
			if inTable {
				content = "<em>" + content + "</em>"
			} else {
				content = "*" + content + "*"
			}
		}
		if docxOnOff(properties.Child("b")) || docxOnOff(properties.Child("bCs")) {
			if inTable {
				content = "<strong>" + content + "</strong>"
			} else {
				content = "**" + content + "**"
			}
		}
	}
	return content
}

func (renderer *docxRenderer) renderHyperlink(node *core.XMLNode, partName string, inTable bool) string {
	content := renderer.renderInlineChildren(node, partName, inTable)
	relationID := node.Attr("id")
	if relationID == "" || content == "" {
		return content
	}
	relation, ok := renderer.relationshipsFor(partName)[relationID]
	if !ok || relation.Target == "" {
		return content
	}
	if inTable {
		return `<a href="` + html.EscapeString(relation.Target) + `">` + content + `</a>`
	}
	return "[" + content + "](" + relation.Target + ")"
}

func (renderer *docxRenderer) renderImages(container *core.XMLNode, partName string, inTable bool) string {
	if container == nil || renderer.err != nil {
		return ""
	}
	candidates := container.Descendants("svgBlip")
	if len(candidates) == 0 {
		candidates = container.Descendants("blip")
	}
	candidates = append(candidates, container.Descendants("imagedata")...)
	if len(candidates) == 0 {
		return ""
	}

	altText := docxImageAltText(container)
	seen := make(map[string]bool)
	images := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		relationID := candidate.Attr("embed")
		if relationID == "" {
			relationID = candidate.Attr("link")
		}
		if relationID == "" {
			relationID = candidate.Attr("id")
		}
		if relationID == "" || seen[relationID] {
			continue
		}
		seen[relationID] = true
		imageURL, err := renderer.resolveImage(partName, relationID, altText)
		if err != nil {
			renderer.err = err
			return ""
		}
		if imageURL == "" {
			continue
		}
		if inTable {
			images = append(images, `<img src="`+html.EscapeString(imageURL)+`" alt="`+html.EscapeString(altText)+`">`)
		} else {
			images = append(images, core.MarkdownImage(altText, imageURL))
		}
	}
	return strings.Join(images, "")
}

func (renderer *docxRenderer) resolveImage(partName, relationID, altText string) (string, error) {
	relation, ok := renderer.relationshipsFor(partName)[relationID]
	if !ok {
		return "", nil
	}
	if relation.External {
		return relation.Target, nil
	}
	target := strings.ReplaceAll(relation.Target, "\\", "/")
	if decoded, err := url.PathUnescape(target); err == nil {
		target = decoded
	}
	partPath := path.Clean(path.Join(path.Dir(partName), target))
	data, ok := renderer.parts[partPath]
	if !ok {
		return "", nil
	}
	mimeType := core.ImageMIMEType(partPath, data)
	handler := renderer.settings.ImageHandlerOrDefault()
	if handler == nil {
		handler = core.DataURIImageHandler
	}
	imageURL, err := handler(renderer.ctx, core.Image{
		Name:     path.Base(partPath),
		MIMEType: mimeType,
		AltText:  altText,
		Data:     append([]byte(nil), data...),
	})
	if err != nil {
		return "", fmt.Errorf("markdown: handle docx image %q: %w", partPath, err)
	}
	return imageURL, nil
}

func (renderer *docxRenderer) relationshipsFor(partName string) map[string]core.Relationship {
	if relationships, ok := renderer.relations[partName]; ok {
		return relationships
	}
	relsName := path.Join(path.Dir(partName), "_rels", path.Base(partName)+".rels")
	relationships := core.ParseRelationships(renderer.parts[relsName])
	renderer.relations[partName] = relationships
	return relationships
}

type docxTableCell struct {
	content string
	colSpan int
	rowSpan int
	column  int
	nested  bool
}

type docxTableRow struct {
	cells []*docxTableCell
}

func (renderer *docxRenderer) renderTable(table *core.XMLNode, partName string) string {
	rowNodes := wrappedDOCXChildren(table, "tr")
	if len(rowNodes) == 0 {
		return ""
	}
	rows := make([]docxTableRow, 0, len(rowNodes))
	activeMerges := make(map[int]*docxTableCell)
	hasSpans := false
	hasNestedTable := false

	for _, rowNode := range rowNodes {
		row := docxTableRow{}
		column := 0
		touched := make(map[int]bool)
		incremented := make(map[*docxTableCell]bool)
		for _, cellNode := range wrappedDOCXChildren(rowNode, "tc") {
			properties := cellNode.Child("tcPr")
			colSpan := 1
			if properties != nil && properties.Child("gridSpan") != nil {
				if value, err := strconv.Atoi(properties.Child("gridSpan").Attr("val")); err == nil && value > 1 {
					colSpan = value
				}
			}
			content, nested := renderer.renderTableCell(cellNode, partName)
			merge := (*core.XMLNode)(nil)
			if properties != nil {
				merge = properties.Child("vMerge")
			}
			mergeValue := ""
			if merge != nil {
				mergeValue = strings.ToLower(merge.Attr("val"))
			}
			horizontalMerge := (*core.XMLNode)(nil)
			if properties != nil {
				horizontalMerge = properties.Child("hMerge")
			}
			horizontalMergeValue := ""
			if horizontalMerge != nil {
				horizontalMergeValue = strings.ToLower(horizontalMerge.Attr("val"))
			}

			if horizontalMerge != nil && horizontalMergeValue != "restart" && len(row.cells) > 0 {
				origin := row.cells[len(row.cells)-1]
				origin.colSpan += colSpan
				if strings.TrimSpace(content) != "" {
					if origin.content != "" {
						origin.content += "<br>"
					}
					origin.content += content
				}
				for index := 0; index < colSpan; index++ {
					touched[column+index] = true
					if mergeValue == "restart" {
						activeMerges[column+index] = origin
					} else {
						delete(activeMerges, column+index)
					}
				}
				column += colSpan
				hasSpans = true
				continue
			}

			if merge != nil && mergeValue != "restart" {
				origin := activeMerges[column]
				if origin != nil {
					if !incremented[origin] {
						origin.rowSpan++
						incremented[origin] = true
					}
					if strings.TrimSpace(content) != "" {
						if origin.content != "" {
							origin.content += "<br>"
						}
						origin.content += content
					}
					for index := 0; index < colSpan; index++ {
						touched[column+index] = true
					}
					column += colSpan
					hasSpans = true
					continue
				}
			}

			cell := &docxTableCell{content: content, colSpan: colSpan, rowSpan: 1, column: column, nested: nested}
			row.cells = append(row.cells, cell)
			for index := 0; index < colSpan; index++ {
				touched[column+index] = true
				if mergeValue == "restart" {
					activeMerges[column+index] = cell
				} else {
					delete(activeMerges, column+index)
				}
			}
			if colSpan > 1 || mergeValue == "restart" || horizontalMergeValue == "restart" {
				hasSpans = true
			}
			hasNestedTable = hasNestedTable || nested
			column += colSpan
		}
		for activeColumn := range activeMerges {
			if !touched[activeColumn] {
				delete(activeMerges, activeColumn)
			}
		}
		rows = append(rows, row)
	}

	tableFormat := core.TableFormatFromContext(renderer.ctx)
	if !hasSpans && !hasNestedTable && tableFormat != core.TableFormatHTML {
		plainRows := make([][]string, 0, len(rows))
		for _, row := range rows {
			values := make([]string, 0, len(row.cells))
			for _, cell := range row.cells {
				values = append(values, cell.content)
			}
			plainRows = append(plainRows, values)
		}
		return core.MarkdownTable(plainRows)
	}
	return renderDOCXHTMLTable(rows)
}

func (renderer *docxRenderer) renderTableCell(cell *core.XMLNode, partName string) (string, bool) {
	parts := make([]string, 0)
	nested := false
	var walk func(*core.XMLNode)
	walk = func(parent *core.XMLNode) {
		for _, child := range parent.Children {
			if renderer.skipNode(child) || child.Name == "tcPr" {
				continue
			}
			switch child.Name {
			case "p":
				if value := renderer.renderParagraph(child, partName, true); value != "" {
					parts = append(parts, value)
				}
			case "tbl":
				nested = true
				if value := renderer.renderTable(child, partName); value != "" {
					parts = append(parts, value)
				}
			case "AlternateContent":
				if selected := selectAlternateContent(child); selected != nil {
					walk(selected)
				}
			case "sdt", "sdtContent", "customXml", "ins", "moveTo", "smartTag":
				walk(child)
			}
		}
	}
	walk(cell)
	return strings.Join(parts, "<br>"), nested
}

func renderDOCXHTMLTable(rows []docxTableRow) string {
	grid := make([]ooxml.TableRow, 0, len(rows))
	for _, row := range rows {
		cells := make([]ooxml.TableCell, 0, len(row.cells))
		for _, cell := range row.cells {
			cells = append(cells, ooxml.TableCell{Content: cell.content, ColSpan: cell.colSpan, RowSpan: cell.rowSpan})
		}
		grid = append(grid, ooxml.TableRow{Cells: cells})
	}
	return ooxml.RenderHTMLTable(grid)
}

func (renderer *docxRenderer) renderNotes(partName, elementName string) []string {
	data := renderer.parts[partName]
	if len(data) == 0 {
		return nil
	}
	root, err := core.ParseXML(data)
	if err != nil {
		return nil
	}
	definitions := make([]string, 0)
	for _, note := range root.Descendants(elementName) {
		id := note.Attr("id")
		value, parseErr := strconv.Atoi(id)
		if parseErr != nil || value < 0 {
			continue
		}
		content := core.JoinBlocks(renderer.renderBlocks(note, partName, false))
		if content == "" {
			continue
		}
		prefix := "[^" + id + "]: "
		if elementName == "endnote" {
			prefix = "[^endnote-" + id + "]: "
		}
		content = strings.ReplaceAll(content, "\n", "\n    ")
		definitions = append(definitions, prefix+content)
	}
	return definitions
}

func (renderer *docxRenderer) listMarker(properties *core.XMLNode) (string, int) {
	if properties == nil || properties.Child("numPr") == nil {
		return "", 0
	}
	numberProperties := properties.Child("numPr")
	numberID := ""
	level := 0
	if node := numberProperties.Child("numId"); node != nil {
		numberID = node.Attr("val")
	}
	if node := numberProperties.Child("ilvl"); node != nil {
		level, _ = strconv.Atoi(node.Attr("val"))
	}
	format := renderer.numbering[numberID][level]
	if format == "bullet" {
		return "-", level
	}
	return "1.", level
}

func (renderer *docxRenderer) skipNode(node *core.XMLNode) bool {
	if node == nil {
		return true
	}
	return node.Name == "del" || node.Name == "moveFrom"
}

func parseDOCXNumbering(data []byte) map[string]map[int]string {
	result := make(map[string]map[int]string)
	if len(data) == 0 {
		return result
	}
	root, err := core.ParseXML(data)
	if err != nil {
		return result
	}
	abstract := make(map[string]map[int]string)
	for _, item := range root.Descendants("abstractNum") {
		levels := make(map[int]string)
		for _, level := range item.ChildrenNamed("lvl") {
			index, _ := strconv.Atoi(level.Attr("ilvl"))
			format := "decimal"
			if node := level.Child("numFmt"); node != nil && node.Attr("val") != "" {
				format = node.Attr("val")
			}
			levels[index] = format
		}
		abstract[item.Attr("abstractNumId")] = levels
	}
	for _, item := range root.Descendants("num") {
		if abstractID := item.Child("abstractNumId"); abstractID != nil {
			result[item.Attr("numId")] = abstract[abstractID.Attr("val")]
		}
	}
	return result
}

func wrappedDOCXChildren(parent *core.XMLNode, target string) []*core.XMLNode {
	result := make([]*core.XMLNode, 0)
	var walk func(*core.XMLNode)
	walk = func(current *core.XMLNode) {
		for _, child := range current.Children {
			if child.Name == "del" || child.Name == "moveFrom" {
				continue
			}
			if child.Name == target {
				result = append(result, child)
				continue
			}
			switch child.Name {
			case "sdt", "sdtContent", "customXml", "ins", "moveTo", "smartTag":
				walk(child)
			}
		}
	}
	walk(parent)
	return result
}

func selectAlternateContent(node *core.XMLNode) *core.XMLNode {
	if node == nil {
		return nil
	}
	if choice := node.Child("Choice"); choice != nil {
		return choice
	}
	return node.Child("Fallback")
}

func docxHeadingLevel(properties *core.XMLNode) int {
	if properties == nil {
		return 0
	}
	if outline := properties.Child("outlineLvl"); outline != nil {
		if level, err := strconv.Atoi(outline.Attr("val")); err == nil && level >= 0 && level < 6 {
			return level + 1
		}
	}
	style := properties.Child("pStyle")
	if style == nil {
		return 0
	}
	value := strings.ToLower(style.Attr("val"))
	value = strings.ReplaceAll(value, " ", "")
	for _, prefix := range []string{"heading", "标题"} {
		if strings.HasPrefix(value, prefix) {
			if level, err := strconv.Atoi(strings.TrimPrefix(value, prefix)); err == nil && level >= 1 && level <= 6 {
				return level
			}
		}
	}
	return 0
}

func docxOnOff(node *core.XMLNode) bool {
	if node == nil {
		return false
	}
	switch strings.ToLower(node.Attr("val")) {
	case "0", "false", "off", "no":
		return false
	default:
		return true
	}
}

func docxImageAltText(container *core.XMLNode) string {
	for _, elementName := range []string{"docPr", "cNvPr", "shape"} {
		for _, node := range container.Descendants(elementName) {
			for _, attribute := range []string{"descr", "title", "alt", "name"} {
				if value := strings.TrimSpace(node.Attr(attribute)); value != "" {
					return value
				}
			}
		}
	}
	return "image"
}

func escapeDOCXMarkdownText(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"*", "\\*",
		"_", "\\_",
		"[", "\\[",
		"]", "\\]",
	)
	return replacer.Replace(value)
}

func renderDOCXSymbol(node *core.XMLNode) string {
	value := strings.TrimPrefix(node.Attr("char"), "0x")
	data, err := hex.DecodeString(value)
	if err != nil || len(data) == 0 {
		return ""
	}
	if len(data) == 2 {
		return string(rune(data[0])<<8 | rune(data[1]))
	}
	return string(data)
}
