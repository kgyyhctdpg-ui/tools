// 一款将 Markdown 文本转换为 Word 文档 (.docx) 的小工具

package docx

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	docxcommon "github.com/scoming-dev/tools/docx/common"
	docxmath "github.com/scoming-dev/tools/docx/math"

	"github.com/88250/lute/ast"
	"github.com/88250/lute/parse"
	"github.com/88250/lute/render"
	"github.com/88250/lute/util"
	"github.com/scoming-dev/tools/docx/color"
	"github.com/scoming-dev/tools/docx/document"
	"github.com/scoming-dev/tools/docx/measurement"
	"github.com/scoming-dev/tools/docx/media"
	"github.com/scoming-dev/tools/docx/schema/soo/wml"
)

// DocxRenderer 描述了 DOCX 渲染器。
type DocxRenderer struct {
	*render.BaseRenderer
	needRenderFootnotesDef bool

	headingStyle int // 标题渲染模式：0 默认，1 一级标题居中分页，2 一级标题左对齐分页

	config     Config
	doc        *document.Document    // DOCX 生成器句柄
	zoom       float64               // 字体、行高大小倍数
	margin     float64               // 页边距
	paragraphs []*document.Paragraph // 当前段落栈
	runs       []*document.Run       // 当前排版栈
	files      []string              // 生成文件后待清理的临时文件路径
	listStack  []listState           // 当前列表编号定义栈

	// 表格相关状态
	currentTable *document.Table
	currentRow   *document.Row
	currentCell  *document.Cell

	// 目录相关状态
	headingBookmarks map[string]string // 标题文本到书签ID的映射
	headingIndex     int               // 当前处理的标题索引

	// 页头和页脚
	header document.Header
	footer document.Footer

	htmlBlockPlugins map[string]HTMLBlockPlugin
	cover            CoverPlugin
}

type listState struct {
	definition          document.Definition
	checkedDefinition   document.Definition
	uncheckedDefinition document.Definition
}

// TocNode 表示目录树的节点
type TocNode struct {
	Title    string     // 标题文本
	Level    int        // 标题级别 (1-6)
	PageNum  int        // 页码 (目前固定为1，可根据实际需要调整)
	ID       string     // 唯一标识符，用作书签名
	Children []*TocNode // 子节点
}

// TocTree 表示完整的目录树
type TocTree struct {
	Root []*TocNode // 根级节点列表
}

// NewDocxRenderer 创建一个 DOCX 渲染器。
func NewDocxRenderer(tree *parse.Tree, options *render.Options, headingStyle int, rendererOptions ...RendererOption) *DocxRenderer {
	doc := document.New()
	// docxcommon.AddSectionBreakWithOrientation(doc, docxcommon.OrientationH)
	ret := &DocxRenderer{
		BaseRenderer:           render.NewBaseRenderer(tree, options),
		needRenderFootnotesDef: false,
		doc:                    doc,
		headingBookmarks:       make(map[string]string),
		headingIndex:           0,
		headingStyle:           normalizeHeadingStyle(headingStyle),
		config:                 DefaultConfig(),
		htmlBlockPlugins:       make(map[string]HTMLBlockPlugin),
	}
	ret.zoom = 1
	ret.margin = ret.config.Page.MarginMM * ret.zoom

	ret.RendererFuncs[ast.NodeDocument] = ret.renderDocument
	ret.RendererFuncs[ast.NodeParagraph] = ret.renderParagraph
	ret.RendererFuncs[ast.NodeText] = ret.renderText
	ret.RendererFuncs[ast.NodeCodeSpan] = ret.renderCodeSpan
	ret.RendererFuncs[ast.NodeCodeSpanOpenMarker] = ret.renderCodeSpanOpenMarker
	ret.RendererFuncs[ast.NodeCodeSpanContent] = ret.renderCodeSpanContent
	ret.RendererFuncs[ast.NodeCodeSpanCloseMarker] = ret.renderCodeSpanCloseMarker
	ret.RendererFuncs[ast.NodeCodeBlock] = ret.renderCodeBlock
	ret.RendererFuncs[ast.NodeCodeBlockFenceOpenMarker] = ret.renderCodeBlockOpenMarker
	ret.RendererFuncs[ast.NodeCodeBlockFenceInfoMarker] = ret.renderCodeBlockInfoMarker
	ret.RendererFuncs[ast.NodeCodeBlockCode] = ret.renderCodeBlockCode
	ret.RendererFuncs[ast.NodeCodeBlockFenceCloseMarker] = ret.renderCodeBlockCloseMarker
	ret.RendererFuncs[ast.NodeMathBlock] = ret.renderMathBlock
	ret.RendererFuncs[ast.NodeMathBlockOpenMarker] = ret.renderMathBlockOpenMarker
	ret.RendererFuncs[ast.NodeMathBlockContent] = ret.renderMathBlockContent
	ret.RendererFuncs[ast.NodeMathBlockCloseMarker] = ret.renderMathBlockCloseMarker
	ret.RendererFuncs[ast.NodeInlineMath] = ret.renderInlineMath
	ret.RendererFuncs[ast.NodeInlineMathOpenMarker] = ret.renderInlineMathOpenMarker
	ret.RendererFuncs[ast.NodeInlineMathContent] = ret.renderInlineMathContent
	ret.RendererFuncs[ast.NodeInlineMathCloseMarker] = ret.renderInlineMathCloseMarker
	ret.RendererFuncs[ast.NodeEmphasis] = ret.renderEmphasis
	ret.RendererFuncs[ast.NodeEmA6kOpenMarker] = ret.renderEmAsteriskOpenMarker
	ret.RendererFuncs[ast.NodeEmA6kCloseMarker] = ret.renderEmAsteriskCloseMarker
	ret.RendererFuncs[ast.NodeEmU8eOpenMarker] = ret.renderEmUnderscoreOpenMarker
	ret.RendererFuncs[ast.NodeEmU8eCloseMarker] = ret.renderEmUnderscoreCloseMarker
	ret.RendererFuncs[ast.NodeStrong] = ret.renderStrong
	ret.RendererFuncs[ast.NodeStrongA6kOpenMarker] = ret.renderStrongA6kOpenMarker
	ret.RendererFuncs[ast.NodeStrongA6kCloseMarker] = ret.renderStrongA6kCloseMarker
	ret.RendererFuncs[ast.NodeStrongU8eOpenMarker] = ret.renderStrongU8eOpenMarker
	ret.RendererFuncs[ast.NodeStrongU8eCloseMarker] = ret.renderStrongU8eCloseMarker
	ret.RendererFuncs[ast.NodeBlockquote] = ret.renderBlockquote
	ret.RendererFuncs[ast.NodeBlockquoteMarker] = ret.renderBlockquoteMarker
	ret.RendererFuncs[ast.NodeHeading] = ret.renderHeading
	ret.RendererFuncs[ast.NodeHeadingC8hMarker] = ret.renderHeadingC8hMarker
	ret.RendererFuncs[ast.NodeList] = ret.renderList
	ret.RendererFuncs[ast.NodeListItem] = ret.renderListItem
	ret.RendererFuncs[ast.NodeThematicBreak] = ret.renderThematicBreak
	ret.RendererFuncs[ast.NodeHardBreak] = ret.renderHardBreak
	ret.RendererFuncs[ast.NodeSoftBreak] = ret.renderSoftBreak
	ret.RendererFuncs[ast.NodeHTMLBlock] = ret.renderHTML
	ret.RendererFuncs[ast.NodeInlineHTML] = ret.renderInlineHTML
	ret.RendererFuncs[ast.NodeLink] = ret.renderLink
	ret.RendererFuncs[ast.NodeImage] = ret.renderImage
	ret.RendererFuncs[ast.NodeBang] = ret.renderBang
	ret.RendererFuncs[ast.NodeOpenBracket] = ret.renderOpenBracket
	ret.RendererFuncs[ast.NodeCloseBracket] = ret.renderCloseBracket
	ret.RendererFuncs[ast.NodeOpenParen] = ret.renderOpenParen
	ret.RendererFuncs[ast.NodeCloseParen] = ret.renderCloseParen
	ret.RendererFuncs[ast.NodeLinkText] = ret.renderLinkText
	ret.RendererFuncs[ast.NodeLinkSpace] = ret.renderLinkSpace
	ret.RendererFuncs[ast.NodeLinkDest] = ret.renderLinkDest
	ret.RendererFuncs[ast.NodeLinkTitle] = ret.renderLinkTitle
	ret.RendererFuncs[ast.NodeStrikethrough] = ret.renderStrikethrough
	ret.RendererFuncs[ast.NodeStrikethrough1OpenMarker] = ret.renderStrikethrough1OpenMarker
	ret.RendererFuncs[ast.NodeStrikethrough1CloseMarker] = ret.renderStrikethrough1CloseMarker
	ret.RendererFuncs[ast.NodeStrikethrough2OpenMarker] = ret.renderStrikethrough2OpenMarker
	ret.RendererFuncs[ast.NodeStrikethrough2CloseMarker] = ret.renderStrikethrough2CloseMarker
	ret.RendererFuncs[ast.NodeTaskListItemMarker] = ret.renderTaskListItemMarker
	ret.RendererFuncs[ast.NodeTable] = ret.renderTable
	ret.RendererFuncs[ast.NodeTableHead] = ret.renderTableHead
	ret.RendererFuncs[ast.NodeTableRow] = ret.renderTableRow
	ret.RendererFuncs[ast.NodeTableCell] = ret.renderTableCell
	ret.RendererFuncs[ast.NodeEmoji] = ret.renderEmoji
	ret.RendererFuncs[ast.NodeEmojiUnicode] = ret.renderEmojiUnicode
	ret.RendererFuncs[ast.NodeEmojiImg] = ret.renderEmojiImg
	ret.RendererFuncs[ast.NodeEmojiAlias] = ret.renderEmojiAlias
	ret.RendererFuncs[ast.NodeFootnotesDef] = ret.renderFootnotesDef
	ret.RendererFuncs[ast.NodeFootnotesRef] = ret.renderFootnotesRef
	ret.RendererFuncs[ast.NodeToC] = ret.renderToC
	ret.RendererFuncs[ast.NodeBackslash] = ret.renderBackslash
	ret.RendererFuncs[ast.NodeBackslashContent] = ret.renderBackslashContent
	for _, option := range rendererOptions {
		if option != nil {
			option(ret)
		}
	}
	ret.margin = ret.config.Page.MarginMM * ret.zoom
	ret.SetStyle()
	ret.setupHeaderFooter()
	if ret.cover != nil {
		if err := ret.cover.RenderCover(ret); err != nil {
			log.Printf("渲染封面失败: %v", err)
		}
	}
	return ret
}

func normalizeHeadingStyle(style int) int {
	switch style {
	case 1, 2:
		return style
	default:
		return 0
	}
}

func (r *DocxRenderer) SetStyle() {
	linkStyle := r.doc.Styles.AddStyle("ListParagraph", wml.ST_StyleTypeCharacter, false)
	linkStyle.SetName("ListParagraph")
	linkStyle.SetBasedOn("DefaultParagraphFont")
	props := linkStyle.RunProperties()
	r.SetContentFont(&props, r.config.Fonts.Content)
}

func (r *DocxRenderer) setupHeaderFooter() {
	// 创建页头
	r.header = r.doc.AddHeader()
	hdrPara := r.header.AddParagraph()
	hdrParaProps := hdrPara.Properties()
	hdrParaProps.SetAlignment(r.config.HeaderFooter.Alignment) // 页头居中对齐

	hdrRun := hdrPara.AddRun()
	hdrRunProps := hdrRun.Properties()
	r.setFontFamily(&hdrRunProps, r.config.Fonts.Content)
	r.setFontSzie(&hdrRunProps, r.config.HeaderFooter.FontSize)
	// hdrRun.AddText("项目文档")    // 可以根据需要修改为实际的项目名称

	// 创建页脚
	r.footer = r.doc.AddFooter()
	ftrPara := r.footer.AddParagraph()
	ftrParaProps := ftrPara.Properties()
	ftrParaProps.SetAlignment(r.config.HeaderFooter.Alignment) // 页脚居中对齐

	ftrRun := ftrPara.AddRun()
	ftrRunProps := ftrRun.Properties()
	r.setFontFamily(&ftrRunProps, r.config.Fonts.Content)
	r.setFontSzie(&ftrRunProps, r.config.HeaderFooter.FontSize)

	// 添加页码：第X页 共Y页
	ftrRun.AddText("第")
	ftrRun.AddField(document.FieldCurrentPage)
	ftrRun.AddText("页 共")
	ftrRun.AddField(document.FieldNumberOfPages)
	ftrRun.AddText("页")

	ftrRun.AddText(r.config.HeaderFooter.FooterSuffix)
}

func (r *DocxRenderer) Render() (output []byte) {
	ast.Walk(r.Tree.Root, func(n *ast.Node, entering bool) ast.WalkStatus {
		extRender := r.ExtRendererFuncs[n.Type]
		if nil != extRender {
			output, status := extRender(n, entering)
			r.WriteString(output)
			return status
		}
		render := r.RendererFuncs[n.Type]
		if nil == render {
			if nil != r.DefaultRendererFunc {
				return r.DefaultRendererFunc(n, entering)
			} else {
				return r.renderDefault(n, entering)
			}
		}
		return render(n, entering)
	})

	if 0 < len(r.FootnotesDefs) {
		output = r.RenderFootnotesDefs(r.Tree.Context)
	}
	return
}

func (r *DocxRenderer) renderDefault(n *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		r.WriteString("not found render function for node [type=" + n.Type.String() + ", Tokens=" + util.BytesToStr(n.Tokens) + "]")
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) renderBackslashContent(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		r.Write(node.Tokens)
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) renderBackslash(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderToC(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		headings := r.headings()
		length := len(headings)
		if 1 > length {
			return ast.WalkContinue
		}
		r.WriteString("<div class=\"toc-div\">")
		for i, heading := range headings {
			level := strconv.Itoa(heading.HeadingLevel)
			spaces := (heading.HeadingLevel - 1) * 2
			r.WriteString(strings.Repeat("&emsp;", spaces))
			r.WriteString("<span class=\"toc-h" + level + "\">")
			r.WriteString("<a class=\"toc-a\" href=\"#toc_h" + level + "_" + strconv.Itoa(i) + "\">" + heading.Text() + "</a></span><br>")
		}
		r.WriteString("</div>\n\n")
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

// buildTocTree 从文档的标题构建目录树
func (r *DocxRenderer) buildTocTree() *TocTree {
	headings := r.headings()
	if len(headings) == 0 {
		return &TocTree{Root: []*TocNode{}}
	}

	tree := &TocTree{Root: []*TocNode{}}
	var stack []*TocNode // 用于维护层级关系的栈

	for i, heading := range headings {
		// 生成唯一的书签ID，使用MD5哈希确保唯一性和规范性
		rawID := fmt.Sprintf("heading_%d_%s", i+1, heading.Text())
		hash := md5.Sum([]byte(rawID))
		bookmarkID := fmt.Sprintf("heading_%s", hex.EncodeToString(hash[:]))

		node := &TocNode{
			Title:    heading.Text(),
			Level:    heading.HeadingLevel,
			PageNum:  i + 1,      // 简单的页码分配，实际使用时可以更精确
			ID:       bookmarkID, // 设置唯一的书签ID
			Children: []*TocNode{},
		}

		// 建立标题文本到书签ID的映射
		r.headingBookmarks[heading.Text()] = bookmarkID

		// 找到合适的父节点
		for len(stack) > 0 && stack[len(stack)-1].Level >= node.Level {
			stack = stack[:len(stack)-1] // 弹出栈顶
		}

		if len(stack) == 0 {
			// 作为根节点
			tree.Root = append(tree.Root, node)
		} else {
			// 作为当前栈顶的子节点
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, node)
		}

		stack = append(stack, node) // 当前节点入栈
	}

	return tree
}

// renderTocInCover 在封面中渲染目录树
func (r *DocxRenderer) renderTocInCover(tree *TocTree) {
	if len(tree.Root) == 0 {
		return
	}

	// 目录标题 - 优化样式
	tocTitlePara := r.doc.AddParagraph()
	tocTitleParaProps := tocTitlePara.Properties()
	tocTitleParaProps.SetAlignment(wml.ST_JcCenter)

	// 设置标题段落间距
	titleSpacing := tocTitleParaProps.Spacing()
	titleSpacing.SetBefore(r.config.TOC.TitleSpacingBefore)
	titleSpacing.SetAfter(r.config.TOC.TitleSpacingAfter)

	tocTitleRun := tocTitlePara.AddRun()
	tocTitleProps := tocTitleRun.Properties()
	r.setFontFamily(&tocTitleProps, r.config.Fonts.Title)
	r.setFontSzie(&tocTitleProps, r.config.TOC.TitleFontSize)
	tocTitleProps.SetBold(true) // 确保标题加粗
	tocTitleRun.AddText("目录")

	// 渲染目录内容
	r.renderTocNodes(tree.Root, 0)
}

// renderTocNodes 递归渲染目录节点，只显示到第二级，模拟WPS原生目录实现
func (r *DocxRenderer) renderTocNodes(nodes []*TocNode, depth int) {
	for _, node := range nodes {
		// 创建目录项段落
		para := r.doc.AddParagraph()
		paraProps := para.Properties()

		// 设置段落样式 - 模拟WPS目录样式
		r.setParagraphSpacing(&para)

		// 根据层级设置左缩进 - WPS风格
		if depth > 0 {
			paraProps.SetFirstLineIndent(r.config.TOC.IndentPerDepth * measurement.Distance(depth))
		}

		// 🔧 关键改进：设置制表位实现专业的页码右对齐
		r.addTOCTabStop(paraProps, depth)

		// 创建内部超链接到书签
		link := para.AddHyperLink()
		link.SetTarget("") // 内部链接不设置外部目标

		// 通过底层XML设置内部链接属性
		linkXML := link.X()
		linkXML.AnchorAttr = &node.ID // 设置锚点为书签ID

		// 添加标题文本作为超链接
		titleRun := link.AddRun()
		titleRunProps := titleRun.Properties()
		r.setFontFamily(&titleRunProps, r.config.Fonts.Content)
		titleRunProps.SetStyle("Hyperlink") // 设置超链接样式

		// 根据层级设置字体样式 - 优化后的专业标准
		r.setFontSzie(&titleRunProps, r.config.TOC.ItemFontSize)

		// 添加标题文本
		titleRun.AddText(node.Title)

		// 在同一个run中添加制表符，确保连接到制表位
		titleRun.AddTab()

		// 添加页码到同一个run中，确保制表符和制表位正确工作
		titleRun.AddText(fmt.Sprintf("%d", node.PageNum))

		// 只递归渲染到第二级
		if len(node.Children) > 0 && depth < 1 {
			r.renderTocNodes(node.Children, depth+1)
		}
	}
}

// addTOCTabStop 为TOC段落添加制表位，实现专业的页码右对齐效果
func (r *DocxRenderer) addTOCTabStop(paraProps document.ParagraphProperties, depth int) {
	// 计算制表位位置，使用标准A4纸张的右边距位置
	var tabPosition measurement.Distance
	tabPosition = 8600

	// 通过底层XML设置制表位
	paraPropsXML := paraProps.X()

	// 清除现有制表位并重新创建
	paraPropsXML.Tabs = wml.NewCT_Tabs()

	// 创建右对齐制表位
	tabStop := wml.NewCT_TabStop()
	tabStop.ValAttr = wml.ST_TabJcRight   // 右对齐
	tabStop.LeaderAttr = wml.ST_TabTlcDot // 点引导符

	// 设置制表位位置 - 直接使用twips值
	twipsPos := int64(tabPosition)
	tabStop.PosAttr = wml.ST_SignedTwipsMeasure{}
	tabStop.PosAttr.Int64 = &twipsPos

	// 添加到制表位集合
	paraPropsXML.Tabs.Tab = append(paraPropsXML.Tabs.Tab, tabStop)

	// 设置段落右缩进，确保页码有足够空间右对齐
	if paraPropsXML.Ind == nil {
		paraPropsXML.Ind = wml.NewCT_Ind()
	}
	rightIndent := int64(360) // 0.25英寸的右缩进，给页码留空间
	paraPropsXML.Ind.RightAttr = &wml.ST_SignedTwipsMeasure{}
	paraPropsXML.Ind.RightAttr.Int64 = &rightIndent
}

func (r *DocxRenderer) RenderFootnotesDefs(context *parse.Context) []byte {
	//if r.needRenderFootnotesDef {
	//	return nil
	//}
	//
	//r.addPage()
	//r.renderThematicBreak(nil, false)
	//for i, def := range context.FootnotesDefs {
	//	r.pdf.SetAnchor(string(def.Tokens))
	//	r.WriteString(fmt.Sprint(i+1) + ". ")
	//	tree := &parse.Tree{Name: "", Context: context}
	//	tree.Context.Tree = tree
	//	tree.Root = &ast.Node{Type: ast.NodeDocument}
	//	tree.Root.AppendChild(def)
	//	r.Tree = tree
	//	r.needRenderFootnotesDef = true
	//	r.Render()
	//	r.Newline()
	//}
	//r.needRenderFootnotesDef = false
	//r.renderFooter()
	return nil
}

func (r *DocxRenderer) renderFootnotesRef(node *ast.Node, entering bool) ast.WalkStatus {
	//x := r.pdf.GetX() + 1
	//r.pdf.SetX(x)
	//y := r.pdf.GetY()
	//r.pdf.SetFont("regular", "R", 8)
	//r.pdf.SetTextColor(66, 133, 244)
	//
	//idx := string(node.Tokens)
	//width, _ := r.pdf.MeasureTextWidth(idx[1:])
	//r.pdf.SetY(y - 4)
	//r.pdf.Cell(nil, idx[1:])
	//
	//x += width
	//r.pdf.SetX(x)
	//r.pdf.SetY(y)
	//font := r.peekFont()
	//r.pdf.SetFont(font.family, font.style, font.size)
	//textColor := r.peekTextColor()
	//r.pdf.SetTextColor(textColor.R, textColor.G, textColor.B)
	return ast.WalkContinue
}

func (r *DocxRenderer) renderFootnotesDef(node *ast.Node, entering bool) ast.WalkStatus {
	if !r.needRenderFootnotesDef {
		return ast.WalkContinue
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) renderCodeBlock(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		if !node.IsFencedCodeBlock {
			// 缩进代码块处理
			r.renderCodeBlockLike(node.Tokens)
			return ast.WalkContinue
		}
	}
	return ast.WalkContinue
}

// renderCodeBlockCode 进行代码块 HTML 渲染，实现语法高亮。
func (r *DocxRenderer) renderCodeBlockCode(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		r.renderCodeBlockLike(node.Tokens)
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) renderCodeBlockLike(content []byte) {
	para := r.doc.AddParagraph()
	r.setParagraphSpacing(&para)
	r.pushPara(&para)
	run := para.AddRun()
	run.Properties().SetStyle("CodeBlock")
	r.pushRun(&run)
	r.WriteString(util.BytesToStr(content))
	run.AddBreak()
	r.popRun()
	r.reRun()
	r.popPara()
}

func (r *DocxRenderer) renderCodeSpanLike(content []byte) {
	para := r.peekPara()
	if para != nil {
		run := r.peekRun()
		if run != nil {
			run.AddText(util.BytesToStr(content))
		} else {
			run := para.AddRun()
			run.Properties().SetStyle("Code")
			r.pushRun(&run)
			r.WriteString(util.BytesToStr(content))
			r.popRun()
			r.reRun()
		}
	}

}

func (r *DocxRenderer) reRun() {
	if nil != r.peekRun() {
		// 如果链接之前有输出的话需要先结束掉，然后重新开一个
		r.popRun()
		para := r.peekPara()
		run := para.AddRun()
		r.pushRun(&run)
	}
}

func (r *DocxRenderer) renderCodeBlockCloseMarker(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderCodeBlockInfoMarker(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderCodeBlockOpenMarker(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderEmojiAlias(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderEmojiImg(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderEmojiUnicode(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderEmoji(node *ast.Node, entering bool) ast.WalkStatus {
	// 暂不渲染 Emoji，字体似乎有问题
	return ast.WalkContinue
}

func (r *DocxRenderer) renderInlineMathCloseMarker(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderInlineMathContent(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		// 尝试使用 OMML 渲染数学公式
		if err := r.renderMathAsOMML(node.Tokens, true); err != nil {
			log.Printf("OMML rendering failed, fallback to image: %s", err)
			// 回退到图片渲染
			r.renderMathAsImage(node.Tokens, true)
		}
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) renderInlineMathOpenMarker(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderInlineMath(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderMathBlockCloseMarker(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderMathBlockContent(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		// 尝试使用 OMML 渲染数学公式
		if err := r.renderMathAsOMML(node.Tokens, false); err != nil {
			log.Printf("OMML rendering failed, fallback to image: %s", err)
			// 回退到图片渲染
			r.renderMathAsImage(node.Tokens, false)
		}
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) renderMathBlockOpenMarker(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderMathBlock(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderTableCell(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		if r.currentRow != nil {
			// 添加新单元格
			cell := (*r.currentRow).AddCell()
			r.currentCell = &cell

			// 创建单元格段落
			para := cell.AddParagraph()
			// r.pushPara(&para)
			// 设置单元格垂直对齐 - 居中
			cellProps := cell.Properties()
			cellProps.SetVerticalAlignment(wml.ST_VerticalJcCenter)
			cellProps.Borders().SetAll(wml.ST_BorderSingle, color.FromHex(r.config.Table.BorderColor), r.config.Table.BorderWidth)

			// 强制设置单元格内容水平居中对齐
			para.Properties().SetAlignment(wml.ST_JcCenter)

			// 为单元格创建run来承载文本内容
			run := para.AddRun()

			// 设置表格字体：仿宋10号
			runProps := run.Properties()
			runProps.SetFontFamily(r.config.Fonts.Content)
			runProps.SetSize(measurement.Point * measurement.Distance(r.config.Table.HeaderFontSize))

			// 检查是否是表头单元格
			if r.isInTableHead(node) {
				// 表头样式：加粗
				runProps.SetBold(true)

				// 设置表头背景灰色
				cellProps := cell.Properties()
				// 创建灰色
				grayColor := color.RGB(217, 217, 217) // 浅灰色 #D9D9D9
				cellProps.SetShading(wml.ST_ShdClear, grayColor, grayColor)
			}
			// r.pushRun(&run)
			// r.pushPara(&para)
			// r.pushPara(&para)
			run.AddText(node.Text())
		}
	} else {
		if r.currentCell != nil {
			// 所有单元格都需要弹出run，因为进入时都创建了run
			r.currentCell = nil
		}
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) tableCols(cell *ast.Node) int {
	for parent := cell.Parent; nil != parent; parent = parent.Parent {
		if nil != parent.TableAligns {
			return len(parent.TableAligns)
		}
	}
	return 0
}

// isInTableHead 判断节点是否在表头中
func (r *DocxRenderer) isInTableHead(node *ast.Node) bool {
	for parent := node.Parent; nil != parent; parent = parent.Parent {
		if parent.Type == ast.NodeTableHead {
			return true
		}
		if parent.Type == ast.NodeTable {
			break
		}
	}
	return false
}

func (r *DocxRenderer) renderTableRow(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		if r.currentTable != nil {
			// 添加新行
			row := (*r.currentTable).AddRow()
			r.currentRow = &row
			r.currentCell = nil
		}
	} else {
		// 行结束
		r.currentRow = nil
		r.currentCell = nil
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) renderTableHead(node *ast.Node, entering bool) ast.WalkStatus {
	// 表头和普通行的处理逻辑相同，由单元格来区分样式
	return ast.WalkContinue
}

func (r *DocxRenderer) renderTable(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		// 计算表格列数
		cols := r.tableCols(node)
		if cols <= 0 {
			cols = 1 // 至少一列
		}

		// 创建一个新的表格
		table := r.doc.AddTable()
		table.Properties().SetWidthPercent(100) // 表格宽度100%
		table.Properties().SetAlignment(wml.ST_JcTableCenter)

		borders := table.Properties().Borders()
		borders.SetAll(wml.ST_BorderSingle, color.FromHex(r.config.Table.BorderColor), r.config.Table.BorderWidth)

		// 将表格存储到当前段落栈中以供子节点使用
		r.currentTable = &table
		r.currentRow = nil
		r.currentCell = nil
	} else {
		// 表格结束，清理状态
		r.currentTable = nil
		r.currentRow = nil
		r.currentCell = nil
		r.addTableAfterSpacing()
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) addTableAfterSpacing() {
	if r.config.Table.AfterLineSpacing <= 0 {
		return
	}
	para := r.doc.AddParagraph()
	para.Properties().Spacing().SetLineSpacing(r.config.Table.AfterLineSpacing, wml.ST_LineSpacingRuleAuto)
}

func (r *DocxRenderer) renderStrikethrough(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		para := r.peekPara()
		if para != nil {
			run := para.AddRun()
			r.pushRun(&run)
			run.Properties().SetStrikeThrough(true)
		}
	} else {
		r.popRun()
		r.reRun()
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) renderStrikethrough1OpenMarker(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderStrikethrough1CloseMarker(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderStrikethrough2OpenMarker(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderStrikethrough2CloseMarker(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderLinkTitle(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderLinkDest(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderLinkSpace(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderLinkText(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		if ast.NodeImage != node.Parent.Type {
			r.Write(node.Tokens)
		}
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) renderCloseParen(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderOpenParen(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderCloseBracket(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderOpenBracket(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderBang(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderImage(node *ast.Node, entering bool) ast.WalkStatus {
	// return ast.WalkContinue
	if entering {
		if 0 == r.DisableTags {
			destTokens := node.ChildByType(ast.NodeLinkDest).Tokens
			src := util.BytesToStr(destTokens)
			src, ok, isTemp := r.DownloadImg(src)
			if ok {
				// 检查是否为SVG文件，如果是则转换为PNG
				imgPath := src
				if r.IsSVG(src) {
					pngPath, err := r.ConvertSVGToPNG(src)
					if err != nil {
						log.Printf("Failed to convert SVG to PNG [%s]: %s", src, err)
						r.DisableTags++
						return ast.WalkContinue
					}
					imgPath = pngPath
				}
				img, err := media.ImageFromFile(imgPath)
				if err != nil {
					log.Printf("Failed to load image [%s]: %s", imgPath, err)
					r.DisableTags++
					return ast.WalkContinue
				}

				imgRef, err := r.doc.AddImage(img)
				if err != nil {
					log.Printf("Failed to add image to document [%s]: %s", imgPath, err)
					r.DisableTags++
					return ast.WalkContinue
				}

				// 为图片创建新的段落并设置居中对齐
				imgPara := r.doc.AddParagraph()
				imgParaProps := imgPara.Properties()
				imgParaProps.SetAlignment(r.config.Image.Alignment)

				// 设置段落间距
				spacing := imgParaProps.Spacing()
				spacing.SetBefore(r.config.Image.SpacingBefore)
				spacing.SetAfter(r.config.Image.SpacingAfter)

				// 在居中的段落中添加图片
				imgRun := imgPara.AddRun()
				inline, err := imgRun.AddDrawingInline(imgRef)
				if err != nil {
					log.Printf("Failed to add drawing inline [%s]: %s", imgPath, err)
					r.DisableTags++
					return ast.WalkContinue
				}

				width, height := r.getImgSize(imgPath)
				inline.SetSize(measurement.Distance(width), measurement.Distance(height))

				if isTemp {
					r.files = append(r.files, imgPath)
				}

				log.Printf("Successfully rendered centered image: %s (size: %.0fx%.0f)", imgPath, width, height)
			}
		}
		r.DisableTags++
		return ast.WalkContinue
	}

	r.DisableTags--
	if 0 == r.DisableTags {
		//r.WriteString("\"")
		//if title := node.ChildByType(ast.NodeLinkTitle); nil != title && nil != title.Tokens {
		//	r.WriteString(" title=\"")
		//	r.Write(title.Tokens)
		//	r.WriteString("\"")
		//}
		//r.WriteString(" />")
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) renderLink(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		dest := node.ChildByType(ast.NodeLinkDest)
		destTokens := dest.Tokens
		destTokens = r.RelativePath(destTokens)
		para := r.peekPara()
		if para != nil {
			link := para.AddHyperLink()
			link.SetTarget(util.BytesToStr(destTokens))
			run := link.AddRun()
			run.Properties().SetStyle("Hyperlink")
			r.pushRun(&run)
		}

	} else {
		r.popRun()
		r.reRun()
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) renderHTML(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		content := string(node.Tokens)
		nodeType := docxcommon.GetNodeValue(content, "s-tag", "type")
		if plugin := r.htmlBlockPlugins[nodeType]; plugin != nil {
			if err := plugin.RenderHTML(r, content); err != nil {
				log.Printf("渲染 HTML 插件 %q 失败: %v", nodeType, err)
			}
			return ast.WalkContinue
		}
		r.renderCodeBlockLike(node.Tokens)
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) renderInlineHTML(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		r.renderCodeSpanLike(node.Tokens)
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) renderDocument(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) Save(docxPath string) error {
	// 在保存前，为所有section设置页头和页脚
	r.applyHeaderFooterToAllSections()

	err := r.doc.SaveToFile(docxPath)
	for _, file := range r.files {
		os.Remove(file)
	}
	return err
}

// applyHeaderFooterToAllSections 为文档中的所有section设置页头和页脚
func (r *DocxRenderer) applyHeaderFooterToAllSections() {
	// 为BodySection设置页头和页脚（适用于没有分节符的文档或第一个section）
	bodySection := r.doc.BodySection()
	bodySection.SetHeader(r.header, wml.ST_HdrFtrDefault)
	bodySection.SetFooter(r.footer, wml.ST_HdrFtrDefault)

	// 遍历所有段落，找到有section属性的段落，并为它们设置页头和页脚
	paragraphs := r.doc.Paragraphs()
	for _, para := range paragraphs {
		paraProps := para.Properties()
		sectPr := paraProps.X().SectPr
		// 检查段落是否有section属性
		if sectPr != nil {
			// 检查是否已经有页头和页脚引用
			hasHeader := false
			hasFooter := false
			for _, ref := range sectPr.EG_HdrFtrReferences {
				if ref.HeaderReference != nil && ref.HeaderReference.TypeAttr == wml.ST_HdrFtrDefault {
					hasHeader = true
				}
				if ref.FooterReference != nil && ref.FooterReference.TypeAttr == wml.ST_HdrFtrDefault {
					hasFooter = true
				}
			}

			// 如果还没有设置，使用AddSection获取Section对象并设置
			// 注意：AddSection会创建新的SectPr，但我们可以通过临时段落来获取Section对象
			if !hasHeader || !hasFooter {
				// 创建一个临时段落来获取Section对象
				tempPara := r.doc.AddParagraph()
				tempSection := tempPara.Properties().AddSection(wml.ST_SectionMarkUnset)

				// 设置页头和页脚
				if !hasHeader {
					tempSection.SetHeader(r.header, wml.ST_HdrFtrDefault)
				}
				if !hasFooter {
					tempSection.SetFooter(r.footer, wml.ST_HdrFtrDefault)
				}

				// 获取新添加的header/footer引用
				tempSectPr := tempPara.Properties().X().SectPr
				var newHeaderRef, newFooterRef *wml.EG_HdrFtrReferences
				for _, ref := range tempSectPr.EG_HdrFtrReferences {
					if ref.HeaderReference != nil && ref.HeaderReference.TypeAttr == wml.ST_HdrFtrDefault {
						newHeaderRef = ref
					}
					if ref.FooterReference != nil && ref.FooterReference.TypeAttr == wml.ST_HdrFtrDefault {
						newFooterRef = ref
					}
				}

				// 将新添加的引用复制到原始的sectPr中
				if !hasHeader && newHeaderRef != nil {
					sectPr.EG_HdrFtrReferences = append(sectPr.EG_HdrFtrReferences, newHeaderRef)
				}
				if !hasFooter && newFooterRef != nil {
					sectPr.EG_HdrFtrReferences = append(sectPr.EG_HdrFtrReferences, newFooterRef)
				}

				// 删除临时段落
				r.doc.RemoveParagraph(tempPara)
			}
		}
	}
}

func (r *DocxRenderer) renderParagraph(node *ast.Node, entering bool) ast.WalkStatus {
	inList := false
	grandparent := node.Parent.Parent
	inTightList := false
	if nil != grandparent && ast.NodeList == grandparent.Type {
		inList = true
		inTightList = grandparent.ListData.Tight
	}

	if inTightList {
		if entering {
			para := r.peekPara()
			if para != nil {
				run := para.AddRun()
				r.pushRun(&run)
			}
		} else {
			r.popRun()
		}
		return ast.WalkContinue
	}

	isFirstParaInList := false
	if inList {
		isFirstParaInList = node.Parent.FirstChild == node
	}

	if entering {
		if !inList {
			para := r.doc.AddParagraph()
			r.setFirstLineIndent(&para)
			r.setParagraphSpacing(&para)
			r.pushPara(&para)
			run := para.AddRun()
			r.pushRun(&run)
		} else {
			if inTightList {
				para := r.peekPara()
				if para != nil {
					run := para.AddRun()
					r.pushRun(&run)
				}
			} else {
				if isFirstParaInList {
					para := r.peekPara()
					if para != nil {
						run := para.AddRun()
						r.pushRun(&run)
					}
				} else {
					para := r.doc.AddParagraph()
					r.setFirstLineIndent(&para)
					r.setParagraphSpacing(&para)
					r.pushPara(&para)
					run := para.AddRun()
					r.pushRun(&run)
				}
			}
		}
	} else {
		if !inList {
			r.peekRun()
			r.popRun()
			r.popPara()
		} else {
			if inTightList {
				r.popRun()
			} else {
				if isFirstParaInList {
					r.popRun()
				} else {
					r.popRun()
					r.popPara()
				}
			}
		}
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) renderText(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		r.WriteString(node.Text())
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) renderCodeSpan(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderCodeSpanOpenMarker(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderCodeSpanContent(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		r.renderCodeSpanLike(node.Tokens)
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) renderCodeSpanCloseMarker(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderEmphasis(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		para := r.peekPara()
		if para != nil {
			run := para.AddRun()
			r.pushRun(&run)
			run.Properties().SetItalic(true)
		}
	} else {
		r.popRun()
		r.reRun()
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) renderEmAsteriskOpenMarker(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderEmAsteriskCloseMarker(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderEmUnderscoreOpenMarker(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderEmUnderscoreCloseMarker(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderStrong(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		para := r.peekPara()
		if para != nil {
			run := para.AddRun()
			r.pushRun(&run)
			run.Properties().SetBold(true)
		}
	} else {
		r.popRun()
		r.reRun()
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) renderStrongA6kOpenMarker(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderStrongA6kCloseMarker(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderStrongU8eOpenMarker(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderStrongU8eCloseMarker(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderBlockquote(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderBlockquoteMarker(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderHeading(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		para := r.doc.AddParagraph()
		r.setParagraphSpacing(&para)

		// 为标题添加书签
		headingText := node.Text()
		if bookmarkID, exists := r.headingBookmarks[headingText]; exists {
			// 添加书签
			para.AddBookmark(bookmarkID)
		}

		// 添加标题内容
		run := para.AddRun()
		switch node.HeadingLevel {
		case 1:
			if r.headingStyle == 1 {
				// 设置一级标题居中对齐
				para.Properties().SetAlignment(r.config.Heading.LevelOneAlignment)
				lineHeight := float64(r.config.Text.ContentSize) * r.config.Paragraph.LineHeightMultiplier
				para.Properties().Spacing().SetLineSpacing(measurement.Distance(lineHeight), wml.ST_LineSpacingRuleAuto)
				props := run.Properties()
				r.SetTitleFont(&props, r.config.Fonts.Title)
				run.AddPageBreak()
			} else if r.headingStyle == 2 {
				para.Properties().SetAlignment(wml.ST_JcLeft)
				lineHeight := float64(r.config.Text.ContentSize) * r.config.Paragraph.LineHeightMultiplier
				para.Properties().Spacing().SetLineSpacing(measurement.Distance(lineHeight), wml.ST_LineSpacingRuleAuto)
				props := run.Properties()
				r.SetTitleFont(&props, r.config.Fonts.Title)
				run.AddPageBreak()
			}
		default:
			props := run.Properties()
			r.SetSubTitleFont(&props, r.config.Fonts.Title)
		}
		if r.headingStyle == 0 {
			props := run.Properties()
			r.SetSubTitleFont(&props, r.config.Fonts.Title)
			r.setFontSzie(&props, r.headingFontSize(node.HeadingLevel))
		}

		run.AddText(headingText)
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) headingFontSize(level int) int {
	if level <= 0 {
		return r.config.Text.SubTitleSize
	}
	index := level - 1
	if index >= len(r.config.Heading.Sizes) {
		return r.config.Heading.Sizes[len(r.config.Heading.Sizes)-1]
	}
	return r.config.Heading.Sizes[index]
}

func (r *DocxRenderer) renderHeadingC8hMarker(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderList(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		r.listStack = append(r.listStack, r.newListState(node))
	} else {
		if len(r.listStack) > 0 {
			r.listStack = r.listStack[:len(r.listStack)-1]
		}
		r.Newline()
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) renderListItem(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		paragraph := r.doc.AddParagraph()
		r.setParagraphSpacing(&paragraph)
		r.setListFirstLineIndent(&paragraph)
		r.pushPara(&paragraph)

		if definition := r.listDefinitionForItem(node); definition.NumberID() > 0 {
			paragraph.SetNumberingDefinition(definition)
		}
	} else {
		r.popPara()
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) newListState(node *ast.Node) listState {
	if node == nil || node.ListData == nil {
		return listState{}
	}
	if isUnorderedTaskList(node.ListData) {
		return listState{
			checkedDefinition:   r.newListDefinition(node.ListData, "☑"),
			uncheckedDefinition: r.newListDefinition(node.ListData, "□"),
		}
	}
	return listState{definition: r.newListDefinition(node.ListData, "")}
}

func (r *DocxRenderer) listDefinitionForItem(node *ast.Node) document.Definition {
	if len(r.listStack) == 0 || node == nil || node.ListData == nil {
		return document.Definition{}
	}
	state := r.listStack[len(r.listStack)-1]
	if isUnorderedTaskList(node.ListData) {
		if node.ListData.Checked {
			return state.checkedDefinition
		}
		return state.uncheckedDefinition
	}
	return state.definition
}

func (r *DocxRenderer) newListDefinition(data *ast.ListData, markerOverride string) document.Definition {
	definition := r.doc.Numbering.AddDefinition()
	level := definition.AddLevel()
	level.SetStart(listStart(data))
	level.SetSuffix(r.config.List.MarkerSuffix)

	switch {
	case isOrderedList(data):
		level.SetFormat(wml.ST_NumberFormatDecimal)
		level.RunProperties().SetSize(measurement.Distance(r.config.List.ThirdLevelFontSize))
		level.RunProperties().SetFontFamily(r.config.List.ThirdLevelFont)
		level.SetText(orderedListPattern(data))
	default:
		level.SetFormat(wml.ST_NumberFormatBullet)
		if isUnorderedTaskList(data) {
			level.RunProperties().SetSize(measurement.Distance(r.config.List.SecondLevelFontSize))
		} else {
			level.RunProperties().SetSize(measurement.Distance(r.config.List.FirstLevelFontSize))
		}
		if markerOverride != "" {
			level.SetText(markerOverride)
		} else {
			level.SetText("•")
		}
	}
	return definition
}

func isOrderedList(data *ast.ListData) bool {
	return data != nil && (data.Typ == 1 || (data.Typ == 3 && data.BulletChar == 0))
}

func isUnorderedTaskList(data *ast.ListData) bool {
	return data != nil && data.Typ == 3 && data.BulletChar != 0
}

func listStart(data *ast.ListData) int {
	if data == nil {
		return 1
	}
	if data.Start > 0 {
		return data.Start
	}
	if data.Num > 0 {
		return data.Num
	}
	return 1
}

func orderedListPattern(data *ast.ListData) string {
	delimiter := byte('.')
	if data != nil && data.Delimiter != 0 {
		delimiter = data.Delimiter
	}
	return "%1" + string(delimiter)
}

func (r *DocxRenderer) getNumberingLevel(nestedLevel int, num int) string {
	numList := make([]string, 0)
	point := ""
	switch nestedLevel {
	case 0:
		numList = []string{"（一）", "（二）", "（三）", "（四）", "（五）", "（六）", "（七）", "（八）", "（九）", "（十）", "（十一）", "（十二）", "（十三）", "（十四）", "（十五）", "（十六）", "（十七）", "（十八）", "（十九）", "（二十）", "（二十一）", "（二十二）", "（二十三）", "（二十四）", "（二十五）"}
	case 1:
		numList = []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13", "14", "15", "16", "17", "18", "19", "20", "21", "22", "23", "24", "25"}
		point = "."
	case 2:
		numList = []string{"（1）", "（2）", "（3）", "（4）", "（5）", "（6）", "（7）", "（8）", "（9）", "（10）", "（11）", "（12）", "（13）", "（14）", "（15）", "（16）", "（17）", "（18）", "（19）", "（20）", "（21）", "（22）", "（23）", "（24）", "（25）"}
	case 3:
		numList = []string{"1）", "2）", "3）", "4）", "5）", "6）", "7）", "8）", "9）", "10）", "11）", "12）", "13）", "14）", "15）", "16）", "17）", "18）", "19）", "20）", "21）", "22）", "23）", "24）", "25）"}
	case 4:
		numList = []string{"①", "②", "③", "④", "⑤", "⑥", "⑦", "⑧", "⑨", "⑩", "⑪", "⑫", "⑬", "⑭", "⑮", "⑯", "⑰", "⑱", "⑲", "⑳", "㉑", "㉒", "㉓", "㉔", "㉕"}
	case 5:
		numList = []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}
	case 6:
		numList = []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z"}
	}
	return numList[num-1] + point
}

func (r *DocxRenderer) renderTaskListItemMarker(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		var attrs [][]string
		if node.TaskListItemChecked {
			attrs = append(attrs, []string{"checked", ""})
		}
		attrs = append(attrs, []string{"disabled", ""}, []string{"type", "checkbox"})
		//r.tag("input", attrs, true)
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) renderThematicBreak(node *ast.Node, entering bool) ast.WalkStatus {
	//r.Newline()
	//r.pdf.SetY(r.pdf.GetY() + 14)
	//r.pdf.SetStrokeColor(106, 115, 125)
	//r.pdf.SetY(r.pdf.GetY() + 12)
	//r.pdf.SetStrokeColor(0, 0, 0)
	//r.Newline()
	return ast.WalkContinue
}

func (r *DocxRenderer) renderHardBreak(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		r.NewBreak()
	} else {
		// fmt.Println("hard break2")
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) renderSoftBreak(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		r.NewBreak()
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) pushPara(para *document.Paragraph) {
	r.paragraphs = append(r.paragraphs, para)
}

func (r *DocxRenderer) popPara() *document.Paragraph {
	ret := r.paragraphs[len(r.paragraphs)-1]
	r.paragraphs = r.paragraphs[:len(r.paragraphs)-1]
	return ret
}

func (r *DocxRenderer) peekPara() *document.Paragraph {
	if len(r.paragraphs) == 0 {
		return nil
	}
	return r.paragraphs[len(r.paragraphs)-1]
}

func (r *DocxRenderer) pushRun(run *document.Run) {
	r.runs = append(r.runs, run)
}

func (r *DocxRenderer) popRun() *document.Run {
	if len(r.runs) == 0 {
		return nil
	}
	ret := r.runs[len(r.runs)-1]
	r.runs = r.runs[:len(r.runs)-1]
	return ret
}

func (r *DocxRenderer) peekRun() *document.Run {
	if 1 > len(r.runs) {
		return nil
	}

	return r.runs[len(r.runs)-1]
}

func (r *DocxRenderer) countParentContainerBlocks(n *ast.Node) (ret int) {
	for parent := n.Parent; nil != parent; parent = parent.Parent {
		if ast.NodeBlockquote == parent.Type || ast.NodeList == parent.Type {
			ret++
		}
	}
	return
}

// WriteByte 输出一个字节 c。
func (r *DocxRenderer) WriteByte(c byte) {
	r.WriteString(string(c))
}

// Write 输出指定的字节数组 content。
func (r *DocxRenderer) Write(content []byte) {
	r.WriteString(util.BytesToStr(content))
}

// WriteString 输出指定的字符串 content。
func (r *DocxRenderer) WriteString(content string) {
	if length := len(content); 0 < length {
		run := r.peekRun()
		if run != nil {
			props := run.Properties()
			r.SetContentFont(&props, r.config.Fonts.Content)
			run.AddText(content)
		}
		r.LastOut = content[length-1]
	}
}

// Newline 会在最新内容不是换行符 \n 时输出一个换行符。
func (r *DocxRenderer) Newline() {
	r.WriteString("\n\r")
}

func (r *DocxRenderer) NewBreak() {
	if r.peekPara() != nil {
		r.popPara()
	}
	if r.peekRun() != nil {
		r.popRun()
	}
	para := r.doc.AddParagraph()
	r.setParagraphSpacing(&para)
	r.setFirstLineIndent(&para)
	run := para.AddRun()
	r.pushRun(&run)
	r.pushPara(&para)
}

func (r *DocxRenderer) DownloadImg(src string) (localPath string, ok, isTemp bool) {
	// 检查是否为base64编码的图片
	if r.isBase64Image(src) {
		data, fileExt, err := r.parseBase64Image(src)
		if err != nil {
			log.Printf("Failed to parse base64 image: %s", err)
			return src, false, false
		}

		// 创建临时文件
		tempFile, err := os.CreateTemp("", "lute-docx-base64*"+fileExt)
		if err != nil {
			log.Printf("Failed to create temp file for base64 image: %s", err)
			return src, false, false
		}

		// 写入解码后的数据
		_, err = tempFile.Write(data)
		if err != nil {
			log.Printf("Failed to write base64 image data: %s", err)
			tempFile.Close()
			os.Remove(tempFile.Name())
			return src, false, false
		}

		tempFile.Close()
		log.Printf("Base64 image saved to temporary file: %s", tempFile.Name())
		return tempFile.Name(), true, true
	}

	if strings.HasPrefix(src, "//") {
		src = "https:" + src
	}

	u, err := url.Parse(src)
	if nil != err {
		log.Printf("image src [%s] is not an valid URL, treat it as local path", src)
		return src, true, false
	}

	if !strings.HasPrefix(u.Scheme, "http") {
		log.Printf("image src [%s] scheme is not [http] or [https], treat it as local path", src)
		return src, true, false
	}

	u, _ = url.Parse(src)

	client := http.Client{
		Timeout: 5 * time.Second,
	}
	req := &http.Request{
		URL: u,
	}
	resp, err := client.Do(req)
	if nil != err {
		log.Printf("download image [%s] failed: %s", src, err)
		return src, false, false
	}
	defer resp.Body.Close()
	if 200 != resp.StatusCode {
		log.Printf("download image [%s] failed, status code is [%d]", src, resp.StatusCode)
		return src, false, false
	}

	data, err := io.ReadAll(resp.Body)
	file, err := os.CreateTemp("", "lute-docx.img.*")
	if nil != err {
		log.Printf("create temp image [%s] failed: %s", src, err)
		return src, false, false
	}
	_, err = file.Write(data)
	if nil != err {
		log.Printf("write temp image [%s] failed: %s", src, err)
		return src, false, false
	}
	file.Close()
	return file.Name(), true, true
}

// getDocumentContentWidth 计算文档的有效内容宽度（像素）
func (r *DocxRenderer) getDocumentContentWidth() float64 {
	// A4纸张标准尺寸: 210mm 宽度
	// 96 DPI 下转换为像素: 210mm * 96 / 25.4 ≈ 794 像素
	pageWidthPx := 210.0 * 96 / 25.4

	// 默认页边距 (通常左右各 25mm)
	marginPx := r.margin * 96 / 25.4 // 将 mm 转换为像素
	if marginPx == 0 {
		marginPx = 25.0 * 96 / 25.4 // 默认 25mm 边距
	}

	// 计算有效内容宽度 (页面宽度 - 左右边距)
	contentWidth := pageWidthPx - (marginPx * 2)

	// 确保有一个合理的最小宽度
	if contentWidth < 300 {
		contentWidth = 300
	}

	log.Printf("Document content width: %.2f px (page: %.2f, margin: %.2f)", contentWidth, pageWidthPx, marginPx)
	return contentWidth
}

// constrainImageSize 限制图片尺寸不超过文档宽度，保持宽高比
func (r *DocxRenderer) constrainImageSize(width, height float64) (float64, float64) {
	maxWidth := r.getDocumentContentWidth()

	// 如果图片宽度不超过限制，直接返回
	if width <= maxWidth {
		return width, height
	}

	// 计算缩放比例，保持宽高比
	scale := maxWidth / width
	newWidth := maxWidth
	newHeight := height * scale

	log.Printf("Image resized from %.2fx%.2f to %.2fx%.2f (scale: %.3f)",
		width, height, newWidth, newHeight, scale)

	return newWidth, newHeight
}

func (r *DocxRenderer) getImgSize(imgPath string) (width, height float64) {
	file, err := os.Open(imgPath)
	if nil != err {
		log.Printf("failed to open image file [%s]: %s", imgPath, err)
		return 400, 300 // 返回默认尺寸而不是Fatal
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if nil != err {
		log.Printf("failed to decode image file [%s]: %s", imgPath, err)
		return 400, 300 // 返回默认尺寸而不是Fatal
	}

	imageRect := img.Bounds()
	k := 1
	w := -128
	h := -128
	if w < 0 {
		w = -imageRect.Dx() * 72 / w / k
	}
	if h < 0 {
		h = -imageRect.Dy() * 72 / h / k
	}
	if w == 0 {
		w = h * imageRect.Dx() / imageRect.Dy()
	}
	if h == 0 {
		h = w * imageRect.Dy() / imageRect.Dx()
	}

	width = float64(w)
	height = float64(h)

	// 应用文档宽度限制
	return r.constrainImageSize(width, height)
}

func (r *DocxRenderer) SetDefaultFontFamily(props *document.RunProperties, fontFamily string) {
	fonts := props.Fonts()
	fontsX := fonts.X()
	times := r.config.Fonts.Latin
	fontsX.AsciiAttr = &times
	fontsX.HAnsiAttr = &times
	fontsX.EastAsiaAttr = &fontFamily
	fontsX.CsAttr = &times
}

// setMixedFont 设置混合字体：中文仿宋，数字Times New Roman
func (r *DocxRenderer) setFontFamily(props *document.RunProperties, fontFamily string) {
	if fontFamily == "" {
		fontFamily = r.config.Fonts.Content
	}
	r.SetDefaultFontFamily(props, fontFamily)
}

func (r *DocxRenderer) setFontSzie(props *document.RunProperties, fontSize int) {
	props.SetSize(measurement.Distance(fontSize))
}

// setFirstLineIndent 设置段落首行缩进
func (r *DocxRenderer) setFirstLineIndent(para *document.Paragraph) {
	props := para.Properties()
	props.SetFirstLineIndent(r.config.Paragraph.FirstLineIndent)
}

// setListFirstLineIndent 设置列表段落首行缩进
func (r *DocxRenderer) setListFirstLineIndent(para *document.Paragraph) {
	props := para.Properties()
	props.SetFirstLineIndent(r.config.List.FirstLineIndent)
}

// setParagraphSpacing 设置段落行距
func (r *DocxRenderer) setParagraphSpacing(para *document.Paragraph) {
	props := para.Properties()
	spacing := props.Spacing()
	lineHeight := float64(r.config.Text.ContentSize) * r.config.Paragraph.LineHeightMultiplier
	spacing.SetLineSpacing(measurement.Distance(lineHeight), wml.ST_LineSpacingRuleAuto)

	// 设置段落两端对齐
	props.SetAlignment(r.config.Paragraph.Alignment)
}

func (r *DocxRenderer) SetContentFont(props *document.RunProperties, fontFamily string) {
	r.setFontFamily(props, fontFamily)
	r.setFontSzie(props, r.config.Text.ContentSize)
}

func (r *DocxRenderer) SetSubTitleFont(props *document.RunProperties, fontFamily string) {
	r.setFontFamily(props, fontFamily)
	r.setFontSzie(props, r.config.Text.SubTitleSize)
}

func (r *DocxRenderer) SetTitleFont(props *document.RunProperties, fontFamily string) {
	r.setFontFamily(props, fontFamily)
	r.setFontSzie(props, r.config.Text.TitleSize)
}

func (r *DocxRenderer) SetNameFont(para *document.Paragraph, props *document.RunProperties, fontFamily string) {
	// 设置段落居中对齐
	paraProps := para.Properties()
	paraProps.SetAlignment(wml.ST_JcCenter)

	// 设置字体
	r.setFontFamily(props, fontFamily)
	r.setFontSzie(props, 36)
}

// isSVG 检查文件是否为SVG格式
func (r *DocxRenderer) IsSVG(imgPath string) bool {
	// 检查文件扩展名
	ext := strings.ToLower(filepath.Ext(imgPath))
	if ext == ".svg" {
		return true
	}

	// 检查文件内容
	data, err := ioutil.ReadFile(imgPath)
	if err != nil {
		return false
	}

	// 检查是否包含SVG标签
	content := string(data)
	return strings.Contains(content, "<svg") || strings.Contains(content, "<?xml")
}

// getSVGSize 从SVG文件中解析尺寸信息，支持本地文件和远程URL
func (r *DocxRenderer) getSVGSize(imgPath string) (width, height float64) {
	var content string

	// 本地文件处理
	data, err := ioutil.ReadFile(imgPath)
	if err != nil {
		log.Printf("failed to read local SVG file [%s]: %s", imgPath, err)
		return 400, 300 // 默认尺寸
	}
	content = string(data)

	// 尝试从SVG标签中提取width和height属性
	svgPattern := `<svg[^>]*(?:width\s*=\s*["']?([^"'\s>]+)["']?)[^>]*(?:height\s*=\s*["']?([^"'\s>]+)["']?)[^>]*>`
	re := regexp.MustCompile(svgPattern)
	matches := re.FindStringSubmatch(content)

	if len(matches) >= 3 {
		widthStr := strings.TrimSuffix(matches[1], "px")
		heightStr := strings.TrimSuffix(matches[2], "px")

		if w, err := strconv.ParseFloat(widthStr, 64); err == nil {
			width = w
			log.Printf("Extracted width from SVG: %s -> %.2f", widthStr, width)
		}
		if h, err := strconv.ParseFloat(heightStr, 64); err == nil {
			height = h
			log.Printf("Extracted height from SVG: %s -> %.2f", heightStr, height)
		}
	}

	// 如果没有找到尺寸，尝试从viewBox中提取
	if width == 0 || height == 0 {
		viewBoxPattern := `viewBox\s*=\s*["']?[^"'\s]*\s+[^"'\s]*\s+([^"'\s]+)\s+([^"'\s]+)["']?`
		re := regexp.MustCompile(viewBoxPattern)
		matches := re.FindStringSubmatch(content)

		if len(matches) >= 3 {
			if w, err := strconv.ParseFloat(matches[1], 64); err == nil && width == 0 {
				width = w
				log.Printf("Extracted width from viewBox: %.2f", width)
			}
			if h, err := strconv.ParseFloat(matches[2], 64); err == nil && height == 0 {
				height = h
				log.Printf("Extracted height from viewBox: %.2f", height)
			}
		}
	}

	// 如果仍然没有找到尺寸，使用默认值
	if width == 0 {
		width = 400
		log.Printf("Using default width: %.2f", width)
	}
	if height == 0 {
		height = 300
		log.Printf("Using default height: %.2f", height)
	}

	log.Printf("Final SVG size for [%s]: %.2f x %.2f", imgPath, width, height)
	return width, height
}

// ConvertSVGToPNG 将SVG文件转换为PNG格式，使用rsvg-convert命令行工具
func (r *DocxRenderer) ConvertSVGToPNG(svgPath string) (pngPath string, err error) {
	// 创建临时PNG文件
	pngFile, err := os.CreateTemp("", "lute-docx-svg-*.png")
	if err != nil {
		return "", err
	}
	pngPath = pngFile.Name()
	pngFile.Close() // 关闭文件，让rsvg-convert可以写入

	// 获取SVG尺寸
	width, height := r.getSVGSize(svgPath)

	// 构建rsvg-convert命令
	// rsvg-convert -f png -o output.png -w 400 -h 300 input.svg
	cmd := exec.Command("rsvg-convert",
		"-f", "png",
		"-o", pngPath,
		"-w", strconv.Itoa(int(width)),
		"-h", strconv.Itoa(int(height)),
		svgPath)

	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("rsvg-convert failed: %s, output: %s", err, string(output))
		return "", fmt.Errorf("rsvg-convert failed: %s", err)
	}

	log.Printf("Successfully converted SVG to PNG: %s -> %s (size: %.0fx%.0f)",
		svgPath, pngPath, width, height)

	// 将生成的PNG添加到临时文件列表中
	r.files = append(r.files, pngPath)

	return pngPath, nil
}

// isBase64Image 检查是否为base64编码的图片
// 支持的格式示例:
// - data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAB...
// - data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQ...
// - data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAwIi...
func (r *DocxRenderer) isBase64Image(src string) bool {
	return strings.HasPrefix(src, "data:image/")
}

// parseBase64Image 解析base64编码的图片
func (r *DocxRenderer) parseBase64Image(src string) (data []byte, fileExt string, err error) {
	// 格式: data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAB...
	if !strings.HasPrefix(src, "data:image/") {
		return nil, "", fmt.Errorf("not a valid base64 image format")
	}

	// 查找base64数据的开始位置
	base64Index := strings.Index(src, "base64,")
	if base64Index == -1 {
		return nil, "", fmt.Errorf("no base64 data found")
	}

	// 提取MIME类型
	mimeType := src[5 : base64Index-1] // 去掉 "data:" 和 ";base64,"

	// 确定文件扩展名
	switch {
	case strings.Contains(mimeType, "png"):
		fileExt = ".png"
	case strings.Contains(mimeType, "jpeg") || strings.Contains(mimeType, "jpg"):
		fileExt = ".jpg"
	case strings.Contains(mimeType, "gif"):
		fileExt = ".gif"
	case strings.Contains(mimeType, "svg+xml"):
		fileExt = ".svg"
	case strings.Contains(mimeType, "webp"):
		fileExt = ".webp"
	case strings.Contains(mimeType, "bmp"):
		fileExt = ".bmp"
	default:
		fileExt = ".png" // 默认为PNG
		log.Printf("Unknown image MIME type [%s], defaulting to PNG", mimeType)
	}

	// 提取并解码base64数据
	base64Data := src[base64Index+7:] // 去掉 "base64,"
	data, err = base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode base64 data: %s", err)
	}

	log.Printf("Parsed base64 image: MIME type [%s], size [%d bytes], extension [%s]", mimeType, len(data), fileExt)
	return data, fileExt, nil
}

// renderMathAsOMML 将LaTeX数学公式转换为OMML格式并插入到文档中
func (r *DocxRenderer) renderMathAsOMML(mathContent []byte, isInline bool) error {
	latex := util.BytesToStr(mathContent)
	converter := docxmath.NewConverter()
	if isInline {
		// 行内公式
		oMath, err := converter.ConvertInline(latex)
		if err != nil {
			return fmt.Errorf("failed to convert inline math: %w", err)
		}

		// 将 OMML 插入到当前段落
		para := r.peekPara()
		if para == nil {
			return fmt.Errorf("no current paragraph for inline math")
		}

		return docxmath.InsertInlineMath(para, oMath)
	} else {
		// 块级公式
		oMathPara, err := converter.ConvertBlock(latex)
		if err != nil {
			return fmt.Errorf("failed to convert block math: %w", err)
		}

		// 创建居中段落并插入 OMML
		para, err := docxmath.InsertBlockMath(r.doc, oMathPara)
		if err != nil {
			return fmt.Errorf("failed to insert block math: %w", err)
		}

		// 设置段落居中对齐
		para.Properties().SetAlignment(r.config.Math.Alignment)

		// 设置段落间距
		spacing := para.Properties().Spacing()
		spacing.SetBefore(r.config.Math.SpacingBefore)
		spacing.SetAfter(r.config.Math.SpacingAfter)

		return nil
	}
}

// renderMathAsImage 将LaTeX数学公式转换为图片并插入到文档中
func (r *DocxRenderer) renderMathAsImage(mathContent []byte, isInline bool) {
	mathStr := util.BytesToStr(mathContent)

	// 生成图片
	imgPath, err := r.ConvertMathToImage(mathStr, isInline)
	if err != nil {
		log.Printf("Failed to convert math formula to image: %s", err)
		// 回退到文本渲染
		if isInline {
			r.renderCodeSpanLike(mathContent)
		} else {
			r.renderCodeBlockLike(mathContent)
		}
		return
	}

	// 加载图片
	img, err := media.ImageFromFile(imgPath)
	if err != nil {
		log.Printf("Failed to load math image [%s]: %s", imgPath, err)
		// 回退到文本渲染
		if isInline {
			r.renderCodeSpanLike(mathContent)
		} else {
			r.renderCodeBlockLike(mathContent)
		}
		return
	}

	// 添加图片到文档
	imgRef, err := r.doc.AddImage(img)
	if err != nil {
		log.Printf("Failed to add math image to document [%s]: %s", imgPath, err)
		// 回退到文本渲染
		if isInline {
			r.renderCodeSpanLike(mathContent)
		} else {
			r.renderCodeBlockLike(mathContent)
		}
		return
	}

	if isInline {
		// 行内公式 - 在当前run中添加，保持文本流的连续性
		run := r.peekRun()
		if run != nil {
			inline, err := run.AddDrawingInline(imgRef)
			if err != nil {
				log.Printf("Failed to add inline math image [%s]: %s", imgPath, err)
				r.renderCodeSpanLike(mathContent)
				return
			}

			// 设置适当的行内图片尺寸
			width, height := r.getMathImageSize(imgPath, true)
			inline.SetSize(measurement.Distance(width), measurement.Distance(height))
		} else {
			r.renderCodeSpanLike(mathContent)
			return
		}
	} else {
		// 块级公式 - 创建新段落并居中
		mathPara := r.doc.AddParagraph()
		mathParaProps := mathPara.Properties()
		mathParaProps.SetAlignment(r.config.Math.Alignment)

		// 设置段落间距
		spacing := mathParaProps.Spacing()
		spacing.SetBefore(r.config.Math.SpacingBefore)
		spacing.SetAfter(r.config.Math.SpacingAfter)

		// 在居中的段落中添加公式图片
		mathRun := mathPara.AddRun()
		inline, err := mathRun.AddDrawingInline(imgRef)
		if err != nil {
			log.Printf("Failed to add block math image [%s]: %s", imgPath, err)
			r.renderCodeBlockLike(mathContent)
			return
		}

		// 设置适当的块级图片尺寸
		width, height := r.getMathImageSize(imgPath, false)
		inline.SetSize(measurement.Distance(width), measurement.Distance(height))
	}

	// 添加到临时文件列表以便清理
	r.files = append(r.files, imgPath)

	log.Printf("Successfully rendered math formula as image: %s", imgPath)
}

// ConvertMathToImage 使用pdflatex和convert将LaTeX数学公式转换为PNG图片
func (r *DocxRenderer) ConvertMathToImage(mathFormula string, isInline bool) (imgPath string, err error) {
	// 创建临时目录
	tempDir, err := ioutil.TempDir("", "lute-math-")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %s", err)
	}
	defer os.RemoveAll(tempDir) // 清理临时目录

	// 创建LaTeX文档
	var latexContent string
	if isInline {
		latexContent = fmt.Sprintf(`\documentclass[12pt]{article}
\usepackage{amsmath}
\usepackage{amsfonts}
\usepackage{amssymb}
\usepackage[utf8]{inputenc}
\usepackage{xcolor}
\pagestyle{empty}
\begin{document}
\thispagestyle{empty}
$%s$
\end{document}`, mathFormula)
	} else {
		latexContent = fmt.Sprintf(`\documentclass[12pt]{article}
\usepackage{amsmath}
\usepackage{amsfonts}
\usepackage{amssymb}
\usepackage[utf8]{inputenc}
\usepackage{xcolor}
\pagestyle{empty}
\begin{document}
\thispagestyle{empty}
\[%s\]
\end{document}`, mathFormula)
	}

	// 写入LaTeX文件
	texFile := filepath.Join(tempDir, "formula.tex")
	err = ioutil.WriteFile(texFile, []byte(latexContent), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write LaTeX file: %s", err)
	}

	// 执行pdflatex生成PDF
	pdflatexCmd := exec.Command("pdflatex",
		"-interaction=nonstopmode",
		"-output-directory", tempDir,
		texFile)

	pdflatexOutput, err := pdflatexCmd.CombinedOutput()
	if err != nil {
		log.Printf("pdflatex output: %s", string(pdflatexOutput))
		return "", fmt.Errorf("pdflatex failed: %s", err)
	}

	pdfFile := filepath.Join(tempDir, "formula.pdf")
	if _, err := os.Stat(pdfFile); os.IsNotExist(err) {
		return "", fmt.Errorf("PDF file was not generated")
	}

	// 创建最终PNG文件
	pngFile, err := os.CreateTemp("", "lute-math-*.png")
	if err != nil {
		return "", fmt.Errorf("failed to create temp PNG file: %s", err)
	}
	imgPath = pngFile.Name()
	pngFile.Close()

	// 使用convert将PDF转换为PNG
	var density string
	if isInline {
		density = "200" // 行内公式使用较小密度
	} else {
		density = "300" // 块级公式使用较高密度
	}

	convertCmd := exec.Command("convert",
		"-density", density,
		"-quality", "100",
		"-colorspace", "RGB",
		"-background", "white",
		"-alpha", "remove",
		"-trim",
		"+repage",
		pdfFile,
		imgPath)

	convertOutput, err := convertCmd.CombinedOutput()
	if err != nil {
		log.Printf("convert output: %s", string(convertOutput))
		return "", fmt.Errorf("convert failed: %s", err)
	}

	// 验证PNG文件是否成功生成
	if _, err := os.Stat(imgPath); os.IsNotExist(err) {
		return "", fmt.Errorf("PNG file was not generated")
	}

	log.Printf("Successfully converted math formula to image: %s (inline: %v)", imgPath, isInline)
	return imgPath, nil
}

// getMathImageSize 获取数学公式图片的适当尺寸
func (r *DocxRenderer) getMathImageSize(imgPath string, isInline bool) (width, height float64) {
	// 获取原始图片尺寸
	origWidth, origHeight := r.getImgSize(imgPath)

	if isInline {
		// 行内公式：确保高度不超过行高
		maxHeight := float64(r.config.Text.ContentSize) * 1.2 // 稍微大于字体尺寸
		if origHeight > maxHeight {
			scale := maxHeight / origHeight
			width = origWidth * scale
			height = maxHeight
		} else {
			width = origWidth
			height = origHeight
		}
	} else {
		// 块级公式：限制最大宽度，保持宽高比
		maxWidth := r.getDocumentContentWidth() * 0.8 // 使用80%的文档宽度
		if origWidth > maxWidth {
			scale := maxWidth / origWidth
			width = maxWidth
			height = origHeight * scale
		} else {
			width = origWidth
			height = origHeight
		}
	}

	log.Printf("Math image size: original(%.0fx%.0f) -> final(%.0fx%.0f), inline: %v",
		origWidth, origHeight, width, height, isInline)

	return width, height
}
