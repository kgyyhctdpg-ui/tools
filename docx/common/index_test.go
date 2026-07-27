package common

import "testing"

func TestFormatLatexFormula(t *testing.T) {
	input := ` \frac {  a+b } { c } =x_ { 1 }^ {2} `
	want := `\frac{a + b}{c} = x_{1}^{2}`

	if got := FormatLatexFormula(input); got != want {
		t.Fatalf("FormatLatexFormula() = %q, want %q", got, want)
	}
}

func TestFormatLatexFormulaEnvironment(t *testing.T) {
	input := `\begin{aligned}a&=b+c\\d&=e-f\end{aligned}`
	want := "\\begin{aligned}\na &= b + c \\\\\nd &= e - f\n\\end{aligned}"

	if got := FormatLatexFormula(input); got != want {
		t.Fatalf("FormatLatexFormula() = %q, want %q", got, want)
	}
}

func TestNormalizeEscapedLatexCommands(t *testing.T) {
	input := `SAF=(\\Delta A/A)\\div(\\Delta F/F)`
	want := `SAF=(\Delta A/A)\div(\Delta F/F)`

	if got := NormalizeEscapedLatexCommands(input); got != want {
		t.Fatalf("NormalizeEscapedLatexCommands() = %q, want %q", got, want)
	}
}

func TestNormalizeEscapedLatexCommandsKeepsRowBreak(t *testing.T) {
	input := `\begin{aligned}a&=b+c\\d&=e-f\end{aligned}`

	if got := NormalizeEscapedLatexCommands(input); got != input {
		t.Fatalf("NormalizeEscapedLatexCommands() = %q, want %q", got, input)
	}
}

func TestFormatLatexInMathBlocks(t *testing.T) {
	input := "before\n$$\n \\frac { a+b } { c } =x_ { 1 }^ {2}  \n$$\nafter"
	want := "before\n$$\n\\frac{a + b}{c} = x_{1}^{2}\n$$\nafter"

	if got := FormatLatexInMathBlocks(input); got != want {
		t.Fatalf("FormatLatexInMathBlocks() = %q, want %q", got, want)
	}
}

func TestFormatLatexInMathBlocksSkipsFencedCode(t *testing.T) {
	input := "```md\n$$ a+b $$\n```\n$$a+b$$"
	want := "```md\n$$ a+b $$\n```\n$$a + b$$"

	if got := FormatLatexInMathBlocks(input); got != want {
		t.Fatalf("FormatLatexInMathBlocks() = %q, want %q", got, want)
	}
}
