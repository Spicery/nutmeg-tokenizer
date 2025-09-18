package tokeniser

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
	Label    []LabelRule    `yaml:"label"`
	Compound []CompoundRule `yaml:"compound"`
	Wildcard []WildcardRule `yaml:"wildcard"`
	Operator []OperatorRule `yaml:"operator"`
}

// BracketRule represents a bracket token rule
type BracketRule struct {
	Text     string   `yaml:"text"`
	ClosedBy []string `yaml:"closed_by"`
	Infix    bool     `yaml:"infix"`
	Prefix   bool     `yaml:"prefix"`
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
}

// LabelRule represents a label token rule
type LabelRule struct {
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

// TokeniserRules holds all the rule maps that can be customized
type TokeniserRules struct {
	StartTokens         map[string]StartTokenData
	LabelTokens         map[string]LabelTokenData
	CompoundTokens      map[string]CompoundTokenData
	PrefixTokens        map[string]bool
	DelimiterMappings   map[string][]string
	DelimiterProperties map[string][2]bool
	WildcardTokens      map[string]bool
	OperatorPrecedences map[string][3]int // [prefix, infix, postfix]
}

// DefaultRules returns the default tokeniser rules
func DefaultRules() *TokeniserRules {
	return &TokeniserRules{
		StartTokens:         getDefaultStartTokens(),
		LabelTokens:         getDefaultLabelTokens(),
		CompoundTokens:      getDefaultCompoundTokens(),
		PrefixTokens:        getDefaultPrefixTokens(),
		DelimiterMappings:   getDefaultDelimiterMappings(),
		DelimiterProperties: getDefaultDelimiterProperties(),
		WildcardTokens:      getDefaultWildcardTokens(),
		OperatorPrecedences: make(map[string][3]int),
	}
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

// ApplyRulesToDefaults applies the rules from a RulesFile to create a new TokeniserRules
func ApplyRulesToDefaults(rules *RulesFile) *TokeniserRules {
	tokeniserRules := DefaultRules()

	// Apply bracket rules
	if len(rules.Bracket) > 0 {
		tokeniserRules.DelimiterMappings = make(map[string][]string)
		tokeniserRules.DelimiterProperties = make(map[string][2]bool)

		for _, rule := range rules.Bracket {
			tokeniserRules.DelimiterMappings[rule.Text] = rule.ClosedBy
			tokeniserRules.DelimiterProperties[rule.Text] = [2]bool{rule.Infix, rule.Prefix}
		}
	}

	// Apply prefix rules
	if len(rules.Prefix) > 0 {
		tokeniserRules.PrefixTokens = make(map[string]bool)
		for _, rule := range rules.Prefix {
			tokeniserRules.PrefixTokens[rule.Text] = true
		}
	}

	// Apply start rules
	if len(rules.Start) > 0 {
		tokeniserRules.StartTokens = make(map[string]StartTokenData)
		for _, rule := range rules.Start {
			tokeniserRules.StartTokens[rule.Text] = StartTokenData{
				Expecting: rule.Expecting,
				ClosedBy:  rule.ClosedBy,
			}
		}
	}

	// Apply label rules
	if len(rules.Label) > 0 {
		tokeniserRules.LabelTokens = make(map[string]LabelTokenData)
		for _, rule := range rules.Label {
			tokeniserRules.LabelTokens[rule.Text] = LabelTokenData{
				Expecting: rule.Expecting,
				In:        rule.In,
			}
		}
	}

	// Apply compound rules
	if len(rules.Compound) > 0 {
		tokeniserRules.CompoundTokens = make(map[string]CompoundTokenData)
		for _, rule := range rules.Compound {
			tokeniserRules.CompoundTokens[rule.Text] = CompoundTokenData{
				Expecting: rule.Expecting,
				In:        rule.In,
			}
		}
	}

	// Apply wildcard rules
	if len(rules.Wildcard) > 0 {
		tokeniserRules.WildcardTokens = make(map[string]bool)
		for _, rule := range rules.Wildcard {
			tokeniserRules.WildcardTokens[rule.Text] = true
		}
	}

	// Apply operator rules
	if len(rules.Operator) > 0 {
		for _, rule := range rules.Operator {
			tokeniserRules.OperatorPrecedences[rule.Text] = rule.Precedence
		}
	}

	return tokeniserRules
}

// Helper functions to get default values (these will copy from the existing global variables)
func getDefaultStartTokens() map[string]StartTokenData {
	return map[string]StartTokenData{
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
		"class": {
			Expecting: []string{},
			ClosedBy:  []string{"end", "endclass"},
		},
		"for": {
			Expecting: []string{"do"},
			ClosedBy:  []string{"end", "endfor"},
		},
		"try": {
			Expecting: []string{"catch"},
			ClosedBy:  []string{"end", "endtry"},
		},
		"transaction": {
			Expecting: []string{},
			ClosedBy:  []string{"end", "endtransaction"},
		},
	}
}

func getDefaultLabelTokens() map[string]LabelTokenData {
	return map[string]LabelTokenData{
		"=>>": {
			Expecting: []string{"do"},
			In:        []string{"def"},
		},
		"do": {
			Expecting: []string{},
			In:        []string{"def", "for"},
		},
		"then": {
			Expecting: []string{},
			In:        []string{"if", "ifnot"},
		},
		"else": {
			Expecting: []string{},
			In:        []string{"if", "ifnot"},
		},
		"catch": {
			Expecting: []string{},
			In:        []string{"try"},
		},
	}
}

func getDefaultCompoundTokens() map[string]CompoundTokenData {
	return map[string]CompoundTokenData{
		"elseif": {
			Expecting: []string{"then"},
			In:        []string{"if"},
		},
		"elseifnot": {
			Expecting: []string{"then"},
			In:        []string{"if"},
		},
	}
}

func getDefaultPrefixTokens() map[string]bool {
	return map[string]bool{
		"return": true,
		"yield":  true,
	}
}

func getDefaultDelimiterMappings() map[string][]string {
	return map[string][]string{
		"(": {")"},
		"[": {"]"},
		"{": {"}"},
	}
}

func getDefaultDelimiterProperties() map[string][2]bool {
	return map[string][2]bool{
		"(": {true, true},  // infix=true, prefix=true
		"[": {true, false}, // infix=true, prefix=false
		"{": {true, true},  // infix=false, prefix=true
	}
}

func getDefaultWildcardTokens() map[string]bool {
	return map[string]bool{
		":": true,
	}
}
