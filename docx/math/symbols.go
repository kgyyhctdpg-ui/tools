package math

// GreekLetters maps LaTeX Greek letter commands to Unicode characters.
var GreekLetters = map[string]string{
	// Lowercase Greek letters
	"alpha":   "α",
	"beta":    "β",
	"gamma":   "γ",
	"delta":   "δ",
	"epsilon": "ε",
	"zeta":    "ζ",
	"eta":     "η",
	"theta":   "θ",
	"iota":    "ι",
	"kappa":   "κ",
	"lambda":  "λ",
	"mu":      "μ",
	"nu":      "ν",
	"xi":      "ξ",
	"omicron": "ο",
	"pi":      "π",
	"rho":     "ρ",
	"sigma":   "σ",
	"tau":     "τ",
	"upsilon": "υ",
	"phi":     "φ",
	"chi":     "χ",
	"psi":     "ψ",
	"omega":   "ω",

	// Uppercase Greek letters
	"Alpha":   "Α",
	"Beta":    "Β",
	"Gamma":   "Γ",
	"Delta":   "Δ",
	"Epsilon": "Ε",
	"Zeta":    "Ζ",
	"Eta":     "Η",
	"Theta":   "Θ",
	"Iota":    "Ι",
	"Kappa":   "Κ",
	"Lambda":  "Λ",
	"Mu":      "Μ",
	"Nu":      "Ν",
	"Xi":      "Ξ",
	"Omicron": "Ο",
	"Pi":      "Π",
	"Rho":     "Ρ",
	"Sigma":   "Σ",
	"Tau":     "Τ",
	"Upsilon": "Υ",
	"Phi":     "Φ",
	"Chi":     "Χ",
	"Psi":     "Ψ",
	"Omega":   "Ω",

	// Variant forms
	"varepsilon": "ε",
	"varphi":     "φ",
	"varpi":      "π",
	"varrho":     "ρ",
	"varsigma":   "ς",
	"vartheta":   "ϑ",
}

// MathOperators maps LaTeX operator commands to Unicode characters.
var MathOperators = map[string]string{
	// N-ary operators
	"int":       "∫",
	"oint":      "∮",
	"sum":       "∑",
	"prod":      "∏",
	"coprod":    "∐",
	"bigcap":    "⋂",
	"bigcup":    "⋃",
	"bigsqcup":  "⊔",
	"bigvee":    "⋁",
	"bigwedge":  "⋀",
	"bigodot":   "⊙",
	"bigotimes": "⊗",
	"bigoplus":  "⊕",
	"biguplus":  "⊎",

	// Binary operators
	"pm":           "±",
	"mp":           "∓",
	"times":        "×",
	"div":          "÷",
	"cdot":         "·",
	"star":         "⋆",
	"asterisk":     "∗",
	"circ":         "∘",
	"bullet":       "∙",
	"oplus":        "⊕",
	"otimes":       "⊗",
	"odot":         "⊙",
	"oslash":       "⊘",
	"cup":          "∪",
	"cap":          "∩",
	"uplus":        "⊎",
	"sqcap":        "⊓",
	"sqcup":        "⊔",
	"triangle":     "△",
	"triangledown": "▽",
	"box":          "□",
	"diamond":      "◇",
	"wedge":        "∧",
	"vee":          "∨",
	"setminus":     "∖",
	"wr":           "≀",
	"amalg":        "⨿",

	// Special symbols
	"infty":          "∞",
	"partial":        "∂",
	"nabla":          "∇",
	"hbar":           "ℏ",
	"dagger":         "†",
	"ddagger":        "‡",
	"ell":            "ℓ",
	"Re":             "ℜ",
	"Im":             "ℑ",
	"aleph":          "ℵ",
	"beth":           "ℶ",
	"gimel":          "ℷ",
	"daleth":         "ℸ",
	"forall":         "∀",
	"exists":         "∃",
	"nexists":        "∄",
	"emptyset":       "∅",
	"neg":            "¬",
	"lnot":           "¬",
	"top":            "⊤",
	"bot":            "⊥",
	"angle":          "∠",
	"measuredangle":  "∡",
	"sphericalangle": "∢",
	"prime":          "′",
	"dprime":         "″",
	"tprime":         "‴",
	"backprime":      "‵",
}

// RelationSymbols maps LaTeX relation commands to Unicode characters.
var RelationSymbols = map[string]string{
	// Equality and inequality
	"eq":     "=",
	"neq":    "≠",
	"ne":     "≠",
	"equiv":  "≡",
	"approx": "≈",
	"cong":   "≅",
	"sim":    "∼",
	"simeq":  "≃",
	"propto": "∝",
	"models": "⊧",

	// Less/greater than
	"lt":         "<",
	"gt":         ">",
	"leq":        "≤",
	"le":         "≤",
	"geq":        "≥",
	"ge":         "≥",
	"ll":         "≪",
	"gg":         "≫",
	"nless":      "≮",
	"ngtr":       "≯",
	"nleq":       "≰",
	"ngeq":       "≱",
	"lesssim":    "≲",
	"gtrsim":     "≳",
	"lessapprox": "⪅",
	"gtrapprox":  "⪆",
	"prec":       "≺",
	"succ":       "≻",
	"preceq":     "≼",
	"succeq":     "≽",
	"precsim":    "≾",
	"succsim":    "≿",
	"nprec":      "⊀",
	"nsucc":      "⊁",

	// Set relations
	"in":         "∈",
	"notin":      "∉",
	"ni":         "∋",
	"notni":      "∌",
	"subset":     "⊂",
	"supset":     "⊃",
	"subseteq":   "⊆",
	"supseteq":   "⊇",
	"nsubset":    "⊄",
	"nsupset":    "⊅",
	"nsubseteq":  "⊈",
	"nsupseteq":  "⊉",
	"sqsubset":   "⊏",
	"sqsupset":   "⊐",
	"sqsubseteq": "⊑",
	"sqsupseteq": "⊒",

	// Arrows
	"rightarrow":        "→",
	"to":                "→",
	"leftarrow":         "←",
	"gets":              "←",
	"leftrightarrow":    "↔",
	"Rightarrow":        "⇒",
	"Leftarrow":         "⇐",
	"Leftrightarrow":    "⇔",
	"uparrow":           "↑",
	"downarrow":         "↓",
	"updownarrow":       "↕",
	"Uparrow":           "⇑",
	"Downarrow":         "⇓",
	"Updownarrow":       "⇕",
	"nearrow":           "↗",
	"searrow":           "↘",
	"swarrow":           "↙",
	"nwarrow":           "↖",
	"mapsto":            "↦",
	"longmapsto":        "⟼",
	"hookrightarrow":    "↪",
	"hookleftarrow":     "↩",
	"looparrowright":    "↬",
	"looparrowleft":     "↫",
	"leftharpoonup":     "↼",
	"rightharpoonup":    "⇀",
	"leftharpoondown":   "↽",
	"rightharpoondown":  "⇁",
	"upharpoonleft":     "↿",
	"upharpoonright":    "↾",
	"downharpoonleft":   "⇃",
	"downharpoonright":  "⇂",
	"rightsquigarrow":   "⇝",
	"leadsto":           "⇝",
	"twoheadrightarrow": "↠",
	"twoheadleftarrow":  "↞",

	// Misc relations
	"perp":      "⊥",
	"mid":       "∣",
	"nmid":      "∤",
	"parallel":  "∥",
	"nparallel": "∦",
	"smile":     "⌣",
	"frown":     "⌢",
	"bowtie":    "⋈",
	"Join":      "⨝",
	"diamond":   "◇",
	"vdots":     "⋮",
	"ddots":     "⋱",
	"cdots":     "⋯",
	"colon":     ":",
	"dblcolon":  "∷",
	"coloneqq":  "≔",
	"eqqcolon":  "≕",
}

// AccentSymbols maps LaTeX accent commands to Unicode characters and OMML accent types.
var AccentSymbols = map[string]AccentInfo{
	"vec":   {Char: "⃗", Type: "arrow"},
	"bar":   {Char: "̄", Type: "bar"},
	"hat":   {Char: "̂", Type: "hat"},
	"tilde": {Char: "̃", Type: "tilde"},
	"dot":   {Char: "̇", Type: "dot"},
	"ddot":  {Char: "̈", Type: "ddot"},
	"acute": {Char: "́", Type: "acute"},
	"grave": {Char: "̀", Type: "grave"},
	"breve": {Char: "̆", Type: "breve"},
	"check": {Char: "̌", Type: "check"},
}

// AccentInfo contains information about an accent symbol.
type AccentInfo struct {
	Char string // Unicode combining character
	Type string // OMML accent type identifier
}

// FunctionNames maps LaTeX function names.
var FunctionNames = map[string]string{
	"sin":    "sin",
	"cos":    "cos",
	"tan":    "tan",
	"cot":    "cot",
	"sec":    "sec",
	"csc":    "csc",
	"arcsin": "arcsin",
	"arccos": "arccos",
	"arctan": "arctan",
	"sinh":   "sinh",
	"cosh":   "cosh",
	"tanh":   "tanh",
	"coth":   "coth",
	"log":    "log",
	"ln":     "ln",
	"lg":     "lg",
	"exp":    "exp",
	"max":    "max",
	"min":    "min",
	"sup":    "sup",
	"inf":    "inf",
	"lim":    "lim",
	"limsup": "lim sup",
	"liminf": "lim inf",
	"det":    "det",
	"dim":    "dim",
	"ker":    "ker",
	"gcd":    "gcd",
	"arg":    "arg",
	"mod":    "mod",
	"bmod":   "mod",
	"pmod":   "mod",
}

// DelimiterPairs maps delimiter names to left/right characters.
var DelimiterPairs = map[string]DelimiterInfo{
	"paren":          {Left: "(", Right: ")"},
	"bracket":        {Left: "[", Right: "]"},
	"brace":          {Left: "{", Right: "}"},
	"angle":          {Left: "⟨", Right: "⟩"},
	"ceil":           {Left: "⌈", Right: "⌉"},
	"floor":          {Left: "⌊", Right: "⌋"},
	"lfloor":         {Left: "⌊", Right: ""},
	"rfloor":         {Left: "", Right: "⌋"},
	"lceil":          {Left: "⌈", Right: ""},
	"rceil":          {Left: "", Right: "⌉"},
	"langle":         {Left: "⟨", Right: "⟩"},
	"rangle":         {Left: "⟩", Right: "⟩"},
	"ulcorner":       {Left: "⌜", Right: ""},
	"urcorner":       {Left: "", Right: "⌝"},
	"llcorner":       {Left: "⌞", Right: ""},
	"lrcorner":       {Left: "", Right: "⌟"},
	"vert":           {Left: "|", Right: "|"},
	"Vert":           {Left: "‖", Right: "‖"},
	"lvert":          {Left: "|", Right: ""},
	"rvert":          {Left: "", Right: "|"},
	"lVert":          {Left: "‖", Right: ""},
	"rVert":          {Left: "", Right: "‖"},
	"leftarrow":      {Left: "←", Right: ""},
	"rightarrow":     {Left: "", Right: "→"},
	"leftrightarrow": {Left: "↔", Right: "↔"},
	"Leftarrow":      {Left: "⇐", Right: ""},
	"Rightarrow":     {Left: "", Right: "⇒"},
	"Leftrightarrow": {Left: "⇔", Right: "⇔"},
	"uparrow":        {Left: "↑", Right: ""},
	"downarrow":      {Left: "", Right: "↓"},
	"updownarrow":    {Left: "↕", Right: "↕"},
	"Uparrow":        {Left: "⇑", Right: ""},
	"Downarrow":      {Left: "", Right: "⇓"},
	"Updownarrow":    {Left: "⇕", Right: "⇕"},
}

// DelimiterInfo contains delimiter character information.
type DelimiterInfo struct {
	Left  string
	Right string
}

// NaryOperators maps LaTeX n-ary operator names to OMML operator characters.
var NaryOperators = map[string]string{
	"int":       "∫",
	"oint":      "∮",
	"sum":       "∑",
	"prod":      "∏",
	"coprod":    "∐",
	"bigcap":    "⋂",
	"bigcup":    "⋃",
	"bigsqcup":  "⊔",
	"bigvee":    "⋁",
	"bigwedge":  "⋀",
	"bigodot":   "⊙",
	"bigotimes": "⊗",
	"bigoplus":  "⊕",
	"biguplus":  "⊎",
	"iint":      "∬",
	"iiint":     "∭",
	"idotsint":  "∫⋯∫",
}

// IsGreekLetter checks if a command is a Greek letter.
func IsGreekLetter(cmd string) bool {
	_, ok := GreekLetters[cmd]
	return ok
}

// IsMathOperator checks if a command is a math operator.
func IsMathOperator(cmd string) bool {
	_, ok := MathOperators[cmd]
	return ok
}

// IsRelationSymbol checks if a command is a relation symbol.
func IsRelationSymbol(cmd string) bool {
	_, ok := RelationSymbols[cmd]
	return ok
}

// IsFunctionName checks if a command is a function name.
func IsFunctionName(cmd string) bool {
	_, ok := FunctionNames[cmd]
	return ok
}

// IsAccent checks if a command is an accent.
func IsAccent(cmd string) bool {
	_, ok := AccentSymbols[cmd]
	return ok
}

// IsNaryOperator checks if a command is an n-ary operator.
func IsNaryOperator(cmd string) bool {
	_, ok := NaryOperators[cmd]
	return ok
}

// GetSymbol returns the Unicode character for a LaTeX command.
func GetSymbol(cmd string) string {
	if s, ok := GreekLetters[cmd]; ok {
		return s
	}
	if s, ok := MathOperators[cmd]; ok {
		return s
	}
	if s, ok := RelationSymbols[cmd]; ok {
		return s
	}
	if s, ok := NaryOperators[cmd]; ok {
		return s
	}
	return ""
}
