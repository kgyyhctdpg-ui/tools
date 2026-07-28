package math

import (
	"fmt"

	mathSchema "github.com/scoming-dev/tools/docx/schema/soo/ofc/math"
)

// Converter provides LaTeX to OMML conversion functionality.
type Converter struct {
	builder *Builder
}

// NewConverter creates a new LaTeX to OMML converter.
func NewConverter() *Converter {
	return &Converter{
		builder: NewBuilder(),
	}
}

// ConvertInline converts a LaTeX formula to inline math OMML.
func (c *Converter) ConvertInline(latex string) (*mathSchema.OMath, error) {
	parser := NewParser(latex)
	expr, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	return c.buildOMathFromExp(expr)
}

// ConvertBlock converts a LaTeX formula to block math OMML.
func (c *Converter) ConvertBlock(latex string) (*mathSchema.OMathPara, error) {
	parser := NewParser(latex)
	expr, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	oMath, err := c.buildOMathFromExp(expr)
	if err != nil {
		return nil, err
	}

	oMathPara := mathSchema.NewOMathPara()
	oMathPara.OMath = append(oMathPara.OMath, oMath)

	return oMathPara, nil
}

func (c *Converter) buildOMathFromExp(expr Exp) (*mathSchema.OMath, error) {
	if g, ok := expr.(*EGrouped); ok {
		return c.builder.BuildOMathFromGrouped(g)
	}
	return c.builder.BuildOMath(expr)
}

// QuickConvert provides a simple interface for converting LaTeX to OMML.
// Returns an OMath for inline use.
func QuickConvert(latex string) (*mathSchema.OMath, error) {
	return NewConverter().ConvertInline(latex)
}

// QuickConvertBlock provides a simple interface for converting LaTeX to block OMML.
func QuickConvertBlock(latex string) (*mathSchema.OMathPara, error) {
	return NewConverter().ConvertBlock(latex)
}

// ValidateLaTeX checks if a LaTeX string can be parsed without errors.
func ValidateLaTeX(latex string) error {
	parser := NewParser(latex)
	_, err := parser.Parse()
	return err
}

// GetSupportedCommands returns a list of supported LaTeX commands.
func GetSupportedCommands() []string {
	commands := []string{
		// Fractions and roots
		"\\frac", "\\dfrac", "\\tfrac", "\\binom",
		"\\sqrt",

		// Subscripts and superscripts
		"_", "^",

		// N-ary operators
		"\\int", "\\oint", "\\sum", "\\prod", "\\coprod",
		"\\bigcap", "\\bigcup", "\\bigsqcup", "\\bigvee", "\\bigwedge",

		// Limits
		"\\lim", "\\limsup", "\\liminf", "\\sup", "\\inf", "\\max", "\\min",

		// Functions
		"\\sin", "\\cos", "\\tan", "\\cot", "\\sec", "\\csc",
		"\\arcsin", "\\arccos", "\\arctan",
		"\\sinh", "\\cosh", "\\tanh", "\\coth",
		"\\log", "\\ln", "\\lg", "\\exp",
		"\\det", "\\dim", "\\ker", "\\gcd", "\\arg",

		// Accents
		"\\vec", "\\bar", "\\hat", "\\tilde", "\\dot", "\\ddot",

		// Delimiters
		"\\left", "\\right",

		// Environments
		"\\begin{matrix}", "\\begin{pmatrix}", "\\begin{bmatrix}",
		"\\begin{vmatrix}", "\\begin{Vmatrix}",
		"\\begin{array}", "\\begin{align}", "\\begin{cases}",

		// Text
		"\\text", "\\textrm", "\\textit", "\\textbf",

		// Spacing
		"\\quad", "\\qquad", "\\,", "\\;", "\\:", "\\!",
	}

	for name := range GreekLetters {
		commands = append(commands, "\\"+name)
	}

	for name := range MathOperators {
		commands = append(commands, "\\"+name)
	}

	for name := range RelationSymbols {
		commands = append(commands, "\\"+name)
	}

	return commands
}
