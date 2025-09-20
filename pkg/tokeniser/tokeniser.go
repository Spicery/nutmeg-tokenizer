package tokeniser

import (
	"fmt"
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
	expectingStack [][]string      // Stack of expecting arrays for context tracking
	rules          *TokeniserRules // Custom rules for this tokeniser instance
}

// Regular expressions for token matching
var (
	identifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*`)
	operatorRegex   = regexp.MustCompile(`^[\*/%\+\-<>~!&^|?=:]+`)
	closeDelimRegex = regexp.MustCompile(`^[\)\]\}]`)
	// numericRegex    = regexp.MustCompile(`^(?:\d+[xobt][0-9A-Z]+(?:_[0-9A-Z]+)*|\d+r[0-9A-Z]+(?:_[0-9A-Z]+)*|\d+(?:_\d+)*)(?:\.[0-9A-Z01T]*(?:_[0-9A-Z01T]+)*)?(?:e[+-]?\d+)?`)
	radixRegex   = regexp.MustCompile(`^(\d+[xobtr])([0-9A-Z]+(?:_[0-9A-Z]+)*)(\.[0-9A-Z]*(?:_[0-9A-Z]+)*)?(?:e([+-]?\d+))?`)
	decimalRegex = regexp.MustCompile(`^(\d+(?:_\d+)*)(\.\d*(?:_\d+)*)?(?:e([+-]?\d+))?`)
	commentRegex = regexp.MustCompile(`^###.*`)
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

// New creates a new tokeniser instance with default rules.
func New(input string) *Tokeniser {
	return &Tokeniser{
		input:          input,
		line:           1,
		column:         1,
		tokens:         make([]*Token, 0),
		expectingStack: make([][]string, 0),
		rules:          nil, // Use global variables for backward compatibility
	}
}

// NewWithRules creates a new tokeniser instance with custom rules.
func NewWithRules(input string, rules *TokeniserRules) *Tokeniser {
	return &Tokeniser{
		input:          input,
		line:           1,
		column:         1,
		tokens:         make([]*Token, 0),
		expectingStack: make([][]string, 0),
		rules:          rules,
	}
}

// Helper methods to access rules with fallback to global variables

func (t *Tokeniser) getStartTokens() map[string]StartTokenData {
	if t.rules != nil {
		return t.rules.StartTokens
	}
	return startTokens
}

func (t *Tokeniser) getLabelTokens() map[string]LabelTokenData {
	if t.rules != nil {
		return t.rules.LabelTokens
	}
	return labelTokens
}

func (t *Tokeniser) getWildcardTokens() map[string]bool {
	if t.rules != nil {
		return t.rules.WildcardTokens
	}
	// Default wildcard is just ":"
	return map[string]bool{":": true}
}

func (t *Tokeniser) getDelimiterMappings() map[string][]string {
	if t.rules != nil {
		return t.rules.DelimiterMappings
	}
	return delimiterMappings
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
func (t *Tokeniser) addTokenAndManageStack(token *Token) error {
	// Check if numeric token is valid before adding it
	if token.Type == NumericLiteral {
		if valid, reason := token.isValidNumber(); !valid {
			// Replace the token with an exception token
			exceptionToken := NewExceptionToken(token.Text, "invalid numeric literal: "+reason, token.Span)
			t.tokens = append(t.tokens, exceptionToken)
			return fmt.Errorf("tokenisation error at line %d, column %d: %s",
				exceptionToken.Span.Start.Line, exceptionToken.Span.Start.Col, *exceptionToken.Reason)
		}
	}

	// Check for newlines after this token's position
	savedPosition := t.position
	savedLine := t.line
	savedColumn := t.column
	sawNewlineAfter := t.skipWhitespaceAndComments()
	t.position = savedPosition // Restore position since we're just peeking ahead
	t.line = savedLine
	t.column = savedColumn
	token.LnAfter = &sawNewlineAfter

	t.tokens = append(t.tokens, token)

	// If this is an exception token, stop processing
	if token.Type == ExceptionToken {
		return fmt.Errorf("tokenisation error at line %d, column %d: %s",
			token.Span.Start.Line, token.Span.Start.Col, *token.Reason)
	}

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
	return nil
} // Tokenise processes the input and returns a slice of tokens.
func (t *Tokeniser) Tokenise() ([]*Token, error) {
	for t.position < len(t.input) {
		if err := t.nextToken(); err != nil {
			return t.tokens, err
		}
	}
	return t.tokens, nil
}

// nextToken processes the next token from the input.
func (t *Tokeniser) nextToken() error {
	// Skip whitespace and comments, tracking if we saw a newline
	sawNewlineBefore := t.skipWhitespaceAndComments()

	if t.position >= len(t.input) {
		return nil
	}

	start := Position{Line: t.line, Col: t.column}

	// Try to match different token types
	if token := t.matchString(); token != nil {
		token.Span.Start = start
		if sawNewlineBefore {
			token.LnBefore = &sawNewlineBefore
		}
		return t.addTokenAndManageStack(token)
	}

	if token := t.matchNumeric(); token != nil {
		token.Span.Start = start
		if sawNewlineBefore {
			token.LnBefore = &sawNewlineBefore
		}
		return t.addTokenAndManageStack(token)
	}

	// Check custom rules first - they take precedence over defaults
	if token := t.matchCustomRules(); token != nil {
		token.Span.Start = start
		if sawNewlineBefore {
			token.LnBefore = &sawNewlineBefore
		}
		return t.addTokenAndManageStack(token)
	}

	if token := t.matchIdentifier(); token != nil {
		token.Span.Start = start
		if sawNewlineBefore {
			token.LnBefore = &sawNewlineBefore
		}
		return t.addTokenAndManageStack(token)
	}

	// Only use default special labels matching if no custom label rules are defined
	if t.rules == nil || len(t.rules.LabelTokens) == 0 {
		if token := t.matchSpecialLabels(); token != nil {
			token.Span.Start = start
			if sawNewlineBefore {
				token.LnBefore = &sawNewlineBefore
			}
			return t.addTokenAndManageStack(token)
		}
	}

	// Only use default operator matching if no custom operator rules are defined
	if t.rules == nil || len(t.rules.OperatorPrecedences) == 0 {
		if token := t.matchOperator(); token != nil {
			token.Span.Start = start
			if sawNewlineBefore {
				token.LnBefore = &sawNewlineBefore
			}
			return t.addTokenAndManageStack(token)
		}
	}

	// Only use default delimiter matching if no custom bracket rules are defined
	if t.rules == nil || len(t.rules.DelimiterMappings) == 0 {
		if token := t.matchDelimiter(); token != nil {
			token.Span.Start = start
			if sawNewlineBefore {
				token.LnBefore = &sawNewlineBefore
			}
			return t.addTokenAndManageStack(token)
		}
	}

	// If nothing matches, create an unclassified token
	r, size := utf8.DecodeRuneInString(t.input[t.position:])
	text := string(r)
	end := Position{Line: t.line, Col: t.column + size}
	span := Span{Start: start, End: end}

	token := NewToken(text, UnclassifiedToken, span)
	if sawNewlineBefore {
		token.LnBefore = &sawNewlineBefore
	}
	t.advance(size)
	return t.addTokenAndManageStack(token)
}

// skipWhitespaceAndComments advances past whitespace characters and comments.
// Returns true if a newline (LF or CR) was encountered in the skipped content.
func (t *Tokeniser) skipWhitespaceAndComments() bool {
	sawNewline := false
	
	for t.position < len(t.input) {
		// Check for comments first
		if match := commentRegex.FindString(t.input[t.position:]); match != "" {
			t.advance(len(match))
			sawNewline = true // End-of-line comments always include a newline conceptually
			continue
		}
		
		// Check for whitespace
		r, size := utf8.DecodeRuneInString(t.input[t.position:])
		if !unicode.IsSpace(r) {
			break
		}
		
		// Check if this whitespace character is a newline
		if r == '\n' || r == '\r' {
			sawNewline = true
		}
		
		t.advance(size)
	}
	
	return sawNewline
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
	// First try to match radix-based numbers (must check before decimal)
	if radixMatch := radixRegex.FindStringSubmatch(t.input[t.position:]); radixMatch != nil {
		return t.parseRadixNumber(radixMatch)
	}

	// Then try to match decimal numbers
	if decimalMatch := decimalRegex.FindStringSubmatch(t.input[t.position:]); decimalMatch != nil {
		return t.parseDecimalNumber(decimalMatch)
	}

	return nil
}

// parseRadixNumber parses a number with radix notation (e.g., 0x, 0o, 0b, 0t, or nr).
func (t *Tokeniser) parseRadixNumber(match []string) *Token {
	fullMatch := match[0]
	radixPart := match[1]
	mantissa := match[2]
	fraction := ""
	exponent := ""

	if len(match) > 3 && match[3] != "" {
		fraction = match[3][1:] // Remove the leading dot
	}
	if len(match) > 4 && match[4] != "" {
		exponent = match[4] // Already without the 'e'
	}

	// Extract radix prefix and determine base
	lastChar := radixPart[len(radixPart)-1]
	radixPrefix := ""
	var base int

	switch lastChar {
	case 'x':
		if radixPart == "0x" {
			radixPrefix = "0x"
			base = 16
		} else {
			// Invalid hex format - should be 0x
			return t.createExceptionToken(fullMatch, "invalid literal")
		}
	case 'o':
		if radixPart == "0o" {
			radixPrefix = "0o"
			base = 8
		} else {
			// Invalid octal format - should be 0o
			return t.createExceptionToken(fullMatch, "invalid literal")
		}
	case 'b':
		if radixPart == "0b" {
			radixPrefix = "0b"
			base = 2
		} else {
			// Invalid binary format - should be 0b
			return t.createExceptionToken(fullMatch, "invalid literal")
		}
	case 't':
		if radixPart == "0t" {
			// Handle balanced ternary
			mantissa = strings.ReplaceAll(mantissa, "_", "")
			if fraction != "" {
				fraction = strings.ReplaceAll(fraction, "_", "")
			}

			end := Position{Line: t.line, Col: t.column + len(fullMatch)}
			span := Span{End: end}
			t.advance(len(fullMatch))

			return NewBalancedTernaryToken(fullMatch, mantissa, fraction, exponent, span)
		} else {
			// Invalid ternary format - should be 0t
			return t.createExceptionToken(fullMatch, "invalid literal")
		}
	case 'r':
		// Parse the radix number (e.g., "2r", "16r", "36r")
		radixStr := radixPart[:len(radixPart)-1]
		radixPrefix = radixPart

		parsedRadix := 0
		for _, digit := range radixStr {
			if digit >= '0' && digit <= '9' {
				parsedRadix = parsedRadix*10 + int(digit-'0')
			} else {
				return t.createExceptionToken(fullMatch, "invalid literal")
			}
		}

		if parsedRadix < 2 || parsedRadix > 36 {
			return t.createExceptionToken(fullMatch, "invalid literal")
		}

		base = parsedRadix
	default:
		return t.createExceptionToken(fullMatch, "invalid literal")
	}

	// Remove underscores from mantissa and fraction
	mantissa = strings.ReplaceAll(mantissa, "_", "")
	if fraction != "" {
		fraction = strings.ReplaceAll(fraction, "_", "")
	}

	end := Position{Line: t.line, Col: t.column + len(fullMatch)}
	span := Span{End: end}
	t.advance(len(fullMatch))

	return NewNumericToken(fullMatch, radixPrefix, base, mantissa, fraction, exponent, span)
}

// parseDecimalNumber parses a decimal number.
func (t *Tokeniser) parseDecimalNumber(match []string) *Token {
	fullMatch := match[0]
	mantissa := match[1]
	fraction := ""
	exponent := ""

	if len(match) > 2 && match[2] != "" {
		fraction = match[2][1:] // Remove the leading dot
	}
	if len(match) > 3 && match[3] != "" {
		exponent = match[3] // Already without the 'e'
	}

	// Remove underscores from mantissa and fraction
	mantissa = strings.ReplaceAll(mantissa, "_", "")
	if fraction != "" {
		fraction = strings.ReplaceAll(fraction, "_", "")
	}

	end := Position{Line: t.line, Col: t.column + len(fullMatch)}
	span := Span{End: end}
	t.advance(len(fullMatch))

	return NewNumericToken(fullMatch, "", 10, mantissa, fraction, exponent, span)
}

// createExceptionToken creates an exception token for invalid numeric formats.
func (t *Tokeniser) createExceptionToken(text, reason string) *Token {
	end := Position{Line: t.line, Col: t.column + len(text)}
	span := Span{End: end}
	t.advance(len(text))
	return NewExceptionToken(text, reason, span)
}

// matchIdentifier attempts to match an identifier.
func (t *Tokeniser) matchIdentifier() *Token {
	match := identifierRegex.FindString(t.input[t.position:])
	if match == "" {
		return nil
	}

	end := Position{Line: t.line, Col: t.column + len(match)}
	span := Span{End: end}

	// Check if it's a start token (only if no custom start rules are defined)
	if t.rules == nil || len(t.rules.StartTokens) == 0 {
		if startData, isStart := startTokens[match]; isStart {
			t.advance(len(match))
			return NewStartToken(match, startData.Expecting, startData.ClosedBy, span)
		}
	}

	// Check if it's an end token - only if no custom start rules are defined
	if t.rules == nil || len(t.rules.StartTokens) == 0 {
		if strings.HasPrefix(match, "end") {
			t.advance(len(match))
			return NewToken(match, EndToken, span)
		}
	}

	// Check if it's a label token (L) - only if no custom label rules are defined
	if t.rules == nil || len(t.rules.LabelTokens) == 0 {
		if labelData, isLabel := labelTokens[match]; isLabel {
			t.advance(len(match))
			return NewLabelToken(match, labelData.Expecting, labelData.In, span)
		}
	}

	// Check if it's a compound token (C) - only if no custom compound rules are defined
	if t.rules == nil || len(t.rules.CompoundTokens) == 0 {
		if compoundData, isCompound := compoundTokens[match]; isCompound {
			t.advance(len(match))
			return NewCompoundToken(match, compoundData.Expecting, compoundData.In, span)
		}
	}

	// Default to VariableToken, but check for prefix tokens
	var tokenType TokenType = VariableToken

	// Check if it's a prefix token (P) - only if no custom prefix rules are defined
	if t.rules == nil || len(t.rules.PrefixTokens) == 0 {
		if prefixTokens[match] {
			tokenType = PrefixToken
		}
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

	// Check for wildcard tokens
	wildcardTokens := t.getWildcardTokens()
	for wildcardText := range wildcardTokens {
		if t.position < len(t.input) && strings.HasPrefix(t.input[t.position:], wildcardText) {
			// For single character wildcards, make sure it's not part of a longer operator
			if len(wildcardText) == 1 && wildcardText == ":" {
				if t.position+1 < len(t.input) && strings.ContainsRune("*/%+-<>~!&^|?=:", rune(t.input[t.position+1])) {
					continue // Skip this wildcard as it's part of a longer operator
				}
			}

			end := Position{Line: t.line, Col: t.column + len(wildcardText)}
			span := Span{End: end}

			// Check if we have context from the expecting stack
			expected := t.getCurrentlyExpected()
			if len(expected) > 0 {
				// Use the first expected token as the basis for the wildcard
				expectedText := expected[0]

				// Check if it's a label token
				labelTokens := t.getLabelTokens()
				if labelData, exists := labelTokens[expectedText]; exists {
					// Create a wildcard token that copies attributes from the expected label
					t.advance(len(wildcardText))
					return NewWildcardLabelTokenWithAttributes(wildcardText, expectedText, labelData.Expecting, labelData.In, span)
				}

				// Check if it's a start token
				startTokens := t.getStartTokens()
				if startData, exists := startTokens[expectedText]; exists {
					// Create wildcard start token
					t.advance(len(wildcardText))
					return NewWildcardStartToken(wildcardText, expectedText, startData.ClosedBy, span)
				}

				// Check if it's an end token (starts with "end")
				if strings.HasPrefix(expectedText, "end") {
					// Create wildcard end token
					t.advance(len(wildcardText))
					return NewWildcardEndToken(wildcardText, expectedText, span)
				}
			}

			// No context available, create unclassified token
			t.advance(len(wildcardText))
			return NewToken(wildcardText, UnclassifiedToken, span)
		}
	}

	return nil
}

// matchCustomRules checks for any custom rules that match at the current position.
// Custom rules take precedence over default rules.
func (t *Tokeniser) matchCustomRules() *Token {
	if t.rules == nil || t.rules.TokenLookup == nil {
		return nil // No custom rules
	}

	// Determine the next token text using proper tokenization rules
	var text string

	// Check for alphanumeric + underbar sequences
	if match := identifierRegex.FindString(t.input[t.position:]); match != "" {
		text = match
	} else if match := operatorRegex.FindString(t.input[t.position:]); match != "" {
		// Check for sign character sequences
		text = match
	} else if t.position < len(t.input) {
		// Everything else is a single character
		r, size := utf8.DecodeRuneInString(t.input[t.position:])
		text = string(r)
		_ = size // We'll use len(text) for advancing
	} else {
		return nil
	}

	// Efficient lookup - single map access
	entry, exists := t.rules.TokenLookup[text]
	if !exists {
		return nil
	}

	end := Position{Line: t.line, Col: t.column + len(text)}
	span := Span{End: end}

	// Process the single rule entry
	switch entry.Type {
	case CustomWildcard:
		// For single character wildcards, make sure it's not part of a longer operator
		if len(text) == 1 && text == ":" {
			if t.position+1 < len(t.input) && strings.ContainsRune("*/%+-<>~!&^|?=:", rune(t.input[t.position+1])) {
				return nil // Skip this wildcard as it's part of a longer operator
			}
		}

		// Check if we have context from the expecting stack
		expected := t.getCurrentlyExpected()
		if len(expected) > 0 {
			// Use the first expected token as the basis for the wildcard
			expectedText := expected[0]

			// Check if it's a label token
			if labelData, exists := t.rules.LabelTokens[expectedText]; exists {
				// Create a wildcard token that copies attributes from the expected label
				t.advance(len(text))
				return NewWildcardLabelTokenWithAttributes(text, expectedText, labelData.Expecting, labelData.In, span)
			}

			// Check if it's a start token
			if startData, exists := t.rules.StartTokens[expectedText]; exists {
				// Create wildcard start token
				t.advance(len(text))
				return NewWildcardStartToken(text, expectedText, startData.ClosedBy, span)
			}

			// Check if it's an end token (starts with "end")
			if strings.HasPrefix(expectedText, "end") {
				// Create wildcard end token
				t.advance(len(text))
				return NewWildcardEndToken(text, expectedText, span)
			}
		}

		// No context available, create unclassified token
		t.advance(len(text))
		return NewToken(text, UnclassifiedToken, span)

	case CustomStart:
		startData := entry.Data.(StartTokenData)
		t.advance(len(text))
		return NewStartToken(text, startData.Expecting, startData.ClosedBy, span)

	case CustomEnd:
		t.advance(len(text))
		return NewToken(text, EndToken, span)

	case CustomLabel:
		labelData := entry.Data.(LabelTokenData)
		t.advance(len(text))
		return NewLabelToken(text, labelData.Expecting, labelData.In, span)

	case CustomCompound:
		compoundData := entry.Data.(CompoundTokenData)
		t.advance(len(text))
		return NewCompoundToken(text, compoundData.Expecting, compoundData.In, span)

	case CustomPrefix:
		t.advance(len(text))
		return NewToken(text, PrefixToken, span)

	case CustomOperator:
		precedence := entry.Data.([3]int)
		t.advance(len(text))
		return NewOperatorToken(text, precedence[0], precedence[1], precedence[2], span)

	case CustomOpenDelimiter:
		delimiterData := entry.Data.(struct {
			ClosedBy []string
			IsInfix  bool
			IsPrefix bool
		})
		t.advance(len(text))
		return NewDelimiterToken(text, delimiterData.ClosedBy, delimiterData.IsInfix, delimiterData.IsPrefix, span)

	case CustomCloseDelimiter:
		t.advance(len(text))
		return NewToken(text, CloseDelimiter, span)
	}

	return nil
}

// matchOperator attempts to match an operator.
func (t *Tokeniser) matchOperator() *Token {
	match := operatorRegex.FindString(t.input[t.position:])
	if match == "" {
		return nil
	}

	// Check if this might be a custom delimiter first (open delimiters)
	delimMappings := t.getDelimiterMappings()
	firstChar := string(match[0])
	if _, isDelimiter := delimMappings[firstChar]; isDelimiter {
		// Let delimiter matching handle this instead
		return nil
	}

	// Check if this might be a closing delimiter
	for _, closedByList := range delimMappings {
		for _, closer := range closedByList {
			if firstChar == closer {
				// Let delimiter matching handle this instead
				return nil
			}
		}
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

// matchDelimiter attempts to match a delimiter using default rules only.
// This is only called when no custom bracket rules are defined.
func (t *Tokeniser) matchDelimiter() *Token {
	if t.position >= len(t.input) {
		return nil
	}

	char := string(t.input[t.position])

	// Check for open delimiters using default mappings
	if closedBy, isOpen := delimiterMappings[char]; isOpen {
		end := Position{Line: t.line, Col: t.column + 1}
		span := Span{End: end}

		props := delimiterProperties[char]
		isInfix, isPrefix := props[0], props[1]

		t.advance(1)
		return NewDelimiterToken(char, closedBy, isInfix, isPrefix, span)
	}

	// Check for close delimiters using default regex
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
