// Package math provides LaTeX to OMML (Office Math Markup Language) conversion.
// Based on texmath library design: https://github.com/jgm/texmath
package math

import (
	"fmt"
	"strings"
)

// ============================================================
// Core Expression Types (based on texmath's Exp type)
// ============================================================

// Exp represents a mathematical expression in the AST.
// This is the core interface, similar to texmath's Exp type.
type Exp interface {
	isExp()
	String() string
}

// ============================================================
// Basic Types
// ============================================================

// ENumber represents a numeric value.
type ENumber struct {
	Value string
}

func (e *ENumber) isExp()         {}
func (e *ENumber) String() string { return e.Value }

// EIdentifier represents a variable name or identifier.
type EIdentifier struct {
	Name string
}

func (e *EIdentifier) isExp()         {}
func (e *EIdentifier) String() string { return e.Name }

// ESymbol represents a mathematical symbol with type information.
// Based on texmath's TeXSymbolType.
type SymbolType int

const (
	SymbolOrd   SymbolType = iota // Ordinary symbol
	SymbolOp                      // Operator (large)
	SymbolBin                     // Binary operator
	SymbolRel                     // Relation
	SymbolOpen                    // Opening delimiter
	SymbolClose                   // Closing delimiter
	SymbolPun                     // Punctuation
)

type ESymbol struct {
	Type SymbolType
	Name string // Symbol name or Unicode character
}

func (e *ESymbol) isExp()         {}
func (e *ESymbol) String() string { return e.Name }

// EOperator represents an operator (like +, -, =).
type EOperator struct {
	Op string
}

func (e *EOperator) isExp()         {}
func (e *EOperator) String() string { return e.Op }

// EText represents text with a specific style.
type TextType int

const (
	TextNormal     TextType = iota
	TextBold                // \mathbf
	TextItalic              // \mathit
	TextRoman               // \mathrm
	TextSansSerif           // \mathsf
	TextMonospace           // \mathtt
	TextTypewriter          // \texttt
)

type EText struct {
	Type    TextType
	Content string
}

func (e *EText) isExp()         {}
func (e *EText) String() string { return e.Content }

// ============================================================
// Grouping and Delimiters
// ============================================================

// EGrouped represents a grouped expression { ... }.
type EGrouped struct {
	Exps []Exp
}

func (e *EGrouped) isExp() {}
func (e *EGrouped) String() string {
	var sb strings.Builder
	sb.WriteString("{")
	for _, exp := range e.Exps {
		sb.WriteString(exp.String())
	}
	sb.WriteString("}")
	return sb.String()
}

// EDelimited represents a delimited expression like (...) or [...].
type EDelimited struct {
	LeftDelimiter  string
	RightDelimiter string
	Exps           []Exp
}

func (e *EDelimited) isExp() {}
func (e *EDelimited) String() string {
	var sb strings.Builder
	sb.WriteString(e.LeftDelimiter)
	for _, exp := range e.Exps {
		sb.WriteString(exp.String())
	}
	sb.WriteString(e.RightDelimiter)
	return sb.String()
}

// ============================================================
// Subscripts and Superscripts
// ============================================================

// ESubscript represents a subscripted expression.
type ESubscript struct {
	Base      Exp
	Subscript Exp
}

func (e *ESubscript) isExp() {}
func (e *ESubscript) String() string {
	return fmt.Sprintf("%s_{%s}", e.Base.String(), e.Subscript.String())
}

// ESuperscript represents a superscripted expression.
type ESuperscript struct {
	Base        Exp
	Superscript Exp
}

func (e *ESuperscript) isExp() {}
func (e *ESuperscript) String() string {
	return fmt.Sprintf("%s^{%s}", e.Base.String(), e.Superscript.String())
}

// ESubSuperscript represents an expression with both subscript and superscript.
type ESubSuperscript struct {
	Base        Exp
	Subscript   Exp
	Superscript Exp
}

func (e *ESubSuperscript) isExp() {}
func (e *ESubSuperscript) String() string {
	return fmt.Sprintf("%s_{%s}^{%s}", e.Base.String(), e.Subscript.String(), e.Superscript.String())
}

// ============================================================
// Fractions and Roots
// ============================================================

// EFraction represents a fraction.
type EFraction struct {
	DisplayStyle bool // true for \dfrac, false for \tfrac
	Numerator    Exp
	Denominator  Exp
	NoBar        bool // true for \binom style (no fraction bar)
}

func (e *EFraction) isExp() {}
func (e *EFraction) String() string {
	if e.NoBar {
		return fmt.Sprintf("\\binom{%s}{%s}", e.Numerator.String(), e.Denominator.String())
	}
	return fmt.Sprintf("\\frac{%s}{%s}", e.Numerator.String(), e.Denominator.String())
}

// ESqrt represents a square root.
type ESqrt struct {
	Radicand Exp
}

func (e *ESqrt) isExp() {}
func (e *ESqrt) String() string {
	return fmt.Sprintf("\\sqrt{%s}", e.Radicand.String())
}

// ERoot represents an nth root.
type ERoot struct {
	Index    Exp // nil for square root
	Radicand Exp
}

func (e *ERoot) isExp() {}
func (e *ERoot) String() string {
	if e.Index == nil {
		return fmt.Sprintf("\\sqrt{%s}", e.Radicand.String())
	}
	return fmt.Sprintf("\\sqrt[%s]{%s}", e.Index.String(), e.Radicand.String())
}

// ============================================================
// Under, Over, and UnderOver (for \overset, \underset, etc.)
// ============================================================

// EUnder represents an expression with something under it.
type EUnder struct {
	Base     Exp
	UnderExp Exp
}

func (e *EUnder) isExp() {}
func (e *EUnder) String() string {
	return fmt.Sprintf("\\underset{%s}{%s}", e.UnderExp.String(), e.Base.String())
}

// EOver represents an expression with something over it.
type EOver struct {
	Base    Exp
	OverExp Exp
}

func (e *EOver) isExp() {}
func (e *EOver) String() string {
	return fmt.Sprintf("\\overset{%s}{%s}", e.OverExp.String(), e.Base.String())
}

// EUnderOver represents an expression with both under and over.
type EUnderOver struct {
	Base     Exp
	UnderExp Exp
	OverExp  Exp
}

func (e *EUnderOver) isExp() {}
func (e *EUnderOver) String() string {
	return fmt.Sprintf("\\overset{%s}{\\underset{%s}{%s}}",
		e.OverExp.String(), e.UnderExp.String(), e.Base.String())
}

// ============================================================
// N-ary Operators (Sum, Product, Integral, etc.)
// ============================================================

// ENary represents an n-ary operator like sum, product, integral.
type ENary struct {
	Operator   string // "∑", "∏", "∫", etc.
	Body       Exp    // The body of the operator
	LowerLimit Exp    // Lower limit (subscript)
	UpperLimit Exp    // Upper limit (superscript)
	HasLimits  bool   // Whether limits are shown
}

func (e *ENary) isExp() {}
func (e *ENary) String() string {
	var sb strings.Builder
	sb.WriteString(e.Operator)
	if e.LowerLimit != nil {
		sb.WriteString(fmt.Sprintf("_{%s}", e.LowerLimit.String()))
	}
	if e.UpperLimit != nil {
		sb.WriteString(fmt.Sprintf("^{%s}", e.UpperLimit.String()))
	}
	if e.Body != nil {
		sb.WriteString(" ")
		sb.WriteString(e.Body.String())
	}
	return sb.String()
}

// ============================================================
// Arrays and Matrices
// ============================================================

// ArrayLine represents a row in an array or matrix.
type ArrayLine []Exp

// EArray represents an array or matrix.
type EArray struct {
	EnvType string      // "matrix", "pmatrix", "bmatrix", "array", etc
	Lines   []ArrayLine // Rows of expressions
	Aligns  []string    // Column alignments (for array environment)
}

func (e *EArray) isExp() {}
func (e *EArray) String() string {
	var sb strings.Builder
	sb.WriteString("\\begin{")
	sb.WriteString(e.EnvType)
	sb.WriteString("}")
	for i, line := range e.Lines {
		for j, exp := range line {
			sb.WriteString(exp.String())
			if j < len(line)-1 {
				sb.WriteString("&")
			}
		}
		if i < len(e.Lines)-1 {
			sb.WriteString("\\\\")
		}
	}
	sb.WriteString("\\end{")
	sb.WriteString(e.EnvType)
	sb.WriteString("}")
	return sb.String()
}

// ============================================================
// Decorations (Accents, etc.)
// ============================================================

// EDecorated represents a decorated expression.
type DecorationType int

const (
	DecorHat   DecorationType = iota // \hat
	DecorBar                         // \bar
	DecorVec                         // \vec
	DecorDot                         // \dot
	DecorDdot                        // \ddot
	DecorTilde                       // \tilde
	DecorAcute                       // \acute
	DecorGrave                       // \grave
	DecorCheck                       // \check
	DecorBreve                       // \breve
)

type EDecorated struct {
	Decoration DecorationType
	Base       Exp
}

func (e *EDecorated) isExp() {}
func (e *EDecorated) String() string {
	decNames := map[DecorationType]string{
		DecorHat:   "hat",
		DecorBar:   "bar",
		DecorVec:   "vec",
		DecorDot:   "dot",
		DecorDdot:  "ddot",
		DecorTilde: "tilde",
	}
	return fmt.Sprintf("\\%s{%s}", decNames[e.Decoration], e.Base.String())
}

// ============================================================
// Space and Phantom
// ============================================================

// ESpace represents spacing.
type ESpace struct {
	Width string // " ", "\\,", "\\;", "\\:", "\\quad", "\\qquad"
}

func (e *ESpace) isExp()         {}
func (e *ESpace) String() string { return e.Width }

// EPhantom represents a phantom expression.
type EPhantom struct {
	Base      Exp
	Direction string // "h" for horizontal, "v" for vertical, "" for both
}

func (e *EPhantom) isExp() {}
func (e *EPhantom) String() string {
	if e.Direction == "" {
		return fmt.Sprintf("\\phantom{%s}", e.Base.String())
	}
	return fmt.Sprintf("\\%sphantom{%s}", e.Direction, e.Base.String())
}

// ============================================================
// Scaled (for \big, \Big, \bigg, \Bigg)
// ============================================================

// EScaled represents a scaled expression.
type EScaled struct {
	Scale string // "big", "Big", "bigg", "Bigg"
	Base  Exp
}

func (e *EScaled) isExp() {}
func (e *EScaled) String() string {
	return fmt.Sprintf("\\%s %s", e.Scale, e.Base.String())
}

// ============================================================
// Styled (for \mathbf, \mathrm, etc.)
// ============================================================

// EStyled represents styled text.
type EStyled struct {
	Style TextType
	Exps  []Exp
}

func (e *EStyled) isExp() {}
func (e *EStyled) String() string {
	var sb strings.Builder
	styleNames := map[TextType]string{
		TextBold:      "mathbf",
		TextItalic:    "mathit",
		TextRoman:     "mathrm",
		TextSansSerif: "mathsf",
		TextMonospace: "mathtt",
	}
	sb.WriteString("\\")
	sb.WriteString(styleNames[e.Style])
	sb.WriteString("{")
	for _, exp := range e.Exps {
		sb.WriteString(exp.String())
	}
	sb.WriteString("}")
	return sb.String()
}

// ============================================================
// Boxed (for \boxed)
// ============================================================

// EBoxed represents a boxed expression.
type EBoxed struct {
	Base Exp
}

func (e *EBoxed) isExp() {}
func (e *EBoxed) String() string {
	return fmt.Sprintf("\\boxed{%s}", e.Base.String())
}

// ============================================================
// Utility Functions
// ============================================================

// IsEmpty checks if an expression is nil or empty.
func IsEmpty(e Exp) bool {
	if e == nil {
		return true
	}
	if g, ok := e.(*EGrouped); ok && len(g.Exps) == 0 {
		return true
	}
	return false
}

// AsExps converts a single Exp to a slice.
func AsExps(e Exp) []Exp {
	if e == nil {
		return nil
	}
	if g, ok := e.(*EGrouped); ok {
		return g.Exps
	}
	return []Exp{e}
}
