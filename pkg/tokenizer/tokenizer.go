package tokenizer

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Tokenizer represents the main tokenizer structure.
type Tokenizer struct {
	input          string
	position       int
	line           int
	column         int
	markStack      []int // Stack of position markers
	lineNoStack    []int // Array to store line numbers for each token
	lineColStack   []int // Array to store column numbers for each token
	tokens         []*Token
	expectingStack [][]string      // Stack of expecting arrays for context tracking
	rules          *TokenizerRules // Custom rules for this tokenizer instance
}

// Regular expressions for token matching
var (
	identifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*`)
	operatorRegex   = regexp.MustCompile(`^[.\*/%\+\-<>~!&^|?=:$]+`)
	radixRegex      = regexp.MustCompile(`^(\d+[xobtr])([0-9A-Z]+(?:_[0-9A-Z]+)*)(\.[0-9A-Z]*(?:_[0-9A-Z]+)*)?(?:e([+-]?\d+))?`)
	decimalRegex    = regexp.MustCompile(`^(\d+(?:_\d+)*)(\.\d*(?:_\d+)*)?(?:e([+-]?\d+))?`)
	commentRegex    = regexp.MustCompile(`^###.*`)
)

// Start token mappings with expecting and closed_by information
type StartTokenData struct {
	Expecting []string
	ClosedBy  []string
	Arity     Arity
}

// Bridge tokens (B) with their attributes
type BridgeTokenData struct {
	Expecting []string
	In        []string
	Arity     Arity
}

type PrefixTokenData struct {
	Arity Arity
}

// Base precedence values for operator characters (from operators.md)
// Should follow this order: .([{*/%+-<>~!&^|?:=
var baseOperatorPrecedence = map[rune]int{
	'.': 10,
	'(': 20,
	'[': 30,
	'{': 40,
	'*': 50,
	'/': 60,
	'%': 70,
	'+': 80,
	'-': 90,
	'<': 100,
	'>': 110,
	'~': 120,
	'!': 130,
	'&': 140,
	'^': 150,
	'|': 160,
	'?': 170,
	'=': 180,
	':': 190,
}

// NewTokenizer creates a new tokenizer instance with default rules.
func NewTokenizer(input string) *Tokenizer {
	return NewTokenizerWithRules(input, DefaultRules())
}

// NewTokenizerWithRules creates a new tokenizer instance with custom rules.
func NewTokenizerWithRules(input string, rules *TokenizerRules) *Tokenizer {
	return &Tokenizer{
		input:          input,
		line:           1,
		column:         1,
		tokens:         make([]*Token, 0),
		expectingStack: make([][]string, 0),
		rules:          rules,
	}
}

// Helper methods to access rules with fallback to global variables

// pushExpecting pushes a new set of expected tokens onto the stack.
func (t *Tokenizer) pushExpecting(expected []string) {
	t.expectingStack = append(t.expectingStack, expected)
}

// popExpecting removes the top set of expected tokens from the stack.
func (t *Tokenizer) popExpecting() {
	if len(t.expectingStack) > 0 {
		t.expectingStack = t.expectingStack[:len(t.expectingStack)-1]
	}
}

func (t *Tokenizer) replaceExpecting(expected []string) {
	if len(t.expectingStack) > 0 {
		t.expectingStack[len(t.expectingStack)-1] = expected
	}
}

// getCurrentlyExpected returns the currently expected tokens, or nil if stack is empty.
func (t *Tokenizer) getCurrentlyExpected() []string {
	if len(t.expectingStack) == 0 {
		return nil
	}
	return t.expectingStack[len(t.expectingStack)-1]
}

// addTokenAndManageStack adds a token to the tokens slice and manages the expecting stack.
func (t *Tokenizer) addTokenAndManageStack(token *Token) error {
	// Check if numeric token is valid before adding it
	if token.Type == NumericLiteralTokenType {
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
	if sawNewlineAfter {
		token.LnAfter = &sawNewlineAfter
	}

	t.tokens = append(t.tokens, token)

	// If this is an exception token, stop processing
	if token.Type == ExceptionTokenType {
		return fmt.Errorf("tokenisation error at line %d, column %d: %s",
			token.Span.Start.Line, token.Span.Start.Col, *token.Reason)
	}

	// Manage the expecting stack based on token type and text
	switch token.Type {
	case StartTokenType:
		// Push expected tokens for this start token
		if len(token.Expecting) > 0 {
			t.pushExpecting(token.Expecting)
		}
	case EndTokenType:
		// Pop the expecting stack
		t.popExpecting()
	case BridgeTokenType:
		// Update expecting for bridge tokens based on their attributes
		if token.Expecting != nil {
			// If the token has explicit expecting, replace current expectations
			t.replaceExpecting(token.Expecting)
		}
	}
	return nil
}

// Tokenize processes the input and returns a slice of tokens.
func (t *Tokenizer) Tokenize() ([]*Token, error) {
	for t.position < len(t.input) {
		if err := t.nextToken(); err != nil {
			return t.tokens, err
		}
	}
	return t.tokens, nil
}

// nextToken processes the next token from the input.
func (t *Tokenizer) nextToken() error {
	// Skip whitespace and comments, tracking if we saw a newline
	sawNewlineBefore := t.skipWhitespaceAndComments()

	if t.position >= len(t.input) {
		return nil
	}

	start := Position{Line: t.line, Col: t.column}

	// Try to match different token types
	{
		token, err := t.matchString()
		if err != nil {
			return err
		}
		if token != nil {
			token.Span.Start = start
			if sawNewlineBefore {
				token.LnBefore = &sawNewlineBefore
			}
			return t.addTokenAndManageStack(token)
		}
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

	// If nothing matches, create an unclassified token
	r, size := utf8.DecodeRuneInString(t.input[t.position:])
	text := string(r)
	end := Position{Line: t.line, Col: t.column + size}
	span := Span{Start: start, End: end}

	token := NewToken(text, UnclassifiedTokenType, span)
	if sawNewlineBefore {
		token.LnBefore = &sawNewlineBefore
	}
	t.advance(size)
	return t.addTokenAndManageStack(token)
}

// skipWhitespaceAndComments advances past whitespace characters and comments.
// Returns true if a newline (LF or CR) was encountered in the skipped content.
func (t *Tokenizer) skipWhitespaceAndComments() bool {
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

// matchNumeric attempts to match a numeric literal.
func (t *Tokenizer) matchNumeric() *Token {
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
func (t *Tokenizer) parseRadixNumber(match []string) *Token {
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

			exponentVal := 0
			if exponent != "" {
				var err error
				exponentVal, err = strconv.Atoi(exponent)
				if err != nil {
					return t.createExceptionToken(fullMatch, fmt.Sprintf("invalid literal: %s", exponent))
				}
			}
			return NewBalancedTernaryToken(fullMatch, mantissa, fraction, exponentVal, span)
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

	exponentVal := 0
	if exponent != "" {
		var err error
		exponentVal, err = strconv.Atoi(exponent)
		if err != nil {
			return t.createExceptionToken(fullMatch, "invalid literal")
		}
	}
	return NewNumericToken(fullMatch, radixPrefix, base, mantissa, fraction, exponentVal, span)
}

// parseDecimalNumber parses a decimal number.
func (t *Tokenizer) parseDecimalNumber(match []string) *Token {
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

	exponentVal := 0
	if exponent != "" {
		var err error
		exponentVal, err = strconv.Atoi(exponent)
		if err != nil {
			return t.createExceptionToken(fullMatch, fmt.Sprintf("invalid literal: %s", err))
		}
	}
	return NewNumericToken(fullMatch, "", 10, mantissa, fraction, exponentVal, span)
}

// createExceptionToken creates an exception token for invalid numeric formats.
func (t *Tokenizer) createExceptionToken(text, reason string) *Token {
	end := Position{Line: t.line, Col: t.column + len(text)}
	span := Span{End: end}
	t.advance(len(text))
	return NewExceptionToken(text, reason, span)
}

// matchCustomRules checks for any custom rules that match at the current position.
// Custom rules take precedence over default rules.
func (t *Tokenizer) matchCustomRules() *Token {
	if t.rules == nil || t.rules.TokenLookup == nil {
		return nil // No custom rules
	}

	// Check for alphanumeric + underbar sequences
	// fmt.Println("Custom rules check at position", t.position, "char:", string(t.input[t.position]))
	is_identifier, text, ok := nextIdOrOp(t)
	if !ok {
		return nil
	}

	// fmt.Println("Custom rules token text:", text)
	// fmt.Println("is_identifier?", is_identifier)

	end := Position{Line: t.line, Col: t.column + len(text)}
	span := Span{End: end}

	// Efficient lookup - single map access
	entry, exists := t.rules.TokenLookup[text]
	if !exists {
		if is_identifier {

			// If it's an identifier and no special type, treat as VariableToken
			t.advance(len(text))
			return NewToken(text, VariableTokenType, span)
		}
		return nil // No matching custom rule
	}

	// Process the single rule entry
	switch entry.Type {
	case CustomWildcard:
		// Check if we have context from the expecting stack
		expected := t.getCurrentlyExpected()
		if len(expected) > 0 {
			// Use the first expected token as the basis for the wildcard
			expectedText := expected[0]

			// Check if it's a bridge token
			if bridgeData, exists := t.rules.BridgeTokens[expectedText]; exists {
				// Create a wildcard token that copies attributes from the expected bridge
				t.advance(len(text))
				return NewWildcardBridgeToken(text, expectedText, bridgeData.Expecting, bridgeData.In, bridgeData.Arity, span)
			}
		}

		// No context available, create unclassified token
		t.advance(len(text))
		return NewToken(text, UnclassifiedTokenType, span)

	case CustomStart:
		startData := entry.Data.(StartTokenData)
		t.advance(len(text))
		return NewStartToken(text, startData.Expecting, startData.ClosedBy, span, startData.Arity)

	case CustomEnd:
		t.advance(len(text))
		return NewToken(text, EndTokenType, span)

	case CustomBridge:
		bridgeData := entry.Data.(BridgeTokenData)
		t.advance(len(text))
		return NewStmntBridgeToken(text, bridgeData.Expecting, bridgeData.In, span)

	case CustomPrefix:
		prefixData := entry.Data.(PrefixTokenData)

		t.advance(len(text))
		return NewPrefixToken(text, PrefixTokenType, span, prefixData.Arity)

	case CustomMark:
		t.advance(len(text))
		return NewToken(text, MarkTokenType, span)

	case CustomOperator:
		precedence := entry.Data.([3]int)
		t.advance(len(text))
		return NewOperatorToken(text, precedence[0], precedence[1], precedence[2], span)

	case CustomOpenDelimiter:
		delimiterData := entry.Data.(struct {
			ClosedBy  []string
			InfixPrec int
			IsPrefix  bool
		})
		t.advance(len(text))
		return NewDelimiterToken(text, delimiterData.ClosedBy, delimiterData.InfixPrec, delimiterData.IsPrefix, span)

	case CustomCloseDelimiter:
		t.advance(len(text))
		return NewToken(text, CloseDelimiterTokenType, span)
	}

	return nil
}

// nextIdOrOp is a helper function that attempts to match an identifier or operator token.
// It returns three values:
// - A boolean indicating if the matched token is an identifier.
// - The matched text.
// - A boolean indicating if a match was found.
func nextIdOrOp(t *Tokenizer) (bool, string, bool) {
	if match := identifierRegex.FindString(t.input[t.position:]); match != "" {
		text := match
		return true, text, true
	}
	if match := operatorRegex.FindString(t.input[t.position:]); match != "" {
		// Check for sign character sequences
		text := match
		return false, text, true
	}
	if t.position < len(t.input) {
		// Everything else is a single character
		r, _ := utf8.DecodeRuneInString(t.input[t.position:])
		text := string(r)
		return false, text, true
	}
	return false, "", false
}

// advance moves the position forward and updates line/column tracking.
func (t *Tokenizer) advance(n int) {
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

func (t *Tokenizer) peek() (rune, bool) {
	if t.position >= len(t.input) {
		return rune(0), false // End of input
	}
	r, b := utf8.DecodeRuneInString(t.input[t.position:])
	return r, b > 0
}

func (t *Tokenizer) tryPeekTripleOpeningQuotes() (rune, bool) {
	return t.tryPeekTripleQuotes(true)
}

func (t *Tokenizer) tryReadTripleClosingQuotes() (rune, bool) {
	return t.tryReadTripleQuotes(false)
}

func (t *Tokenizer) tryReadTripleOpeningQuotes() (rune, bool) {
	return t.tryReadTripleQuotes(true)
}

func (t *Tokenizer) tryReadTripleQuotes(is_opening bool) (rune, bool) {
	r, b := t.tryPeekTripleQuotes(is_opening)

	if b {
		// Consume all three quotes
		t.consume() // Consume the first quote
		t.consume() // Consume the second quote
		t.consume() // Consume the third quote
	}

	return r, b
}

func (t *Tokenizer) tryPeekTripleQuotes(is_opening bool) (rune, bool) {
	// Peek the first character to check if it’s a valid quote character
	r1, ok1 := t.peek()
	if !ok1 {
		return 0, false // End of input
	}
	if is_opening {
		if !isOpeningQuoteChar(r1) {
			return 0, false // Invalid opening quote character
		}
	} else {
		if !isClosingQuoteChar(r1) {
			return 0, false // Invalid closing quote character
		}
	}

	// Check if the next two characters match the first one
	r2, ok2 := t.peekN(2)
	r3, ok3 := t.peekN(3)
	if !(ok2 && ok3 && r2 == r1 && r3 == r1) {
		return 0, false // Not a triple quote
	}

	return r1, true // Successfully read triple quotes
}

func (t *Tokenizer) peekN(n int) (rune, bool) {
	currentPos := t.position // Start at the current position
	var r rune
	var size int

	// Iterate through the input to locate the nth rune
	for range n {
		if currentPos >= len(t.input) {
			// If we reach the end of input before finding the nth rune, return false
			return 0, false
		}

		r, size = utf8.DecodeRuneInString(t.input[currentPos:])
		if r == utf8.RuneError {
			// Handle invalid UTF-8 character by returning false
			return 0, false
		}

		// Advance to the next rune
		currentPos += size
	}

	// Return the nth rune and true (indicating success)
	return r, true
}

func isOpeningQuoteChar(r rune) bool {
	return r == '\'' || r == '"' || r == '`' || r == '«'
}

func isClosingQuoteChar(r rune) bool {
	return r == '\'' || r == '"' || r == '`' || r == '»'
}

// Consume the current rune and advance the position
func (t *Tokenizer) consume() rune {
	r, ok := t.peek()
	if !ok {
		return r
	}
	t.advance(utf8.RuneLen(r)) // Move the byte position forward
	return r
}

func (t *Tokenizer) markPosition() {
	// Mark the current position in the input
	t.markStack = append(t.markStack, t.position)
	t.lineNoStack = append(t.lineNoStack, t.line)
	t.lineColStack = append(t.lineColStack, t.column)
}

// Reset the position to the last marked position
func (t *Tokenizer) resetPosition() {
	// Reset the position to the last marked position
	if len(t.markStack) == 0 {
		return
	}
	n1 := len(t.markStack) - 1
	t.position = t.markStack[n1]
	t.line = t.lineNoStack[n1]
	t.column = t.lineColStack[n1]
	t.markStack = t.markStack[:n1]
	t.lineNoStack = t.lineNoStack[:n1]
	t.lineColStack = t.lineColStack[:n1]
}

func (t *Tokenizer) popMark() string {
	// Pop the last marked position and return the corresponding substring
	if len(t.markStack) == 0 {
		return ""
	}
	n1 := len(t.markStack) - 1
	start := t.markStack[n1]
	t.markStack = t.markStack[:n1]
	t.lineNoStack = t.lineNoStack[:n1]
	t.lineColStack = t.lineColStack[:n1]
	return t.input[start:t.position]
}

// hasMoreInput checks whether there is any remaining input to be processed.
// It returns true if the current position has not reached the end of the input
// string, indicating that there is more content to tokenize.
func (t *Tokenizer) hasMoreInput() bool {
	return t.position < len(t.input)
}

func (t *Tokenizer) tryConsumeRune(char rune) bool {
	// Check if the next rune matches the given character
	r, ok := t.peek()
	if !ok {
		return false // End of input
	}
	if r != char {
		return false // No match
	}
	t.consume() // Consume the character
	return true
}

func (t *Tokenizer) readRestOfLine() string {
	// Read the rest of the line until a newline or end of input
	var text strings.Builder
	for t.hasMoreInput() {
		r, _ := t.peek()
		if r == '\n' || r == '\r' {
			break // End of line
		}
		text.WriteRune(t.consume())
	}
	t.tryConsumeNewline()
	return text.String()
}

func (t *Tokenizer) tryConsumeNewline() bool {
	// Consume '\r' and optionally '\n' to handle both '\n' and '\r\n' line endings.
	// IMPORTANT: This direct indexing is only safe because we are testing against
	// the ASCII range. In this range, the UTF-8 encoding is identical to the ASCII.
	if t.hasMoreInput() && t.input[t.position] == '\r' {
		t.consume() // Consume '\r'
		if t.hasMoreInput() && t.input[t.position] == '\n' {
			t.consume() // Consume '\n' if it follows
		}
		return true
	} else if t.hasMoreInput() && t.input[t.position] == '\n' {
		t.consume() // Consume '\n'
		return true
	}
	return false // No newline consumed
}

// Calculates the closing indent if we are on the last line of a multiline string.
func textIsWhitespaceFollowedBy3Quotes(text string, quote rune) (bool, string) {
	// Check if the text is whitespace followed by three quotes
	if len(text) < 3 {
		return false, ""
	}
	var indent strings.Builder
	whitespace := true
	count := 0
	for _, r := range text {
		if whitespace {
			if unicode.IsSpace(r) {
				indent.WriteRune(r)
				continue
			} else if r == quote {
				whitespace = false
				count++
			} else {
				return false, ""
			}
		} else {
			if r == quote {
				count++
				if count >= 3 {
					return true, indent.String()
				}
			} else {
				return false, ""
			}
		}
	}
	return false, ""
}

func (t *Tokenizer) tryConsumeText(text string) bool {
	// Check if the next characters match the given text
	if strings.HasPrefix(t.input[t.position:], text) {
		t.consumeN(len(text)) // Consume the matching text
		return true
	}
	return false
}

func (t *Tokenizer) consumeN(n int) {
	// Consume n runes from the input
	for range n {
		if t.hasMoreInput() {
			t.consume()
		} else {
			break // Stop if we reach the end of input
		}
	}
}

func (t *Tokenizer) skipSpacesUpToNewline() {
	// Skip whitespace characters
	for t.hasMoreInput() {
		r, ok := t.peek()
		if !ok || r == '\n' || r == '\r' {
			break
		}
		if !unicode.IsSpace(r) {
			break // Stop if a non-whitespace character is found
		}
		t.consume() // Consume the whitespace character
	}
}

func (t *Tokenizer) consumeTripleClosingQuotes(quote rune) error {
	r, b := t.tryReadTripleClosingQuotes()
	if !b {
		return fmt.Errorf("missing triple quotes at line %d, column %d", t.line, t.column)
	}
	if r != quote {
		return fmt.Errorf("expected %c, but found %c at line %d, column %d", quote, r, t.line, t.column)
	}
	return nil
}
