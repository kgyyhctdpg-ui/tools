package math

import (
	"fmt"

	"github.com/scoming-dev/tools/docx/document"
	mathSchema "github.com/scoming-dev/tools/docx/schema/soo/ofc/math"
	wml "github.com/scoming-dev/tools/docx/schema/soo/wml"
)

// InsertInlineMath inserts an inline math formula into a paragraph.
func InsertInlineMath(para *document.Paragraph, oMath *mathSchema.OMath) error {
	if para == nil {
		return fmt.Errorf("paragraph is nil")
	}
	if oMath == nil {
		return fmt.Errorf("oMath is nil")
	}

	// Get the underlying CT_P structure
	ctP := para.X()

	// Create the embedding structure:
	// CT_P -> EG_PContent -> EG_ContentRunContent -> EG_RunLevelElts -> EG_MathContent -> OMath

	// Create EG_PContent
	egPContent := wml.NewEG_PContent()

	// Create EG_ContentRunContent
	contentRunContent := wml.NewEG_ContentRunContent()

	// Create EG_RunLevelElts
	runLevelElts := wml.NewEG_RunLevelElts()

	// Create EG_MathContent
	mathContent := wml.NewEG_MathContent()
	mathContent.OMath = oMath

	// Build the structure
	runLevelElts.EG_MathContent = append(runLevelElts.EG_MathContent, mathContent)
	contentRunContent.EG_RunLevelElts = append(contentRunContent.EG_RunLevelElts, runLevelElts)
	egPContent.EG_ContentRunContent = append(egPContent.EG_ContentRunContent, contentRunContent)

	// Add to paragraph
	ctP.EG_PContent = append(ctP.EG_PContent, egPContent)

	return nil
}

// InsertBlockMath inserts a block math formula as a new paragraph and returns the paragraph.
func InsertBlockMath(doc *document.Document, oMathPara *mathSchema.OMathPara) (*document.Paragraph, error) {
	if doc == nil {
		return nil, fmt.Errorf("document is nil")
	}
	if oMathPara == nil {
		return nil, fmt.Errorf("oMathPara is nil")
	}

	// Create a new paragraph
	para := doc.AddParagraph()

	// Get the underlying CT_P structure
	ctP := para.X()

	// Create the embedding structure for block math:
	// CT_P -> EG_PContent -> EG_ContentRunContent -> EG_RunLevelElts -> EG_MathContent -> OMathPara

	// Create EG_PContent
	egPContent := wml.NewEG_PContent()

	// Create EG_ContentRunContent
	contentRunContent := wml.NewEG_ContentRunContent()

	// Create EG_RunLevelElts
	runLevelElts := wml.NewEG_RunLevelElts()

	// Create EG_MathContent
	mathContent := wml.NewEG_MathContent()
	mathContent.OMathPara = oMathPara

	// Build the structure
	runLevelElts.EG_MathContent = append(runLevelElts.EG_MathContent, mathContent)
	contentRunContent.EG_RunLevelElts = append(contentRunContent.EG_RunLevelElts, runLevelElts)
	egPContent.EG_ContentRunContent = append(egPContent.EG_ContentRunContent, contentRunContent)

	// Set the paragraph content (replace any existing content)
	ctP.EG_PContent = []*wml.EG_PContent{egPContent}

	return &para, nil
}

// AppendInlineMathToRun appends an inline math formula to an existing run context.
// This is useful when you want to add math to the current run instead of creating a new one.
func AppendInlineMathToRun(ctP *wml.CT_P, oMath *mathSchema.OMath) error {
	if ctP == nil {
		return fmt.Errorf("CT_P is nil")
	}
	if oMath == nil {
		return fmt.Errorf("oMath is nil")
	}

	// Try to append to existing EG_PContent if it exists
	if len(ctP.EG_PContent) > 0 {
		// Find or create the right structure
		lastPContent := ctP.EG_PContent[len(ctP.EG_PContent)-1]

		// Create new EG_ContentRunContent for the math
		contentRunContent := wml.NewEG_ContentRunContent()
		runLevelElts := wml.NewEG_RunLevelElts()
		mathContent := wml.NewEG_MathContent()
		mathContent.OMath = oMath

		runLevelElts.EG_MathContent = append(runLevelElts.EG_MathContent, mathContent)
		contentRunContent.EG_RunLevelElts = append(contentRunContent.EG_RunLevelElts, runLevelElts)
		lastPContent.EG_ContentRunContent = append(lastPContent.EG_ContentRunContent, contentRunContent)

		return nil
	}

	// No existing content, create new structure
	egPContent := wml.NewEG_PContent()
	contentRunContent := wml.NewEG_ContentRunContent()
	runLevelElts := wml.NewEG_RunLevelElts()
	mathContent := wml.NewEG_MathContent()
	mathContent.OMath = oMath

	runLevelElts.EG_MathContent = append(runLevelElts.EG_MathContent, mathContent)
	contentRunContent.EG_RunLevelElts = append(contentRunContent.EG_RunLevelElts, runLevelElts)
	egPContent.EG_ContentRunContent = append(egPContent.EG_ContentRunContent, contentRunContent)
	ctP.EG_PContent = append(ctP.EG_PContent, egPContent)

	return nil
}

// InsertOMathToCTP directly inserts OMath to a CT_P structure.
// This is a lower-level function for more control.
func InsertOMathToCTP(ctP *wml.CT_P, oMath *mathSchema.OMath) error {
	return AppendInlineMathToRun(ctP, oMath)
}

// InsertOMathParaToDocument creates a new paragraph with OMathPara in the document.
func InsertOMathParaToDocument(doc *document.Document, oMathPara *mathSchema.OMathPara) (*wml.CT_P, error) {
	para, err := InsertBlockMath(doc, oMathPara)
	if err != nil {
		return nil, err
	}
	return para.X(), nil
}

// MathRun represents a math run that can be styled.
type MathRun struct {
	OMath *mathSchema.OMath
}

// NewMathRun creates a new math run from an OMath structure.
func NewMathRun(oMath *mathSchema.OMath) *MathRun {
	return &MathRun{OMath: oMath}
}

// SetText sets the text styling for all text runs in the math formula.
// This can be used to set font, size, etc.
func (mr *MathRun) SetText(fontName string, fontSize int32) {
	// This would require traversing the OMath structure and setting
	// styling properties on each CT_R element.
	// For now, this is a placeholder for future enhancement.
}

// Helper function to create a simple text OMath
func CreateTextOMath(text string) *mathSchema.OMath {
	oMath := mathSchema.NewOMath()
	elem := mathSchema.NewEG_OMathMathElements()

	r := mathSchema.NewCT_R()
	t := mathSchema.NewCT_Text()
	t.Content = text
	r.Choice = append(r.Choice, &mathSchema.CT_RChoice{
		T: []*mathSchema.CT_Text{t},
	})

	elem.R = r
	oMath.EG_OMathMathElements = append(oMath.EG_OMathMathElements, elem)

	return oMath
}
