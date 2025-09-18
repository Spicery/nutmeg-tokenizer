package tokeniser

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Tokeniser represents the main tokeniser structure.
type Tokeniser struct {
	input    string
	position int
	line     int
	column   int
	tokens   []*Token
}

// Regular expressions for token matching
var (
	identifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*`)
	operatorRegex   = regexp.MustCompile(`^[\+\-\*/=<>!&|%^~]+`)
	closeDelimRegex = regexp.MustCompile(`^[\)\]\}]`)
	numericRegex    = regexp.MustCompile(`^(?:0[bB][01]+|0[oO][0-7]+|0[xX][0-9a-fA-F]+|\d+)(?:\.\d*)?(?:[eE][+-]?\d+)?`)
	commentRegex    = regexp.MustCompile(`^###.*`)
)

// Start token mappings
var startTokens = map[string][]string{
	"def":         {"end"},
	"if":          {"end"},
	"ifnot":       {"end"},
	"fn":          {"end"},
	"for":         {"end"},
	"class":       {"end"},
	"interface":   {"end"},
	"try":         {"end"},
	"transaction": {"end"},
}

// Label tokens (L) with their attributes
type LabelTokenData struct {
	Expecting []string
	In        []string
}

var labelTokens = map[string]LabelTokenData{
	"then": {
		Expecting: []string{"else", "elseif", "elseifnot", "catch"},
		In:        []string{"try", "if"},
	},
	"else": {
		Expecting: []string{},
		In:        []string{"if", "try"},
	},
}

// Compound tokens (C) with their attributes
type CompoundTokenData struct {
	Expecting []string
	In        []string
}

var compoundTokens = map[string]CompoundTokenData{
	"catch": {
		Expecting: []string{"then", ":"},
		In:        []string{"try"},
	},
	"elseif": {
		Expecting: []string{"then", ":"},
		In:        []string{"if"},
	},
	"elseifnot": {
		Expecting: []string{"then", ":"},
		In:        []string{"if"},
	},
}

// Prefix tokens (P)
var prefixTokens = map[string]bool{
	"return": true,
	"yield":  true,
}

// Operator precedence mappings
var operatorPrecedence = map[string][3]int{
	"+":  {0, 5, 0}, // prefix=0, infix=5, postfix=0
	"-":  {8, 5, 0}, // prefix=8, infix=5, postfix=0
	"*":  {0, 6, 0}, // prefix=0, infix=6, postfix=0
	"/":  {0, 6, 0}, // prefix=0, infix=6, postfix=0
	"=":  {0, 1, 0}, // prefix=0, infix=1, postfix=0
	"<":  {0, 3, 0}, // prefix=0, infix=3, postfix=0
	">":  {0, 3, 0}, // prefix=0, infix=3, postfix=0
	"<=": {0, 3, 0}, // prefix=0, infix=3, postfix=0
	">=": {0, 3, 0}, // prefix=0, infix=3, postfix=0
	"==": {0, 2, 0}, // prefix=0, infix=2, postfix=0
	"!=": {0, 2, 0}, // prefix=0, infix=2, postfix=0
	"&&": {0, 1, 0}, // prefix=0, infix=1, postfix=0
	"||": {0, 1, 0}, // prefix=0, infix=1, postfix=0
}

// Delimiter mappings
var delimiterMappings = map[string]string{
	"(": ")",
	"[": "]",
	"{": "}",
}

// Delimiter properties
var delimiterProperties = map[string][2]bool{
	"(": {true, true},  // infix=true, prefix=true
	"[": {true, false}, // infix=true, prefix=false
	"{": {true, true},  // infix=false, prefix=true
}

// New creates a new tokeniser instance.
func New(input string) *Tokeniser {
	return &Tokeniser{
		input:  input,
		line:   1,
		column: 1,
		tokens: make([]*Token, 0),
	}
}

// Tokenise processes the input and returns a slice of tokens.
func (t *Tokeniser) Tokenise() ([]*Token, error) {
	for t.position < len(t.input) {
		if err := t.nextToken(); err != nil {
			return nil, err
		}
	}
	return t.tokens, nil
}

// nextToken processes the next token from the input.
func (t *Tokeniser) nextToken() error {
	// Skip whitespace
	t.skipWhitespace()

	if t.position >= len(t.input) {
		return nil
	}

	start := Position{Line: t.line, Col: t.column}

	// Skip comments
	if t.matchComment() {
		return nil
	}

	// Try to match different token types
	if token := t.matchString(); token != nil {
		token.Span.Start = start
		t.tokens = append(t.tokens, token)
		return nil
	}

	if token := t.matchNumeric(); token != nil {
		token.Span.Start = start
		t.tokens = append(t.tokens, token)
		return nil
	}

	if token := t.matchIdentifier(); token != nil {
		token.Span.Start = start
		t.tokens = append(t.tokens, token)
		return nil
	}

	if token := t.matchOperator(); token != nil {
		token.Span.Start = start
		t.tokens = append(t.tokens, token)
		return nil
	}

	if token := t.matchDelimiter(); token != nil {
		token.Span.Start = start
		t.tokens = append(t.tokens, token)
		return nil
	}

	// If nothing matches, create an unclassified token
	r, size := utf8.DecodeRuneInString(t.input[t.position:])
	text := string(r)
	end := Position{Line: t.line, Col: t.column + size}
	span := Span{Start: start, End: end}

	token := NewToken(text, UnclassifiedToken, span)
	t.tokens = append(t.tokens, token)
	t.advance(size)

	return nil
}

// skipWhitespace advances past whitespace characters.
func (t *Tokeniser) skipWhitespace() {
	for t.position < len(t.input) {
		r, size := utf8.DecodeRuneInString(t.input[t.position:])
		if !unicode.IsSpace(r) {
			break
		}
		t.advance(size)
	}
}

// matchComment checks for and skips comments.
func (t *Tokeniser) matchComment() bool {
	match := commentRegex.FindString(t.input[t.position:])
	if match != "" {
		t.advance(len(match))
		return true
	}
	return false
}

// matchString attempts to match a string literal.
func (t *Tokeniser) matchString() *Token {
	if t.position >= len(t.input) {
		return nil
	}

	quote := t.input[t.position]
	if quote != '"' && quote != '\'' && quote != '`' {
		return nil
	}

	start := t.position
	t.advance(1) // Skip opening quote

	var value strings.Builder
	escaped := false

	for t.position < len(t.input) {
		r, size := utf8.DecodeRuneInString(t.input[t.position:])

		if escaped {
			// Handle escape sequences
			switch r {
			case 'n':
				value.WriteRune('\n')
			case 't':
				value.WriteRune('\t')
			case 'r':
				value.WriteRune('\r')
			case '\\':
				value.WriteRune('\\')
			case '"', '\'', '`':
				value.WriteRune(r)
			default:
				value.WriteRune(r)
			}
			escaped = false
		} else if r == '\\' {
			escaped = true
		} else if byte(r) == quote {
			t.advance(size) // Skip closing quote
			break
		} else {
			value.WriteRune(r)
		}

		t.advance(size)
	}

	text := t.input[start:t.position]
	end := Position{Line: t.line, Col: t.column}
	span := Span{End: end}

	return NewStringToken(text, value.String(), span)
}

// matchNumeric attempts to match a numeric literal.
func (t *Tokeniser) matchNumeric() *Token {
	match := numericRegex.FindString(t.input[t.position:])
	if match == "" {
		return nil
	}

	radix := 10
	mantissa := match
	var fraction, exponent string

	// Determine radix and extract components
	if strings.HasPrefix(match, "0b") || strings.HasPrefix(match, "0B") {
		radix = 2
		mantissa = match[2:]
	} else if strings.HasPrefix(match, "0o") || strings.HasPrefix(match, "0O") {
		radix = 8
		mantissa = match[2:]
	} else if strings.HasPrefix(match, "0x") || strings.HasPrefix(match, "0X") {
		radix = 16
		mantissa = match[2:]
	}

	// Extract decimal point and fraction
	if dotIndex := strings.Index(mantissa, "."); dotIndex != -1 {
		fraction = mantissa[dotIndex+1:]
		mantissa = mantissa[:dotIndex]

		// Remove exponent from fraction if present
		if eIndex := strings.IndexAny(fraction, "eE"); eIndex != -1 {
			exponent = fraction[eIndex+1:]
			fraction = fraction[:eIndex]
		}
	} else if eIndex := strings.IndexAny(mantissa, "eE"); eIndex != -1 {
		exponent = mantissa[eIndex+1:]
		mantissa = mantissa[:eIndex]
	}

	end := Position{Line: t.line, Col: t.column + len(match)}
	span := Span{End: end}

	t.advance(len(match))
	return NewNumericToken(match, radix, mantissa, fraction, exponent, span)
}

// matchIdentifier attempts to match an identifier.
func (t *Tokeniser) matchIdentifier() *Token {
	match := identifierRegex.FindString(t.input[t.position:])
	if match == "" {
		return nil
	}

	end := Position{Line: t.line, Col: t.column + len(match)}
	span := Span{End: end}

	var tokenType TokenType = VariableToken

	// Check if it's a start token
	if closedBy, isStart := startTokens[match]; isStart {
		// Create the full closed_by list including end{TEXT} and end
		fullClosedBy := make([]string, 0, len(closedBy)+1)

		// Add the specific end token (e.g., "enddef" for "def")
		endSpecific := "end" + match
		fullClosedBy = append(fullClosedBy, endSpecific)

		// Add all the configured closing tokens
		fullClosedBy = append(fullClosedBy, closedBy...)

		t.advance(len(match))
		return NewStartToken(match, fullClosedBy, span)
	}

	// Check if it's an end token
	if strings.HasPrefix(match, "end") {
		tokenType = EndToken
	} else if labelData, isLabel := labelTokens[match]; isLabel {
		// Check if it's a label token (L)
		t.advance(len(match))
		return NewLabelToken(match, labelData.Expecting, labelData.In, span)
	} else if compoundData, isCompound := compoundTokens[match]; isCompound {
		// Check if it's a compound token (C)
		t.advance(len(match))
		return NewCompoundToken(match, compoundData.Expecting, compoundData.In, span)
	} else if prefixTokens[match] {
		// Check if it's a prefix token (P)
		tokenType = PrefixToken
	}
	// Otherwise, default to VariableToken

	t.advance(len(match))
	return NewToken(match, tokenType, span)
}

// matchOperator attempts to match an operator.
func (t *Tokeniser) matchOperator() *Token {
	match := operatorRegex.FindString(t.input[t.position:])
	if match == "" {
		return nil
	}

	// Try to find the longest matching operator
	longestMatch := ""
	for op := range operatorPrecedence {
		if strings.HasPrefix(match, op) && len(op) > len(longestMatch) {
			longestMatch = op
		}
	}

	if longestMatch == "" {
		longestMatch = match
	}

	end := Position{Line: t.line, Col: t.column + len(longestMatch)}
	span := Span{End: end}

	precedence, exists := operatorPrecedence[longestMatch]
	var prefix, infix, postfix int
	if exists {
		prefix, infix, postfix = precedence[0], precedence[1], precedence[2]
	}

	t.advance(len(longestMatch))
	return NewOperatorToken(longestMatch, prefix, infix, postfix, span)
}

// matchDelimiter attempts to match a delimiter.
func (t *Tokeniser) matchDelimiter() *Token {
	if t.position >= len(t.input) {
		return nil
	}

	char := string(t.input[t.position])

	// Check for open delimiters
	if closedBy, isOpen := delimiterMappings[char]; isOpen {
		end := Position{Line: t.line, Col: t.column + 1}
		span := Span{End: end}

		props := delimiterProperties[char]
		isInfix, isPrefix := props[0], props[1]

		t.advance(1)
		return NewDelimiterToken(char, closedBy, isInfix, isPrefix, span)
	}

	// Check for close delimiters
	if closeDelimRegex.MatchString(char) {
		end := Position{Line: t.line, Col: t.column + 1}
		span := Span{End: end}

		t.advance(1)
		return NewToken(char, CloseDelimiter, span)
	}

	return nil
}

// advance moves the position forward and updates line/column tracking.
func (t *Tokeniser) advance(n int) {
	for i := 0; i < n && t.position < len(t.input); i++ {
		if t.input[t.position] == '\n' {
			t.line++
			t.column = 1
		} else {
			t.column++
		}
		t.position++
	}
}
