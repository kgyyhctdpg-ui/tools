package renderer

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/88250/lute/ast"
	"github.com/scoming-dev/tools/docx/schema/soo/wml"
)

const (
	maxTOCFieldLevel    = 2
	tocFieldInstruction = `TOC \o "1-3" \h \z \u`
)

// TocNode represents a table-of-contents entry.
type TocNode struct {
	Title    string
	Level    int
	PageNum  int
	ID       string
	Children []*TocNode
}

// TocTree contains the root table-of-contents entries.
type TocTree struct {
	Root []*TocNode
}

func (r *DocxRenderer) renderToC(_ *ast.Node, entering bool) ast.WalkStatus {
	if entering && len(r.headings()) > 0 {
		r.renderWpsTOCField(r.buildTocTree())
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) headings() (ret []*ast.Node) {
	for n := r.Tree.Root.FirstChild; nil != n; n = n.Next {
		r.headings0(n, &ret)
	}
	return
}

func (r *DocxRenderer) headings0(n *ast.Node, headings *[]*ast.Node) {
	if ast.NodeHeading == n.Type {
		*headings = append(*headings, n)
		return
	}
	if ast.NodeList == n.Type || ast.NodeListItem == n.Type || ast.NodeBlockquote == n.Type {
		for c := n.FirstChild; nil != c; c = c.Next {
			r.headings0(c, headings)
		}
	}
}

func (r *DocxRenderer) buildTocTree() *TocTree {
	headings := r.headings()
	if len(headings) == 0 {
		return &TocTree{Root: []*TocNode{}}
	}

	tree := &TocTree{Root: []*TocNode{}}
	var stack []*TocNode

	for i, heading := range headings {
		rawID := fmt.Sprintf("heading_%d_%s", i+1, heading.Text())
		hash := md5.Sum([]byte(rawID))
		bookmarkID := fmt.Sprintf("heading_%s", hex.EncodeToString(hash[:]))

		node := &TocNode{
			Title:    heading.Text(),
			Level:    heading.HeadingLevel,
			PageNum:  i + 1,
			ID:       bookmarkID,
			Children: []*TocNode{},
		}

		r.headingBookmarks[heading.Text()] = bookmarkID

		for len(stack) > 0 && stack[len(stack)-1].Level >= node.Level {
			stack = stack[:len(stack)-1]
		}

		if len(stack) == 0 {
			tree.Root = append(tree.Root, node)
		} else {
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, node)
		}

		stack = append(stack, node)
	}

	return tree
}

func (r *DocxRenderer) renderTocInCover(tree *TocTree) {
	if len(tree.Root) == 0 {
		return
	}

	tocTitlePara := r.doc.AddParagraph()
	tocTitleParaProps := tocTitlePara.Properties()
	tocTitleParaProps.SetAlignment(wml.ST_JcCenter)

	titleSpacing := tocTitleParaProps.Spacing()
	titleSpacing.SetBefore(r.config.TOC.TitleSpacingBefore)
	titleSpacing.SetAfter(r.config.TOC.TitleSpacingAfter)

	tocTitleRun := tocTitlePara.AddRun()
	tocTitleProps := tocTitleRun.Properties()
	r.setFontFamily(&tocTitleProps, r.config.Fonts.Title)
	r.setFontSize(&tocTitleProps, r.config.TOC.TitleFontSize)
	tocTitleProps.SetBold(true)
	tocTitleRun.AddText("目录")

	r.renderWpsTOCField(tree)
}

func (r *DocxRenderer) renderWpsTOCField(tree *TocTree) {
	entries := flattenTocEntries(tree)
	para := r.doc.AddParagraph()
	run := para.AddRun()
	run.AddRawXML(r.wpsTOCFieldParagraphsXML(entries))
	r.addPageBreakAfterTOC()
}

func (r *DocxRenderer) wpsTOCFieldParagraphsXML(entries []*TocNode) string {
	if len(entries) == 0 {
		return `<w:p>` + tocFieldStartXML(tocFieldInstruction) + `<w:r><w:fldChar w:fldCharType="end"/></w:r></w:p>`
	}

	var builder strings.Builder
	for i, entry := range entries {
		builder.WriteString(`<w:p>`)
		builder.WriteString(r.tocParagraphPropertiesXML(entry.Level))
		if i == 0 {
			builder.WriteString(tocFieldStartXML(tocFieldInstruction))
		}
		builder.WriteString(r.tocEntryResultXML(entry))
		if i == len(entries)-1 {
			builder.WriteString(`<w:r><w:fldChar w:fldCharType="end"/></w:r>`)
		}
		builder.WriteString(`</w:p>`)
	}
	return builder.String()
}

func (r *DocxRenderer) addPageBreakAfterTOC() {
	para := r.doc.AddParagraph()
	run := para.AddRun()
	run.AddPageBreak()
}

func flattenTocEntries(tree *TocTree) []*TocNode {
	if tree == nil {
		return nil
	}
	var entries []*TocNode
	var walk func(nodes []*TocNode)
	walk = func(nodes []*TocNode) {
		for _, node := range nodes {
			if node.Level <= maxTOCFieldLevel {
				entries = append(entries, node)
			}
			walk(node.Children)
		}
	}
	walk(tree.Root)
	return entries
}

func tocStyleID(level int) string {
	if level < 1 {
		level = 1
	}
	if level > 9 {
		level = 9
	}
	return fmt.Sprintf("TOC%d", level)
}

func (r *DocxRenderer) tocParagraphPropertiesXML(level int) string {
	lineHeight := float64(r.config.Text.ContentSize) * r.config.Paragraph.LineHeightMultiplier
	lineTwips := pointsToTwips(lineHeight)
	depth := level - 1
	if depth < 0 {
		depth = 0
	}
	leftTwips := pointsToTwips(r.config.TOC.IndentPerDepth * float64(depth))

	var builder strings.Builder
	builder.WriteString(`<w:pPr>`)
	builder.WriteString(`<w:pStyle w:val="` + escapeXMLAttr(tocStyleID(level)) + `"/>`)
	builder.WriteString(`<w:tabs><w:tab w:val="right" w:leader="dot" w:pos="8600"/></w:tabs>`)
	builder.WriteString(`<w:spacing w:line="` + fmt.Sprintf("%d", lineTwips) + `" w:lineRule="auto"/>`)
	if leftTwips > 0 {
		builder.WriteString(`<w:ind w:left="` + fmt.Sprintf("%d", leftTwips) + `"/>`)
	}
	builder.WriteString(`<w:jc w:val="both"/>`)
	builder.WriteString(`</w:pPr>`)
	return builder.String()
}

func (r *DocxRenderer) tocEntryResultXML(entry *TocNode) string {
	bookmark := tocBookmarkName(entry)
	runProps := r.tocRunPropertiesXML()
	return `<w:hyperlink w:anchor="` + escapeXMLAttr(bookmark) + `" w:history="1">` +
		`<w:r>` + runProps + `<w:t>` + escapeXMLText(entry.Title) + `</w:t></w:r>` +
		`<w:r><w:tab/></w:r>` +
		tocPageReferenceXML(bookmark, entry.PageNum, runProps) +
		`</w:hyperlink>`
}

func (r *DocxRenderer) tocRunPropertiesXML() string {
	size := r.config.TOC.ItemFontSize * 2
	font := escapeXMLAttr(r.config.Fonts.Content)
	latin := escapeXMLAttr(r.config.Fonts.Latin)
	return `<w:rPr><w:rFonts w:ascii="` + latin + `" w:hAnsi="` + latin + `" w:eastAsia="` + font + `"/>` +
		`<w:sz w:val="` + fmt.Sprintf("%d", size) + `"/><w:szCs w:val="` + fmt.Sprintf("%d", size) + `"/></w:rPr>`
}

func tocPageReferenceXML(bookmark string, pageNum int, runProps string) string {
	instruction := ` PAGEREF ` + bookmark + ` \h `
	return `<w:r><w:fldChar w:fldCharType="begin" w:dirty="true"/></w:r>` +
		`<w:r><w:instrText xml:space="preserve">` + escapeXMLText(instruction) + `</w:instrText></w:r>` +
		`<w:r><w:fldChar w:fldCharType="separate"/></w:r>` +
		`<w:r>` + runProps + `<w:t>` + fmt.Sprintf("%d", pageNum) + `</w:t></w:r>` +
		`<w:r><w:fldChar w:fldCharType="end"/></w:r>`
}

func tocBookmarkName(entry *TocNode) string {
	if entry == nil || entry.PageNum <= 0 {
		return "_Toc0"
	}
	return fmt.Sprintf("_Toc%d", entry.PageNum-1)
}

func tocFieldStartXML(instruction string) string {
	return `<w:r><w:fldChar w:fldCharType="begin" w:dirty="true"/></w:r>` +
		`<w:r><w:instrText xml:space="preserve">` + escapeXMLText(instruction) + `</w:instrText></w:r>` +
		`<w:r><w:fldChar w:fldCharType="separate"/></w:r>`
}

func escapeXMLText(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	)
	return replacer.Replace(value)
}

func escapeXMLAttr(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
	)
	return replacer.Replace(value)
}

func pointsToTwips(value float64) int {
	if value <= 0 {
		return 0
	}
	return int(value*20 + 0.5)
}
