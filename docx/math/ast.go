package math

// Expr is an alias for Exp for backward compatibility.
// New code should use Exp directly.
type Expr = Exp

// The following types are kept for backward compatibility.
// They are aliases for the new texmath-inspired types.

// Text represents plain text content.
// Deprecated: Use EText instead.
type Text = EText

// Number represents a numeric value.
// Deprecated: Use ENumber instead.
type Number = ENumber

// Identifier represents a variable name or symbol.
// Deprecated: Use EIdentifier instead.
type Identifier = EIdentifier

// Group represents a grouped expression { ... }.
// Deprecated: Use EGrouped instead.
type Group = EGrouped

// Sequence represents a sequence of expressions.
// Deprecated: Use EGrouped instead.
type Sequence = EGrouped

// Fraction represents a fraction \frac{num}{den}.
// Deprecated: Use EFraction instead.
type Fraction = EFraction

// Radical represents a root expression \sqrt{x} or \sqrt[n]{x}.
// Deprecated: Use ERoot instead.
type Radical = ERoot

// SubSup represents subscript and/or superscript structure.
// Deprecated: Use ESubSuperscript instead.
type SubSup = ESubSuperscript

// Nary represents n-ary operators (integral, sum, product, etc).
// Deprecated: Use ENary instead.
type Nary = ENary

// Matrix represents a matrix environment.
// Deprecated: Use EArray instead.
type Matrix = EArray

// Delimiter represents delimited expressions (parentheses, brackets, etc).
// Deprecated: Use EDelimited instead.
type Delimiter = EDelimited

// Accent represents an accented expression (vector arrow, bar, etc).
// Deprecated: Use EDecorated instead.
type Accent = EDecorated

// Binom represents a binomial coefficient \binom{a}{b}.
// Deprecated: Use EFraction with NoBar=true instead.
type Binom = EFraction

// Operator represents a mathematical operator symbol.
// Deprecated: Use EOperator instead.
type Operator = EOperator

// Space represents spacing in the formula.
// Deprecated: Use ESpace instead.
type Space = ESpace

// LimLow represents a limit structure with subscript (like lim_{x->0}).
// Deprecated: Use EUnder instead.
type LimLow = EUnder

// LimUpp represents a limit structure with superscript (like overset).
// Deprecated: Use EOver instead.
type LimUpp = EOver

// Func represents a function with argument.
// Deprecated: Use EGrouped with EText for function name instead.
type Func struct {
	Name string
	Arg  Exp
}

// Array represents an array of equations.
// Deprecated: Use EArray instead.
type Array = EArray
