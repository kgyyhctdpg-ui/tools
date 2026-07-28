package math

import (
	"fmt"

	mathSchema "github.com/scoming-dev/tools/docx/schema/soo/ofc/math"
)

// Builder converts AST expressions to OMML structures.
type Builder struct{}

// NewBuilder creates a new OMML builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// BuildOMath converts an expression to inline math OMML.
func (b *Builder) BuildOMath(expr Exp) (*mathSchema.OMath, error) {
	oMath := mathSchema.NewOMath()

	elem, err := b.buildElement(expr)
	if err != nil {
		return nil, err
	}

	if elem != nil {
		oMath.EG_OMathMathElements = append(oMath.EG_OMathMathElements, elem)
	}

	return oMath, nil
}

// BuildOMathPara converts an expression to block math OMML.
func (b *Builder) BuildOMathPara(expr Exp) (*mathSchema.OMathPara, error) {
	oMathPara := mathSchema.NewOMathPara()

	oMath, err := b.BuildOMath(expr)
	if err != nil {
		return nil, err
	}

	oMathPara.OMath = append(oMathPara.OMath, oMath)

	return oMathPara, nil
}

// buildOMathArg builds an OMath argument container.
func (b *Builder) buildOMathArg(expr Exp) (*mathSchema.CT_OMathArg, error) {
	arg := mathSchema.NewCT_OMathArg()

	if expr == nil {
		return arg, nil
	}

	elem, err := b.buildElement(expr)
	if err != nil {
		return nil, err
	}

	if elem != nil {
		arg.EG_OMathMathElements = append(arg.EG_OMathMathElements, elem)
	}

	return arg, nil
}

// buildElement builds an OMML element from an expression.
func (b *Builder) buildElement(expr Exp) (*mathSchema.EG_OMathMathElements, error) {
	if expr == nil {
		return nil, nil
	}

	elem := mathSchema.NewEG_OMathMathElements()

	switch e := expr.(type) {
	case *EText:
		r, err := b.buildTextRun(e.Content, e.Type)
		if err != nil {
			return nil, err
		}
		elem.R = r
		return elem, nil

	case *ENumber:
		r, err := b.buildTextRun(e.Value, TextNormal)
		if err != nil {
			return nil, err
		}
		elem.R = r
		return elem, nil

	case *EIdentifier:
		r, err := b.buildTextRun(e.Name, TextItalic)
		if err != nil {
			return nil, err
		}
		elem.R = r
		return elem, nil

	case *ESymbol:
		r, err := b.buildTextRun(e.Name, TextNormal)
		if err != nil {
			return nil, err
		}
		elem.R = r
		return elem, nil

	case *EOperator:
		r, err := b.buildTextRun(e.Op, TextNormal)
		if err != nil {
			return nil, err
		}
		elem.R = r
		return elem, nil

	case *ESpace:
		r, err := b.buildTextRun(" ", TextNormal)
		if err != nil {
			return nil, err
		}
		elem.R = r
		return elem, nil

	case *EGrouped:
		// For grouped expressions, we need to handle each element
		// Return nil here - the parent will handle adding elements
		return nil, nil

	case *EFraction:
		f, err := b.buildFraction(e)
		if err != nil {
			return nil, err
		}
		elem.F = f
		return elem, nil

	case *ERoot:
		rad, err := b.buildRoot(e)
		if err != nil {
			return nil, err
		}
		elem.Rad = rad
		return elem, nil

	case *ESqrt:
		rad, err := b.buildSqrt(e)
		if err != nil {
			return nil, err
		}
		elem.Rad = rad
		return elem, nil

	case *ESubscript:
		ss, err := b.buildSubscript(e)
		if err != nil {
			return nil, err
		}
		return ss, nil

	case *ESuperscript:
		ss, err := b.buildSuperscript(e)
		if err != nil {
			return nil, err
		}
		return ss, nil

	case *ESubSuperscript:
		ss, err := b.buildSubSuperscript(e)
		if err != nil {
			return nil, err
		}
		return ss, nil

	case *ENary:
		nary, err := b.buildNary(e)
		if err != nil {
			return nil, err
		}
		elem.Nary = nary
		return elem, nil

	case *EUnder:
		lim, err := b.buildUnder(e)
		if err != nil {
			return nil, err
		}
		elem.LimLow = lim
		return elem, nil

	case *EOver:
		lim, err := b.buildOver(e)
		if err != nil {
			return nil, err
		}
		elem.LimUpp = lim
		return elem, nil

	case *EDecorated:
		acc, err := b.buildDecorated(e)
		if err != nil {
			return nil, err
		}
		elem.Acc = acc
		return elem, nil

	case *EDelimited:
		d, err := b.buildDelimited(e)
		if err != nil {
			return nil, err
		}
		elem.D = d
		return elem, nil

	case *EArray:
		m, err := b.buildArray(e)
		if err != nil {
			return nil, err
		}
		elem.M = m
		return elem, nil

	default:
		return nil, fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// buildTextRun creates a text run element.
func (b *Builder) buildTextRun(text string, _ TextType) (*mathSchema.CT_R, error) {
	r := mathSchema.NewCT_R()

	t := mathSchema.NewCT_Text()
	t.Content = text

	r.Choice = append(r.Choice, &mathSchema.CT_RChoice{
		T: []*mathSchema.CT_Text{t},
	})

	return r, nil
}

// createChar creates a CT_Char element with the given value.
func createChar(val string) *mathSchema.CT_Char {
	chr := mathSchema.NewCT_Char()
	chr.ValAttr = val
	return chr
}

// buildFraction creates a fraction element.
func (b *Builder) buildFraction(f *EFraction) (*mathSchema.CT_F, error) {
	ctF := mathSchema.NewCT_F()

	// Build numerator
	num, err := b.buildOMathArg(f.Numerator)
	if err != nil {
		return nil, err
	}
	ctF.Num = num

	// Build denominator
	den, err := b.buildOMathArg(f.Denominator)
	if err != nil {
		return nil, err
	}
	ctF.Den = den

	// Set fraction properties
	ctF.FPr = mathSchema.NewCT_FPr()
	if f.NoBar {
		ctF.FPr.Type = mathSchema.NewCT_FType()
		ctF.FPr.Type.ValAttr = mathSchema.ST_FTypeNoBar
	}

	return ctF, nil
}

// buildRoot creates a radical (root) element with optional index.
func (b *Builder) buildRoot(r *ERoot) (*mathSchema.CT_Rad, error) {
	ctRad := mathSchema.NewCT_Rad()

	// Build radicand (the expression under the root)
	radicand, err := b.buildOMathArg(r.Radicand)
	if err != nil {
		return nil, err
	}
	ctRad.E = radicand

	// Build index (root degree) if present
	if r.Index != nil {
		index, err := b.buildOMathArg(r.Index)
		if err != nil {
			return nil, err
		}
		ctRad.Deg = index
	}

	// Set radical properties
	ctRad.RadPr = mathSchema.NewCT_RadPr()
	if r.Index == nil {
		// Square root - hide degree
		ctRad.RadPr.DegHide = mathSchema.NewCT_OnOff()
	}

	return ctRad, nil
}

// buildSqrt creates a square root element.
func (b *Builder) buildSqrt(s *ESqrt) (*mathSchema.CT_Rad, error) {
	ctRad := mathSchema.NewCT_Rad()

	// Build radicand
	radicand, err := b.buildOMathArg(s.Radicand)
	if err != nil {
		return nil, err
	}
	ctRad.E = radicand

	// Set radical properties - hide degree for square root
	ctRad.RadPr = mathSchema.NewCT_RadPr()
	ctRad.RadPr.DegHide = mathSchema.NewCT_OnOff()

	return ctRad, nil
}

// buildSubscript creates a subscript element.
func (b *Builder) buildSubscript(s *ESubscript) (*mathSchema.EG_OMathMathElements, error) {
	ctSub := mathSchema.NewCT_SSub()

	// Build base
	baseArg, err := b.buildOMathArg(s.Base)
	if err != nil {
		return nil, err
	}
	ctSub.E = baseArg

	// Build subscript
	subArg, err := b.buildOMathArg(s.Subscript)
	if err != nil {
		return nil, err
	}
	ctSub.Sub = subArg

	ctSub.SSubPr = mathSchema.NewCT_SSubPr()

	elem := mathSchema.NewEG_OMathMathElements()
	elem.SSub = ctSub
	return elem, nil
}

// buildSuperscript creates a superscript element.
func (b *Builder) buildSuperscript(s *ESuperscript) (*mathSchema.EG_OMathMathElements, error) {
	ctSup := mathSchema.NewCT_SSup()

	// Build base
	baseArg, err := b.buildOMathArg(s.Base)
	if err != nil {
		return nil, err
	}
	ctSup.E = baseArg

	// Build superscript
	supArg, err := b.buildOMathArg(s.Superscript)
	if err != nil {
		return nil, err
	}
	ctSup.Sup = supArg

	ctSup.SSupPr = mathSchema.NewCT_SSupPr()

	elem := mathSchema.NewEG_OMathMathElements()
	elem.SSup = ctSup
	return elem, nil
}

// buildSubSuperscript creates a subscript+superscript element.
func (b *Builder) buildSubSuperscript(ss *ESubSuperscript) (*mathSchema.EG_OMathMathElements, error) {
	ctSubSup := mathSchema.NewCT_SSubSup()

	// Build base
	baseArg, err := b.buildOMathArg(ss.Base)
	if err != nil {
		return nil, err
	}
	ctSubSup.E = baseArg

	// Build subscript
	subArg, err := b.buildOMathArg(ss.Subscript)
	if err != nil {
		return nil, err
	}
	ctSubSup.Sub = subArg

	// Build superscript
	supArg, err := b.buildOMathArg(ss.Superscript)
	if err != nil {
		return nil, err
	}
	ctSubSup.Sup = supArg

	ctSubSup.SSubSupPr = mathSchema.NewCT_SSubSupPr()

	elem := mathSchema.NewEG_OMathMathElements()
	elem.SSubSup = ctSubSup
	return elem, nil
}

// buildNary creates an n-ary operator element.
func (b *Builder) buildNary(n *ENary) (*mathSchema.CT_Nary, error) {
	ctNary := mathSchema.NewCT_Nary()

	// Set the operator character
	ctNary.NaryPr = mathSchema.NewCT_NaryPr()
	ctNary.NaryPr.Chr = createChar(n.Operator)

	// Build lower limit (subscript)
	if n.LowerLimit != nil {
		subArg, err := b.buildOMathArg(n.LowerLimit)
		if err != nil {
			return nil, err
		}
		ctNary.Sub = subArg
	}

	// Build upper limit (superscript)
	if n.UpperLimit != nil {
		supArg, err := b.buildOMathArg(n.UpperLimit)
		if err != nil {
			return nil, err
		}
		ctNary.Sup = supArg
	}

	// Build the base expression
	if n.Body != nil {
		baseArg, err := b.buildOMathArg(n.Body)
		if err != nil {
			return nil, err
		}
		ctNary.E = baseArg
	}

	return ctNary, nil
}

// buildUnder creates a lower limit element (underset).
func (b *Builder) buildUnder(u *EUnder) (*mathSchema.CT_LimLow, error) {
	ctLimLow := mathSchema.NewCT_LimLow()

	// Build the base expression
	if u.Base != nil {
		baseArg, err := b.buildOMathArg(u.Base)
		if err != nil {
			return nil, err
		}
		ctLimLow.E = baseArg
	}

	// Build the under expression
	if u.UnderExp != nil {
		limArg, err := b.buildOMathArg(u.UnderExp)
		if err != nil {
			return nil, err
		}
		ctLimLow.Lim = limArg
	}

	return ctLimLow, nil
}

// buildOver creates an upper limit element (overset).
func (b *Builder) buildOver(o *EOver) (*mathSchema.CT_LimUpp, error) {
	ctLimUpp := mathSchema.NewCT_LimUpp()

	// Build the base expression
	if o.Base != nil {
		baseArg, err := b.buildOMathArg(o.Base)
		if err != nil {
			return nil, err
		}
		ctLimUpp.E = baseArg
	}

	// Build the over expression
	if o.OverExp != nil {
		limArg, err := b.buildOMathArg(o.OverExp)
		if err != nil {
			return nil, err
		}
		ctLimUpp.Lim = limArg
	}

	return ctLimUpp, nil
}

// buildDecorated creates an accent element.
func (b *Builder) buildDecorated(d *EDecorated) (*mathSchema.CT_Acc, error) {
	ctAcc := mathSchema.NewCT_Acc()

	// Map decoration type to character
	accentChar := ""
	switch d.Decoration {
	case DecorHat:
		accentChar = "^"
	case DecorBar:
		accentChar = "-"
	case DecorVec:
		accentChar = "→"
	case DecorDot:
		accentChar = "."
	case DecorDdot:
		accentChar = ".."
	case DecorTilde:
		accentChar = "~"
	case DecorAcute:
		accentChar = "´"
	case DecorGrave:
		accentChar = "`"
	case DecorCheck:
		accentChar = "ˇ"
	case DecorBreve:
		accentChar = "˘"
	}

	ctAcc.AccPr = mathSchema.NewCT_AccPr()
	if accentChar != "" {
		ctAcc.AccPr.Chr = createChar(accentChar)
	}

	// Build the expression being accented
	arg, err := b.buildOMathArg(d.Base)
	if err != nil {
		return nil, err
	}
	ctAcc.E = arg

	return ctAcc, nil
}

// buildDelimited creates a delimiter (brackets) element.
func (b *Builder) buildDelimited(d *EDelimited) (*mathSchema.CT_D, error) {
	ctD := mathSchema.NewCT_D()

	// Set delimiter properties
	ctD.DPr = mathSchema.NewCT_DPr()

	// Set left delimiter
	if d.LeftDelimiter != "" && d.LeftDelimiter != "." {
		ctD.DPr.BegChr = createChar(d.LeftDelimiter)
	}

	// Set right delimiter
	if d.RightDelimiter != "" && d.RightDelimiter != "." {
		ctD.DPr.EndChr = createChar(d.RightDelimiter)
	}

	// Build the expressions inside the delimiter
	for _, exp := range d.Exps {
		arg, err := b.buildOMathArg(exp)
		if err != nil {
			return nil, err
		}
		ctD.E = append(ctD.E, arg)
	}

	return ctD, nil
}

// buildArray creates a matrix/array element.
func (b *Builder) buildArray(a *EArray) (*mathSchema.CT_M, error) {
	ctM := mathSchema.NewCT_M()

	// Set matrix properties
	ctM.MPr = mathSchema.NewCT_MPr()

	// For pmatrix, bmatrix, etc., the delimiters are handled outside the matrix
	// Build each row
	for _, line := range a.Lines {
		mcRow := mathSchema.NewCT_MR()

		for _, cell := range line {
			// Build cell element
			mcCell, err := b.buildOMathArg(cell)
			if err != nil {
				return nil, err
			}

			mcRow.E = append(mcRow.E, mcCell)
		}

		ctM.Mr = append(ctM.Mr, mcRow)
	}

	return ctM, nil
}

// BuildOMathFromGrouped builds an OMath from a grouped expression.
// This is used when the top-level expression is a grouped/sequence.
func (b *Builder) BuildOMathFromGrouped(g *EGrouped) (*mathSchema.OMath, error) {
	oMath := mathSchema.NewOMath()

	for _, expr := range g.Exps {
		elem, err := b.buildElement(expr)
		if err != nil {
			return nil, err
		}

		// Handle nested grouped expressions
		if innerG, ok := expr.(*EGrouped); ok {
			// Recursively process inner grouped
			for _, innerExpr := range innerG.Exps {
				innerElem, err := b.buildElement(innerExpr)
				if err != nil {
					return nil, err
				}
				if innerElem != nil {
					oMath.EG_OMathMathElements = append(oMath.EG_OMathMathElements, innerElem)
				}
			}
		} else if elem != nil {
			oMath.EG_OMathMathElements = append(oMath.EG_OMathMathElements, elem)
		}
	}

	return oMath, nil
}
