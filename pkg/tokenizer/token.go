package tokenizer

import (
	"encoding/json"
	"strings"
)

// TokenType represents the different types of tokens.
type TokenType string

const (
	// Literal constants
	NumericLiteralTokenType     TokenType = "n" // Numeric literals with radix support
	StringLiteralTokenType      TokenType = "s" // String literals with quotes and escapes
	MultiLineStringTokenType    TokenType = "m" // String literals with quotes and escapes
	InterpolatedStringTokenType TokenType = "i" // Interpolated string literals e.g. `Hello, \(name)!`
	ExpressionTokenType         TokenType = "e" // Expression tokens (e.g., (1 + 2))

	// Identifier tokens
	StartTokenType    TokenType = "S" // Form start tokens (def, if, while)
	EndTokenType      TokenType = "E" // Form end tokens (end, endif, endwhile)
	BridgeTokenType   TokenType = "B" // Bridge tokens (=>, =>, else, catch)
	PrefixTokenType   TokenType = "P" // Prefix operators (return, yield)
	VariableTokenType TokenType = "V" // Variable identifiers

	// Other tokens
	OperatorTokenType       TokenType = "O" // Infix/postfix operators
	OpenDelimiterTokenType  TokenType = "[" // Opening brackets/braces/parentheses
	CloseDelimiterTokenType TokenType = "]" // Closing brackets/braces/parentheses
	MarkTokenType           TokenType = "M" // Marks (commas, semicolons)
	UnclassifiedTokenType   TokenType = "U" // Unclassified tokens
	ExceptionTokenType      TokenType = "X" // Exception tokens for invalid constructs
)

// Position represents a line and column position in the source file.
type Position struct {
	Line int `json:"line"`
	Col  int `json:"col"`
}

// Span represents the start and end positions of a token.
type Span struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// MarshalJSON implements custom JSON marshaling for Span.
func (s Span) MarshalJSON() ([]byte, error) {
	arr := [4]int{s.Start.Line, s.Start.Col, s.End.Line, s.End.Col}
	return json.Marshal(arr)
}

// UnmarshalJSON implements custom JSON unmarshaling for Span.
func (s *Span) UnmarshalJSON(data []byte) error {
	var arr [4]int
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}
	s.Start = Position{Line: arr[0], Col: arr[1]}
	s.End = Position{Line: arr[2], Col: arr[3]}
	return nil
}

type Arity int

const (
	Zero Arity = iota
	One
	Many
)

// Token represents a single token from the Nutmeg source code.
type Token struct {
	// Common fields for all tokens
	Text  string    `json:"text"`
	Span  Span      `json:"span"`
	Type  TokenType `json:"type"`
	Alias *string   `json:"alias,omitempty"` // The node alias, if any

	// String token fields
	Quote     string   `json:"quote,omitempty"`
	Value     *string  `json:"value,omitempty"`
	Specifier *string  `json:"specifier,omitempty"`
	Subtokens []*Token `json:"subtokens,omitempty"`

	// Numeric token fields
	Radix    *string `json:"radix,omitempty"` // Textual radix prefix (e.g., "0x", "2r", "0t", "" for decimal)
	Base     *int    `json:"base,omitempty"`  // Numeric base (e.g., 16, 2, 3, 10)
	Mantissa *string `json:"mantissa,omitempty"`
	Fraction *string `json:"fraction,omitempty"`
	Exponent *int    `json:"exponent,omitempty"`
	Balanced *bool   `json:"balanced,omitempty"` // For balanced ternary numbers

	// Start token, Bridge token, and Compound token fields
	Expecting []string `json:"expecting,omitempty"` // For start tokens (immediate next tokens) and bridge tokens (what can follow them)
	In        []string `json:"in,omitempty"`        // For bridge and compound tokens - what can contain them
	ClosedBy  []string `json:"closed_by,omitempty"` // For start tokens and delimiter tokens - what can close them
	Arity     *Arity   `json:"arity,omitempty"`     // For start tokens - whether they introduce a single statement block

	// Operator token fields
	Precedence *[3]int `json:"precedence,omitempty"` // [prefix, infix, postfix] precedence values

	// Delimiter fields (for '[' tokens)
	InfixPrecedence *int  `json:"infix,omitempty"`  // For delimiter infix usage
	Prefix          *bool `json:"prefix,omitempty"` // For delimiter prefix usage

	// Exception token fields
	Reason *string `json:"reason,omitempty"` // For exception tokens - explanation of the error

	// Newline tracking fields
	LnBefore *bool `json:"ln_before,omitempty"` // True if token was preceded by a newline
	LnAfter  *bool `json:"ln_after,omitempty"`  // True if token was followed by a newline
}

func (t *Token) SetQuote(r rune) {
	switch r {
	case '\'':
		t.Quote = "single"
	case '"':
		t.Quote = "double"
	case '`':
		t.Quote = "backtick"
	default:
		t.Quote = string(r)
	}
}

// NewToken creates a new token with the basic required fields.
func NewToken(text string, tokenType TokenType, span Span) *Token {
	return &Token{
		Text: text,
		Type: tokenType,
		Span: span,
	}
}

// NewStringToken creates a new string token with interpreted value.
func NewStringToken(text, value string, span Span) *Token {
	return &Token{
		Text:  text,
		Type:  StringLiteralTokenType,
		Span:  span,
		Value: &value,
	}
}

func NewMultiLineStringToken(text, value string, span Span) *Token {
	return &Token{
		Text:  text,
		Type:  MultiLineStringTokenType,
		Span:  span,
		Value: &value,
	}
}

func NewInterpolatedStringToken(text string, subtokens []*Token, span Span) *Token {
	return &Token{
		Text:      text,
		Type:      StringLiteralTokenType,
		Span:      span,
		Subtokens: subtokens,
	}
}

func NewExpressionToken(text string, span Span) *Token {
	return &Token{
		Text:  text,
		Type:  ExpressionTokenType,
		Span:  span,
		Value: &text,
	}
}

// NewNumericToken creates a new numeric token with radix and components.
func NewNumericToken(text string, radix string, base int, mantissa, fraction string, exponent int, span Span) *Token {
	token := &Token{
		Text:     text,
		Type:     NumericLiteralTokenType,
		Span:     span,
		Radix:    &radix,
		Base:     &base,
		Mantissa: &mantissa,
	}

	if fraction != "" {
		token.Fraction = &fraction
	}
	if exponent != 0 {
		token.Exponent = &exponent
	}

	return token
}

// NewBalancedTernaryToken creates a new balanced ternary numeric token.
func NewBalancedTernaryToken(text string, mantissa, fraction string, exponent int, span Span) *Token {
	radixPrefix := "0t"
	base := 3
	balanced := true
	token := &Token{
		Text:     text,
		Type:     NumericLiteralTokenType,
		Span:     span,
		Radix:    &radixPrefix,
		Base:     &base,
		Mantissa: &mantissa,
		Balanced: &balanced,
	}

	if fraction != "" {
		token.Fraction = &fraction
	}
	if exponent != 0 {
		token.Exponent = &exponent
	}

	return token
}

// NewStartToken creates a new start token with expecting and closed_by tokens.
func NewStartToken(text string, expecting, closedBy []string, span Span, arity Arity) *Token {
	return &Token{
		Text:      text,
		Type:      StartTokenType,
		Span:      span,
		Expecting: expecting,
		ClosedBy:  closedBy,
		Arity:     &arity,
	}
}

// NewOperatorToken creates a new operator token with precedence values.
func NewOperatorToken(text string, prefix, infix, postfix int, span Span) *Token {
	token := &Token{
		Text: text,
		Type: OperatorTokenType,
		Span: span,
	}

	// Only set precedence if at least one value is non-zero
	if prefix > 0 || infix > 0 || postfix > 0 {
		precedence := [3]int{prefix, infix, postfix}
		token.Precedence = &precedence
	}

	return token
}

// NewDelimiterToken creates a new open delimiter token.
func NewDelimiterToken(text string, closedBy []string, isInfix int, isPrefix bool, span Span) *Token {
	return &Token{
		Text:            text,
		Type:            OpenDelimiterTokenType,
		Span:            span,
		ClosedBy:        closedBy,
		InfixPrecedence: &isInfix,
		Prefix:          &isPrefix,
	}
}

// NewStmntBridgeToken creates a new bridge token with expecting and in attributes.
func NewStmntBridgeToken(text string, expecting, in []string, span Span) *Token {
	return NewBridgeToken(text, expecting, in, Many, span)
}

// NewExprBridgeToken creates a new compound token with expecting and in attributes.
func NewExprBridgeToken(text string, expecting, in []string, span Span) *Token {
	return NewBridgeToken(text, expecting, in, One, span)
}

func NewBridgeToken(text string, expecting, in []string, arity Arity, span Span) *Token {
	return &Token{
		Text:      text,
		Type:      BridgeTokenType,
		Span:      span,
		Expecting: expecting,
		In:        in,
		Arity:     &arity,
	}
}

// NewWildcardBridgeToken creates a wildcard bridge token with copied attributes.
func NewWildcardBridgeToken(text, expectedText string, expecting, in []string, arity Arity, span Span) *Token {
	return &Token{
		Text:      text,
		Type:      BridgeTokenType,
		Span:      span,
		Expecting: expecting,
		In:        in,
		Alias:     &expectedText,
		Arity:     &arity,
	}
}

func NewUnclassifiedToken(text string, span Span) *Token {
	return &Token{
		Text: text,
		Type: UnclassifiedTokenType,
		Span: span,
	}
}

// NewExceptionToken creates a new exception token with an error reason.
func NewExceptionToken(text, reason string, span Span) *Token {
	return &Token{
		Text:   text,
		Type:   ExceptionTokenType,
		Span:   span,
		Reason: &reason,
	}
}

// isValidNumber checks if a numeric token represents a valid number.
func (t *Token) isValidNumber() (bool, string) {
	if t.Type != NumericLiteralTokenType {
		return true, "" // Non-numeric tokens are always valid
	}

	if t.Base == nil || t.Mantissa == nil {
		return false, "missing base or mantissa"
	}

	base := *t.Base
	mantissa := *t.Mantissa
	isBalanced := t.Balanced != nil && *t.Balanced

	// Check prefix validity for x/o/b/t notation
	text := t.Text
	if strings.Contains(text, "x") || strings.Contains(text, "o") || strings.Contains(text, "b") || strings.Contains(text, "t") {
		// Find the prefix character
		var prefixIndex int
		var found bool
		for _, chars := range []string{"x", "o", "b", "t"} {
			if idx := strings.Index(text, chars); idx != -1 {
				prefixIndex = idx
				found = true
				break
			}
		}
		if found {
			prefix := text[:prefixIndex]
			if prefix != "0" {
				return false, "invalid literal"
			}
		}
	}

	// Validate mantissa digits
	if !isValidDigitsForRadix(mantissa, base, isBalanced) {
		return false, "invalid literal"
	}

	// Validate fraction digits if present
	if t.Fraction != nil && *t.Fraction != "" {
		if !isValidDigitsForRadix(*t.Fraction, base, isBalanced) {
			return false, "invalid literal"
		}
	}

	return true, ""
}

// isValidDigitsForRadix checks if all characters in a string are valid digits for the given radix.
func isValidDigitsForRadix(digits string, radix int, allowBalancedTernary bool) bool {
	for _, char := range digits {
		// Skip underscores - they're allowed as separators
		if char == '_' {
			continue
		}
		if !isValidDigitForRadix(char, radix, allowBalancedTernary) {
			return false
		}
	}
	return true
}

// isValidDigitForRadix checks if a character is a valid digit for the given radix.
func isValidDigitForRadix(char rune, radix int, allowBalancedTernary bool) bool {
	// Handle balanced ternary special case
	if allowBalancedTernary && radix == 3 && char == 'T' {
		return true
	}

	// Handle numeric digits 0-9
	if char >= '0' && char <= '9' {
		return int(char-'0') < radix
	}

	// Handle alphabetic digits A-Z (for radix > 10)
	if char >= 'A' && char <= 'Z' {
		return int(char-'A'+10) < radix
	}

	return false
}
