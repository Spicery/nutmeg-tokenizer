package tokeniser

import "encoding/json"

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

	// Start token fields
	Expecting []string `json:"expecting,omitempty"` // For start tokens

	// Operator token fields
	Precedence *[3]int `json:"precedence,omitempty"` // [prefix, infix, postfix] precedence values

	// Delimiter fields (for '[' tokens)
	ClosedBy *string `json:"closed_by,omitempty"` // For delimiter tokens
	Infix    *bool   `json:"infix,omitempty"`     // For delimiter infix usage
	Prefix   *bool   `json:"prefix,omitempty"`    // For delimiter prefix usage
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

// NewStartToken creates a new start token with expected closing tokens.
func NewStartToken(text string, expecting []string, span Span) *Token {
	return &Token{
		Text:      text,
		Type:      StartToken,
		Span:      span,
		Expecting: expecting,
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
func NewDelimiterToken(text, closedBy string, isInfix, isPrefix bool, span Span) *Token {
	return &Token{
		Text:     text,
		Type:     OpenDelimiter,
		Span:     span,
		ClosedBy: &closedBy,
		Infix:    &isInfix,
		Prefix:   &isPrefix,
	}
}
