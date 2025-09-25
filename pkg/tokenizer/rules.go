package tokenizer

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// RulesFile represents the structure of a YAML rules file
type RulesFile struct {
	Bracket  []BracketRule  `yaml:"bracket"`
	Prefix   []PrefixRule   `yaml:"prefix"`
	Start    []StartRule    `yaml:"start"`
	Bridge   []BridgeRule   `yaml:"bridge"`
	Wildcard []WildcardRule `yaml:"wildcard"`
	Operator []OperatorRule `yaml:"operator"`
	Mark     []MarkRule     `yaml:"mark"`
}

type MarkRule struct {
	Text string `yaml:"text"`
}

// BracketRule represents a bracket token rule
type BracketRule struct {
	Text      string   `yaml:"text"`
	ClosedBy  []string `yaml:"closed_by"`
	InfixPrec int      `yaml:"infix"`
	Prefix    bool     `yaml:"prefix"`
}

// PrefixRule represents a prefix token rule
type PrefixRule struct {
	Text string `yaml:"text"`
}

// StartRule represents a start token rule
type StartRule struct {
	Text      string   `yaml:"text"`
	ClosedBy  []string `yaml:"closed_by"`
	Expecting []string `yaml:"expecting"`
	Single    bool     `yaml:"single"`
}

// BridgeRule represents a bridge token rule
type BridgeRule struct {
	Text      string   `yaml:"text"`
	Expecting []string `yaml:"expecting"`
	In        []string `yaml:"in"`
}

// CompoundRule represents a compound token rule
type CompoundRule struct {
	Text      string   `yaml:"text"`
	Expecting []string `yaml:"expecting"`
	In        []string `yaml:"in"`
}

// WildcardRule represents a wildcard token rule
type WildcardRule struct {
	Text string `yaml:"text"`
}

// OperatorRule represents an operator token rule
type OperatorRule struct {
	Text       string `yaml:"text"`
	Precedence [3]int `yaml:"precedence"` // [prefix, infix, postfix]
}

// CustomRuleType represents the type of custom rule
type CustomRuleType int

const (
	CustomWildcard CustomRuleType = iota
	CustomStart
	CustomEnd
	CustomBridge
	CustomPrefix
	CustomOperator
	CustomOpenDelimiter
	CustomCloseDelimiter
	CustomMark
)

// CustomRuleEntry holds the rule type and any associated data
type CustomRuleEntry struct {
	Type CustomRuleType
	Data interface{} // Can be StartTokenData, BridgeTokenData, etc.
}

// TokenizerRules holds all the rule maps that can be customized
type TokenizerRules struct {
	StartTokens         map[string]StartTokenData
	BridgeTokens        map[string]BridgeTokenData
	PrefixTokens        map[string]bool
	DelimiterMappings   map[string][]string
	DelimiterProperties map[string]DelimiterProp
	WildcardTokens      map[string]bool
	OperatorPrecedences map[string][3]int // [prefix, infix, postfix]
	MarkTokens          map[string]bool

	// Precomputed lookup map for efficient matching
	TokenLookup map[string]CustomRuleEntry
}

// DefaultRules returns the default tokenizer rules
func DefaultRules() *TokenizerRules {
	rules := &TokenizerRules{
		StartTokens:         getDefaultStartTokens(),
		BridgeTokens:        getDefaultBridgeTokens(),
		PrefixTokens:        getDefaultPrefixTokens(),
		DelimiterMappings:   getDefaultDelimiterMappings(),
		DelimiterProperties: getDefaultDelimiterProperties(),
		WildcardTokens:      getDefaultWildcardTokens(),
		OperatorPrecedences: getDefaultOperatorPrecedences(),
		MarkTokens:          map[string]bool{",": true, ";": true},
	}

	// Build the precomputed lookup map
	// Note: Default rules should never have conflicts, so we panic if there's an error
	if err := rules.BuildTokenLookup(); err != nil {
		panic(fmt.Sprintf("Invalid default rules: %v", err))
	}

	return rules
}

// LoadRulesFile loads and parses a YAML rules file
func LoadRulesFile(filename string) (*RulesFile, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules file '%s': %w", filename, err)
	}

	var rules RulesFile
	if err := yaml.Unmarshal(data, &rules); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in rules file '%s': %w", filename, err)
	}

	return &rules, nil
}

// ApplyRulesToDefaults applies the rules from a RulesFile to create a new TokenizerRules.
// Returns an error if there are conflicting token definitions.
func ApplyRulesToDefaults(rules *RulesFile) (*TokenizerRules, error) {
	tokenizerRules := DefaultRules()

	// Apply bracket rules
	if len(rules.Bracket) > 0 {
		tokenizerRules.DelimiterMappings = make(map[string][]string)
		tokenizerRules.DelimiterProperties = make(map[string]DelimiterProp)

		for _, rule := range rules.Bracket {
			tokenizerRules.DelimiterMappings[rule.Text] = rule.ClosedBy
			tokenizerRules.DelimiterProperties[rule.Text] = DelimiterProp{rule.InfixPrec, rule.Prefix}
		}
	}

	// Apply prefix rules
	if len(rules.Prefix) > 0 {
		tokenizerRules.PrefixTokens = make(map[string]bool)
		for _, rule := range rules.Prefix {
			tokenizerRules.PrefixTokens[rule.Text] = true
		}
	}

	// Apply mark rules
	if len(rules.Mark) > 0 {
		tokenizerRules.MarkTokens = make(map[string]bool)
		for _, rule := range rules.Mark {
			tokenizerRules.MarkTokens[rule.Text] = true
		}
	}

	// Apply start rules
	if len(rules.Start) > 0 {
		tokenizerRules.StartTokens = make(map[string]StartTokenData)
		for _, rule := range rules.Start {
			tokenizerRules.StartTokens[rule.Text] = StartTokenData{
				Expecting: rule.Expecting,
				ClosedBy:  rule.ClosedBy,
			}
		}
	}

	// Apply bridge rules
	if len(rules.Bridge) > 0 {
		tokenizerRules.BridgeTokens = make(map[string]BridgeTokenData)
		for _, rule := range rules.Bridge {
			tokenizerRules.BridgeTokens[rule.Text] = BridgeTokenData{
				Expecting: rule.Expecting,
				In:        rule.In,
			}
		}
	}

	// Apply wildcard rules
	if len(rules.Wildcard) > 0 {
		tokenizerRules.WildcardTokens = make(map[string]bool)
		for _, rule := range rules.Wildcard {
			tokenizerRules.WildcardTokens[rule.Text] = true
		}
	}

	// Apply operator rules
	if len(rules.Operator) > 0 {
		for _, rule := range rules.Operator {
			tokenizerRules.OperatorPrecedences[rule.Text] = rule.Precedence
		}
	}

	// Build the precomputed lookup map for efficient matching
	if err := tokenizerRules.BuildTokenLookup(); err != nil {
		return nil, err
	}

	return tokenizerRules, nil
}

// Helper functions to get default values (these will copy from the existing global variables)

func getDefaultOperatorPrecedences() map[string][3]int {
	m := make(map[string][3]int)
	updateOperatorPrecedence(m, ".")
	updateOperatorPrecedence(m, "*")
	updateOperatorPrecedence(m, "/")
	updateOperatorPrecedence(m, "+")
	updateOperatorPrecedence(m, "-")
	updateOperatorPrecedence(m, "<")
	updateOperatorPrecedence(m, ">")
	updateOperatorPrecedence(m, "<=")
	updateOperatorPrecedence(m, ">=")
	updateOperatorPrecedence(m, "==")
	updateOperatorPrecedence(m, "..<")
	updateOperatorPrecedence(m, "..=")
	updateOperatorPrecedence(m, ":=")
	updateOperatorPrecedence(m, "<-")
	updateOperatorPrecedence(m, "<--")
	m["in"] = [3]int{0, 3000, 0}
	return m
}

func getDefaultStartTokens() map[string]StartTokenData {
	return map[string]StartTokenData{
		"def": {
			Expecting: []string{"=>>"},
			ClosedBy:  []string{"end", "enddef"},
			Arity:     One,
		},
		"let": {
			Expecting: []string{},
			ClosedBy:  []string{"end", "endlet"},
			Arity:     Many,
		},
		"switch": {
			Expecting: []string{"case", "else"},
			ClosedBy:  []string{"end", "endswitch"},
			Arity:     One,
		},
		"if": {
			Expecting: []string{"then"},
			ClosedBy:  []string{"end", "endif"},
			Arity:     One,
		},
		"ifnot": {
			Expecting: []string{"then"},
			ClosedBy:  []string{"end", "endifnot"},
			Arity:     One,
		},
		"fn": {
			Expecting: []string{"=>>"},
			ClosedBy:  []string{"end", "endfn"},
			Arity:     One,
		},
		"class": {
			Expecting: []string{},
			ClosedBy:  []string{"end", "endclass"},
			Arity:     One,
		},
		"for": {
			Expecting: []string{"do"},
			ClosedBy:  []string{"end", "endfor"},
			Arity:     One,
		},
		"try": {
			Expecting: []string{"catch", "else"},
			ClosedBy:  []string{"end", "endtry"},
			Arity:     Many,
		},
		"transaction": {
			Expecting: []string{"catch", "else"},
			ClosedBy:  []string{"end", "endtransaction"},
			Arity:     Many,
		},
	}
}

func getDefaultBridgeTokens() map[string]BridgeTokenData {
	return map[string]BridgeTokenData{
		"case": {
			Expecting: []string{"then"},
			In:        []string{"switch"},
			Arity:     One,
		},
		"=>>": {
			Expecting: []string{"end", "enddef", "endfn"},
			In:        []string{"def"},
			Arity:     Many,
		},
		"do": {
			Expecting: []string{"end", "endfor"},
			In:        []string{"def", "for"},
			Arity:     Many,
		},
		"then": {
			Expecting: []string{"case", "elseif", "else", "end", "endif", "endifnot", "endswitch", "endcase"},
			In:        []string{"if", "ifnot", "switch"},
			Arity:     Many,
		},
		"elseif": {
			Expecting: []string{"then"},
			In:        []string{"if", "ifnot"},
			Arity:     One,
		},
		"elseifnot": {
			Expecting: []string{"then"},
			In:        []string{"if", "ifnot"},
			Arity:     Many,
		},
		"else": {
			Expecting: []string{"end", "endif", "endifnot", "endswitch", "endcase"},
			In:        []string{"if", "ifnot", "switch"},
			Arity:     Many,
		},
		"endcase": {
			Expecting: []string{"end", "endswitch"},
			In:        []string{"switch"},
			Arity:     Zero,
		},
		"catch": {
			Expecting: []string{},
			In:        []string{"try"},
			Arity:     One,
		},
	}
}

func getDefaultPrefixTokens() map[string]bool {
	return map[string]bool{
		"return": true,
		"yield":  true,
		"const":  true,
		"var":    true,
		"val":    true,
	}
}

func getDefaultDelimiterMappings() map[string][]string {
	return map[string][]string{
		"(": {")"},
		"[": {"]"},
		"{": {"}"},
	}
}

type DelimiterProp struct {
	InfixPrec int
	Prefix    bool
}

func getDefaultDelimiterProperties() map[string]DelimiterProp {
	_, a, _ := calculateOperatorPrecedence("(")
	_, b, _ := calculateOperatorPrecedence("[")
	_, c, _ := calculateOperatorPrecedence("{")
	return map[string]DelimiterProp{
		"(": {a, true}, // infix=true, prefix=true
		"[": {b, true}, // infix=true, prefix=false
		"{": {c, true}, // infix=false, prefix=true
	}
}

func getDefaultWildcardTokens() map[string]bool {
	return map[string]bool{
		":": true,
	}
}

// BuildTokenLookup creates the precomputed lookup map for efficient token matching.
// Returns an error if a token is defined in multiple rules.
func (rules *TokenizerRules) BuildTokenLookup() error {
	rules.TokenLookup = make(map[string]CustomRuleEntry)
	tokenSources := make(map[string]string) // Track which rule type defined each token

	// Helper function to add a token and check for duplicates
	addToken := func(token string, ruleType CustomRuleType, ruleTypeName string, data interface{}) error {
		if existingSource, exists := tokenSources[token]; exists {
			return fmt.Errorf("token '%s' is defined in both %s and %s rules", token, existingSource, ruleTypeName)
		}
		tokenSources[token] = ruleTypeName
		rules.TokenLookup[token] = CustomRuleEntry{
			Type: ruleType,
			Data: data,
		}
		return nil
	}

	// Add wildcard tokens
	for token := range rules.WildcardTokens {
		if err := addToken(token, CustomWildcard, "wildcard", nil); err != nil {
			return err
		}
	}

	// Add start tokens
	for token, data := range rules.StartTokens {
		if err := addToken(token, CustomStart, "start", data); err != nil {
			return err
		}
	}

	// Add bridge tokens
	for token, data := range rules.BridgeTokens {
		if err := addToken(token, CustomBridge, "bridge", data); err != nil {
			return err
		}
	}

	// Add prefix tokens
	for token := range rules.PrefixTokens {
		if err := addToken(token, CustomPrefix, "prefix", nil); err != nil {
			return err
		}
	}

	// Add mark tokens
	for token := range rules.MarkTokens {
		if err := addToken(token, CustomMark, "mark", nil); err != nil {
			return err
		}
	}

	// Add operator tokens
	for token, precedence := range rules.OperatorPrecedences {
		if err := addToken(token, CustomOperator, "operator", precedence); err != nil {
			return err
		}
	}

	// Add open delimiter tokens
	for token, closedBy := range rules.DelimiterMappings {
		props := rules.DelimiterProperties[token]
		delimiterData := struct {
			ClosedBy  []string
			InfixPrec int
			IsPrefix  bool
		}{
			ClosedBy:  closedBy,
			InfixPrec: props.InfixPrec,
			IsPrefix:  props.Prefix,
		}
		if err := addToken(token, CustomOpenDelimiter, "bracket", delimiterData); err != nil {
			return err
		}
	}

	// Add close delimiter tokens (derived from closed_by fields)
	// Note: These can legitimately appear multiple times from different brackets
	closeDelimiters := make(map[string]bool)
	for _, closedByList := range rules.DelimiterMappings {
		for _, closer := range closedByList {
			if !closeDelimiters[closer] {
				closeDelimiters[closer] = true
				// Don't check for duplicates for close delimiters since they're derived
				rules.TokenLookup[closer] = CustomRuleEntry{
					Type: CustomCloseDelimiter,
					Data: nil,
				}
			}
		}
	}

	// Add end tokens (derived from start token closed_by fields)
	// Note: These can legitimately appear multiple times from different start tokens
	endTokens := make(map[string]bool)
	for _, startData := range rules.StartTokens {
		for _, endToken := range startData.ClosedBy {
			if !endTokens[endToken] {
				endTokens[endToken] = true
				// Don't check for duplicates for end tokens since they're derived
				rules.TokenLookup[endToken] = CustomRuleEntry{
					Type: CustomEnd,
					Data: nil,
				}
			}
		}
	}

	return nil
}

func updateOperatorPrecedence(m map[string][3]int, operator string) {
	prefix, infix, postfix := calculateOperatorPrecedence(operator)
	m[operator] = [3]int{prefix, infix, postfix}
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

	if operator == "-" || operator == "+" {
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
}
