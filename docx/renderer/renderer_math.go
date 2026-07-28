package renderer

import (
	"fmt"
	"log"
	"strings"

	"github.com/88250/lute/ast"
	"github.com/88250/lute/util"
	docxmath "github.com/scoming-dev/tools/docx/math"
)

func (r *DocxRenderer) renderInlineMathCloseMarker(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderInlineMathContent(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		if err := r.renderInlineMathAsOMML(node.Tokens); err != nil {
			log.Printf("OMML rendering failed, fallback to math text: %s", err)
			r.renderCodeSpanLike(node.Tokens)
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
		if err := r.renderBlockMathAsOMML(node.Tokens); err != nil {
			log.Printf("OMML rendering failed, fallback to math text: %s", err)
			r.renderCodeBlockLike(node.Tokens)
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

// renderInlineMathAsOMML converts inline LaTeX math to OMML and appends it to the current run.
func (r *DocxRenderer) renderInlineMathAsOMML(mathContent []byte) error {
	latex := strings.TrimSpace(util.BytesToStr(mathContent))
	converter := docxmath.NewConverter()
	oMath, err := converter.ConvertInline(latex)
	if err != nil {
		log.Printf("Failed to convert inline math to structured OMML, using text OMML: %s", err)
		oMath = docxmath.CreateTextOMath(latex)
	}

	para := r.peekPara()
	if para == nil {
		return fmt.Errorf("no current paragraph for inline math")
	}
	run := r.peekRun()
	if run == nil {
		newRun := para.AddRun()
		run = &newRun
	}
	run.AddRawXML(docxmath.OMathXMLWithFontSize(oMath, r.config.Text.ContentSize))
	return nil
}

// renderBlockMathAsOMML converts block LaTeX math to an independent OMML paragraph.
func (r *DocxRenderer) renderBlockMathAsOMML(mathContent []byte) error {
	latex := strings.TrimSpace(util.BytesToStr(mathContent))
	converter := docxmath.NewConverter()
	oMathPara, err := converter.ConvertBlock(latex)
	if err != nil {
		log.Printf("Failed to convert block math to structured OMML, using text OMML: %s", err)
		oMathPara = docxmath.TextOMathPara(latex)
	}

	para := r.doc.AddParagraph()
	para.Properties().SetAlignment(r.config.Math.Alignment)

	spacing := para.Properties().Spacing()
	spacing.SetBefore(r.config.Math.SpacingBefore)
	spacing.SetAfter(r.config.Math.SpacingAfter)

	run := para.AddRun()
	run.AddRawXML(docxmath.OMathParaXMLWithFontSize(oMathPara, r.config.Text.ContentSize))
	return nil
}
