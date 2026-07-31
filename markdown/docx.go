package markdown

import (
	"context"
	"encoding/hex"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
)

type docxConverter struct {
	engine *MarkItDown
}

func newDOCXConverter(engine *MarkItDown) Converter {
	converter := &docxConverter{engine: engine}
	return newExtensionConverter(
		[]string{".docx"},
		[]string{"application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		converter.convert,
	)
}

func (converter *docxConverter) convert(ctx context.Context, data []byte, _ StreamInfo) (*Result, error) {
	parts, err := officeParts(data, func(name string) bool {
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
	root, err := parseXML(documentXML)
	if err != nil {
		return nil, fmt.Errorf("markdown: parse docx document: %w", err)
	}

	renderer := &docxRenderer{
		ctx:       ctx,
		engine:    converter.engine,
		parts:     parts,
		relations: make(map[string]map[string]relationship),
		numbering: parseDOCXNumbering(parts["word/numbering.xml"]),
	}
	body := root.first("body")
	if body == nil {
		return nil, fmt.Errorf("markdown: docx document has no body")
	}
	blocks := renderer.renderBlocks(body, "word/document.xml", false)
	blocks = append(blocks, renderer.renderNotes("word/footnotes.xml", "footnote")...)
	blocks = append(blocks, renderer.renderNotes("word/endnotes.xml", "endnote")...)
	if renderer.err != nil {
		return nil, renderer.err
	}

	title, metadata := coreProperties(parts["docProps/core.xml"])
	return &Result{Title: title, Markdown: joinMarkdownBlocks(blocks), Metadata: metadata}, nil
}

type docxRenderer struct {
	ctx       context.Context
	engine    *MarkItDown
	parts     map[string][]byte
	relations map[string]map[string]relationship
	numbering map[string]map[int]string
	err       error
}

func (renderer *docxRenderer) renderBlocks(parent *xmlNode, partName string, inTable bool) []string {
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

func (renderer *docxRenderer) renderParagraph(paragraph *xmlNode, partName string, inTable bool) string {
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

	properties := paragraph.child("pPr")
	if level := docxHeadingLevel(properties); level > 0 {
		return strings.Repeat("#", level) + " " + content
	}
	if marker, indent := renderer.listMarker(properties); marker != "" {
		return strings.Repeat("  ", indent) + marker + " " + content
	}
	style := ""
	if properties != nil && properties.child("pStyle") != nil {
		style = strings.ToLower(properties.child("pStyle").attr("val"))
	}
	if strings.Contains(style, "quote") {
		return "> " + strings.ReplaceAll(content, "\n", "\n> ")
	}
	return content
}

func (renderer *docxRenderer) renderInline(node *xmlNode, partName string, inTable bool) string {
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
			return htmlEscapeText(node.Text)
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
		return "[^" + node.attr("id") + "]"
	case "endnoteReference":
		return "[^endnote-" + node.attr("id") + "]"
	case "sdt", "sdtContent", "customXml", "ins", "moveTo", "smartTag", "fldSimple":
		return renderer.renderInlineChildren(node, partName, inTable)
	case "instrText", "pPr", "rPr", "bookmarkStart", "bookmarkEnd", "proofErr", "lastRenderedPageBreak":
		return ""
	default:
		return renderer.renderInlineChildren(node, partName, inTable)
	}
}

func (renderer *docxRenderer) renderInlineChildren(node *xmlNode, partName string, inTable bool) string {
	if node == nil {
		return ""
	}
	var out strings.Builder
	for _, child := range node.Children {
		out.WriteString(renderer.renderInline(child, partName, inTable))
	}
	return out.String()
}

func (renderer *docxRenderer) renderRun(run *xmlNode, partName string, inTable bool) string {
	properties := run.child("rPr")
	hasNativeMath := len(run.descendants("oMath")) > 0 || len(run.descendants("oMathPara")) > 0
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
		if docxOnOff(properties.child("vertAlign")) || properties.child("vertAlign") != nil {
			vertical := properties.child("vertAlign").attr("val")
			if vertical == "superscript" {
				content = "<sup>" + content + "</sup>"
			} else if vertical == "subscript" {
				content = "<sub>" + content + "</sub>"
			}
		}
		if docxOnOff(properties.child("strike")) || docxOnOff(properties.child("dstrike")) {
			if inTable {
				content = "<del>" + content + "</del>"
			} else {
				content = "~~" + content + "~~"
			}
		}
		if docxOnOff(properties.child("i")) || docxOnOff(properties.child("iCs")) {
			if inTable {
				content = "<em>" + content + "</em>"
			} else {
				content = "*" + content + "*"
			}
		}
		if docxOnOff(properties.child("b")) || docxOnOff(properties.child("bCs")) {
			if inTable {
				content = "<strong>" + content + "</strong>"
			} else {
				content = "**" + content + "**"
			}
		}
	}
	return content
}

func (renderer *docxRenderer) renderHyperlink(node *xmlNode, partName string, inTable bool) string {
	content := renderer.renderInlineChildren(node, partName, inTable)
	relationID := node.attr("id")
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

func (renderer *docxRenderer) renderImages(container *xmlNode, partName string, inTable bool) string {
	if container == nil || renderer.err != nil {
		return ""
	}
	candidates := container.descendants("svgBlip")
	if len(candidates) == 0 {
		candidates = container.descendants("blip")
	}
	candidates = append(candidates, container.descendants("imagedata")...)
	if len(candidates) == 0 {
		return ""
	}

	altText := docxImageAltText(container)
	seen := make(map[string]bool)
	images := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		relationID := candidate.attr("embed")
		if relationID == "" {
			relationID = candidate.attr("link")
		}
		if relationID == "" {
			relationID = candidate.attr("id")
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
			images = append(images, markdownImage(altText, imageURL))
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
	mimeType := imageMIMEType(partPath, data)
	handler := renderer.engine.imageHandler
	if handler == nil {
		handler = DataURIImageHandler
	}
	imageURL, err := handler(renderer.ctx, Image{
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

func (renderer *docxRenderer) relationshipsFor(partName string) map[string]relationship {
	if relationships, ok := renderer.relations[partName]; ok {
		return relationships
	}
	relsName := path.Join(path.Dir(partName), "_rels", path.Base(partName)+".rels")
	relationships := parseRelationships(renderer.parts[relsName])
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

func (renderer *docxRenderer) renderTable(table *xmlNode, partName string) string {
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
			properties := cellNode.child("tcPr")
			colSpan := 1
			if properties != nil && properties.child("gridSpan") != nil {
				if value, err := strconv.Atoi(properties.child("gridSpan").attr("val")); err == nil && value > 1 {
					colSpan = value
				}
			}
			content, nested := renderer.renderTableCell(cellNode, partName)
			merge := (*xmlNode)(nil)
			if properties != nil {
				merge = properties.child("vMerge")
			}
			mergeValue := ""
			if merge != nil {
				mergeValue = strings.ToLower(merge.attr("val"))
			}
			horizontalMerge := (*xmlNode)(nil)
			if properties != nil {
				horizontalMerge = properties.child("hMerge")
			}
			horizontalMergeValue := ""
			if horizontalMerge != nil {
				horizontalMergeValue = strings.ToLower(horizontalMerge.attr("val"))
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

	config, _ := renderer.ctx.Value(conversionConfigKey{}).(conversionConfig)
	if !hasSpans && !hasNestedTable && config.tableFormat != TableFormatHTML {
		plainRows := make([][]string, 0, len(rows))
		for _, row := range rows {
			values := make([]string, 0, len(row.cells))
			for _, cell := range row.cells {
				values = append(values, cell.content)
			}
			plainRows = append(plainRows, values)
		}
		return markdownTable(plainRows)
	}
	return renderDOCXHTMLTable(rows)
}

func (renderer *docxRenderer) renderTableCell(cell *xmlNode, partName string) (string, bool) {
	parts := make([]string, 0)
	nested := false
	var walk func(*xmlNode)
	walk = func(parent *xmlNode) {
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
	if len(rows) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString("<table>")
	for rowIndex, row := range rows {
		out.WriteString("\n<tr>")
		tag := "td"
		if rowIndex == 0 {
			tag = "th"
		}
		for _, cell := range row.cells {
			writeDOCXHTMLCell(&out, tag, cell)
		}
		out.WriteString("</tr>")
	}
	out.WriteString("\n</table>")
	return out.String()
}

func writeDOCXHTMLCell(out *strings.Builder, tag string, cell *docxTableCell) {
	out.WriteString("<")
	out.WriteString(tag)
	if cell.colSpan > 1 {
		out.WriteString(` colspan="`)
		out.WriteString(strconv.Itoa(cell.colSpan))
		out.WriteString(`"`)
	}
	if cell.rowSpan > 1 {
		out.WriteString(` rowspan="`)
		out.WriteString(strconv.Itoa(cell.rowSpan))
		out.WriteString(`"`)
	}
	out.WriteString(">")
	out.WriteString(cell.content)
	out.WriteString("</")
	out.WriteString(tag)
	out.WriteString(">")
}

func (renderer *docxRenderer) renderNotes(partName, elementName string) []string {
	data := renderer.parts[partName]
	if len(data) == 0 {
		return nil
	}
	root, err := parseXML(data)
	if err != nil {
		return nil
	}
	definitions := make([]string, 0)
	for _, note := range root.descendants(elementName) {
		id := note.attr("id")
		value, parseErr := strconv.Atoi(id)
		if parseErr != nil || value < 0 {
			continue
		}
		content := joinMarkdownBlocks(renderer.renderBlocks(note, partName, false))
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

func (renderer *docxRenderer) listMarker(properties *xmlNode) (string, int) {
	if properties == nil || properties.child("numPr") == nil {
		return "", 0
	}
	numberProperties := properties.child("numPr")
	numberID := ""
	level := 0
	if node := numberProperties.child("numId"); node != nil {
		numberID = node.attr("val")
	}
	if node := numberProperties.child("ilvl"); node != nil {
		level, _ = strconv.Atoi(node.attr("val"))
	}
	format := renderer.numbering[numberID][level]
	if format == "bullet" {
		return "-", level
	}
	return "1.", level
}

func (renderer *docxRenderer) skipNode(node *xmlNode) bool {
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
	root, err := parseXML(data)
	if err != nil {
		return result
	}
	abstract := make(map[string]map[int]string)
	for _, item := range root.descendants("abstractNum") {
		levels := make(map[int]string)
		for _, level := range item.children("lvl") {
			index, _ := strconv.Atoi(level.attr("ilvl"))
			format := "decimal"
			if node := level.child("numFmt"); node != nil && node.attr("val") != "" {
				format = node.attr("val")
			}
			levels[index] = format
		}
		abstract[item.attr("abstractNumId")] = levels
	}
	for _, item := range root.descendants("num") {
		if abstractID := item.child("abstractNumId"); abstractID != nil {
			result[item.attr("numId")] = abstract[abstractID.attr("val")]
		}
	}
	return result
}

func wrappedDOCXChildren(parent *xmlNode, target string) []*xmlNode {
	result := make([]*xmlNode, 0)
	var walk func(*xmlNode)
	walk = func(current *xmlNode) {
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

func selectAlternateContent(node *xmlNode) *xmlNode {
	if node == nil {
		return nil
	}
	if choice := node.child("Choice"); choice != nil {
		return choice
	}
	return node.child("Fallback")
}

func docxHeadingLevel(properties *xmlNode) int {
	if properties == nil {
		return 0
	}
	if outline := properties.child("outlineLvl"); outline != nil {
		if level, err := strconv.Atoi(outline.attr("val")); err == nil && level >= 0 && level < 6 {
			return level + 1
		}
	}
	style := properties.child("pStyle")
	if style == nil {
		return 0
	}
	value := strings.ToLower(style.attr("val"))
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

func docxOnOff(node *xmlNode) bool {
	if node == nil {
		return false
	}
	switch strings.ToLower(node.attr("val")) {
	case "0", "false", "off", "no":
		return false
	default:
		return true
	}
}

func docxImageAltText(container *xmlNode) string {
	for _, elementName := range []string{"docPr", "cNvPr", "shape"} {
		for _, node := range container.descendants(elementName) {
			for _, attribute := range []string{"descr", "title", "alt", "name"} {
				if value := strings.TrimSpace(node.attr(attribute)); value != "" {
					return value
				}
			}
		}
	}
	return "image"
}

func imageMIMEType(name string, data []byte) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".webp":
		return "image/webp"
	case ".bmp", ".dib":
		return "image/bmp"
	case ".tif", ".tiff":
		return "image/tiff"
	case ".emf":
		return "image/x-emf"
	case ".wmf":
		return "image/wmf"
	default:
		return http.DetectContentType(data)
	}
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

func renderDOCXSymbol(node *xmlNode) string {
	value := strings.TrimPrefix(node.attr("char"), "0x")
	data, err := hex.DecodeString(value)
	if err != nil || len(data) == 0 {
		return ""
	}
	if len(data) == 2 {
		return string(rune(data[0])<<8 | rune(data[1]))
	}
	return string(data)
}
