package tokenizer

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
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.Tokenize()

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
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.Tokenize()

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(tokens) != 1 {
				t.Errorf("Expected 1 token, got %d", len(tokens))
				return
			}

			token := tokens[0]
			if token.Type != StringLiteralTokenType {
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
	// Helper function to create int pointers
	intPtr := func(i int) *int { return &i }

	tests := []struct {
		input            string
		expectedRadix    string
		expectedBase     int
		expectedMantissa string
		expectedFraction string
		expectedExponent *int
	}{
		// Existing decimal tests
		{"42", "", 10, "42", "", nil},
		{"3.14", "", 10, "3", "14", nil},
		{"1.5e10", "", 10, "1", "5", intPtr(10)},
		{"2e-3", "", 10, "2", "", intPtr(-3)},

		// Traditional prefix tests
		{"0x2A", "0x", 16, "2A", "", nil},
		{"0xFF", "0x", 16, "FF", "", nil},
		{"0x1A", "0x", 16, "1A", "", nil},
		{"0b101010", "0b", 2, "101010", "", nil},
		{"0b1010", "0b", 2, "1010", "", nil},
		{"0b11", "0b", 2, "11", "", nil},
		{"0o52", "0o", 8, "52", "", nil},
		{"0o127", "0o", 8, "127", "", nil},
		{"0o777", "0o", 8, "777", "", nil},

		// New rR notation tests - basic
		{"2r1010", "2r", 2, "1010", "", nil},
		{"8r77", "8r", 8, "77", "", nil},
		{"16rFF", "16r", 16, "FF", "", nil},
		{"36rHELLO", "36r", 36, "HELLO", "", nil},
		{"16rDEADBEEF", "16r", 16, "DEADBEEF", "", nil},
		{"36rZEBRA", "36r", 36, "ZEBRA", "", nil},

		// rR notation with floating point
		{"2r1010.11", "2r", 2, "1010", "11", nil},
		{"16rFF.A", "16r", 16, "FF", "A", nil},
		{"36rHELLO.WORLD", "36r", 36, "HELLO", "WORLD", nil},
		{"8r123.456", "8r", 8, "123", "456", nil},
		{"16r1A.BC", "16r", 16, "1A", "BC", nil},

		// rR notation with scientific notation
		{"10r123e5", "10r", 10, "123", "", intPtr(5)},
		{"16rABe-2", "16r", 16, "AB", "", intPtr(-2)},
		{"2r101e+3", "2r", 2, "101", "", intPtr(3)},
		{"8r777e10", "8r", 8, "777", "", intPtr(10)},

		// rR notation with both fraction and exponent
		{"10r12.34e5", "10r", 10, "12", "34", intPtr(5)},
		{"16rAB.CDe-2", "16r", 16, "AB", "CD", intPtr(-2)},
		{"2r10.11e+3", "2r", 2, "10", "11", intPtr(3)},

		// Edge cases for radix values
		{"2r101", "2r", 2, "101", "", nil},  // minimum radix
		{"36rZ", "36r", 36, "Z", "", nil},   // maximum radix
		{"12rAB", "12r", 12, "AB", "", nil}, // double digit radix

		// Complex cases
		{"16rDEAD.BEEFe10", "16r", 16, "DEAD", "BEEF", intPtr(10)},
		{"36rHELLO.WORLDe-5", "36r", 36, "HELLO", "WORLD", intPtr(-5)},
		{"8r123.456e+7", "8r", 8, "123", "456", intPtr(7)},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.Tokenize()

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(tokens) != 1 {
				t.Errorf("Expected 1 token, got %d", len(tokens))
				return
			}

			token := tokens[0]
			if token.Type != NumericLiteralTokenType {
				t.Errorf("Expected numeric token, got %s", token.Type)
			}

			if token.Radix == nil || *token.Radix != tt.expectedRadix {
				t.Errorf("Expected radix '%s', got %v", tt.expectedRadix, token.Radix)
			}

			if token.Base == nil || *token.Base != tt.expectedBase {
				t.Errorf("Expected base %d, got %v", tt.expectedBase, token.Base)
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

			if tt.expectedExponent == nil {
				if token.Exponent != nil {
					t.Errorf("Expected no exponent, got %d", *token.Exponent)
				}
			} else {
				if token.Exponent == nil || *token.Exponent != *tt.expectedExponent {
					t.Errorf("Expected exponent %d, got %v", *tt.expectedExponent, token.Exponent)
				}
			}
		})
	}
}

func TestEnhancedNumericEdgeCases(t *testing.T) {
	// Helper function to create int pointers
	intPtr := func(i int) *int { return &i }

	tests := []struct {
		name             string
		input            string
		expectedTokens   int
		expectedRadix    string
		expectedBase     int
		expectedMantissa string
		expectedFraction string
		expectedExponent *int
	}{
		// Test proper case sensitivity - only uppercase digits allowed
		{"Lowercase e exponent", "10r123e5", 1, "10r", 10, "123", "", intPtr(5)},

		// Test boundary radix values
		{"Minimum radix", "2r1", 1, "2r", 2, "1", "", nil},
		{"Maximum radix", "36rZ", 1, "36r", 36, "Z", "", nil},
		{"Near maximum radix", "35rY", 1, "35r", 35, "Y", "", nil},

		// Test multi-digit radix values
		{"Double digit radix 10", "10r9", 1, "10r", 10, "9", "", nil},
		{"Double digit radix 16", "16rF", 1, "16r", 16, "F", "", nil},
		{"Double digit radix 36", "36rZ", 1, "36r", 36, "Z", "", nil},

		// Test comprehensive digit ranges for different bases
		{"Base 2 max digits", "2r1", 1, "2r", 2, "1", "", nil},
		{"Base 8 max digits", "8r7", 1, "8r", 8, "7", "", nil},
		{"Base 10 max digits", "10r9", 1, "10r", 10, "9", "", nil},
		{"Base 16 max digits", "16rF", 1, "16r", 16, "F", "", nil},
		{"Base 36 max digits", "36rZ", 1, "36r", 36, "Z", "", nil},

		// Test floating point in various bases
		{"Binary floating point", "2r1.1", 1, "2r", 2, "1", "1", nil},
		{"Octal floating point", "8r7.6", 1, "8r", 8, "7", "6", nil},
		{"Hex floating point", "16rF.E", 1, "16r", 16, "F", "E", nil},
		{"Base 36 floating point", "36rZ.Y", 1, "36r", 36, "Z", "Y", nil},

		// Test scientific notation in various bases
		{"Binary scientific", "2r1e2", 1, "2r", 2, "1", "", intPtr(2)},
		{"Hex scientific", "16rFe10", 1, "16r", 16, "F", "", intPtr(10)},
		{"Base 36 scientific", "36rZe5", 1, "36r", 36, "Z", "", intPtr(5)},

		// Test combined floating point and scientific notation
		{"Hex float scientific", "16rA.Be-2", 1, "16r", 16, "A", "B", intPtr(-2)},
		{"Base 36 float scientific", "36rH.ELLOe+10", 1, "36r", 36, "H", "ELLO", intPtr(10)},

		// Test complex realistic examples
		{"Hex color value", "16rFFFFFF", 1, "16r", 16, "FFFFFF", "", nil},
		{"Base 36 word", "36rHELLOWORLD", 1, "36r", 36, "HELLOWORLD", "", nil},
		{"Large hex number", "16rDEADBEEF", 1, "16r", 16, "DEADBEEF", "", nil},

		// Test edge cases with scientific notation avoiding lowercase conflicts
		{"Hex with exponent", "16rABCe5", 1, "16r", 16, "ABC", "", intPtr(5)},
		{"Base 36 avoiding E digit", "36rHELLe10", 1, "36r", 36, "HELL", "", intPtr(10)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.Tokenize()

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
			if token.Type != NumericLiteralTokenType {
				t.Errorf("Expected numeric token, got %s", token.Type)
				return
			}

			if token.Radix == nil || *token.Radix != tt.expectedRadix {
				t.Errorf("Expected radix '%s', got %v", tt.expectedRadix, token.Radix)
			}

			if token.Base == nil || *token.Base != tt.expectedBase {
				t.Errorf("Expected base %d, got %v", tt.expectedBase, token.Base)
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

			if tt.expectedExponent == nil {
				if token.Exponent != nil {
					t.Errorf("Expected no exponent, got %d", *token.Exponent)
				}
			} else {
				if token.Exponent == nil || *token.Exponent != *tt.expectedExponent {
					t.Errorf("Expected exponent %d, got %v", *tt.expectedExponent, token.Exponent)
				}
			}
		})
	}
}

func TestNumericWithUnderscores(t *testing.T) {
	// Helper function to create int pointers
	intPtr := func(i int) *int { return &i }

	tests := []struct {
		name             string
		input            string
		expectedRadix    string
		expectedBase     int
		expectedMantissa string
		expectedFraction string
		expectedExponent *int
	}{
		// Decimal numbers with underscores
		{"Decimal with underscores", "1_234_567", "", 10, "1234567", "", nil},
		{"Decimal float with underscores", "3_14.15_92", "", 10, "314", "1592", nil},
		{"Decimal scientific with underscores", "1_23e45", "", 10, "123", "", intPtr(45)},
		{"Decimal float scientific with underscores", "1_23.45_67e89", "", 10, "123", "4567", intPtr(89)},

		// Binary with underscores
		{"Binary with underscores", "0b1010_1100", "0b", 2, "10101100", "", nil},
		{"Binary float with underscores", "0b10_10.11_01", "0b", 2, "1010", "1101", nil},

		// Octal with underscores
		{"Octal with underscores", "0o12_34_56", "0o", 8, "123456", "", nil},
		{"Octal float with underscores", "0o12_3.45_6", "0o", 8, "123", "456", nil},

		// Hexadecimal with underscores
		{"Hex with underscores", "0xDE_AD_BE_EF", "0x", 16, "DEADBEEF", "", nil},
		{"Hex float with underscores", "0xFF_AA.BB_CC", "0x", 16, "FFAA", "BBCC", nil},

		// rR notation with underscores
		{"Binary rR with underscores", "2r10_10_11", "2r", 2, "101011", "", nil},
		{"Hex rR with underscores", "16rDE_AD_BE_EF", "16r", 16, "DEADBEEF", "", nil},
		{"Base 36 with underscores", "36rHE_LL_O", "36r", 36, "HELLO", "", nil},
		{"rR float with underscores", "16rAB_C.DE_F", "16r", 16, "ABC", "DEF", nil},
		{"rR scientific with underscores", "10r12_3e45", "10r", 10, "123", "", intPtr(45)},
		{"rR float scientific with underscores", "16rAB_C.DE_Fe10", "16r", 16, "ABC", "DEF", intPtr(10)},

		// Edge cases
		{"Single underscore in mantissa", "1_2", "", 10, "12", "", nil},
		{"Multiple underscores", "1_2_3_4", "", 10, "1234", "", nil},
		{"Underscores in fraction only", "12.3_4_5", "", 10, "12", "345", nil},
		{"Complex case", "36rA_B_C.D_E_Fe+12", "36r", 36, "ABC", "DEF", intPtr(12)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.Tokenize()

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(tokens) != 1 {
				t.Errorf("Expected 1 token, got %d", len(tokens))
				return
			}

			token := tokens[0]
			if token.Type != NumericLiteralTokenType {
				t.Errorf("Expected numeric token, got %s", token.Type)
			}

			if token.Radix == nil || *token.Radix != tt.expectedRadix {
				t.Errorf("invalid numeric literal")
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

			if tt.expectedExponent == nil {
				if token.Exponent != nil {
					t.Errorf("Expected no exponent, got %d", *token.Exponent)
				}
			} else {
				if token.Exponent == nil || *token.Exponent != *tt.expectedExponent {
					t.Errorf("Expected exponent %d, got %v", *tt.expectedExponent, token.Exponent)
				}
			}
		})
	}
}

func TestBalancedTernaryTokens(t *testing.T) {
	// Helper function to create int pointers
	intPtr := func(i int) *int { return &i }

	tests := []struct {
		name             string
		input            string
		expectedRadix    string
		expectedBase     int
		expectedMantissa string
		expectedFraction string
		expectedExponent *int
		expectedBalanced bool
	}{
		// Basic balanced ternary integers
		{"Simple balanced ternary", "0t10", "0t", 3, "10", "", nil, true},
		{"Negative balanced ternary", "0tT1", "0t", 3, "T1", "", nil, true},
		{"Complex balanced ternary", "0t1T0", "0t", 3, "1T0", "", nil, true},
		{"All zeros", "0t000", "0t", 3, "000", "", nil, true},
		{"All ones", "0t111", "0t", 3, "111", "", nil, true},
		{"All T (negative)", "0tTTT", "0t", 3, "TTT", "", nil, true},

		// Balanced ternary with fractions
		{"Mixed integer and fraction", "0t1.T", "0t", 3, "1", "T", nil, true},
		{"Complex fraction", "0tT.01", "0t", 3, "T", "01", nil, true},
		{"Long fraction", "0t10.1T0", "0t", 3, "10", "1T0", nil, true},

		// Balanced ternary with scientific notation
		{"Balanced ternary with exponent", "0tTTe-2", "0t", 3, "TT", "", intPtr(-2), true},
		{"Integer with positive exponent", "0t10e+3", "0t", 3, "10", "", intPtr(3), true},
		{"Fraction with exponent", "0t1.Te5", "0t", 3, "1", "T", intPtr(5), true},
		{"Complex with exponent", "0t1T0.01e-4", "0t", 3, "1T0", "01", intPtr(-4), true},

		// Balanced ternary with underscores
		{"Underscores in mantissa", "0t1_0_T", "0t", 3, "10T", "", nil, true},
		{"Underscores in fraction", "0t10.1_T_0", "0t", 3, "10", "1T0", nil, true},
		{"Underscores in both", "0t1_T.0_1", "0t", 3, "1T", "01", nil, true},
		{"Complex with underscores", "0t1_T_0.T_0_1e+2", "0t", 3, "1T0", "T01", intPtr(2), true},

		// Edge cases
		{"Single digit zero", "0t0", "0t", 3, "0", "", nil, true},
		{"Single digit one", "0t1", "0t", 3, "1", "", nil, true},
		{"Single digit T", "0tT", "0t", 3, "T", "", nil, true},
		{"Multiple underscores", "0t1_1_1_T_T_T", "0t", 3, "111TTT", "", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.Tokenize()

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(tokens) != 1 {
				t.Errorf("Expected 1 token, got %d", len(tokens))
				return
			}

			token := tokens[0]
			if token.Type != NumericLiteralTokenType {
				t.Errorf("Expected numeric token, got %s", token.Type)
			}

			if token.Radix == nil || *token.Radix != tt.expectedRadix {
				t.Errorf("invalid numeric literal")
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

			if tt.expectedExponent == nil {
				if token.Exponent != nil {
					t.Errorf("Expected no exponent, got %d", *token.Exponent)
				}
			} else {
				if token.Exponent == nil || *token.Exponent != *tt.expectedExponent {
					t.Errorf("Expected exponent %d, got %v", *tt.expectedExponent, token.Exponent)
				}
			}

			if token.Balanced == nil || *token.Balanced != tt.expectedBalanced {
				t.Errorf("Expected balanced %t, got %v", tt.expectedBalanced, token.Balanced)
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
		{"def", StartTokenType, []string{"=>>"}},
		{"if", StartTokenType, []string{"then"}},
		{"class", StartTokenType, []string{}},
		{"fn", StartTokenType, []string{"=>>"}},
		{"for", StartTokenType, []string{"do"}},
		{"try", StartTokenType, []string{"catch", "else"}},
		{"transaction", StartTokenType, []string{"catch", "else"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.Tokenize()

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
		{"+", [3]int{80, 2080, 0}}, // + has base precedence 40, only infix enabled (40+2000=2040)
		{"-", [3]int{90, 2090, 0}}, // - has base precedence 50, both prefix (50) and infix (50+2000=2050) enabled
		{"*", [3]int{0, 2050, 0}},  // * has base precedence 10, only infix enabled (10+2000=2010)
		{"==", [3]int{0, 2179, 0}}, // = has base precedence 140, repeated so 139, only infix enabled (139+2000=2139)
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.Tokenize()

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(tokens) != 1 {
				t.Errorf("Expected 1 token, got %d", len(tokens))
				return
			}

			token := tokens[0]
			if token.Type != OperatorTokenType {
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
		infixPrec    int
		isPrefix     bool
	}{
		{"(", OpenDelimiterTokenType, []string{")"}, 2020, true},
		{"[", OpenDelimiterTokenType, []string{"]"}, 2030, true},
		{"{", OpenDelimiterTokenType, []string{"}"}, 2040, true}, // Updated: now supports infix usage for f{x} syntax
		{")", CloseDelimiterTokenType, nil, 0, false},
		{"]", CloseDelimiterTokenType, nil, 0, false},
		{"}", CloseDelimiterTokenType, nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.Tokenize()

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

			if tt.expectedType == OpenDelimiterTokenType {
				if len(token.ClosedBy) != len(tt.closedBy) {
					t.Errorf("Expected closed by %v, got %v", tt.closedBy, token.ClosedBy)
				} else {
					for i, expected := range tt.closedBy {
						if token.ClosedBy[i] != expected {
							t.Errorf("Expected closed by '%s' at index %d, got '%s'", expected, i, token.ClosedBy[i])
						}
					}
				}

				if token.InfixPrecedence == nil || *token.InfixPrecedence != tt.infixPrec {
					t.Errorf("Expected infix %d, got %v", tt.infixPrec, token.InfixPrecedence)
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
		// Bridge tokens (L)
		{"=>>", BridgeTokenType},
		{"do", BridgeTokenType},
		{"then", BridgeTokenType},
		{"else", BridgeTokenType},

		// Unclassified tokens (U)
		{":", UnclassifiedTokenType}, // bare wildcard without context

		// Compound tokens (C)
		{"catch", BridgeTokenType},
		{"elseif", BridgeTokenType},
		{"elseifnot", BridgeTokenType},

		// Prefix tokens (P)
		{"return", PrefixTokenType},
		{"yield", PrefixTokenType},

		// End tokens (E)
		{"end", EndTokenType},
		{"enddef", EndTokenType},
		{"endclass", EndTokenType},

		// Variable tokens (V) - should default to this for unknown identifiers
		{"myVariable", VariableTokenType},
		{"unknown", VariableTokenType},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.Tokenize()

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
	tokenizer := NewTokenizer(input)
	tokens, err := tokenizer.Tokenize()

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
	tokenizer := NewTokenizer(input)
	tokens, err := tokenizer.Tokenize()

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
	rules := DefaultRules()
	rules.WildcardTokens = map[string]bool{"***": true} // Use '*' as wildcard instead of ':'

	// Build the precomputed lookup map
	if err := rules.BuildTokenLookup(); err != nil {
		t.Fatalf("Failed to build token lookup: %v", err)
	}

	// Test with custom wildcard in a def context
	tokenizer := NewTokenizerWithRules("def foo ***", rules)
	tokens, err := tokenizer.Tokenize()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(tokens) != 3 {
		t.Fatalf("Expected 3 tokens, got %d", len(tokens))
	}

	// Third token should be a wildcard token behaving like "=>>"
	wildcardToken := tokens[2]
	if wildcardToken.Text != "***" {
		t.Errorf("Expected wildcard token text to be '***', got '%s'", wildcardToken.Text)
	}

	if wildcardToken.Type != BridgeTokenType {
		t.Errorf("Expected wildcard token type to be Bridge, got %s", wildcardToken.Type)
	}

	if wildcardToken.Alias == nil || *wildcardToken.Alias != "=>>" {
		t.Errorf("Expected wildcard token alias to be '=>>', got '%v'", wildcardToken.Alias)
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

// TestExceptionTokens tests that invalid numeric literals produce exception tokens.
func TestExceptionTokens(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "Invalid base 10 digits",
			input: "10rAB",
		},
		{
			name:  "Invalid base 9 digits",
			input: "9rAB",
		},
		{
			name:  "Invalid base 35 digits",
			input: "35rYZ",
		},
		{
			name:  "Invalid binary digits",
			input: "2r123",
		},
		{
			name:  "Invalid octal digits",
			input: "8r89",
		},
		{
			name:  "Invalid hex prefix digits",
			input: "0xGHI",
		},
		{
			name:  "Invalid fraction digits",
			input: "8r12.89",
		},
		{
			name:  "Invalid balanced ternary wrong radix",
			input: "4t0T1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.Tokenize()

			// Should get an error
			if err == nil {
				t.Errorf("Expected an error, but got none")
				return
			}

			// Should still have one token (the exception token)
			if len(tokens) != 1 {
				t.Errorf("Expected 1 token (exception), got %d", len(tokens))
				return
			}

			token := tokens[0]
			if token.Type != ExceptionTokenType {
				t.Errorf("Expected exception token, got %s", token.Type)
			}
		})
	}
}

func TestNewlineTracking(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []struct {
			text     string
			lnBefore *bool
			lnAfter  *bool
		}
	}{
		{
			name:  "Single line, no newlines",
			input: "a b c",
			expected: []struct {
				text     string
				lnBefore *bool
				lnAfter  *bool
			}{
				{"a", nil, nil}, // no newlines before or after
				{"b", nil, nil}, // no newlines before or after
				{"c", nil, nil}, // no newlines before or after
			},
		},
		{
			name:  "Simple newline between tokens",
			input: "a\nb",
			expected: []struct {
				text     string
				lnBefore *bool
				lnAfter  *bool
			}{
				{"a", nil, boolPtr(true)}, // newline after
				{"b", boolPtr(true), nil}, // newline before
			},
		},
		{
			name:  "Multiple newlines",
			input: "a\n\nb",
			expected: []struct {
				text     string
				lnBefore *bool
				lnAfter  *bool
			}{
				{"a", nil, boolPtr(true)}, // newline after
				{"b", boolPtr(true), nil}, // newline before
			},
		},
		{
			name:  "Mixed spaces and newlines",
			input: "a  \n  b",
			expected: []struct {
				text     string
				lnBefore *bool
				lnAfter  *bool
			}{
				{"a", nil, boolPtr(true)}, // newline after (in the whitespace)
				{"b", boolPtr(true), nil}, // newline before (in the whitespace)
			},
		},
		{
			name:  "Three tokens on separate lines",
			input: "x\ny\nz",
			expected: []struct {
				text     string
				lnBefore *bool
				lnAfter  *bool
			}{
				{"x", nil, boolPtr(true)},           // newline after
				{"y", boolPtr(true), boolPtr(true)}, // newlines before and after
				{"z", boolPtr(true), nil},           // newline before
			},
		},
		{
			name:  "Comment treated as newline",
			input: "a ### comment\nb",
			expected: []struct {
				text     string
				lnBefore *bool
				lnAfter  *bool
			}{
				{"a", nil, boolPtr(true)}, // comment after is treated as newline
				{"b", boolPtr(true), nil}, // newline before
			},
		},
		{
			name:  "Complex multi-line example",
			input: "def foo(x)\n    return x + 1\nend",
			expected: []struct {
				text     string
				lnBefore *bool
				lnAfter  *bool
			}{
				{"def", nil, nil},              // followed by space
				{"foo", nil, nil},              // followed by (
				{"(", nil, nil},                // followed by x
				{"x", nil, nil},                // followed by )
				{")", nil, boolPtr(true)},      // followed by newline
				{"return", boolPtr(true), nil}, // preceded by newline, followed by space
				{"x", nil, nil},                // followed by space
				{"+", nil, nil},                // followed by space
				{"1", nil, boolPtr(true)},      // followed by newline
				{"end", boolPtr(true), nil},    // preceded by newline
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.Tokenize()

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(tokens) != len(tt.expected) {
				t.Errorf("Expected %d tokens, got %d", len(tt.expected), len(tokens))
				return
			}

			for i, token := range tokens {
				expected := tt.expected[i]

				if token.Text != expected.text {
					t.Errorf("Token %d: expected text %q, got %q", i, expected.text, token.Text)
				}

				// Check LnBefore
				if expected.lnBefore == nil {
					if token.LnBefore != nil {
						t.Errorf("Token %d (%q): expected LnBefore to be nil, got %v", i, token.Text, *token.LnBefore)
					}
				} else {
					if token.LnBefore == nil {
						t.Errorf("Token %d (%q): expected LnBefore to be %v, got nil", i, token.Text, *expected.lnBefore)
					} else if *token.LnBefore != *expected.lnBefore {
						t.Errorf("Token %d (%q): expected LnBefore to be %v, got %v", i, token.Text, *expected.lnBefore, *token.LnBefore)
					}
				}

				// Check LnAfter
				if expected.lnAfter == nil {
					if token.LnAfter != nil {
						t.Errorf("Token %d (%q): expected LnAfter to be nil, got %v", i, token.Text, *token.LnAfter)
					}
				} else {
					if token.LnAfter == nil {
						t.Errorf("Token %d (%q): expected LnAfter to be %v, got nil", i, token.Text, *expected.lnAfter)
					} else if *token.LnAfter != *expected.lnAfter {
						t.Errorf("Token %d (%q): expected LnAfter to be %v, got %v", i, token.Text, *expected.lnAfter, *token.LnAfter)
					}
				}
			}
		})
	}
}

func TestNewlineJSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []map[string]interface{}
	}{
		{
			name:  "Token with newline before",
			input: "\na",
			expected: []map[string]interface{}{
				{
					"text":      "a",
					"ln_before": true,
				},
			},
		},
		{
			name:  "Token with newline after",
			input: "a\n",
			expected: []map[string]interface{}{
				{
					"text":     "a",
					"ln_after": true,
				},
			},
		},
		{
			name:  "Token with newlines before and after",
			input: "\na\n",
			expected: []map[string]interface{}{
				{
					"text":      "a",
					"ln_before": true,
					"ln_after":  true,
				},
			},
		},
		{
			name:  "Token without newlines should not have ln_before/ln_after fields",
			input: "a",
			expected: []map[string]interface{}{
				{
					"text": "a",
					// ln_before and ln_after should not be present in JSON
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.Tokenize()

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(tokens) != len(tt.expected) {
				t.Errorf("Expected %d tokens, got %d", len(tt.expected), len(tokens))
				return
			}

			for i, token := range tokens {
				expected := tt.expected[i]

				// Serialize token to JSON
				jsonBytes, err := json.Marshal(token)
				if err != nil {
					t.Errorf("Failed to marshal token to JSON: %v", err)
					continue
				}

				// Parse JSON back to map
				var actual map[string]interface{}
				if err := json.Unmarshal(jsonBytes, &actual); err != nil {
					t.Errorf("Failed to unmarshal JSON: %v", err)
					continue
				}

				// Check expected fields are present and correct
				for key, expectedValue := range expected {
					if actualValue, exists := actual[key]; !exists {
						t.Errorf("Token %d: expected field %q to be present in JSON", i, key)
					} else if actualValue != expectedValue {
						t.Errorf("Token %d: expected %q to be %v, got %v", i, key, expectedValue, actualValue)
					}
				}

				// Check that ln_before and ln_after are only present when they should be
				if token.LnBefore == nil {
					if _, exists := actual["ln_before"]; exists {
						t.Errorf("Token %d: ln_before should not be present in JSON when LnBefore is nil", i)
					}
				}
				if token.LnAfter == nil {
					if _, exists := actual["ln_after"]; exists {
						t.Errorf("Token %d: ln_after should not be present in JSON when LnAfter is nil", i)
					}
				}
			}
		})
	}
}

// Helper function to create bool pointers for test expectations
func boolPtr(b bool) *bool {
	return &b
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
