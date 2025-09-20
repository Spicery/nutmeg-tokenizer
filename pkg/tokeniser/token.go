package tokeniser

import (
	"encoding/json"
	"strings"
)

// TokenType represents the different types of tokens in the Nutmeg language.
type TokenType string

const (
	// Literal constants
	NumericLiteral TokenType = "n" // Numeric literals with radix support
	StringLiteral  TokenType = "s" // String literals with quotes and escapes

	// Identifier tokens
	StartToken    TokenType = "S" // Form start tokens (def, if, while)
	EndToken      TokenType = "E" // Form end tokens (end, endif, endwhile)
	CompoundToken TokenType = "C" // Multi-part constructs
	LabelToken    TokenType = "L" // Label identifiers
	PrefixToken   TokenType = "P" // Prefix operators (return, yield)
	VariableToken TokenType = "V" // Variable identifiers

	// Other tokens
	OperatorToken     TokenType = "O" // Infix/postfix operators
	OpenDelimiter     TokenType = "[" // Opening brackets/braces/parentheses
	CloseDelimiter    TokenType = "]" // Closing brackets/braces/parentheses
	UnclassifiedToken TokenType = "U" // Unclassified tokens
	ExceptionToken    TokenType = "X" // Exception tokens for invalid constructs
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

// Token represents a single token from the Nutmeg source code.
type Token struct {
	// Common fields for all tokens
	Text string    `json:"text"`
	Span Span      `json:"span"`
	Type TokenType `json:"type"`

	// String token fields
	Value *string `json:"value,omitempty"`

	// Numeric token fields
	Radix    *int    `json:"radix,omitempty"`
	Mantissa *string `json:"mantissa,omitempty"`
	Fraction *string `json:"fraction,omitempty"`
	Exponent *string `json:"exponent,omitempty"`
	Balanced *bool   `json:"balanced,omitempty"` // For balanced ternary numbers

	// Start token, Label token, and Compound token fields
	Expecting []string `json:"expecting,omitempty"` // For start tokens (immediate next tokens) and label/compound tokens (what can follow them)
	In        []string `json:"in,omitempty"`        // For label and compound tokens - what can contain them
	ClosedBy  []string `json:"closed_by,omitempty"` // For start tokens and delimiter tokens - what can close them

	// Operator token fields
	Precedence *[3]int `json:"precedence,omitempty"` // [prefix, infix, postfix] precedence values

	// Delimiter fields (for '[' tokens)
	Infix  *bool `json:"infix,omitempty"`  // For delimiter infix usage
	Prefix *bool `json:"prefix,omitempty"` // For delimiter prefix usage

	// Exception token fields
	Reason *string `json:"reason,omitempty"` // For exception tokens - explanation of the error
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
		Type:  StringLiteral,
		Span:  span,
		Value: &value,
	}
}

// NewNumericToken creates a new numeric token with radix and components.
func NewNumericToken(text string, radix int, mantissa, fraction, exponent string, span Span) *Token {
	token := &Token{
		Text:     text,
		Type:     NumericLiteral,
		Span:     span,
		Radix:    &radix,
		Mantissa: &mantissa,
	}

	if fraction != "" {
		token.Fraction = &fraction
	}
	if exponent != "" {
		token.Exponent = &exponent
	}

	return token
}

// NewBalancedTernaryToken creates a new balanced ternary numeric token.
func NewBalancedTernaryToken(text string, mantissa, fraction, exponent string, span Span) *Token {
	radix := 3
	balanced := true
	token := &Token{
		Text:     text,
		Type:     NumericLiteral,
		Span:     span,
		Radix:    &radix,
		Mantissa: &mantissa,
		Balanced: &balanced,
	}

	if fraction != "" {
		token.Fraction = &fraction
	}
	if exponent != "" {
		token.Exponent = &exponent
	}

	return token
}

// NewStartToken creates a new start token with expecting and closed_by tokens.
func NewStartToken(text string, expecting, closedBy []string, span Span) *Token {
	return &Token{
		Text:      text,
		Type:      StartToken,
		Span:      span,
		Expecting: expecting,
		ClosedBy:  closedBy,
	}
}

// NewOperatorToken creates a new operator token with precedence values.
func NewOperatorToken(text string, prefix, infix, postfix int, span Span) *Token {
	token := &Token{
		Text: text,
		Type: OperatorToken,
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
func NewDelimiterToken(text string, closedBy []string, isInfix, isPrefix bool, span Span) *Token {
	return &Token{
		Text:     text,
		Type:     OpenDelimiter,
		Span:     span,
		ClosedBy: closedBy,
		Infix:    &isInfix,
		Prefix:   &isPrefix,
	}
}

// NewLabelToken creates a new label token with expecting and in attributes.
func NewLabelToken(text string, expecting, in []string, span Span) *Token {
	return &Token{
		Text:      text,
		Type:      LabelToken,
		Span:      span,
		Expecting: expecting,
		In:        in,
	}
}

// NewCompoundToken creates a new compound token with expecting and in attributes.
func NewCompoundToken(text string, expecting, in []string, span Span) *Token {
	return &Token{
		Text:      text,
		Type:      CompoundToken,
		Span:      span,
		Expecting: expecting,
		In:        in,
	}
}

// NewWildcardLabelToken creates a new wildcard label token.
// For now, this creates a basic label token. The context-dependent logic
// to copy attributes from expected tokens will be implemented later.
func NewWildcardLabelToken(text string, span Span) *Token {
	value := ""
	return &Token{
		Text:  text,
		Type:  LabelToken,
		Span:  span,
		Value: &value,
	}
}

// NewWildcardLabelTokenWithAttributes creates a wildcard label token with copied attributes.
func NewWildcardLabelTokenWithAttributes(text, expectedText string, expecting, in []string, span Span) *Token {
	return &Token{
		Text:      text,
		Type:      LabelToken,
		Span:      span,
		Expecting: expecting,
		In:        in,
		Value:     &expectedText,
	}
}

// NewWildcardStartToken creates a wildcard start token with copied attributes.
func NewWildcardStartToken(text, expectedText string, closedBy []string, span Span) *Token {
	return &Token{
		Text:     text,
		Type:     StartToken,
		Span:     span,
		ClosedBy: closedBy,
		Value:    &expectedText,
	}
}

// NewWildcardEndToken creates a wildcard end token.
func NewWildcardEndToken(text, expectedText string, span Span) *Token {
	return &Token{
		Text:  text,
		Type:  EndToken,
		Span:  span,
		Value: &expectedText,
	}
}

// NewExceptionToken creates a new exception token with an error reason.
func NewExceptionToken(text, reason string, span Span) *Token {
	return &Token{
		Text:   text,
		Type:   ExceptionToken,
		Span:   span,
		Reason: &reason,
	}
}

// isValidNumber checks if a numeric token represents a valid number.
func (t *Token) isValidNumber() (bool, string) {
	if t.Type != NumericLiteral {
		return true, "" // Non-numeric tokens are always valid
	}

	if t.Radix == nil || t.Mantissa == nil {
		return false, "missing radix or mantissa"
	}

	radix := *t.Radix
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
	if !isValidDigitsForRadix(mantissa, radix, isBalanced) {
		return false, "invalid literal"
	}

	// Validate fraction digits if present
	if t.Fraction != nil && *t.Fraction != "" {
		if !isValidDigitsForRadix(*t.Fraction, radix, isBalanced) {
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
