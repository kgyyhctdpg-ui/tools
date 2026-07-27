package math

import (
	"fmt"

	mathSchema "github.com/scoming-dev/tools/docx/schema/soo/ofc/math"
)

// Converter provides LaTeX to OMML conversion functionality.
type Converter struct {
	parser  *Parser
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
	// Parse LaTeX to AST
	c.parser = NewParser(latex)
	expr, err := c.parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Build OMML from AST
	return c.buildOMathFromExp(expr)
}

// ConvertBlock converts a LaTeX formula to block math OMML.
func (c *Converter) ConvertBlock(latex string) (*mathSchema.OMathPara, error) {
	// Parse LaTeX to AST
	c.parser = NewParser(latex)
	expr, err := c.parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Build OMML from AST
	oMath, err := c.buildOMathFromExp(expr)
	if err != nil {
		return nil, err
	}

	// Wrap in OMathPara for block math
	oMathPara := mathSchema.NewOMathPara()
	oMathPara.OMath = append(oMathPara.OMath, &oMath.CT_OMath)

	return oMathPara, nil
}

// ConvertWithError converts and returns detailed error information.
func (c *Converter) ConvertWithError(latex string, isInline bool) (interface{}, error) {
	if isInline {
		return c.ConvertInline(latex)
	}
	return c.ConvertBlock(latex)
}

// buildOMathFromExp builds an OMath from an expression, handling special cases.
func (c *Converter) buildOMathFromExp(expr Exp) (*mathSchema.OMath, error) {
	// Handle grouped expressions specially at the top level
	if g, ok := expr.(*EGrouped); ok {
		return c.builder.BuildOMathFromGrouped(g)
	}

	// For all other expressions, use the standard builder
	return c.builder.BuildOMath(expr)
}

// ConvertAndInsert converts LaTeX to OMML and inserts it into a paragraph.
// This is a convenience method that combines conversion and insertion.
func (c *Converter) ConvertAndInsert(latex string, isInline bool, insertFunc func(*mathSchema.OMath) error) error {
	oMath, err := c.ConvertInline(latex)
	if err != nil {
		return err
	}

	return insertFunc(oMath)
}

// ConversionResult contains the result of a conversion with potential warnings.
type ConversionResult struct {
	OMath    *mathSchema.OMath
	Warnings []string // Non-fatal issues during conversion
	Errors   []string // Fatal errors
}

// ConvertWithWarnings converts and collects any warnings.
func (c *Converter) ConvertWithWarnings(latex string) (*ConversionResult, error) {
	result := &ConversionResult{}

	// Parse LaTeX to AST
	c.parser = NewParser(latex)
	expr, err := c.parser.Parse()
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		return result, err
	}

	// Build OMML
	oMath, err := c.buildOMathFromExp(expr)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		return result, err
	}

	result.OMath = oMath
	return result, nil
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

	// Add Greek letters
	for name := range GreekLetters {
		commands = append(commands, "\\"+name)
	}

	// Add math operators
	for name := range MathOperators {
		commands = append(commands, "\\"+name)
	}

	// Add relation symbols
	for name := range RelationSymbols {
		commands = append(commands, "\\"+name)
	}

	return commands
}

// Error types for better error handling
type (
	// BuildError represents an error during OMML building.
	BuildError struct {
		Message string
		Expr    Exp
	}

	// UnsupportedError represents an unsupported LaTeX feature.
	UnsupportedError struct {
		Feature string
		Pos     int
	}
)

func (e *BuildError) Error() string {
	return fmt.Sprintf("build error: %s (expr type: %T)", e.Message, e.Expr)
}

func (e *UnsupportedError) Error() string {
	return fmt.Sprintf("unsupported feature at position %d: %s", e.Pos, e.Feature)
}
