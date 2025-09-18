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
		input            string
		expectedRadix    int
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
		expecting    []string
	}{
		{"def", StartToken, []string{"enddef", "end"}},
		{"if", StartToken, []string{"endif", "end"}},
		{"class", StartToken, []string{"endclass", "end"}},
		{"fn", StartToken, []string{"endfn", "end"}},
		{"try", StartToken, []string{"endtry", "end"}},
		{"transaction", StartToken, []string{"endtransaction", "end"}},
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

			if len(token.Expecting) != len(tt.expecting) {
				t.Errorf("Expected %d expecting tokens, got %d", len(tt.expecting), len(token.Expecting))
				return
			}

			for i, expected := range tt.expecting {
				if token.Expecting[i] != expected {
					t.Errorf("Expected expecting token '%s' at index %d, got '%s'", expected, i, token.Expecting[i])
				}
			}
		})
	}
}

func TestOperatorTokens(t *testing.T) {
	tests := []struct {
		input              string
		expectedPrecedence [3]int // [prefix, infix, postfix]
	}{
		{"+", [3]int{0, 5, 0}},
		{"-", [3]int{8, 5, 0}},
		{"*", [3]int{0, 6, 0}},
		{"==", [3]int{0, 2, 0}},
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

			if token.Precedence == nil {
				t.Errorf("Expected precedence to be set")
				return
			}

			if *token.Precedence != tt.expectedPrecedence {
				t.Errorf("Expected precedence %v, got %v", tt.expectedPrecedence, *token.Precedence)
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
		{"{", OpenDelimiter, "}", true, true}, // Updated: now supports infix usage for f{x} syntax
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
				if token.ClosedBy == nil || *token.ClosedBy != tt.closedBy {
					t.Errorf("Expected closed by '%s', got %v", tt.closedBy, token.ClosedBy)
				}

				if token.Infix == nil || *token.Infix != tt.isInfix {
					t.Errorf("Expected infix %t, got %v", tt.isInfix, token.Infix)
				}

				if token.Prefix == nil || *token.Prefix != tt.isPrefix {
					t.Errorf("Expected prefix %t, got %v", tt.isPrefix, token.Prefix)
				}
			}
		})
	}
}

func TestKeywordClassification(t *testing.T) {
	tests := []struct {
		input        string
		expectedType TokenType
	}{
		// Label tokens (L)
		{"then", LabelToken},
		{"else", LabelToken},

		// Compound tokens (C)
		{"catch", CompoundToken},
		{"elseif", CompoundToken},
		{"elseifnot", CompoundToken},

		// Prefix tokens (P)
		{"return", PrefixToken},
		{"yield", PrefixToken},

		// End tokens (E)
		{"end", EndToken},
		{"enddef", EndToken},
		{"endclass", EndToken},

		// Variable tokens (V) - should default to this for unknown identifiers
		{"myVariable", VariableToken},
		{"unknown", VariableToken},
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
