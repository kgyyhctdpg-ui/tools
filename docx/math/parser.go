package math

import (
	"fmt"
	"strings"
	"unicode"
)

// TokenType represents the type of a lexical token.
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenError
	TokenBackslash       // \
	TokenLBrace          // {
	TokenRBrace          // }
	TokenLBracket        // [
	TokenRBracket        // ]
	TokenLParen          // (
	TokenRParen          // )
	TokenCaret           // ^
	TokenUnderscore      // _
	TokenCommand         // \frac, \sqrt, etc.
	TokenText            // plain text
	TokenNumber          // number
	TokenOperator        // +, -, *, /, =, etc.
	TokenAmpersand       // & (for matrix)
	TokenDoubleBackslash // \\ (for matrix row separator)
)

// Token represents a lexical token.
type Token struct {
	Type  TokenType
	Value string
	Pos   int
}

// Lexer performs lexical analysis on LaTeX formula strings.
type Lexer struct {
	input string
	pos   int
	width int
}

// NewLexer creates a new lexer for the given input.
func NewLexer(input string) *Lexer {
	return &Lexer{input: input}
}

// next returns the next character from the input.
func (l *Lexer) next() rune {
	if l.pos >= len(l.input) {
		l.width = 0
		return 0
	}
	r, w := rune(l.input[l.pos]), 1
	if r >= 0x80 {
		r, w = utf8DecodeRuneInString(l.input[l.pos:])
	}
	l.pos += w
	l.width = w
	return r
}

// backup moves back one character.
func (l *Lexer) backup() {
	l.pos -= l.width
}

// peek returns the next character without consuming it.
func (l *Lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// NextToken returns the next token from the lexer.
func (l *Lexer) NextToken() Token {
	startPos := l.pos

	r := l.next()
	if r == 0 {
		return Token{Type: TokenEOF, Pos: startPos}
	}

	switch r {
	case '\\':
		// Check for \\ (row separator in matrix)
		if l.peek() == '\\' {
			l.next()
			return Token{Type: TokenDoubleBackslash, Value: "\\\\", Pos: startPos}
		}
		// Parse command
		cmd := l.parseCommand()
		return Token{Type: TokenCommand, Value: cmd, Pos: startPos}

	case '{':
		return Token{Type: TokenLBrace, Value: "{", Pos: startPos}
	case '}':
		return Token{Type: TokenRBrace, Value: "}", Pos: startPos}
	case '[':
		return Token{Type: TokenLBracket, Value: "[", Pos: startPos}
	case ']':
		return Token{Type: TokenRBracket, Value: "]", Pos: startPos}
	case '(':
		return Token{Type: TokenLParen, Value: "(", Pos: startPos}
	case ')':
		return Token{Type: TokenRParen, Value: ")", Pos: startPos}
	case '^':
		return Token{Type: TokenCaret, Value: "^", Pos: startPos}
	case '_':
		return Token{Type: TokenUnderscore, Value: "_", Pos: startPos}
	case '&':
		return Token{Type: TokenAmpersand, Value: "&", Pos: startPos}
	case '+', '-', '*', '/', '=', '<', '>', '!', '|':
		return Token{Type: TokenOperator, Value: string(r), Pos: startPos}
	case ',':
		return Token{Type: TokenText, Value: ",", Pos: startPos}
	case ' ':
		// Skip spaces in math mode (except for specific spaces like \, \;)
		return l.NextToken()
	}

	// Number
	if unicode.IsDigit(r) || r == '.' {
		l.backup()
		num := l.parseNumber()
		return Token{Type: TokenNumber, Value: num, Pos: startPos}
	}

	// Text (letters and other characters)
	if unicode.IsLetter(r) {
		l.backup()
		text := l.parseText()
		return Token{Type: TokenText, Value: text, Pos: startPos}
	}

	// Any other character as text
	return Token{Type: TokenText, Value: string(r), Pos: startPos}
}

// parseCommand parses a LaTeX command after the backslash.
func (l *Lexer) parseCommand() string {
	var cmd strings.Builder

	// Parse command name (letters only)
	for {
		r := l.peek()
		if unicode.IsLetter(r) {
			l.next()
			cmd.WriteRune(r)
		} else {
			break
		}
	}

	// Handle single-character commands like \*, \+, etc
	if cmd.Len() == 0 {
		r := l.next()
		if r != 0 && r != ' ' {
			cmd.WriteRune(r)
		}
	}

	return cmd.String()
}

// parseNumber parses a number.
func (l *Lexer) parseNumber() string {
	var num strings.Builder
	hasDot := false

	for {
		r := l.peek()
		if unicode.IsDigit(r) {
			l.next()
			num.WriteRune(r)
		} else if r == '.' && !hasDot {
			l.next()
			num.WriteRune('.')
			hasDot = true
		} else {
			break
		}
	}

	return num.String()
}

// parseText parses text (letters and other characters).
func (l *Lexer) parseText() string {
	var text strings.Builder

	for {
		r := l.peek()
		if unicode.IsLetter(r) {
			l.next()
			text.WriteRune(r)
		} else {
			break
		}
	}

	return text.String()
}

// ParseError represents a parsing error.
type ParseError struct {
	Message string
	Pos     int
	Input   string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("parse error at position %d: %s", e.Pos, e.Message)
}

// Parser performs syntactic analysis on LaTeX formulas.
type Parser struct {
	lexer  *Lexer
	tokens []Token
	pos    int
}

// NewParser creates a new parser for the given LaTeX formula.
func NewParser(latex string) *Parser {
	lexer := NewLexer(latex)
	return &Parser{lexer: lexer}
}

// Parse parses the LaTeX formula and returns an AST expression.
func (p *Parser) Parse() (Exp, error) {
	// Tokenize the input
	p.tokenize()
	if p.pos >= len(p.tokens) {
		return nil, &ParseError{Message: "empty input", Pos: 0, Input: p.lexer.input}
	}

	return p.parseExpr()
}

// tokenize converts the input into tokens.
func (p *Parser) tokenize() {
	for {
		tok := p.lexer.NextToken()
		p.tokens = append(p.tokens, tok)
		if tok.Type == TokenEOF || tok.Type == TokenError {
			break
		}
	}
}

// current returns the current token.
func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

// advance moves to the next token.
func (p *Parser) advance() Token {
	tok := p.current()
	p.pos++
	return tok
}

// expect advances and returns the current token if it matches the expected type.
func (p *Parser) expect(typ TokenType) (Token, error) {
	tok := p.current()
	if tok.Type != typ {
		return Token{}, &ParseError{
			Message: fmt.Sprintf("expected %v, got %v", typ, tok.Type),
			Pos:     tok.Pos,
			Input:   p.lexer.input,
		}
	}
	p.advance()
	return tok, nil
}

// parseExpr parses an expression (lowest precedence).
func (p *Parser) parseExpr() (Exp, error) {
	return p.parseSubSup()
}

// parseSubSup parses subscript and superscript expressions.
func (p *Parser) parseSubSup() (Exp, error) {
	// Parse base (may include sequence)
	base, err := p.parseTerm()
	if err != nil {
		return nil, err
	}

	var sub, sup Exp

	// Handle subscript
	if p.current().Type == TokenUnderscore {
		p.advance()
		if p.current().Type == TokenLBrace {
			p.advance()
			sub, err = p.parseExpr()
			if err != nil {
				return nil, err
			}
			_, err = p.expect(TokenRBrace)
			if err != nil {
				return nil, err
			}
		} else {
			sub, err = p.parseAtom()
			if err != nil {
				return nil, err
			}
		}
	}

	// Handle superscript (can come after subscript)
	if p.current().Type == TokenCaret {
		p.advance()
		if p.current().Type == TokenLBrace {
			p.advance()
			sup, err = p.parseExpr()
			if err != nil {
				return nil, err
			}
			_, err = p.expect(TokenRBrace)
			if err != nil {
				return nil, err
			}
		} else {
			sup, err = p.parseAtom()
			if err != nil {
				return nil, err
			}
		}
	}

	// Build the result
	var result Exp
	if sub != nil || sup != nil {
		result = &ESubSuperscript{Base: base, Subscript: sub, Superscript: sup}
	} else {
		result = base
	}

	// Continue parsing if there are more terms (like + y^2 = z^2)
	// This handles expressions like "x^2 + y^2 = z^2"
	exprs := []Exp{result}

	for {
		tok := p.current()
		if tok.Type == TokenOperator {
			p.advance()
			exprs = append(exprs, &EOperator{Op: tok.Value})

			// Parse the next term (with potential subscripts/superscripts)
			nextExpr, err := p.parseSubSupRecursive()
			if err != nil {
				break
			}
			exprs = append(exprs, nextExpr)
		} else if tok.Type == TokenEOF || tok.Type == TokenRBrace ||
			tok.Type == TokenRBracket || tok.Type == TokenRParen ||
			tok.Type == TokenAmpersand || tok.Type == TokenDoubleBackslash {
			break
		} else {
			break
		}
	}

	if len(exprs) == 1 {
		return exprs[0], nil
	}
	return &EGrouped{Exps: exprs}, nil
}

// parseSubSupRecursive parses subscript/superscript without the continuation logic
func (p *Parser) parseSubSupRecursive() (Exp, error) {
	base, err := p.parseTerm()
	if err != nil {
		return nil, err
	}

	var sub, sup Exp

	// Handle subscript
	if p.current().Type == TokenUnderscore {
		p.advance()
		if p.current().Type == TokenLBrace {
			p.advance()
			sub, err = p.parseExpr()
			if err != nil {
				return nil, err
			}
			_, err = p.expect(TokenRBrace)
			if err != nil {
				return nil, err
			}
		} else {
			sub, err = p.parseAtom()
			if err != nil {
				return nil, err
			}
		}
	}

	// Handle superscript
	if p.current().Type == TokenCaret {
		p.advance()
		if p.current().Type == TokenLBrace {
			p.advance()
			sup, err = p.parseExpr()
			if err != nil {
				return nil, err
			}
			_, err = p.expect(TokenRBrace)
			if err != nil {
				return nil, err
			}
		} else {
			sup, err = p.parseAtom()
			if err != nil {
				return nil, err
			}
		}
	}

	if sub != nil || sup != nil {
		return &ESubSuperscript{Base: base, Subscript: sub, Superscript: sup}, nil
	}

	return base, nil
}

// parseTerm parses terms with operators.
func (p *Parser) parseTerm() (Exp, error) {
	exprs := []Exp{}

	for {
		expr, err := p.parseFactor()
		if err != nil {
			// If we have at least one expression, return it as a sequence
			if len(exprs) > 0 {
				if len(exprs) == 1 {
					return exprs[0], nil
				}
				return &EGrouped{Exps: exprs}, nil
			}
			return nil, err
		}
		exprs = append(exprs, expr)

		// Check if we should continue parsing
		tok := p.current()
		if tok.Type == TokenOperator {
			p.advance()
			exprs = append(exprs, &EOperator{Op: tok.Value})
		} else if tok.Type == TokenEOF || tok.Type == TokenRBrace ||
			tok.Type == TokenRBracket || tok.Type == TokenRParen ||
			tok.Type == TokenAmpersand || tok.Type == TokenDoubleBackslash ||
			tok.Type == TokenUnderscore || tok.Type == TokenCaret {
			break
		}
	}

	if len(exprs) == 0 {
		return nil, &ParseError{Message: "empty expression", Pos: p.current().Pos, Input: p.lexer.input}
	}
	if len(exprs) == 1 {
		return exprs[0], nil
	}
	return &EGrouped{Exps: exprs}, nil
}

// parseFactor parses factors (atomic expressions with possible grouping).
func (p *Parser) parseFactor() (Exp, error) {
	return p.parseAtom()
}

// parseAtom parses atomic expressions.
func (p *Parser) parseAtom() (Exp, error) {
	tok := p.current()

	switch tok.Type {
	case TokenEOF:
		return nil, &ParseError{Message: "unexpected end of input", Pos: tok.Pos, Input: p.lexer.input}

	case TokenLBrace:
		// Grouped expression
		p.advance()
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		_, err = p.expect(TokenRBrace)
		if err != nil {
			return nil, err
		}
		// Unwrap EGrouped if it's a single group
		if g, ok := expr.(*EGrouped); ok {
			return g, nil
		}
		return &EGrouped{Exps: []Exp{expr}}, nil

	case TokenLParen:
		// Parenthesized expression - treat as delimiter
		p.advance()
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		_, err = p.expect(TokenRParen)
		if err != nil {
			return nil, err
		}
		return &EDelimited{LeftDelimiter: "(", RightDelimiter: ")", Exps: AsExps(expr)}, nil

	case TokenLBracket:
		// Bracketed expression
		p.advance()
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		_, err = p.expect(TokenRBracket)
		if err != nil {
			return nil, err
		}
		return &EDelimited{LeftDelimiter: "[", RightDelimiter: "]", Exps: AsExps(expr)}, nil

	case TokenNumber:
		p.advance()
		return &ENumber{Value: tok.Value}, nil

	case TokenText:
		p.advance()
		// Check if this is a known symbol
		symbol := GetSymbol(tok.Value)
		if symbol != "" {
			return &ESymbol{Type: SymbolOrd, Name: symbol}, nil
		}
		return &EIdentifier{Name: tok.Value}, nil

	case TokenOperator:
		p.advance()
		return &EOperator{Op: tok.Value}, nil

	case TokenCommand:
		return p.parseCommand(tok.Value)

	default:
		return nil, &ParseError{
			Message: fmt.Sprintf("unexpected token type %v", tok.Type),
			Pos:     tok.Pos,
			Input:   p.lexer.input,
		}
	}
}

// parseCommand parses a LaTeX command.
func (p *Parser) parseCommand(cmd string) (Exp, error) {
	switch cmd {
	case "frac", "dfrac", "tfrac":
		return p.parseFraction(cmd == "dfrac")

	case "sqrt":
		return p.parseRadical()

	case "binom":
		return p.parseBinom()

	case "vec", "bar", "hat", "tilde", "dot", "ddot", "acute", "grave", "breve", "check":
		return p.parseAccent(cmd)

	case "int", "oint", "sum", "prod", "coprod", "bigcap", "bigcup",
		"bigsqcup", "bigvee", "bigwedge", "bigodot", "bigotimes", "bigoplus",
		"biguplus", "iint", "iiint", "idotsint":
		return p.parseNary(cmd)

	case "lim", "limsup", "liminf", "sup", "inf", "max", "min":
		return p.parseLimit(cmd)

	case "sin", "cos", "tan", "cot", "sec", "csc",
		"arcsin", "arccos", "arctan",
		"sinh", "cosh", "tanh", "coth",
		"log", "ln", "lg", "exp",
		"det", "dim", "ker", "gcd", "arg":
		return p.parseFunction(cmd)

	case "begin":
		return p.parseEnvironment()

	case "left":
		return p.parseLeftRight()

	case "text", "textrm", "textit", "textbf":
		return p.parseTextCommand(cmd)

	case "quad", "qquad":
		p.advance()
		return &ESpace{Width: "\\" + cmd}, nil

	case ",", ";", ":", "!":
		p.advance()
		return &ESpace{Width: "\\" + cmd}, nil

	default:
		// Check if it's a Greek letter or other symbol
		p.advance()
		symbol := GetSymbol(cmd)
		if symbol != "" {
			// Determine symbol type
			symType := SymbolOrd
			if _, isOp := MathOperators[cmd]; isOp {
				symType = SymbolOp
			} else if _, isRel := RelationSymbols[cmd]; isRel {
				symType = SymbolRel
			}
			return &ESymbol{Type: symType, Name: symbol}, nil
		}
		// Unknown command - treat as text
		return &EText{Type: TextNormal, Content: "\\" + cmd}, nil
	}
}

// parseFraction parses \frac{num}{den}.
func (p *Parser) parseFraction(displayStyle bool) (Exp, error) {
	p.advance() // consume \frac

	_, err := p.expect(TokenLBrace)
	if err != nil {
		return nil, err
	}
	num, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	_, err = p.expect(TokenRBrace)
	if err != nil {
		return nil, err
	}

	_, err = p.expect(TokenLBrace)
	if err != nil {
		return nil, err
	}
	den, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	_, err = p.expect(TokenRBrace)
	if err != nil {
		return nil, err
	}

	return &EFraction{DisplayStyle: displayStyle, Numerator: num, Denominator: den}, nil
}

// parseRadical parses \sqrt{x} or \sqrt[n]{x}.
func (p *Parser) parseRadical() (Exp, error) {
	p.advance() // consume \sqrt

	var index Exp

	// Check for optional index [n]
	if p.current().Type == TokenLBracket {
		p.advance()
		var err error
		index, err = p.parseExpr()
		if err != nil {
			return nil, err
		}
		_, err = p.expect(TokenRBracket)
		if err != nil {
			return nil, err
		}
	}

	_, err := p.expect(TokenLBrace)
	if err != nil {
		return nil, err
	}
	radicand, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	_, err = p.expect(TokenRBrace)
	if err != nil {
		return nil, err
	}

	return &ERoot{Index: index, Radicand: radicand}, nil
}

// parseBinom parses \binom{top}{bot}.
func (p *Parser) parseBinom() (Exp, error) {
	p.advance() // consume \binom

	_, err := p.expect(TokenLBrace)
	if err != nil {
		return nil, err
	}
	top, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	_, err = p.expect(TokenRBrace)
	if err != nil {
		return nil, err
	}

	_, err = p.expect(TokenLBrace)
	if err != nil {
		return nil, err
	}
	bot, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	_, err = p.expect(TokenRBrace)
	if err != nil {
		return nil, err
	}

	return &EFraction{Numerator: top, Denominator: bot, NoBar: true}, nil
}

// parseAccent parses accent commands like \vec{x}, \bar{x}.
func (p *Parser) parseAccent(cmd string) (Exp, error) {
	p.advance() // consume command

	_, err := p.expect(TokenLBrace)
	if err != nil {
		return nil, err
	}
	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	_, err = p.expect(TokenRBrace)
	if err != nil {
		return nil, err
	}

	// Map accent command to DecorationType
	decType := DecorHat
	switch cmd {
	case "vec":
		decType = DecorVec
	case "bar":
		decType = DecorBar
	case "hat":
		decType = DecorHat
	case "tilde":
		decType = DecorTilde
	case "dot":
		decType = DecorDot
	case "ddot":
		decType = DecorDdot
	case "acute":
		decType = DecorAcute
	case "grave":
		decType = DecorGrave
	case "check":
		decType = DecorCheck
	case "breve":
		decType = DecorBreve
	}

	return &EDecorated{Decoration: decType, Base: expr}, nil
}

// parseNary parses n-ary operators like \int_a^b, \sum_{i=1}^n.
func (p *Parser) parseNary(cmd string) (Exp, error) {
	p.advance() // consume command

	var sub, sup Exp
	var err error

	// Handle subscript (lower limit)
	if p.current().Type == TokenUnderscore {
		p.advance()
		if p.current().Type == TokenLBrace {
			p.advance()
			sub, err = p.parseExpr()
			if err != nil {
				return nil, err
			}
			_, err = p.expect(TokenRBrace)
			if err != nil {
				return nil, err
			}
		} else {
			sub, err = p.parseAtom()
			if err != nil {
				return nil, err
			}
		}
	}

	// Handle superscript (upper limit)
	if p.current().Type == TokenCaret {
		p.advance()
		if p.current().Type == TokenLBrace {
			p.advance()
			sup, err = p.parseExpr()
			if err != nil {
				return nil, err
			}
			_, err = p.expect(TokenRBrace)
			if err != nil {
				return nil, err
			}
		} else {
			sup, err = p.parseAtom()
			if err != nil {
				return nil, err
			}
		}
	}

	// Get operator character
	opChar := NaryOperators[cmd]
	if opChar == "" {
		opChar = cmd
	}

	// Parse the base expression (what follows the operator)
	base, err := p.parseTerm()
	if err != nil {
		// No base is okay for some operators
		base = nil
	}

	return &ENary{Operator: opChar, Body: base, LowerLimit: sub, UpperLimit: sup, HasLimits: sub != nil || sup != nil}, nil
}

// parseLimit parses limit expressions like \lim_{x \to 0}.
func (p *Parser) parseLimit(cmd string) (Exp, error) {
	p.advance() // consume command

	var limit Exp
	var err error

	// Handle subscript (limit expression)
	if p.current().Type == TokenUnderscore {
		p.advance()
		if p.current().Type == TokenLBrace {
			p.advance()
			limit, err = p.parseExpr()
			if err != nil {
				return nil, err
			}
			_, err = p.expect(TokenRBrace)
			if err != nil {
				return nil, err
			}
		} else {
			limit, err = p.parseAtom()
			if err != nil {
				return nil, err
			}
		}
	}

	// Parse the base expression
	base, err := p.parseTerm()
	if err != nil {
		base = nil
	}

	// For limits, use EUnder structure
	return &EUnder{Base: base, UnderExp: &EGrouped{Exps: []Exp{&EText{Type: TextNormal, Content: cmd}, limit}}}, nil
}

// parseFunction parses function names like \sin{x} or \sin x.
func (p *Parser) parseFunction(cmd string) (Exp, error) {
	p.advance() // consume command

	// Function name as text
	funcName := &EText{Type: TextRoman, Content: cmd}

	// Function can have argument in braces or just next atom
	var arg Exp
	var err error

	if p.current().Type == TokenLBrace {
		p.advance()
		arg, err = p.parseExpr()
		if err != nil {
			return nil, err
		}
		_, err = p.expect(TokenRBrace)
		if err != nil {
			return nil, err
		}
	} else {
		arg, err = p.parseAtom()
		if err != nil {
			// No argument is okay
			arg = nil
		}
	}

	if arg == nil {
		return funcName, nil
	}
	return &EGrouped{Exps: []Exp{funcName, arg}}, nil
}

// parseEnvironment parses \begin{...}...\end{...}.
func (p *Parser) parseEnvironment() (Exp, error) {
	p.advance() // consume \begin

	// Parse environment name
	if p.current().Type != TokenLBrace {
		return nil, &ParseError{
			Message: "expected { after \\begin",
			Pos:     p.current().Pos,
			Input:   p.lexer.input,
		}
	}
	p.advance()

	envTok := p.current()
	if envTok.Type != TokenText {
		return nil, &ParseError{
			Message: "expected environment name",
			Pos:     envTok.Pos,
			Input:   p.lexer.input,
		}
	}
	envType := envTok.Value
	p.advance()

	_, err := p.expect(TokenRBrace)
	if err != nil {
		return nil, err
	}

	// Parse environment content based on type
	switch envType {
	case "matrix", "pmatrix", "bmatrix", "vmatrix", "Vmatrix":
		return p.parseMatrixContent(envType)

	case "array", "eqnarray", "align", "cases":
		return p.parseArrayContent(envType)

	default:
		return nil, &ParseError{
			Message: fmt.Sprintf("unsupported environment: %s", envType),
			Pos:     envTok.Pos,
			Input:   p.lexer.input,
		}
	}
}

// parseMatrixContent parses matrix content.
func (p *Parser) parseMatrixContent(envType string) (Exp, error) {
	lines := []ArrayLine{}
	currentLine := []Exp{}

	for {
		tok := p.current()
		if tok.Type == TokenEOF {
			return nil, &ParseError{
				Message: "unexpected end of matrix",
				Pos:     tok.Pos,
				Input:   p.lexer.input,
			}
		}

		if tok.Type == TokenCommand && tok.Value == "end" {
			p.advance()
			_, err := p.expect(TokenLBrace)
			if err != nil {
				return nil, err
			}
			endTok := p.current()
			if endTok.Type != TokenText || endTok.Value != envType {
				return nil, &ParseError{
					Message: fmt.Sprintf("mismatched \\end{%s}", envType),
					Pos:     endTok.Pos,
					Input:   p.lexer.input,
				}
			}
			p.advance()
			_, err = p.expect(TokenRBrace)
			if err != nil {
				return nil, err
			}
			// Add last row if not empty
			if len(currentLine) > 0 {
				lines = append(lines, currentLine)
			}
			return &EArray{EnvType: envType, Lines: lines}, nil
		}

		if tok.Type == TokenAmpersand {
			p.advance()
			continue
		}

		if tok.Type == TokenDoubleBackslash {
			p.advance()
			if len(currentLine) > 0 {
				lines = append(lines, currentLine)
				currentLine = []Exp{}
			}
			continue
		}

		// Parse cell content
		cell, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		currentLine = append(currentLine, cell)
	}
}

// parseArrayContent parses array/eqnarray content.
func (p *Parser) parseArrayContent(envType string) (Exp, error) {
	lines := []ArrayLine{}
	currentLine := []Exp{}

	for {
		tok := p.current()
		if tok.Type == TokenEOF {
			return nil, &ParseError{
				Message: "unexpected end of array",
				Pos:     tok.Pos,
				Input:   p.lexer.input,
			}
		}

		if tok.Type == TokenCommand && tok.Value == "end" {
			p.advance()
			_, err := p.expect(TokenLBrace)
			if err != nil {
				return nil, err
			}
			endTok := p.current()
			if endTok.Type != TokenText || endTok.Value != envType {
				return nil, &ParseError{
					Message: fmt.Sprintf("mismatched \\end{%s}", envType),
					Pos:     endTok.Pos,
					Input:   p.lexer.input,
				}
			}
			p.advance()
			_, err = p.expect(TokenRBrace)
			if err != nil {
				return nil, err
			}
			if len(currentLine) > 0 {
				lines = append(lines, currentLine)
			}
			return &EArray{EnvType: envType, Lines: lines}, nil
		}

		if tok.Type == TokenAmpersand {
			p.advance()
			continue
		}

		if tok.Type == TokenDoubleBackslash {
			p.advance()
			if len(currentLine) > 0 {
				lines = append(lines, currentLine)
				currentLine = []Exp{}
			}
			continue
		}

		cell, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		currentLine = append(currentLine, cell)
	}
}

// parseLeftRight parses \left(...\right).
func (p *Parser) parseLeftRight() (Exp, error) {
	p.advance() // consume \left

	// Get left delimiter
	leftTok := p.current()
	var left string
	if leftTok.Type == TokenLParen {
		left = "("
		p.advance()
	} else if leftTok.Type == TokenLBracket {
		left = "["
		p.advance()
	} else if leftTok.Type == TokenLBrace {
		left = "{"
		p.advance()
	} else if leftTok.Type == TokenCommand {
		if info, ok := DelimiterPairs[leftTok.Value]; ok {
			left = info.Left
			p.advance()
		} else {
			left = leftTok.Value
			p.advance()
		}
	} else if leftTok.Type == TokenOperator && leftTok.Value == "|" {
		left = "|"
		p.advance()
	} else {
		left = "."
		p.advance()
	}

	// Parse content
	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	// Expect \right
	rightTok := p.current()
	if rightTok.Type != TokenCommand || rightTok.Value != "right" {
		return nil, &ParseError{
			Message: "expected \\right",
			Pos:     rightTok.Pos,
			Input:   p.lexer.input,
		}
	}
	p.advance()

	// Get right delimiter
	rightTok = p.current()
	var right string
	if rightTok.Type == TokenRParen {
		right = ")"
		p.advance()
	} else if rightTok.Type == TokenRBracket {
		right = "]"
		p.advance()
	} else if rightTok.Type == TokenRBrace {
		right = "}"
		p.advance()
	} else if rightTok.Type == TokenCommand {
		if info, ok := DelimiterPairs[rightTok.Value]; ok {
			right = info.Right
			p.advance()
		} else {
			right = rightTok.Value
			p.advance()
		}
	} else if rightTok.Type == TokenOperator && rightTok.Value == "|" {
		right = "|"
		p.advance()
	} else {
		right = "."
		p.advance()
	}

	return &EDelimited{LeftDelimiter: left, RightDelimiter: right, Exps: AsExps(expr)}, nil
}

// parseTextCommand parses \text{...}.
func (p *Parser) parseTextCommand(cmd string) (Exp, error) {
	p.advance() // consume command

	_, err := p.expect(TokenLBrace)
	if err != nil {
		return nil, err
	}

	// Collect text content until }
	var content strings.Builder
	for {
		tok := p.current()
		if tok.Type == TokenRBrace {
			p.advance()
			break
		}
		if tok.Type == TokenEOF {
			return nil, &ParseError{
				Message: "unclosed \\text",
				Pos:     tok.Pos,
				Input:   p.lexer.input,
			}
		}
		content.WriteString(tok.Value)
		p.advance()
	}

	textType := TextNormal
	switch cmd {
	case "textbf":
		textType = TextBold
	case "textit":
		textType = TextItalic
	case "textrm":
		textType = TextRoman
	}

	return &EText{Type: textType, Content: content.String()}, nil
}

// utf8DecodeRuneInString decodes a UTF-8 rune from a string.
func utf8DecodeRuneInString(s string) (rune, int) {
	if len(s) == 0 {
		return 0, 0
	}
	// Simple UTF-8 decode
	b0 := s[0]
	if b0 < 0x80 {
		return rune(b0), 1
	}
	// Multi-byte sequence
	var r rune
	var size int
	if b0 < 0xE0 {
		r = rune(b0&0x1F)<<6 | rune(s[1]&0x3F)
		size = 2
	} else if b0 < 0xF0 {
		r = rune(b0&0x0F)<<12 | rune(s[1]&0x3F)<<6 | rune(s[2]&0x3F)
		size = 3
	} else {
		r = rune(b0&0x07)<<18 | rune(s[1]&0x3F)<<12 | rune(s[2]&0x3F)<<6 | rune(s[3]&0x3F)
		size = 4
	}
	return r, size
}
