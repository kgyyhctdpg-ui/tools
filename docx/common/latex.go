package common

import (
	"regexp"
	"strings"
)

// FormatLatexInMathBlocks formats LaTeX content between $$ delimiters outside fenced code blocks.
func FormatLatexInMathBlocks(content string) string {
	lines := splitLinesKeepingEnd(content)
	var out strings.Builder
	var pending strings.Builder
	inFence := false
	fenceMarker := ""

	for _, line := range lines {
		lineText := trimLineEnding(line)
		if inFence {
			out.WriteString(line)
			if isMarkdownFenceClose(lineText, fenceMarker) {
				inFence = false
				fenceMarker = ""
			}
			continue
		}

		if marker, ok := markdownFenceMarker(lineText); ok {
			out.WriteString(formatLatexInTextChunk(pending.String()))
			pending.Reset()
			inFence = true
			fenceMarker = marker
			out.WriteString(line)
			continue
		}

		pending.WriteString(line)
	}

	out.WriteString(formatLatexInTextChunk(pending.String()))
	return out.String()
}

func formatLatexInTextChunk(content string) string {
	var out strings.Builder

	for i := 0; i < len(content); {
		if content[i] == '`' {
			end := findClosingBacktickRun(content, i)
			if end == -1 {
				out.WriteString(content[i:])
				break
			}
			out.WriteString(content[i:end])
			i = end
			continue
		}

		if strings.HasPrefix(content[i:], "$$") && !isEscapedAt(content, i) {
			out.WriteString("$$")
			i += 2

			end := findMathDelimiter(content, i)
			if end == -1 {
				out.WriteString(content[i:])
				break
			}

			out.WriteString(formatLatexMathBlockBody(content[i:end]))
			out.WriteString("$$")
			i = end + 2
			continue
		}

		out.WriteByte(content[i])
		i++
	}

	return out.String()
}

func formatLatexMathBlockBody(body string) string {
	formatted := FormatLatexFormula(body)
	if formatted == "" {
		return ""
	}
	if strings.ContainsAny(body, "\r\n") || strings.Contains(formatted, "\n") {
		return "\n" + formatted + "\n"
	}
	return formatted
}

// RemoveSpacesBeforeMathBlockAndLineBreak removes spaces before line breaks and normalizes $$ delimiter lines.
func RemoveSpacesBeforeMathBlockAndLineBreak(content string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		line = strings.TrimRight(line, " \t")
		if strings.TrimSpace(line) == "$$" {
			line = "$$"
		}
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

// FormatLatexFormula normalizes a LaTeX math formula with conservative spacing rules.
func FormatLatexFormula(latex string) string {
	latex = strings.ReplaceAll(latex, "\r\n", "\n")
	latex = strings.ReplaceAll(latex, "\r", "\n")
	latex = NormalizeEscapedLatexCommands(latex)

	lines := strings.Split(latex, "\n")
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			parts = append(parts, line)
		}
	}

	formatted := normalizeLatexSpacing(strings.Join(parts, " "))
	formatted = formatLatexEnvironmentLines(formatted)
	return trimLatexLines(formatted)
}

// NormalizeEscapedLatexCommands converts escaped command residue such as \\Delta
// to \Delta while preserving real LaTeX row breaks like \\d &= e.
func NormalizeEscapedLatexCommands(latex string) string {
	runes := []rune(latex)
	out := make([]rune, 0, len(runes))

	for i := 0; i < len(runes); i++ {
		if i+2 < len(runes) && runes[i] == '\\' && runes[i+1] == '\\' && isLatexLetter(runes[i+2]) {
			commandEnd := i + 2
			for commandEnd < len(runes) && isLatexLetter(runes[commandEnd]) {
				commandEnd++
			}

			command := string(runes[i+2 : commandEnd])
			if shouldUnescapeLatexCommand(command, previousNonWhitespaceRune(runes, i)) {
				out = append(out, '\\')
				i++
				continue
			}
		}

		out = append(out, runes[i])
	}

	return string(out)
}

func normalizeLatexSpacing(input string) string {
	runes := []rune(input)
	out := make([]rune, 0, len(runes))

	for i := 0; i < len(runes); {
		r := runes[i]
		if isLatexWhitespace(r) {
			next := nextNonWhitespace(runes, i+1)
			out = appendLatexSpaceIfNeeded(out, next)
			i++
			continue
		}

		switch r {
		case '\\':
			command, next := readLatexCommand(runes, i)
			if command == `\\` {
				out = appendLatexRowBreak(out)
			} else if isLatexOperatorCommand(command) {
				out = appendLatexSpacedToken(out, command, nextNonWhitespace(runes, next))
			} else if isLatexTextCommand(command) {
				var ok bool
				out, i, ok = appendLatexTextCommand(out, runes, command, next)
				if ok {
					continue
				}
				out = append(out, []rune(command)...)
			} else {
				out = append(out, []rune(command)...)
			}
			i = next
		case '{', '[', '(':
			out = trimTrailingLatexSpace(out)
			out = append(out, r)
			i++
		case '}', ']', ')':
			out = trimTrailingLatexSpace(out)
			out = append(out, r)
			i++
		case '^', '_':
			out = trimTrailingLatexSpace(out)
			out = append(out, r)
			i++
			for i < len(runes) && isLatexWhitespace(runes[i]) {
				i++
			}
		case '+', '=', '<', '>', '*', '/':
			out = appendLatexSpacedToken(out, string(r), nextNonWhitespace(runes, i+1))
			i++
		case '-':
			next := nextNonWhitespace(runes, i+1)
			if isLatexBinaryMinus(out, next) {
				out = appendLatexSpacedToken(out, "-", next)
			} else {
				out = trimTrailingLatexSpace(out)
				out = append(out, r)
			}
			i++
		case '&':
			out = appendLatexSpacedToken(out, "&", nextNonWhitespace(runes, i+1))
			i++
		case ',':
			out = trimTrailingLatexSpace(out)
			out = append(out, r)
			if next := nextNonWhitespace(runes, i+1); next != 0 && !isLatexCloser(next) {
				out = append(out, ' ')
			}
			i++
		default:
			out = append(out, r)
			i++
		}
	}

	return strings.TrimSpace(string(trimTrailingLatexSpace(out)))
}

func formatLatexEnvironmentLines(input string) string {
	input = regexp.MustCompile(`\\begin\{([^}]+)\}\s*`).ReplaceAllString(input, "\\begin{${1}}\n")
	input = regexp.MustCompile(`\s*\\end\{([^}]+)\}`).ReplaceAllString(input, "\n\\end{${1}}")
	input = regexp.MustCompile(`[ \t]*\n[ \t]*`).ReplaceAllString(input, "\n")
	input = regexp.MustCompile(`\n{2,}`).ReplaceAllString(input, "\n")
	return input
}

func trimLatexLines(input string) string {
	lines := strings.Split(input, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

func appendLatexTextCommand(out []rune, runes []rune, command string, start int) ([]rune, int, bool) {
	i := start
	for i < len(runes) && isLatexWhitespace(runes[i]) {
		i++
	}
	if i >= len(runes) || runes[i] != '{' {
		return out, start, false
	}

	group, next, ok := readBalancedLatexGroup(runes, i)
	if !ok {
		return out, start, false
	}

	out = append(out, []rune(command)...)
	out = append(out, '{')
	out = append(out, []rune(strings.TrimSpace(group))...)
	out = append(out, '}')
	return out, next, true
}

func readBalancedLatexGroup(runes []rune, start int) (string, int, bool) {
	if start >= len(runes) || runes[start] != '{' {
		return "", start, false
	}

	depth := 0
	for i := start; i < len(runes); i++ {
		switch runes[i] {
		case '\\':
			if i+1 < len(runes) {
				i++
			}
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return string(runes[start+1 : i]), i + 1, true
			}
		}
	}

	return "", start, false
}

func appendLatexSpaceIfNeeded(out []rune, next rune) []rune {
	if len(out) == 0 || next == 0 {
		return out
	}
	prev := lastNonWhitespace(out)
	if prev == 0 || prev == '\n' || prev == ' ' {
		return out
	}
	if isLatexOpener(prev) || isLatexCloser(next) || isLatexOperatorRune(prev) || isLatexOperatorRune(next) || next == '^' || next == '_' || next == '&' {
		return trimTrailingLatexSpace(out)
	}
	if out[len(out)-1] != ' ' && out[len(out)-1] != '\n' {
		out = append(out, ' ')
	}
	return out
}

func appendLatexSpacedToken(out []rune, token string, next rune) []rune {
	out = trimTrailingLatexSpace(out)
	if prev := lastNonWhitespace(out); prev != 0 && prev != '\n' && !isLatexOpener(prev) && !isLatexOperatorRune(prev) && prev != '&' {
		out = append(out, ' ')
	}
	out = append(out, []rune(token)...)
	if next != 0 && next != '\n' && !isLatexCloser(next) && !isLatexOperatorRune(next) && next != '&' {
		out = append(out, ' ')
	}
	return out
}

func appendLatexRowBreak(out []rune) []rune {
	out = trimTrailingLatexSpace(out)
	if prev := lastNonWhitespace(out); prev != 0 && prev != '\n' {
		out = append(out, ' ')
	}
	out = append(out, '\\', '\\', '\n')
	return out
}

func isLatexBinaryMinus(out []rune, next rune) bool {
	if next == 0 || isLatexCloser(next) || isLatexOperatorRune(next) || next == '&' {
		return false
	}
	prev := lastNonWhitespace(out)
	return prev != 0 && prev != '\n' && !isLatexOpener(prev) && !isLatexOperatorRune(prev) && prev != '^' && prev != '_' && prev != '&'
}

func readLatexCommand(runes []rune, start int) (string, int) {
	i := start + 1
	if i >= len(runes) {
		return `\`, i
	}

	if runes[i] == '\\' {
		return `\\`, i + 1
	}

	if isLatexLetter(runes[i]) {
		for i < len(runes) && isLatexLetter(runes[i]) {
			i++
		}
		return string(runes[start:i]), i
	}

	return string(runes[start : i+1]), i + 1
}

func nextNonWhitespace(runes []rune, start int) rune {
	for i := start; i < len(runes); i++ {
		if !isLatexWhitespace(runes[i]) {
			return runes[i]
		}
	}
	return 0
}

func lastNonWhitespace(runes []rune) rune {
	for i := len(runes) - 1; i >= 0; i-- {
		if !isLatexWhitespace(runes[i]) {
			return runes[i]
		}
	}
	return 0
}

func trimTrailingLatexSpace(runes []rune) []rune {
	for len(runes) > 0 {
		last := runes[len(runes)-1]
		if last != ' ' && last != '\t' {
			break
		}
		runes = runes[:len(runes)-1]
	}
	return runes
}

func isLatexWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

func isLatexLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isLatexOpener(r rune) bool {
	return r == '{' || r == '[' || r == '('
}

func isLatexCloser(r rune) bool {
	return r == '}' || r == ']' || r == ')' || r == ','
}

func isLatexOperatorRune(r rune) bool {
	switch r {
	case '+', '-', '=', '<', '>', '*', '/':
		return true
	default:
		return false
	}
}

func isLatexOperatorCommand(command string) bool {
	switch command {
	case `\le`, `\leq`, `\ge`, `\geq`, `\ne`, `\neq`, `\approx`, `\sim`, `\simeq`, `\equiv`,
		`\times`, `\cdot`, `\div`, `\pm`, `\mp`, `\to`, `\rightarrow`, `\leftarrow`, `\Rightarrow`,
		`\in`, `\notin`, `\subset`, `\subseteq`, `\supset`, `\supseteq`, `\cup`, `\cap`:
		return true
	default:
		return false
	}
}

func shouldUnescapeLatexCommand(command string, prev rune) bool {
	if !isKnownEscapedLatexCommand(command) {
		return false
	}
	if isAlwaysUnescapedLatexCommand(command) || isLatexOperatorCommand(`\`+command) {
		return true
	}
	return prev == 0 || isLatexOpener(prev) || isLatexOperatorRune(prev) || prev == '^' || prev == '_' || prev == '&' || prev == ','
}

func isAlwaysUnescapedLatexCommand(command string) bool {
	switch command {
	case "begin", "end", "left", "right":
		return true
	default:
		return false
	}
}

func isKnownEscapedLatexCommand(command string) bool {
	switch command {
	case "alpha", "beta", "gamma", "delta", "epsilon", "varepsilon", "zeta", "eta", "theta", "vartheta",
		"iota", "kappa", "lambda", "mu", "nu", "xi", "pi", "varpi", "rho", "varrho", "sigma", "varsigma",
		"tau", "upsilon", "phi", "varphi", "chi", "psi", "omega",
		"Alpha", "Beta", "Gamma", "Delta", "Epsilon", "Zeta", "Eta", "Theta", "Iota", "Kappa",
		"Lambda", "Mu", "Nu", "Xi", "Pi", "Rho", "Sigma", "Tau", "Upsilon", "Phi", "Chi", "Psi", "Omega",
		"frac", "dfrac", "tfrac", "binom", "sqrt", "sum", "prod", "coprod", "int", "oint", "lim",
		"sin", "cos", "tan", "cot", "sec", "csc", "arcsin", "arccos", "arctan", "sinh", "cosh", "tanh",
		"log", "ln", "lg", "exp", "max", "min", "det", "dim", "ker", "gcd", "arg",
		"vec", "bar", "hat", "tilde", "dot", "ddot", "overline", "underline", "text", "operatorname",
		"mathrm", "mathbf", "mathit", "mathsf", "mathtt", "left", "right", "begin", "end":
		return true
	default:
		return isLatexOperatorCommand(`\` + command)
	}
}

func previousNonWhitespaceRune(runes []rune, index int) rune {
	for i := index - 1; i >= 0; i-- {
		if !isLatexWhitespace(runes[i]) {
			return runes[i]
		}
	}
	return 0
}

func isLatexTextCommand(command string) bool {
	switch command {
	case `\text`, `\operatorname`, `\mbox`:
		return true
	default:
		return false
	}
}

func splitLinesKeepingEnd(content string) []string {
	if content == "" {
		return nil
	}

	lines := make([]string, 0, strings.Count(content, "\n")+1)
	start := 0
	for start < len(content) {
		idx := strings.IndexByte(content[start:], '\n')
		if idx == -1 {
			lines = append(lines, content[start:])
			break
		}
		end := start + idx + 1
		lines = append(lines, content[start:end])
		start = end
	}
	return lines
}

func trimLineEnding(line string) string {
	return strings.TrimRight(line, "\r\n")
}

func markdownFenceMarker(line string) (string, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if strings.HasPrefix(trimmed, "```") {
		return repeatedFenceMarker(trimmed, '`'), true
	}
	if strings.HasPrefix(trimmed, "~~~") {
		return repeatedFenceMarker(trimmed, '~'), true
	}
	return "", false
}

func repeatedFenceMarker(line string, marker byte) string {
	i := 0
	for i < len(line) && line[i] == marker {
		i++
	}
	return line[:i]
}

func isMarkdownFenceClose(line, marker string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	return strings.HasPrefix(trimmed, marker)
}

func findClosingBacktickRun(content string, start int) int {
	count := 0
	for start+count < len(content) && content[start+count] == '`' {
		count++
	}

	for i := start + count; i < len(content); i++ {
		if content[i] != '`' {
			continue
		}

		run := 0
		for i+run < len(content) && content[i+run] == '`' {
			run++
		}
		if run >= count {
			return i + run
		}
		i += run - 1
	}
	return -1
}

func findMathDelimiter(content string, start int) int {
	for i := start; i < len(content)-1; i++ {
		if strings.HasPrefix(content[i:], "$$") && !isEscapedAt(content, i) {
			return i
		}
	}
	return -1
}

func isEscapedAt(content string, index int) bool {
	backslashes := 0
	for i := index - 1; i >= 0 && content[i] == '\\'; i-- {
		backslashes++
	}
	return backslashes%2 == 1
}
