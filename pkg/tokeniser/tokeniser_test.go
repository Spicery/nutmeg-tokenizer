package tokeniser

import (
	"encoding/json"
	"os"
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
		{"def", StartToken, []string{"=>>"}},
		{"if", StartToken, []string{"then"}},
		{"class", StartToken, []string{}},
		{"fn", StartToken, []string{}},
		{"for", StartToken, []string{"do"}},
		{"try", StartToken, []string{"catch", "else"}},
		{"transaction", StartToken, []string{"else"}},
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
		{"+", [3]int{0, 2040, 0}},  // + has base precedence 40, only infix enabled (40+2000=2040)
		{"-", [3]int{50, 2050, 0}}, // - has base precedence 50, both prefix (50) and infix (50+2000=2050) enabled
		{"*", [3]int{0, 2010, 0}},  // * has base precedence 10, only infix enabled (10+2000=2010)
		{"==", [3]int{0, 2139, 0}}, // = has base precedence 140, repeated so 139, only infix enabled (139+2000=2139)
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
		closedBy     []string
		isInfix      bool
		isPrefix     bool
	}{
		{"(", OpenDelimiter, []string{")"}, true, true},
		{"[", OpenDelimiter, []string{"]"}, true, false},
		{"{", OpenDelimiter, []string{"}"}, true, true}, // Updated: now supports infix usage for f{x} syntax
		{")", CloseDelimiter, nil, false, false},
		{"]", CloseDelimiter, nil, false, false},
		{"}", CloseDelimiter, nil, false, false},
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
				if len(token.ClosedBy) != len(tt.closedBy) {
					t.Errorf("Expected closed by %v, got %v", tt.closedBy, token.ClosedBy)
				} else {
					for i, expected := range tt.closedBy {
						if token.ClosedBy[i] != expected {
							t.Errorf("Expected closed by '%s' at index %d, got '%s'", expected, i, token.ClosedBy[i])
						}
					}
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
		{"=>>", LabelToken},
		{"do", LabelToken},
		{"then", LabelToken},
		{"else", LabelToken},

		// Unclassified tokens (U)
		{":", UnclassifiedToken}, // bare wildcard without context

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

func TestCustomRulesWildcard(t *testing.T) {
	// Create a custom rules set with a different wildcard
	rules := &TokeniserRules{
		StartTokens:         getDefaultStartTokens(),
		LabelTokens:         getDefaultLabelTokens(),
		CompoundTokens:      getDefaultCompoundTokens(),
		PrefixTokens:        getDefaultPrefixTokens(),
		DelimiterMappings:   getDefaultDelimiterMappings(),
		DelimiterProperties: getDefaultDelimiterProperties(),
		WildcardTokens:      map[string]bool{"*": true}, // Use * instead of :
		OperatorPrecedences: make(map[string][3]int),
	}

	// Test with custom wildcard in a def context
	tokeniser := NewWithRules("def foo *", rules)
	tokens, err := tokeniser.Tokenise()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(tokens) != 3 {
		t.Fatalf("Expected 3 tokens, got %d", len(tokens))
	}

	// Third token should be a wildcard token behaving like "=>>"
	wildcardToken := tokens[2]
	if wildcardToken.Text != "*" {
		t.Errorf("Expected wildcard token text to be '*', got '%s'", wildcardToken.Text)
	}

	if wildcardToken.Type != LabelToken {
		t.Errorf("Expected wildcard token type to be Label, got %s", wildcardToken.Type)
	}

	if wildcardToken.Value == nil || *wildcardToken.Value != "=>>" {
		t.Errorf("Expected wildcard token value to be '=>>', got '%v'", wildcardToken.Value)
	}
}

func TestLoadRulesFile(t *testing.T) {
	// Create a temporary rules file
	rulesContent := `wildcard:
  - text: "#"
prefix:
  - text: "custom_return"`

	tmpFile := "/tmp/test_rules.yaml"
	err := writeFile(tmpFile, rulesContent)
	if err != nil {
		t.Fatalf("Failed to create temp rules file: %v", err)
	}

	// Load the rules file
	rules, err := LoadRulesFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load rules file: %v", err)
	}

	// Check that the rules were loaded correctly
	if len(rules.Wildcard) != 1 || rules.Wildcard[0].Text != "#" {
		t.Errorf("Expected wildcard rule with text '#', got %+v", rules.Wildcard)
	}

	if len(rules.Prefix) != 1 || rules.Prefix[0].Text != "custom_return" {
		t.Errorf("Expected prefix rule with text 'custom_return', got %+v", rules.Prefix)
	}
}

// Helper function for writing test files
func writeFile(filename, content string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}
