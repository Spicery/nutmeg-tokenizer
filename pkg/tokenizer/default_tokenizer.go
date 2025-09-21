package tokenizer

import (
	"strings"
)

// nextTokenDefault tries to match the next token using the default token matching rules.
// It is used when no custom rules are defined, or when no custom rule matches the input.
// It returns a non-nil error if tokenization fails.

// func (t *Tokenizer) nextTokenDefault(start Position, sawNewlineBefore bool) error {

// 	if token := t.matchIdentifier(); token != nil {
// 		token.Span.Start = start
// 		if sawNewlineBefore {
// 			token.LnBefore = &sawNewlineBefore
// 		}
// 		return t.addTokenAndManageStack(token)
// 	}

// 	if token := t.matchOperator(); token != nil {
// 		token.Span.Start = start
// 		if sawNewlineBefore {
// 			token.LnBefore = &sawNewlineBefore
// 		}
// 		return t.addTokenAndManageStack(token)
// 	}

// 	if token := t.matchDelimiter(); token != nil {
// 		token.Span.Start = start
// 		if sawNewlineBefore {
// 			token.LnBefore = &sawNewlineBefore
// 		}
// 		return t.addTokenAndManageStack(token)
// 	}

// 	// If nothing matches, create an unclassified token
// 	r, size := utf8.DecodeRuneInString(t.input[t.position:])
// 	text := string(r)
// 	end := Position{Line: t.line, Col: t.column + size}
// 	span := Span{Start: start, End: end}

// 	token := NewToken(text, UnclassifiedToken, span)
// 	if sawNewlineBefore {
// 		token.LnBefore = &sawNewlineBefore
// 	}
// 	t.advance(size)
// 	return t.addTokenAndManageStack(token)
// }

// matchIdentifier attempts to match an identifier.
func (t *Tokenizer) matchIdentifier() *Token {
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

	// Check if it's a bridge token (B) - only if no custom bridge rules are defined
	if t.rules == nil || len(t.rules.BridgeTokens) == 0 {
		if labelData, isLabel := bridgeTokens[match]; isLabel {
			t.advance(len(match))
			return NewStmntBridgeToken(match, labelData.Expecting, labelData.In, span)
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

// // matchSpecialLabels attempts to match special label sequences like '=>>' and wildcard ':'
// func (t *Tokenizer) matchSpecialLabels() *Token {
// 	// Check for '=>>' special label
// 	if strings.HasPrefix(t.input[t.position:], "=>>") {
// 		end := Position{Line: t.line, Col: t.column + 3}
// 		span := Span{End: end}

// 		labelData := bridgeTokens["=>>"]
// 		t.advance(3)
// 		return NewStmntBridgeToken("=>>", labelData.Expecting, labelData.In, span)
// 	}

// 	// Check for wildcard tokens
// 	wildcardTokens := t.getWildcardTokens()
// 	for wildcardText := range wildcardTokens {
// 		if t.position < len(t.input) && strings.HasPrefix(t.input[t.position:], wildcardText) {
// 			// For single character wildcards, make sure it's not part of a longer operator
// 			if len(wildcardText) == 1 && wildcardText == ":" {
// 				if t.position+1 < len(t.input) && strings.ContainsRune("*/%+-<>~!&^|?=:", rune(t.input[t.position+1])) {
// 					continue // Skip this wildcard as it's part of a longer operator
// 				}
// 			}

// 			end := Position{Line: t.line, Col: t.column + len(wildcardText)}
// 			span := Span{End: end}

// 			// Check if we have context from the expecting stack
// 			expected := t.getCurrentlyExpected()
// 			if len(expected) > 0 {
// 				// Use the first expected token as the basis for the wildcard
// 				expectedText := expected[0]

// 				// Check if it's a label token
// 				labelTokens := t.getBridgeTokens()
// 				if labelData, exists := labelTokens[expectedText]; exists {
// 					// Create a wildcard token that copies attributes from the expected label
// 					t.advance(len(wildcardText))
// 					return NewWildcardBridgeTokenWithAttributes(wildcardText, expectedText, labelData.Expecting, labelData.In, span)
// 				}

// 				// Check if it's a start token
// 				startTokens := t.getStartTokens()
// 				if startData, exists := startTokens[expectedText]; exists {
// 					// Create wildcard start token
// 					t.advance(len(wildcardText))
// 					return NewWildcardStartToken(wildcardText, expectedText, startData.ClosedBy, span)
// 				}

// 				// Check if it's an end token (starts with "end")
// 				if strings.HasPrefix(expectedText, "end") {
// 					// Create wildcard end token
// 					t.advance(len(wildcardText))
// 					return NewWildcardEndToken(wildcardText, expectedText, span)
// 				}
// 			}

// 			// No context available, create unclassified token
// 			t.advance(len(wildcardText))
// 			return NewToken(wildcardText, UnclassifiedToken, span)
// 		}
// 	}

// 	return nil
// }

// matchOperator attempts to match an operator.
func (t *Tokenizer) matchOperator() *Token {
	// fmt.Println("matchOperator called at position", t.position, "input:", t.input[t.position:])

	match := operatorRegex.FindString(t.input[t.position:])
	if match == "" {
		return nil
	}

	// fmt.Println("Operator matched:", match)

	end := Position{Line: t.line, Col: t.column + len(match)}
	span := Span{End: end}

	if match == "=>>" {
		labelData := bridgeTokens["=>>"]
		t.advance(len(match))
		return NewStmntBridgeToken(match, labelData.Expecting, labelData.In, span)
	}

	if match == ":" {
		t.advance(len(match))
		expected := t.getCurrentlyExpected()
		if len(expected) > 0 {
			expectedText := expected[0] // Use the leading token only.
			labelTokens := t.getBridgeTokens()
			if labelData, exists := labelTokens[expectedText]; exists {
				return NewWildcardBridgeToken(match, expectedText, labelData.Expecting, labelData.In, span)
			}
		}
		return NewUnclassifiedToken(match, span)
	}

	// Calculate precedence using the new rules
	prefix, infix, postfix := calculateOperatorPrecedence(match)

	t.advance(len(match))
	return NewOperatorToken(match, prefix, infix, postfix, span)
}

// matchDelimiter attempts to match a delimiter using default rules only.
// This is only called when no custom bracket rules are defined.
// func (t *Tokenizer) matchDelimiter() *Token {
// 	if t.position >= len(t.input) {
// 		return nil
// 	}

// 	char := string(t.input[t.position])

// 	// Check for open delimiters using default mappings
// 	if closedBy, isOpen := delimiterMappings[char]; isOpen {
// 		end := Position{Line: t.line, Col: t.column + 1}
// 		span := Span{End: end}

// 		props := delimiterProperties[char]
// 		isInfix, isPrefix := props[0], props[1]

// 		t.advance(1)
// 		return NewDelimiterToken(char, closedBy, isInfix, isPrefix, span)
// 	}

// 	// Check for close delimiters using default regex
// 	if closeDelimRegex.MatchString(char) {
// 		end := Position{Line: t.line, Col: t.column + 1}
// 		span := Span{End: end}

// 		t.advance(1)
// 		return NewToken(char, CloseDelimiter, span)
// 	}

// 	return nil
// }
