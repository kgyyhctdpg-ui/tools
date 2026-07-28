package math

import (
	"fmt"

	"github.com/scoming-dev/tools/docx/document"
	mathSchema "github.com/scoming-dev/tools/docx/schema/soo/ofc/math"
)

// InsertInlineMath appends an inline Office Math expression to the paragraph.
func InsertInlineMath(para *document.Paragraph, oMath *mathSchema.OMath) error {
	if para == nil {
		return fmt.Errorf("paragraph is nil")
	}
	if oMath == nil {
		return fmt.Errorf("oMath is nil")
	}

	run := para.AddRun()
	run.AddRawXML(OMathXML(oMath))
	return nil
}

// InsertBlockMath creates a paragraph containing a block Office Math expression.
func InsertBlockMath(doc *document.Document, oMathPara *mathSchema.OMathPara) (*document.Paragraph, error) {
	if doc == nil {
		return nil, fmt.Errorf("document is nil")
	}
	if oMathPara == nil {
		return nil, fmt.Errorf("oMathPara is nil")
	}

	para := doc.AddParagraph()
	run := para.AddRun()
	run.AddRawXML(OMathParaXML(oMathPara))
	return &para, nil
}

// CreateTextOMath creates a plain-text Office Math expression.
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
