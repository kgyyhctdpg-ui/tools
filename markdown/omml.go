package markdown

import (
	"strings"
	"unicode/utf8"
)

func ommlToLatex(node *xmlNode) string {
	if node == nil {
		return ""
	}

	switch node.Name {
	case "oMath", "oMathPara", "e", "num", "den", "sub", "sup", "deg", "lim", "fName":
		return ommlSequence(node.Children)
	case "r":
		return ommlRun(node)
	case "t":
		return latexText(node.Text)
	case "f":
		return ommlFraction(node)
	case "sSub":
		return latexScripts(ommlChild(node, "e"), ommlChild(node, "sub"), "")
	case "sSup":
		return latexScripts(ommlChild(node, "e"), "", ommlChild(node, "sup"))
	case "sSubSup":
		return latexScripts(ommlChild(node, "e"), ommlChild(node, "sub"), ommlChild(node, "sup"))
	case "sPre":
		return latexPrescripts(ommlChild(node, "e"), ommlChild(node, "sub"), ommlChild(node, "sup"))
	case "rad":
		return ommlRadical(node)
	case "nary":
		return ommlNary(node)
	case "limLow":
		return latexScripts(ommlOperator(ommlChild(node, "e")), ommlChild(node, "lim"), "")
	case "limUpp":
		return "\\overset{" + ommlChild(node, "lim") + "}{" + ommlChild(node, "e") + "}"
	case "acc":
		return ommlAccent(node)
	case "bar":
		return ommlBar(node)
	case "d":
		return ommlDelimiter(node)
	case "m":
		return ommlMatrix(node)
	case "eqArr":
		return ommlEquationArray(node)
	case "func":
		return ommlFunction(node)
	case "groupChr":
		return ommlGroupCharacter(node)
	case "borderBox":
		return "\\boxed{" + ommlChild(node, "e") + "}"
	case "phant":
		return "\\phantom{" + ommlChild(node, "e") + "}"
	case "box":
		return ommlChild(node, "e")
	}

	if isOMMLProperty(node.Name) {
		return ""
	}
	return ommlSequence(node.Children)
}

func ommlSequence(nodes []*xmlNode) string {
	var out strings.Builder
	for _, node := range nodes {
		out.WriteString(ommlToLatex(node))
	}
	return out.String()
}

func ommlRun(node *xmlNode) string {
	var out strings.Builder
	for _, child := range node.Children {
		if child.Name == "t" {
			out.WriteString(latexText(child.Text))
			continue
		}
		if !isOMMLProperty(child.Name) {
			out.WriteString(ommlToLatex(child))
		}
	}
	return out.String()
}

func ommlFraction(node *xmlNode) string {
	numerator := ommlChild(node, "num")
	denominator := ommlChild(node, "den")
	fractionType := ommlPropertyValue(node.child("fPr"), "type")
	switch fractionType {
	case "lin":
		return "{" + numerator + "}/{" + denominator + "}"
	case "noBar":
		return "\\genfrac{}{}{0pt}{}{" + numerator + "}{" + denominator + "}"
	default:
		return "\\frac{" + numerator + "}{" + denominator + "}"
	}
}

func ommlRadical(node *xmlNode) string {
	degree := ommlChild(node, "deg")
	content := ommlChild(node, "e")
	hidden := ommlPropertyEnabled(node.child("radPr"), "degHide")
	if degree == "" || hidden {
		return "\\sqrt{" + content + "}"
	}
	return "\\sqrt[" + degree + "]{" + content + "}"
}

func ommlNary(node *xmlNode) string {
	symbol := ommlPropertyValue(node.child("naryPr"), "chr")
	if symbol == "" {
		symbol = "∫"
	}
	operator := latexNaryOperator(symbol)
	subscript := ""
	superscript := ""
	if !ommlPropertyEnabled(node.child("naryPr"), "subHide") {
		subscript = ommlChild(node, "sub")
	}
	if !ommlPropertyEnabled(node.child("naryPr"), "supHide") {
		superscript = ommlChild(node, "sup")
	}
	return latexScripts(operator, subscript, superscript) + ommlChild(node, "e")
}

func ommlAccent(node *xmlNode) string {
	character := ommlPropertyValue(node.child("accPr"), "chr")
	command := map[string]string{
		"̂": "hat", "^": "hat", "̄": "bar", "¯": "bar", "~": "tilde", "̃": "tilde",
		"⃗": "vec", "→": "vec", "̇": "dot", ".": "dot", "̈": "ddot", "¨": "ddot",
		"́": "acute", "`": "grave", "̀": "grave", "̆": "breve", "̌": "check",
	}[character]
	if command == "" {
		command = "hat"
	}
	return "\\" + command + "{" + ommlChild(node, "e") + "}"
}

func ommlBar(node *xmlNode) string {
	position := ommlPropertyValue(node.child("barPr"), "pos")
	command := "overline"
	if position == "bot" {
		command = "underline"
	}
	return "\\" + command + "{" + ommlChild(node, "e") + "}"
}

func ommlDelimiter(node *xmlNode) string {
	properties := node.child("dPr")
	begin := ommlPropertyValue(properties, "begChr")
	end := ommlPropertyValue(properties, "endChr")
	separator := ommlPropertyValue(properties, "sepChr")
	if begin == "" {
		begin = "("
	}
	if end == "" {
		end = ")"
	}
	if separator == "" {
		separator = "|"
	}
	arguments := make([]string, 0)
	for _, child := range node.Children {
		if child.Name == "e" {
			arguments = append(arguments, ommlToLatex(child))
		}
	}
	return "\\left" + latexDelimiter(begin) + strings.Join(arguments, "\\middle"+latexDelimiter(separator)) + "\\right" + latexDelimiter(end)
}

func ommlMatrix(node *xmlNode) string {
	rows := make([]string, 0)
	for _, row := range node.children("mr") {
		cells := make([]string, 0)
		for _, cell := range row.children("e") {
			cells = append(cells, ommlToLatex(cell))
		}
		rows = append(rows, strings.Join(cells, " & "))
	}
	return "\\begin{matrix}" + strings.Join(rows, " \\\\ ") + "\\end{matrix}"
}

func ommlEquationArray(node *xmlNode) string {
	rows := make([]string, 0)
	for _, row := range node.children("e") {
		rows = append(rows, ommlToLatex(row))
	}
	return "\\begin{aligned}" + strings.Join(rows, " \\\\ ") + "\\end{aligned}"
}

func ommlFunction(node *xmlNode) string {
	name := strings.TrimSpace(ommlChild(node, "fName"))
	argument := ommlChild(node, "e")
	known := map[string]bool{
		"sin": true, "cos": true, "tan": true, "cot": true, "sec": true, "csc": true,
		"arcsin": true, "arccos": true, "arctan": true, "sinh": true, "cosh": true,
		"tanh": true, "log": true, "ln": true, "exp": true, "max": true, "min": true,
		"lim": true, "det": true, "gcd": true,
	}
	if known[name] {
		name = "\\" + name
	} else {
		name = "\\operatorname{" + name + "}"
	}
	return name + "\\left(" + argument + "\\right)"
}

func ommlGroupCharacter(node *xmlNode) string {
	character := ommlPropertyValue(node.child("groupChrPr"), "chr")
	position := ommlPropertyValue(node.child("groupChrPr"), "pos")
	content := ommlChild(node, "e")
	if character == "⏟" || position == "bot" {
		return "\\underbrace{" + content + "}"
	}
	return "\\overbrace{" + content + "}"
}

func ommlChild(node *xmlNode, name string) string {
	if node == nil {
		return ""
	}
	return ommlToLatex(node.child(name))
}

func ommlPropertyValue(properties *xmlNode, name string) string {
	if properties == nil {
		return ""
	}
	property := properties.child(name)
	if property == nil {
		return ""
	}
	return property.attr("val")
}

func ommlPropertyEnabled(properties *xmlNode, name string) bool {
	if properties == nil {
		return false
	}
	property := properties.child(name)
	if property == nil {
		return false
	}
	switch strings.ToLower(property.attr("val")) {
	case "0", "false", "off", "no":
		return false
	default:
		return true
	}
}

func isOMMLProperty(name string) bool {
	return name == "ctrlPr" || name == "argPr" || name == "rPr" || strings.HasSuffix(name, "Pr") ||
		name == "chr" || name == "begChr" || name == "endChr" || name == "sepChr" ||
		name == "type" || name == "pos" || name == "degHide" || name == "subHide" || name == "supHide"
}

func latexScripts(base, subscript, superscript string) string {
	if base == "" {
		base = "{}"
	} else if utf8.RuneCountInString(base) > 1 && !strings.HasPrefix(base, "\\") && !strings.HasPrefix(base, "{") {
		base = "{" + base + "}"
	}
	if subscript != "" {
		base += "_{" + subscript + "}"
	}
	if superscript != "" {
		base += "^{" + superscript + "}"
	}
	return base
}

func latexPrescripts(base, subscript, superscript string) string {
	result := "{}"
	if subscript != "" {
		result += "_{" + subscript + "}"
	}
	if superscript != "" {
		result += "^{" + superscript + "}"
	}
	return result + "{" + base + "}"
}

func ommlOperator(value string) string {
	trimmed := strings.TrimSpace(value)
	known := map[string]bool{"lim": true, "max": true, "min": true, "det": true, "gcd": true}
	if known[trimmed] {
		return "\\" + trimmed
	}
	return value
}

func latexNaryOperator(value string) string {
	if command, ok := map[string]string{
		"∫": "\\int", "∬": "\\iint", "∭": "\\iiint", "∮": "\\oint",
		"∑": "\\sum", "∏": "\\prod", "∐": "\\coprod", "⋂": "\\bigcap",
		"⋃": "\\bigcup", "⋀": "\\bigwedge", "⋁": "\\bigvee", "⊕": "\\bigoplus",
		"⊗": "\\bigotimes", "⊙": "\\bigodot", "⊎": "\\biguplus",
	}[value]; ok {
		return command
	}
	return latexText(value)
}

func latexDelimiter(value string) string {
	switch value {
	case "{":
		return "\\{"
	case "}":
		return "\\}"
	case "[", "]", "(", ")", "|", ".":
		return value
	case "‖", "∥":
		return "\\|"
	case "⌊":
		return "\\lfloor"
	case "⌋":
		return "\\rfloor"
	case "⌈":
		return "\\lceil"
	case "⌉":
		return "\\rceil"
	case "⟨", "〈":
		return "\\langle"
	case "⟩", "〉":
		return "\\rangle"
	default:
		return latexText(value)
	}
}

func latexText(value string) string {
	var out strings.Builder
	for _, character := range value {
		if replacement, ok := latexSymbols[character]; ok {
			out.WriteString(replacement)
			continue
		}
		switch character {
		case '\\':
			out.WriteString("\\backslash ")
		case '{':
			out.WriteString("\\{")
		case '}':
			out.WriteString("\\}")
		case '#', '$', '%', '&', '_':
			out.WriteByte('\\')
			out.WriteRune(character)
		case '^':
			out.WriteString("\\hat{}")
		case '~':
			out.WriteString("\\sim ")
		default:
			out.WriteRune(character)
		}
	}
	return out.String()
}

var latexSymbols = map[rune]string{
	'α': "\\alpha ", 'β': "\\beta ", 'γ': "\\gamma ", 'δ': "\\delta ", 'ε': "\\epsilon ",
	'ζ': "\\zeta ", 'η': "\\eta ", 'θ': "\\theta ", 'ι': "\\iota ", 'κ': "\\kappa ",
	'λ': "\\lambda ", 'μ': "\\mu ", 'ν': "\\nu ", 'ξ': "\\xi ", 'π': "\\pi ",
	'ρ': "\\rho ", 'σ': "\\sigma ", 'τ': "\\tau ", 'υ': "\\upsilon ", 'φ': "\\phi ",
	'χ': "\\chi ", 'ψ': "\\psi ", 'ω': "\\omega ", 'ϑ': "\\vartheta ", 'ς': "\\varsigma ",
	'Γ': "\\Gamma ", 'Δ': "\\Delta ", 'Θ': "\\Theta ", 'Λ': "\\Lambda ", 'Ξ': "\\Xi ",
	'Π': "\\Pi ", 'Σ': "\\Sigma ", 'Υ': "\\Upsilon ", 'Φ': "\\Phi ", 'Ψ': "\\Psi ", 'Ω': "\\Omega ",
	'±': "\\pm ", '∓': "\\mp ", '×': "\\times ", '÷': "\\div ", '·': "\\cdot ",
	'∞': "\\infty ", '∂': "\\partial ", '∇': "\\nabla ", '∀': "\\forall ", '∃': "\\exists ",
	'∅': "\\emptyset ", '∈': "\\in ", '∉': "\\notin ", '∋': "\\ni ", '⊂': "\\subset ",
	'⊃': "\\supset ", '⊆': "\\subseteq ", '⊇': "\\supseteq ", '∪': "\\cup ", '∩': "\\cap ",
	'≤': "\\leq ", '≥': "\\geq ", '≠': "\\neq ", '≈': "\\approx ", '≡': "\\equiv ",
	'∼': "\\sim ", '≅': "\\cong ", '∝': "\\propto ", '⊥': "\\perp ", '∥': "\\parallel ",
	'→': "\\to ", '←': "\\leftarrow ", '↔': "\\leftrightarrow ", '⇒': "\\Rightarrow ",
	'⇐': "\\Leftarrow ", '⇔': "\\Leftrightarrow ", '↦': "\\mapsto ", '…': "\\ldots ",
	'⋯': "\\cdots ", '⋮': "\\vdots ", '⋱': "\\ddots ", '¬': "\\neg ",
}
