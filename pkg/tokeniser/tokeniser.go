package tokeniser

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Tokeniser represents the main tokeniser structure.
type Tokeniser struct {
	input          string
	position       int
	line           int
	column         int
	tokens         []*Token
	expectingStack [][]string // Stack of expecting arrays for context tracking
}

// Regular expressions for token matching
var (
	identifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*`)
	operatorRegex   = regexp.MustCompile(`^[\*/%\+\-<>~!&^|?=:]+`)
	closeDelimRegex = regexp.MustCompile(`^[\)\]\}]`)
	numericRegex    = regexp.MustCompile(`^(?:0[bB][01]+|0[oO][0-7]+|0[xX][0-9a-fA-F]+|\d+)(?:\.\d*)?(?:[eE][+-]?\d+)?`)
	commentRegex    = regexp.MustCompile(`^###.*`)
)

// Start token mappings with expecting and closed_by information
type StartTokenData struct {
	Expecting []string
	ClosedBy  []string
}

var startTokens = map[string]StartTokenData{
	"def": {
		Expecting: []string{"=>>"},
		ClosedBy:  []string{"end", "enddef"},
	},
	"if": {
		Expecting: []string{"then"},
		ClosedBy:  []string{"end", "endif"},
	},
	"ifnot": {
		Expecting: []string{"then"},
		ClosedBy:  []string{"end", "endifnot"},
	},
	"fn": {
		Expecting: []string{},
		ClosedBy:  []string{"end", "endfn"},
	},
	"for": {
		Expecting: []string{"do"},
		ClosedBy:  []string{"end", "endfor"},
	},
	"class": {
		Expecting: []string{},
		ClosedBy:  []string{"end", "endclass"},
	},
	"interface": {
		Expecting: []string{},
		ClosedBy:  []string{"end", "endinterface"},
	},
	"try": {
		Expecting: []string{"catch", "else"},
		ClosedBy:  []string{"end", "endtry"},
	},
	"transaction": {
		Expecting: []string{"else"},
		ClosedBy:  []string{"end", "endtransaction"},
	},
}

// Label tokens (L) with their attributes
type LabelTokenData struct {
	Expecting []string
	In        []string
}

var labelTokens = map[string]LabelTokenData{
	"=>>": {
		Expecting: []string{"do"},
		In:        []string{},
	},
	"do": {
		Expecting: []string{},
		In:        []string{"for", "def"},
	},
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

// Base precedence values for operator characters (from operators.md)
var baseOperatorPrecedence = map[rune]int{
	'*': 10,
	'/': 20,
	'%': 30,
	'+': 40,
	'-': 50,
	'<': 60,
	'>': 70,
	'~': 80,
	'!': 90,
	'&': 100,
	'^': 110,
	'|': 120,
	'?': 130,
	'=': 140,
	':': 150,
}

// calculateOperatorPrecedence calculates precedence based on rules in operators.md
func calculateOperatorPrecedence(operator string) (prefix, infix, postfix int) {
	if len(operator) == 0 {
		return 0, 0, 0
	}

	firstChar := rune(operator[0])
	basePrecedence, exists := baseOperatorPrecedence[firstChar]
	if !exists {
		// Fallback for unknown operators
		basePrecedence = 1000
	}

	// If the first character is repeated, subtract 1
	if len(operator) > 1 && rune(operator[1]) == firstChar {
		basePrecedence--
	}

	// Role adjustments as per updated operators.md:
	// - Only minus ("-") has prefix capability enabled (unary negation)
	// - All operators have infix capability (add 2000 to base precedence)
	// - No operators have postfix capability (set to 0)

	if operator == "-" {
		// Unary minus: enabled for both prefix and infix
		prefix = basePrecedence
		infix = basePrecedence + 2000
		postfix = 0
	} else {
		// All other operators: only infix enabled
		prefix = 0
		infix = basePrecedence + 2000
		postfix = 0
	}

	return prefix, infix, postfix
} // Delimiter mappings
var delimiterMappings = map[string][]string{
	"(": {")"},
	"[": {"]"},
	"{": {"}"},
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
		input:          input,
		line:           1,
		column:         1,
		tokens:         make([]*Token, 0),
		expectingStack: make([][]string, 0),
	}
}

// pushExpecting pushes a new set of expected tokens onto the stack.
func (t *Tokeniser) pushExpecting(expected []string) {
	t.expectingStack = append(t.expectingStack, expected)
}

// popExpecting removes the top set of expected tokens from the stack.
func (t *Tokeniser) popExpecting() {
	if len(t.expectingStack) > 0 {
		t.expectingStack = t.expectingStack[:len(t.expectingStack)-1]
	}
}

// getCurrentlyExpected returns the currently expected tokens, or nil if stack is empty.
func (t *Tokeniser) getCurrentlyExpected() []string {
	if len(t.expectingStack) == 0 {
		return nil
	}
	return t.expectingStack[len(t.expectingStack)-1]
}

// addTokenAndManageStack adds a token to the tokens slice and manages the expecting stack.
func (t *Tokeniser) addTokenAndManageStack(token *Token) {
	t.tokens = append(t.tokens, token)

	// Manage the expecting stack based on token type and text
	switch token.Type {
	case StartToken:
		// Push expected tokens for this start token
		if len(token.Expecting) > 0 {
			t.pushExpecting(token.Expecting)
		}
	case EndToken:
		// Pop the expecting stack
		t.popExpecting()
	case LabelToken, CompoundToken:
		// Update expecting for label and compound tokens based on their attributes
		switch token.Text {
		case "=>>":
			// After =>> we expect do
			t.pushExpecting([]string{"do"})
		case "do":
			// After do in "for x do" or "def f(x) =>> do", we expect end
			t.popExpecting() // Remove the "do" expectation
			t.pushExpecting([]string{"end"})
		default:
			// For other label/compound tokens, check if they have their own expectations
			if labelData, exists := labelTokens[token.Text]; exists {
				// Replace current expectations with what this label expects
				if len(t.expectingStack) > 0 {
					t.popExpecting() // Remove current expectations
				}
				if len(labelData.Expecting) > 0 {
					t.pushExpecting(labelData.Expecting)
				}
				// If labelData.Expecting is empty, we leave the stack with nothing expected
			} else if compoundData, exists := compoundTokens[token.Text]; exists {
				// Handle compound tokens the same way
				if len(t.expectingStack) > 0 {
					t.popExpecting() // Remove current expectations
				}
				if len(compoundData.Expecting) > 0 {
					t.pushExpecting(compoundData.Expecting)
				}
			}
		}
	}
} // Tokenise processes the input and returns a slice of tokens.
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
		t.addTokenAndManageStack(token)
		return nil
	}

	if token := t.matchNumeric(); token != nil {
		token.Span.Start = start
		t.addTokenAndManageStack(token)
		return nil
	}

	if token := t.matchIdentifier(); token != nil {
		token.Span.Start = start
		t.addTokenAndManageStack(token)
		return nil
	}

	if token := t.matchSpecialLabels(); token != nil {
		token.Span.Start = start
		t.addTokenAndManageStack(token)
		return nil
	}

	if token := t.matchOperator(); token != nil {
		token.Span.Start = start
		t.addTokenAndManageStack(token)
		return nil
	}

	if token := t.matchDelimiter(); token != nil {
		token.Span.Start = start
		t.addTokenAndManageStack(token)
		return nil
	}

	// If nothing matches, create an unclassified token
	r, size := utf8.DecodeRuneInString(t.input[t.position:])
	text := string(r)
	end := Position{Line: t.line, Col: t.column + size}
	span := Span{Start: start, End: end}

	token := NewToken(text, UnclassifiedToken, span)
	t.addTokenAndManageStack(token)
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
	if startData, isStart := startTokens[match]; isStart {
		t.advance(len(match))
		return NewStartToken(match, startData.Expecting, startData.ClosedBy, span)
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

// matchSpecialLabels attempts to match special label sequences like '=>>' and wildcard ':'
func (t *Tokeniser) matchSpecialLabels() *Token {
	// Check for '=>>' special label
	if strings.HasPrefix(t.input[t.position:], "=>>") {
		end := Position{Line: t.line, Col: t.column + 3}
		span := Span{End: end}

		labelData := labelTokens["=>>"]
		t.advance(3)
		return NewLabelToken("=>>", labelData.Expecting, labelData.In, span)
	}

	// Check for wildcard ':'
	if t.position < len(t.input) && t.input[t.position] == ':' {
		// Make sure it's a single ':' and not part of a longer operator
		if t.position+1 >= len(t.input) || !strings.ContainsRune("*/%+-<>~!&^|?=:", rune(t.input[t.position+1])) {
			end := Position{Line: t.line, Col: t.column + 1}
			span := Span{End: end}

			// Check if we have context from the expecting stack
			expected := t.getCurrentlyExpected()
			if len(expected) > 0 {
				// Use the first expected token as the basis for the wildcard
				expectedText := expected[0]

				// Check if it's a label token
				if labelData, exists := labelTokens[expectedText]; exists {
					// Create a wildcard token that copies attributes from the expected label
					t.advance(1)
					return NewWildcardLabelTokenWithAttributes(":", expectedText, labelData.Expecting, labelData.In, span)
				}

				// Check if it's a start token
				if startData, exists := startTokens[expectedText]; exists {
					// Create wildcard start token
					t.advance(1)
					return NewWildcardStartToken(":", expectedText, startData.ClosedBy, span)
				}

				// Check if it's an end token (starts with "end")
				if strings.HasPrefix(expectedText, "end") {
					// Create wildcard end token
					t.advance(1)
					return NewWildcardEndToken(":", expectedText, span)
				}
			}

			// No context available, create unclassified token
			t.advance(1)
			return NewToken(":", UnclassifiedToken, span)
		}
	}

	return nil
}

// matchOperator attempts to match an operator.
func (t *Tokeniser) matchOperator() *Token {
	match := operatorRegex.FindString(t.input[t.position:])
	if match == "" {
		return nil
	}

	// Special case: single ':' is treated as a wildcard simple-label, not an operator
	if match == ":" {
		// This should be handled elsewhere as a label token
		return nil
	}

	// Special case: '=>>' is treated as a simple-label, not an operator
	if strings.HasPrefix(match, "=>>") {
		// This should be handled elsewhere as a label token
		return nil
	}

	// Use the entire match as the operator (greedy matching)
	operator := match

	end := Position{Line: t.line, Col: t.column + len(operator)}
	span := Span{End: end}

	// Calculate precedence using the new rules
	prefix, infix, postfix := calculateOperatorPrecedence(operator)

	t.advance(len(operator))
	return NewOperatorToken(operator, prefix, infix, postfix, span)
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
