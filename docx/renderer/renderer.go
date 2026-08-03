// Package renderer contains the Markdown-to-DOCX rendering implementation.
package renderer

import (
	"log"
	"os"

	"github.com/88250/lute/ast"
	"github.com/88250/lute/parse"
	"github.com/88250/lute/render"
	"github.com/88250/lute/util"
	"github.com/scoming-dev/tools/docx/document"
	"github.com/scoming-dev/tools/docx/schema/soo/wml"
)

// HeadingStyle limits the supported visual modes for Markdown heading rendering.
type HeadingStyle int

const (
	// HeadingStyleDefault renders headings in place with configured heading sizes.
	HeadingStyleDefault = iota
	// HeadingStyleLevelOneCenterPageBreak page-breaks before level-one headings and centers them.
	HeadingStyleLevelOneCenterPageBreak
	// HeadingStyleLevelOneLeftPageBreak page-breaks before level-one headings and left-aligns them.
	HeadingStyleLevelOneLeftPageBreak
)

// DocxRenderer converts a Lute Markdown tree into a DOCX document model.
type DocxRenderer struct {
	*render.BaseRenderer

	headingStyle HeadingStyle

	config     Config
	doc        *document.Document
	zoom       float64
	margin     float64
	paragraphs []*document.Paragraph
	runs       []*document.Run
	files      []string
	listStack  []listState

	currentTable *document.Table
	currentRow   *document.Row
	currentCell  *document.Cell

	headingBookmarks map[string]string

	header document.Header
	footer document.Footer

	htmlBlockPlugins map[string]HTMLBlockPlugin
	cover            CoverPlugin
}

// NewDocxRenderer 创建一个 DOCX 渲染器。
func NewDocxRenderer(tree *parse.Tree, options *render.Options, headingStyle int, rendererOptions ...RendererOption) *DocxRenderer {
	doc := document.New()
	ret := &DocxRenderer{
		BaseRenderer:     render.NewBaseRenderer(tree, options),
		doc:              doc,
		headingBookmarks: make(map[string]string),
		headingStyle:     normalizeHeadingStyle(headingStyle),
		config:           DefaultConfig(),
		htmlBlockPlugins: make(map[string]HTMLBlockPlugin),
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
	ret.RendererFuncs[ast.NodeFootnotesDef] = ret.renderNoop
	ret.RendererFuncs[ast.NodeFootnotesRef] = ret.renderNoop
	ret.RendererFuncs[ast.NodeToC] = ret.renderToC
	ret.RendererFuncs[ast.NodeBackslash] = ret.renderBackslash
	ret.RendererFuncs[ast.NodeBackslashContent] = ret.renderBackslashContent
	for _, option := range rendererOptions {
		if option != nil {
			option(ret)
		}
	}
	ret.margin = ret.config.Page.MarginMM * ret.zoom
	ret.setupHeaderFooter()
	if ret.cover != nil {
		if err := ret.cover.RenderCover(ret); err != nil {
			log.Printf("渲染封面失败: %v", err)
		}
	}
	return ret
}

func normalizeHeadingStyle(style int) HeadingStyle {
	switch style {
	case HeadingStyleLevelOneCenterPageBreak:
		return HeadingStyleLevelOneCenterPageBreak
	case HeadingStyleLevelOneLeftPageBreak:
		return HeadingStyleLevelOneLeftPageBreak
	default:
		return HeadingStyleDefault
	}
}

func (r *DocxRenderer) setupHeaderFooter() {
	r.header = r.doc.AddHeader()
	hdrPara := r.header.AddParagraph()
	hdrParaProps := hdrPara.Properties()
	hdrParaProps.SetAlignment(r.config.HeaderFooter.Alignment)

	hdrRun := hdrPara.AddRun()
	hdrRunProps := hdrRun.Properties()
	r.setFontFamily(&hdrRunProps, r.config.Fonts.Content)
	r.setFontSize(&hdrRunProps, r.config.HeaderFooter.FontSize)

	r.footer = r.doc.AddFooter()
	ftrPara := r.footer.AddParagraph()
	ftrParaProps := ftrPara.Properties()
	ftrParaProps.SetAlignment(r.config.HeaderFooter.Alignment)

	ftrRun := ftrPara.AddRun()
	ftrRunProps := ftrRun.Properties()
	r.setFontFamily(&ftrRunProps, r.config.Fonts.Content)
	r.setFontSize(&ftrRunProps, r.config.HeaderFooter.FontSize)

	ftrRun.AddText("第")
	ftrRun.AddField(document.FieldCurrentPage)
	ftrRun.AddText("页 共")
	ftrRun.AddField(document.FieldNumberOfPages)
	ftrRun.AddText("页")

	ftrRun.AddText(r.config.HeaderFooter.FooterSuffix)
}

// Render walks the Markdown AST and writes the rendered content into the document model.
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
	return
}

func (r *DocxRenderer) renderNoop(_ *ast.Node, _ bool) ast.WalkStatus {
	return ast.WalkContinue
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

func (r *DocxRenderer) renderCodeBlock(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		if !node.IsFencedCodeBlock {
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
		nodeType := getNodeValue(content, "s-tag", "type")
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

// Save writes the rendered document to a DOCX file and removes renderer-owned temporary files.
func (r *DocxRenderer) Save(docxPath string) error {
	r.applyHeaderFooterToAllSections()

	err := r.doc.SaveToFile(docxPath)
	for _, file := range r.files {
		_ = os.Remove(file)
	}
	return err
}

func (r *DocxRenderer) applyHeaderFooterToAllSections() {
	bodySection := r.doc.BodySection()
	bodySection.SetHeader(r.header, wml.ST_HdrFtrDefault)
	bodySection.SetFooter(r.footer, wml.ST_HdrFtrDefault)

	paragraphs := r.doc.Paragraphs()
	for _, para := range paragraphs {
		paraProps := para.Properties()
		sectPr := paraProps.X().SectPr
		if sectPr != nil {
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

			// The document compatibility layer exposes header/footer references through Section.
			// A temporary paragraph lets us copy only the missing references onto an existing SectPr.
			if !hasHeader || !hasFooter {
				tempPara := r.doc.AddParagraph()
				tempSection := tempPara.Properties().AddSection(wml.ST_SectionMarkUnset)

				if !hasHeader {
					tempSection.SetHeader(r.header, wml.ST_HdrFtrDefault)
				}
				if !hasFooter {
					tempSection.SetFooter(r.footer, wml.ST_HdrFtrDefault)
				}

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

				if !hasHeader && newHeaderRef != nil {
					sectPr.EG_HdrFtrReferences = append(sectPr.EG_HdrFtrReferences, newHeaderRef)
				}
				if !hasFooter && newFooterRef != nil {
					sectPr.EG_HdrFtrReferences = append(sectPr.EG_HdrFtrReferences, newFooterRef)
				}

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

func (r *DocxRenderer) renderHardBreak(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		r.NewBreak()
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

// NewBreak starts a new paragraph while preserving renderer state.
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

// SetDefaultFontFamily applies the configured Latin font and the requested East Asian font.
func (r *DocxRenderer) SetDefaultFontFamily(props *document.RunProperties, fontFamily string) {
	fonts := props.Fonts()
	fontsX := fonts.X()
	times := r.config.Fonts.Latin
	fontsX.AsciiAttr = &times
	fontsX.HAnsiAttr = &times
	fontsX.EastAsiaAttr = &fontFamily
	fontsX.CsAttr = &times
}

func (r *DocxRenderer) setFontFamily(props *document.RunProperties, fontFamily string) {
	if fontFamily == "" {
		fontFamily = r.config.Fonts.Content
	}
	r.SetDefaultFontFamily(props, fontFamily)
}

func (r *DocxRenderer) setFontSize(props *document.RunProperties, fontSize int) {
	props.SetSize(float64(fontSize))
}

func (r *DocxRenderer) setFirstLineIndent(para *document.Paragraph) {
	props := para.Properties()
	props.SetFirstLineIndent(r.config.Paragraph.FirstLineIndent)
}

func (r *DocxRenderer) setListFirstLineIndent(para *document.Paragraph) {
	props := para.Properties()
	props.SetFirstLineIndent(r.config.List.FirstLineIndent)
}

func (r *DocxRenderer) setParagraphSpacing(para *document.Paragraph) {
	props := para.Properties()
	spacing := props.Spacing()
	lineHeight := float64(r.config.Text.ContentSize) * r.config.Paragraph.LineHeightMultiplier
	spacing.SetLineSpacing(float64(lineHeight), wml.ST_LineSpacingRuleAuto)
	props.SetAlignment(r.config.Paragraph.Alignment)
}

// SetContentFont applies the configured body text size and requested font family.
func (r *DocxRenderer) SetContentFont(props *document.RunProperties, fontFamily string) {
	r.setFontFamily(props, fontFamily)
	r.setFontSize(props, r.config.Text.ContentSize)
}

// SetSubTitleFont applies the configured subtitle text size and requested font family.
func (r *DocxRenderer) SetSubTitleFont(props *document.RunProperties, fontFamily string) {
	r.setFontFamily(props, fontFamily)
	r.setFontSize(props, r.config.Text.SubTitleSize)
}

// SetTitleFont applies the configured title text size and requested font family.
func (r *DocxRenderer) SetTitleFont(props *document.RunProperties, fontFamily string) {
	r.setFontFamily(props, fontFamily)
	r.setFontSize(props, r.config.Text.TitleSize)
}

// SetNameFont applies the legacy centered-name typography.
func (r *DocxRenderer) SetNameFont(para *document.Paragraph, props *document.RunProperties, fontFamily string) {
	paraProps := para.Properties()
	paraProps.SetAlignment(wml.ST_JcCenter)

	r.setFontFamily(props, fontFamily)
	r.setFontSize(props, 36)
}
