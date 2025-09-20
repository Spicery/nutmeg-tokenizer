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
		// Existing decimal tests
		{"42", 10, "42", "", ""},
		{"3.14", 10, "3", "14", ""},
		{"1.5e10", 10, "1", "5", "10"},
		{"2e-3", 10, "2", "", "-3"},

		// Traditional prefix tests
		{"0x2A", 16, "2A", "", ""},
		{"0xFF", 16, "FF", "", ""},
		{"0x1A", 16, "1A", "", ""},
		{"0b101010", 2, "101010", "", ""},
		{"0b1010", 2, "1010", "", ""},
		{"0b11", 2, "11", "", ""},
		{"0o52", 8, "52", "", ""},
		{"0o127", 8, "127", "", ""},
		{"0o777", 8, "777", "", ""},

		// New rR notation tests - basic
		{"2r1010", 2, "1010", "", ""},
		{"8r77", 8, "77", "", ""},
		{"16rFF", 16, "FF", "", ""},
		{"36rHELLO", 36, "HELLO", "", ""},
		{"10rAB", 10, "AB", "", ""},
		{"16rDEADBEEF", 16, "DEADBEEF", "", ""},
		{"36rZEBRA", 36, "ZEBRA", "", ""},

		// rR notation with floating point
		{"2r1010.11", 2, "1010", "11", ""},
		{"16rFF.A", 16, "FF", "A", ""},
		{"36rHELLO.WORLD", 36, "HELLO", "WORLD", ""},
		{"8r123.456", 8, "123", "456", ""},
		{"16r1A.BC", 16, "1A", "BC", ""},

		// rR notation with scientific notation
		{"10r123e5", 10, "123", "", "5"},
		{"16rABe-2", 16, "AB", "", "-2"},
		{"2r101e+3", 2, "101", "", "+3"},
		{"8r777e10", 8, "777", "", "10"},

		// rR notation with both fraction and exponent
		{"10r12.34e5", 10, "12", "34", "5"},
		{"16rAB.CDe-2", 16, "AB", "CD", "-2"},
		{"2r10.11e+3", 2, "10", "11", "+3"},

		// Edge cases for radix values
		{"2r101", 2, "101", "", ""}, // minimum radix
		{"36rZ", 36, "Z", "", ""},   // maximum radix
		{"9rAB", 9, "AB", "", ""},   // single digit radix
		{"35rYZ", 35, "YZ", "", ""}, // near maximum radix
		{"12rAB", 12, "AB", "", ""}, // double digit radix

		// Complex cases
		{"16rDEAD.BEEFe10", 16, "DEAD", "BEEF", "10"},
		{"36rHELLO.WORLDe-5", 36, "HELLO", "WORLD", "-5"},
		{"8r123.456e+7", 8, "123", "456", "+7"},
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

func TestEnhancedNumericEdgeCases(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedTokens   int
		expectedRadix    int
		expectedMantissa string
		expectedFraction string
		expectedExponent string
	}{
		// Test proper case sensitivity - only uppercase digits allowed
		{"Lowercase e exponent", "10r123e5", 1, 10, "123", "", "5"},

		// Test boundary radix values
		{"Minimum radix", "2r1", 1, 2, "1", "", ""},
		{"Maximum radix", "36rZ", 1, 36, "Z", "", ""},
		{"Near maximum radix", "35rY", 1, 35, "Y", "", ""},

		// Test multi-digit radix values
		{"Double digit radix 10", "10r9", 1, 10, "9", "", ""},
		{"Double digit radix 16", "16rF", 1, 16, "F", "", ""},
		{"Double digit radix 36", "36rZ", 1, 36, "Z", "", ""},

		// Test comprehensive digit ranges for different bases
		{"Base 2 max digits", "2r1", 1, 2, "1", "", ""},
		{"Base 8 max digits", "8r7", 1, 8, "7", "", ""},
		{"Base 10 max digits", "10r9", 1, 10, "9", "", ""},
		{"Base 16 max digits", "16rF", 1, 16, "F", "", ""},
		{"Base 36 max digits", "36rZ", 1, 36, "Z", "", ""},

		// Test floating point in various bases
		{"Binary floating point", "2r1.1", 1, 2, "1", "1", ""},
		{"Octal floating point", "8r7.6", 1, 8, "7", "6", ""},
		{"Hex floating point", "16rF.E", 1, 16, "F", "E", ""},
		{"Base 36 floating point", "36rZ.Y", 1, 36, "Z", "Y", ""},

		// Test scientific notation in various bases
		{"Binary scientific", "2r1e2", 1, 2, "1", "", "2"},
		{"Hex scientific", "16rFe10", 1, 16, "F", "", "10"},
		{"Base 36 scientific", "36rZe5", 1, 36, "Z", "", "5"},

		// Test combined floating point and scientific notation
		{"Hex float scientific", "16rA.Be-2", 1, 16, "A", "B", "-2"},
		{"Base 36 float scientific", "36rH.ELLOe+10", 1, 36, "H", "ELLO", "+10"},

		// Test complex realistic examples
		{"Hex color value", "16rFFFFFF", 1, 16, "FFFFFF", "", ""},
		{"Base 36 word", "36rHELLOWORLD", 1, 36, "HELLOWORLD", "", ""},
		{"Large hex number", "16rDEADBEEF", 1, 16, "DEADBEEF", "", ""},

		// Test edge cases with scientific notation avoiding lowercase conflicts
		{"Hex with exponent", "16rABCe5", 1, 16, "ABC", "", "5"},
		{"Base 36 avoiding E digit", "36rHELLe10", 1, 36, "HELL", "", "10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokeniser := New(tt.input)
			tokens, err := tokeniser.Tokenise()

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(tokens) != tt.expectedTokens {
				t.Errorf("Expected %d tokens, got %d", tt.expectedTokens, len(tokens))
				return
			}

			if tt.expectedTokens == 0 {
				return // No token to verify
			}

			token := tokens[0]
			if token.Type != NumericLiteral {
				t.Errorf("Expected numeric token, got %s", token.Type)
				return
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

	// Build the precomputed lookup map
	if err := rules.BuildTokenLookup(); err != nil {
		t.Fatalf("Failed to build token lookup: %v", err)
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
