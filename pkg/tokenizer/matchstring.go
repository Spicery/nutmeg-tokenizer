package tokenizer

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// matchString attempts to match a string literal.
func (t *Tokenizer) matchString() (*Token, error) {
	if t.position >= len(t.input) {
		return nil, nil
	}

	r, ok := t.peek()
	if !ok || !isOpeningQuoteChar(r) {
		if r == '@' {
			return t.matchRawString()
		}
		return nil, nil
	}

	_, ok = t.tryPeekTripleOpeningQuotes()
	if ok {
		return t.readMultilineString(false)
	}
	return t.readString(false, r)
}

func (t *Tokenizer) matchRawString() (*Token, error) {
	t.consume() // Consume the '@'
	tagText := ""
	r, ok := t.peek()
	if ok && (unicode.IsLetter(r) || r == '_') {
		tagText = t.takeTagText()
	}
	r, ok = t.peek()
	if ok && isOpeningQuoteChar(r) {
		_, is_triple := t.tryPeekTripleOpeningQuotes()
		var token *Token
		var terr error
		if is_triple {
			token, terr = t.readMultilineString(true)
		} else {
			token, terr = t.readRawString(false, r)
		}
		if terr != nil {
			return token, terr
		}
		if token.Specifier != nil && tagText != "" && *token.Specifier != tagText {
			return nil, fmt.Errorf("tag specifier '%s' does not match existing specifier '%s' at line %d, column %d", tagText, *token.Specifier, t.line, t.column)
		}
		if tagText != "" {
			token.Specifier = &tagText
		}
		return token, nil
	} else {
		return nil, fmt.Errorf("expected string after @ at line %d, column %d", t.line, t.column)
	}
}

// TODO: I think this is a repeat of readSpecifier
func (t *Tokenizer) takeTagText() string {
	var text strings.Builder

	for t.hasMoreInput() {
		r, _ := t.peek()
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') {
			break // Stop if the character is not part of an identifier
		}
		t.consume() // Consume the character
		text.WriteRune(r)
	}

	return text.String()
}

func (t *Tokenizer) readString(unquoted bool, default_quote rune) (*Token, error) {
	start_position := t.position
	startLine, startCol := t.line, t.column
	currPosition := t.position
	currSpan := Span{Position{startLine, startCol}, Position{-1, -1}}
	quote := default_quote
	if !unquoted {
		quote = getMatchingCloseQuote(t.consume()) // Consume the opening quote
	}
	var value strings.Builder
	var interpolationTokens []*Token

	for {
		if !t.hasMoreInput() {
			return nil, fmt.Errorf("unterminated string at line %d, column %d", startLine, startCol)
		}
		beforeBackSlash := Position{t.line, t.column}
		r := t.consume()
		if !unquoted && r == quote { // Closing quote found
			break
		}
		if r == '\\' && t.hasMoreInput() { // Handle escape or interpolation
			next, _ := t.peek()
			if next == '(' || next == '[' || next == '{' {
				// End the current StringToken and handle interpolation
				if value.Len() > 0 {
					textString := t.input[currPosition:t.position]
					currSpan.End = beforeBackSlash
					valueString := value.String()
					current := NewStringToken(textString, valueString, currSpan)
					current.SetQuote(quote)
					interpolationTokens = append(interpolationTokens, current)
					value.Reset()
				}
				interpolatedToken, err := t.readStringInterpolation()
				if err != nil {
					return nil, err
				}
				interpolationTokens = append(interpolationTokens, interpolatedToken)
				currPosition = t.position
				currSpan = Span{Position{t.line, t.column}, Position{-1, -1}}
			} else {
				value.WriteString(handleEscapeSequence(t))
			}
		} else if r == '\n' || r == '\r' { // Handle newlines
			if unquoted {
				if r == '\r' {
					t.tryConsumeRune('\n') // Consume '\n' if it follows
				}
				break
			}
			return nil, fmt.Errorf("line break in string, at line %d, column %d", startLine, startCol)
		} else {
			value.WriteRune(r)
		}
	}

	// Add the final StringToken if there's remaining text
	if value.Len() > 0 {
		textString := t.input[currPosition:t.position]
		currSpan.End.Line, currSpan.End.Col = t.line, t.column
		token := NewStringToken(textString, value.String(), currSpan)
		token.SetQuote(quote)
		interpolationTokens = append(interpolationTokens, token)
	}

	// Reconstruct the original text.
	end_position := t.position
	text := t.input[start_position:end_position]

	// Is this just a literal string?
	if len(interpolationTokens) == 1 && interpolationTokens[0].Type == StringLiteralTokenType {
		interpolationTokens[0].Text = text
		return interpolationTokens[0], nil
	}

	// Combine into a StringInterpolationToken if interpolation occurred
	compoundToken := NewInterpolatedStringToken(text, interpolationTokens, Span{Position{startLine, startCol}, Position{t.line, t.column}})
	compoundToken.SetQuote(quote)
	compoundToken.Type = InterpolatedStringTokenType
	return compoundToken, nil
}

// Helper to check if brackets match
func matches(open, close rune) bool {
	return (open == '(' && close == ')') || (open == '[' && close == ']') || (open == '{' && close == '}')
}

func (t *Tokenizer) readStringInterpolation() (*Token, error) {
	span := Span{Position{t.line, t.column}, Position{-1, -1}}
	state := 0       // State 0: inside expression, State 1: inside string
	var stack []rune // Pushdown stack

	t.markPosition()                   // Mark the position for the interpolation
	openingRune := t.consume()         // Consume the opening bracket
	stack = append(stack, openingRune) // Push opening bracket onto stack

	for {
		if !t.hasMoreInput() {
			return nil, fmt.Errorf("unterminated interpolation, at line %d, Column: %d", span.Start.Line, span.Start.Col)
		}
		r := t.consume()
		switch state {
		case 0: // Inside expression
			switch r {
			case '\\': // Escape sequence
				handleEscapeSequence(t)
			case '(', '[', '{': // Opening brackets
				stack = append(stack, r)
			case ')', ']', '}': // Closing brackets
				if len(stack) > 0 && matches(stack[len(stack)-1], r) {
					stack = stack[:len(stack)-1] // Pop stack
					if len(stack) == 0 {         // End of interpolation
						text := t.popMark() // Pop the marked position
						span.End.Line, span.End.Col = t.line, t.column
						token := NewExpressionToken(text, span)
						return token, nil
					}
				} else {
					return nil, fmt.Errorf("mismatched bracket, at line %d, Column: %d", span.Start.Line, span.Start.Col)
				}
			case '"', '\'', '`', '«': // Enter string state
				stack = append(stack, getMatchingCloseQuote(r))
				state = 1
			case 'r', '\n': // Line breaks are not allowed
				return nil, fmt.Errorf("line break in interpolation, at line %d, Column: %d", t.line, t.column)
			}
		case 1: // Inside string
			switch r {
			case '\\': // Escape sequence
				if t.hasMoreInput() {
					next, _ := t.peek()
					if next == '(' || next == '[' || next == '{' {
						_, err := t.readStringInterpolation()
						if err != nil {
							return nil, err
						}
					} else {
						handleEscapeSequence(t)
					}
				} else {
					return nil, fmt.Errorf("unterminated escape sequence, at line %d, Column: %d", span.Start.Line, span.Start.Col)
				}
			case stack[len(stack)-1]: // Matching closing quote
				stack = stack[:len(stack)-1] // Pop stack
				state = 0
			}
		}
	}
}

// Helper method to process escape sequences
func handleEscapeSequence(t *Tokenizer) string {
	var value strings.Builder
	r := t.consume() // Consume the escape character
	switch r {
	case 'b':
		value.WriteRune('\b')
	case 'f':
		value.WriteRune('\f')
	case 'n':
		value.WriteRune('\n')
	case 'r':
		value.WriteRune('\r')
	case 't':
		value.WriteRune('\t')
	case '\\', '/', '"', '\'', '`', '»': // Escaped backslash, slash, or matching quote
		value.WriteRune(r)
	case 'u': // Unicode escape sequence
		value.WriteString(t.readUnicodeEscape())
	case '_': // Non-standard escape sequence: \_
		// Expand into no characters (do nothing)
		// This has a couple of use-cases. 1. It helps break up a dense sequence
		// of characters, making it easier to read. 2. It can be used to introduce
		// a non-standard identifier.
	default:
		value.WriteRune('\\') // Keep invalid escape sequences as-is
		value.WriteRune(r)
	}

	return value.String()
}

func (t *Tokenizer) readUnicodeEscape() string {
	var code strings.Builder
	for range 4 {
		if t.hasMoreInput() {
			r, size := utf8.DecodeRuneInString(t.input[t.position:])
			if r == utf8.RuneError {
				break // Handle invalid UTF-8
			}
			code.WriteRune(r)
			t.position += size // Advance by the size of the rune
		} else {
			break // Stop if there are fewer than 4 runes remaining
		}
	}
	code_string := code.String()
	if code.Len() == 4 {
		if decoded, err := decodeUnicodeEscape(code_string); err == nil {
			return string(decoded)
		}
	}
	return "\\u" + code.String()
}

// Decode a Unicode escape sequence (\uXXXX) into a rune
func decodeUnicodeEscape(code string) (rune, error) {
	if r, err := strconv.ParseInt(code, 16, 32); err == nil {
		return rune(r), nil
	} else {
		return 0, err
	}
}

func (t *Tokenizer) readMultilineString(rawFlag bool) (*Token, error) {
	startPosition := t.position
	startLine, startCol := t.line, t.column
	var subTokens []*Token

	openingQuote, closingIndent, specifier, nlines, terr := t.findClosingIndent()
	if terr != nil {
		return nil, terr
	}
	closingQuote := getMatchingCloseQuote(openingQuote) // Get the matching closing quote

	// Discard the rest of this line, which are the opening quotes.
	t.readRestOfLine()

	// The next N lines should be either all whitespace or start with the
	// closing indent.
	for range nlines {
		var tok *Token
		var err error
		if t.tryConsumeText(closingIndent) {
			if rawFlag {
				tok, err = t.readRawString(true, openingQuote)
				if err != nil {
					return nil, err
				}
			} else {
				tok, err = t.readString(true, openingQuote)
				if err != nil {
					return nil, err
				}
			}
		} else {
			tok = NewStringToken("", "", Span{Position{t.line, t.column}, Position{t.line, t.column}})
			tok.SetQuote(openingQuote)
		}
		subTokens = append(subTokens, tok)
	}

	// Discard the rest of the next line, which will be the closing quotes.
	t.skipSpacesUpToNewline()

	terr = t.consumeTripleClosingQuotes(closingQuote)
	if terr != nil {
		return nil, terr // Return error if closing quotes are malformed
	}

	originalText := t.input[startPosition:t.position]

	// Add the multiline string token
	token := NewMultiLineStringToken(originalText, "", Span{Position{startLine, startCol}, Position{t.line, t.column}})
	token.Specifier = &specifier
	token.SetQuote(openingQuote)
	token.Subtokens = subTokens

	return token, nil
}

func (t *Tokenizer) findClosingIndent() (rune, string, string, int, error) {
	t.markPosition()

	// Validate and consume the opening triple quotes
	opening_quote, ok := t.tryReadTripleOpeningQuotes()
	if !ok {
		return 0, "", "", 0, fmt.Errorf("malformed opening triple quotes at line %d, column %d", t.line, t.column)
	}
	closing_quote := getMatchingCloseQuote(opening_quote) // Get the matching closing quote

	// Ensure no other non-space characters appear on the opening line
	specifier, terr := t.readSpecifier()
	if terr != nil {
		return 0, "", "", 0, terr
	}

	// Now read each line in order until we find the closing line.
	startLine, startCol := t.line, t.column
	lines := []string{}
	var match bool
	var closingIndent string
	for t.hasMoreInput() {
		line := t.readRestOfLine()
		match, closingIndent = textIsWhitespaceFollowedBy3Quotes(line, closing_quote)
		if match {
			break
		}
		lines = append(lines, line)
	}

	if !match {
		return 0, "", "", 0, fmt.Errorf("closing triple quote not found at line %d, column %d", t.line, t.column)
	}

	for i, line := range lines {
		// Allow empty lines
		if line == "" {
			continue
		}
		// Check if the line starts with the closing indent
		if !strings.HasPrefix(line, closingIndent) {
			return 0, "", "", 0, fmt.Errorf("not indented consistently with the closing triple quote at line %d, column %d", startLine+i, startCol)
		}
	}

	t.resetPosition()
	return closing_quote, closingIndent, specifier, len(lines), nil
}

func getMatchingCloseQuote(openingQuote rune) rune {
	// Return the matching closing quote for the given opening quote
	if openingQuote == '«' {
		return '»'
	}
	return openingQuote // For other quotes, return the same character
}

// Method to read the specifier of a multi-line string / code-fence.
func (t *Tokenizer) readSpecifier() (string, error) {
	// Read all the characters until a newline or end of input.
	var text strings.Builder
	for t.hasMoreInput() {
		r := t.consume()
		if r == '\n' || r == '\r' {
			if r == '\r' {
				t.tryConsumeRune('\n') // Consume \n if it follows
			}
			break // End of line
		}
		text.WriteRune(r)
	}
	strtext := strings.TrimSpace(text.String())
	if strings.Contains(strtext, " ") {
		return "", fmt.Errorf("spaces inside code-fence specifier at line %d, column %d", t.line, t.column)
	}
	//  Check the specifier matches the regex ^\w*$. This reserves wriggle room
	//  for future expansion.
	if len(strtext) > 0 {
		m, e := regexp.MatchString(`^[a-zA-Z_]\w*$`, strtext)
		if !m || e != nil {
			return "", fmt.Errorf("invalid code-fence specifier at line %d, column %d", t.line, t.column)
		}
	}
	return strtext, nil
}

func (t *Tokenizer) readRawString(unquoted bool, default_quote rune) (*Token, error) {
	startPosition := t.position
	startLine, startCol := t.line, t.column
	quote := default_quote
	if !unquoted {
		quote = getMatchingCloseQuote(t.consume()) // Consume the opening quote
	}
	var text strings.Builder

	for {
		if !t.hasMoreInput() {
			return nil, fmt.Errorf("unterminated raw string at line %d, column %d", startLine, startCol)
		}
		r := t.consume()
		if r == quote { // Closing quote found
			break
		} else if r == '\n' || r == '\r' { // Handle newlines
			if unquoted {
				if r == '\r' {
					t.tryConsumeRune('\n') // Consume '\n' if it follows
				}
				break
			}
			return nil, fmt.Errorf("line break in raw string at line %d, column %d", startLine, startCol)
		}
		// Backslashes are treated as normal characters in raw strings
		text.WriteRune(r)
	}

	// Add the raw string token
	originalText := t.input[startPosition:t.position]
	token := NewStringToken(originalText, text.String(), Span{Position{startLine, startCol}, Position{t.line, t.column}})
	token.SetQuote(quote)
	return token, nil
}
