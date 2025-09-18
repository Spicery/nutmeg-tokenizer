package tokeniser

import (
	"encoding/json"
	"testing"
)

func TestBasicTokenisation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int // expected number of tokens
	}{
		{"Empty input", "", 0},
		{"Single identifier", "hello", 1},
		{"Multiple identifiers", "hello world", 2},
		{"Number", "42", 1},
		{"String", `"hello"`, 1},
		{"Operator", "+", 1},
		{"Delimiter", "(", 1},
		{"Complex expression", "def foo(x) x + 1 end", 9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokeniser := New(tt.input)
			tokens, err := tokeniser.Tokenise()

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(tokens) != tt.expected {
				t.Errorf("Expected %d tokens, got %d", tt.expected, len(tokens))
			}
		})
	}
}

func TestStringTokens(t *testing.T) {
	tests := []struct {
		input         string
		expectedText  string
		expectedValue string
	}{
		{`"hello"`, `"hello"`, "hello"},
		{`'world'`, `'world'`, "world"},
		{"`backtick`", "`backtick`", "backtick"},
		{`"escaped\n"`, `"escaped\n"`, "escaped\n"},
		{`"quote\"test"`, `"quote\"test"`, `quote"test`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokeniser := New(tt.input)
			tokens, err := tokeniser.Tokenise()

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(tokens) != 1 {
				t.Errorf("Expected 1 token, got %d", len(tokens))
				return
			}

			token := tokens[0]
			if token.Type != StringLiteral {
				t.Errorf("Expected string token, got %s", token.Type)
			}

			if token.Text != tt.expectedText {
				t.Errorf("Expected text '%s', got '%s'", tt.expectedText, token.Text)
			}

			if token.Value == nil {
				t.Errorf("Expected value to be set")
				return
			}

			if *token.Value != tt.expectedValue {
				t.Errorf("Expected value '%s', got '%s'", tt.expectedValue, *token.Value)
			}
		})
	}
}

func TestNumericTokens(t *testing.T) {
	tests := []struct {
		input           string
		expectedRadix   int
		expectedMantissa string
		expectedFraction string
		expectedExponent string
	}{
		{"42", 10, "42", "", ""},
		{"0x2A", 16, "2A", "", ""},
		{"0b101010", 2, "101010", "", ""},
		{"0o52", 8, "52", "", ""},
		{"3.14", 10, "3", "14", ""},
		{"1.5e10", 10, "1", "5", "10"},
		{"2e-3", 10, "2", "", "-3"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokeniser := New(tt.input)
			tokens, err := tokeniser.Tokenise()

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(tokens) != 1 {
				t.Errorf("Expected 1 token, got %d", len(tokens))
				return
			}

			token := tokens[0]
			if token.Type != NumericLiteral {
				t.Errorf("Expected numeric token, got %s", token.Type)
			}

			if token.Radix == nil || *token.Radix != tt.expectedRadix {
				t.Errorf("Expected radix %d, got %v", tt.expectedRadix, token.Radix)
			}

			if token.Mantissa == nil || *token.Mantissa != tt.expectedMantissa {
				t.Errorf("Expected mantissa '%s', got %v", tt.expectedMantissa, token.Mantissa)
			}

			if tt.expectedFraction == "" {
				if token.Fraction != nil {
					t.Errorf("Expected no fraction, got '%s'", *token.Fraction)
				}
			} else {
				if token.Fraction == nil || *token.Fraction != tt.expectedFraction {
					t.Errorf("Expected fraction '%s', got %v", tt.expectedFraction, token.Fraction)
				}
			}

			if tt.expectedExponent == "" {
				if token.Exponent != nil {
					t.Errorf("Expected no exponent, got '%s'", *token.Exponent)
				}
			} else {
				if token.Exponent == nil || *token.Exponent != tt.expectedExponent {
					t.Errorf("Expected exponent '%s', got %v", tt.expectedExponent, token.Exponent)
				}
			}
		})
	}
}

func TestStartTokens(t *testing.T) {
	tests := []struct {
		input        string
		expectedType TokenType
		closedBy     []string
	}{
		{"def", StartToken, []string{"end"}},
		{"if", StartToken, []string{"endif", "end"}},
		{"while", StartToken, []string{"endwhile", "end"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokeniser := New(tt.input)
			tokens, err := tokeniser.Tokenise()

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(tokens) != 1 {
				t.Errorf("Expected 1 token, got %d", len(tokens))
				return
			}

			token := tokens[0]
			if token.Type != tt.expectedType {
				t.Errorf("Expected token type %s, got %s", tt.expectedType, token.Type)
			}

			if len(token.ClosedBy) != len(tt.closedBy) {
				t.Errorf("Expected %d closing tokens, got %d", len(tt.closedBy), len(token.ClosedBy))
				return
			}

			for i, expected := range tt.closedBy {
				if token.ClosedBy[i] != expected {
					t.Errorf("Expected closing token '%s' at index %d, got '%s'", expected, i, token.ClosedBy[i])
				}
			}
		})
	}
}

func TestOperatorTokens(t *testing.T) {
	tests := []struct {
		input           string
		expectedPrefix  int
		expectedInfix   int
		expectedPostfix int
	}{
		{"+", 0, 5, 0},
		{"-", 8, 5, 0},
		{"*", 0, 6, 0},
		{"==", 0, 2, 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokeniser := New(tt.input)
			tokens, err := tokeniser.Tokenise()

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(tokens) != 1 {
				t.Errorf("Expected 1 token, got %d", len(tokens))
				return
			}

			token := tokens[0]
			if token.Type != OperatorToken {
				t.Errorf("Expected operator token, got %s", token.Type)
			}

			if tt.expectedPrefix == 0 {
				if token.Prefix != nil {
					t.Errorf("Expected no prefix precedence, got %d", *token.Prefix)
				}
			} else {
				if token.Prefix == nil || *token.Prefix != tt.expectedPrefix {
					t.Errorf("Expected prefix precedence %d, got %v", tt.expectedPrefix, token.Prefix)
				}
			}

			if tt.expectedInfix == 0 {
				if token.Infix != nil {
					t.Errorf("Expected no infix precedence, got %d", *token.Infix)
				}
			} else {
				if token.Infix == nil || *token.Infix != tt.expectedInfix {
					t.Errorf("Expected infix precedence %d, got %v", tt.expectedInfix, token.Infix)
				}
			}

			if tt.expectedPostfix == 0 {
				if token.Postfix != nil {
					t.Errorf("Expected no postfix precedence, got %d", *token.Postfix)
				}
			} else {
				if token.Postfix == nil || *token.Postfix != tt.expectedPostfix {
					t.Errorf("Expected postfix precedence %d, got %v", tt.expectedPostfix, token.Postfix)
				}
			}
		})
	}
}

func TestDelimiterTokens(t *testing.T) {
	tests := []struct {
		input        string
		expectedType TokenType
		closedBy     string
		isInfix      bool
		isPrefix     bool
	}{
		{"(", OpenDelimiter, ")", true, true},
		{"[", OpenDelimiter, "]", true, false},
		{"{", OpenDelimiter, "}", false, true},
		{")", CloseDelimiter, "", false, false},
		{"]", CloseDelimiter, "", false, false},
		{"}", CloseDelimiter, "", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokeniser := New(tt.input)
			tokens, err := tokeniser.Tokenise()

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(tokens) != 1 {
				t.Errorf("Expected 1 token, got %d", len(tokens))
				return
			}

			token := tokens[0]
			if token.Type != tt.expectedType {
				t.Errorf("Expected token type %s, got %s", tt.expectedType, token.Type)
			}

			if tt.expectedType == OpenDelimiter {
				if token.DelimiterClosedBy == nil || *token.DelimiterClosedBy != tt.closedBy {
					t.Errorf("Expected closed by '%s', got %v", tt.closedBy, token.DelimiterClosedBy)
				}

				if token.InfixDelimiter == nil || *token.InfixDelimiter != tt.isInfix {
					t.Errorf("Expected infix %t, got %v", tt.isInfix, token.InfixDelimiter)
				}

				if token.PrefixDelimiter == nil || *token.PrefixDelimiter != tt.isPrefix {
					t.Errorf("Expected prefix %t, got %v", tt.isPrefix, token.PrefixDelimiter)
				}
			}
		})
	}
}

func TestJSONSerialization(t *testing.T) {
	input := `def hello(name) "Hello, " + name end`
	tokeniser := New(input)
	tokens, err := tokeniser.Tokenise()

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	// Test that all tokens can be serialized to JSON
	for i, token := range tokens {
		jsonBytes, err := json.Marshal(token)
		if err != nil {
			t.Errorf("Failed to serialize token %d to JSON: %v", i, err)
			continue
		}

		// Test that the JSON can be deserialized back
		var deserializedToken Token
		err = json.Unmarshal(jsonBytes, &deserializedToken)
		if err != nil {
			t.Errorf("Failed to deserialize token %d from JSON: %v", i, err)
			continue
		}

		// Basic checks
		if deserializedToken.Text != token.Text {
			t.Errorf("Token %d text mismatch after JSON round-trip: expected '%s', got '%s'", i, token.Text, deserializedToken.Text)
		}

		if deserializedToken.Type != token.Type {
			t.Errorf("Token %d type mismatch after JSON round-trip: expected '%s', got '%s'", i, token.Type, deserializedToken.Type)
		}
	}
}

func TestCommentsAreIgnored(t *testing.T) {
	input := `hello ### this is a comment
world`
	tokeniser := New(input)
	tokens, err := tokeniser.Tokenise()

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if len(tokens) != 2 {
		t.Errorf("Expected 2 tokens (ignoring comment), got %d", len(tokens))
		return
	}

	if tokens[0].Text != "hello" {
		t.Errorf("Expected first token to be 'hello', got '%s'", tokens[0].Text)
	}

	if tokens[1].Text != "world" {
		t.Errorf("Expected second token to be 'world', got '%s'", tokens[1].Text)
	}
}