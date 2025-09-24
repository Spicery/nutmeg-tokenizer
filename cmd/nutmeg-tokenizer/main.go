package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/spicery/nutmeg-tokenizer/pkg/tokenizer"
	"gopkg.in/yaml.v3"
)

const (
	version = "0.1.0"
	usage   = `nutmeg-tokenizer - A tokenizer for the Nutmeg programming language

Usage:
  nutmeg-tokenizer [options]

Options:
  -h, --help            Show this help message
  -v, --version         Show version information
  --input <file>        Input file (defaults to stdin)
  --output <file>       Output file (defaults to stdout)
  --rules <file>        YAML rules file for custom tokenisation rules (optional)
  --make-rules          Generate default rules YAML to stdout
  --exit0               Exit with code 0 even on tokenisation errors (suppress stderr)

Examples:
  nutmeg-tokenizer                                   # Read from stdin, write to stdout
  nutmeg-tokenizer --input source.nutmeg             # Read from file, write to stdout
  nutmeg-tokenizer --output tokens.json              # Read from stdin, write to file
  nutmeg-tokenizer --input source.nutmeg --output tokens.json  # Read from file, write to file
  nutmeg-tokenizer --rules custom.yaml --input source.nutmeg   # Use custom rules
  nutmeg-tokenizer --make-rules                      # Generate default rules configuration
  echo "def foo end" | nutmeg-tokenizer              # Read from stdin, write to stdout

The tokenizer outputs one JSON token object per line.
See docs/rules_file.md for information about custom rules files.
`
)

func main() {
	var showHelp, showVersion, exit0, makeRules bool
	var inputFile, outputFile, rulesFile string

	flag.BoolVar(&showHelp, "h", false, "Show help")
	flag.BoolVar(&showHelp, "help", false, "Show help")
	flag.BoolVar(&showVersion, "v", false, "Show version")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&exit0, "exit0", false, "Exit with code 0 even on errors")
	flag.BoolVar(&makeRules, "make-rules", false, "Generate default rules YAML")
	flag.StringVar(&inputFile, "input", "", "Input file (defaults to stdin)")
	flag.StringVar(&outputFile, "output", "", "Output file (defaults to stdout)")
	flag.StringVar(&rulesFile, "rules", "", "YAML rules file (optional)")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}

	flag.Parse()

	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if showVersion {
		fmt.Printf("nutmeg-tokenizer version %s\n", version)
		os.Exit(0)
	}

	if makeRules {
		err := generateDefaultConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating default rules: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Reject any positional arguments
	if len(flag.Args()) > 0 {
		fmt.Fprintf(os.Stderr, "Error: Unexpected positional arguments. Use --input and --output flags instead.\n\n")
		flag.Usage()
		os.Exit(1)
	}

	var input string
	var err error

	// Read input
	if inputFile == "" {
		// Read from stdin
		input, err = readFromStdin()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Read from file
		input, err = readFromFile(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file '%s': %v\n", inputFile, err)
			os.Exit(1)
		}
	}

	// Load rules if specified
	var t *tokenizer.Tokenizer
	if rulesFile != "" {
		rules, err := tokenizer.LoadRulesFile(rulesFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading rules file '%s': %v\n", rulesFile, err)
			os.Exit(1)
		}

		tokenizerRules, err := tokenizer.ApplyRulesToDefaults(rules)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error applying rules: %v\n", err)
			os.Exit(1)
		}
		t = tokenizer.NewTokenizerWithRules(input, tokenizerRules)
	} else {
		t = tokenizer.NewTokenizer(input)
	}

	// Process input
	tokens, tokenizeErr := t.Tokenize()

	// Prepare output destination
	var output io.Writer
	var outputCloser io.Closer

	if outputFile == "" {
		// Write to stdout
		output = os.Stdout
	} else {
		// Write to file
		file, err := os.Create(outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file '%s': %v\n", outputFile, err)
			os.Exit(1)
		}
		output = file
		outputCloser = file
	}

	// Output tokens as JSON, one per line (even if there was an error)
	for _, token := range tokens {
		jsonBytes, err := json.Marshal(token)
		if err != nil {
			fmt.Fprintf(os.Stderr, "JSON encoding error: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintln(output, string(jsonBytes))
	}

	// Close output file if we opened one
	if outputCloser != nil {
		if err := outputCloser.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error closing output file '%s': %v\n", outputFile, err)
			os.Exit(1)
		}
	}

	// Handle tokenisation error after outputting tokens
	if tokenizeErr != nil {
		if exit0 {
			// With --exit0, exit normally despite error
			os.Exit(0)
		} else {
			// Without --exit0, print error to stderr and exit with error code
			fmt.Fprintf(os.Stderr, "Tokenization error: %v\n", tokenizeErr)
			os.Exit(1)
		}
	}
}

// readFromStdin reads all input from stdin.
func readFromStdin() (string, error) {
	bytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// readFromFile reads the contents of a file.
func readFromFile(filename string) (string, error) {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// generateDefaultConfig outputs the default configuration in YAML format to stdout.
func generateDefaultConfig() error {
	rules := tokenizer.DefaultRules()

	// Convert TokenizerRules to RulesFile format
	rulesFile := &tokenizer.RulesFile{}

	// Convert bracket rules
	for text, closedBy := range rules.DelimiterMappings {
		props := rules.DelimiterProperties[text]
		rulesFile.Bracket = append(rulesFile.Bracket, tokenizer.BracketRule{
			Text:      text,
			ClosedBy:  closedBy,
			InfixPrec: props.InfixPrec,
			Prefix:    props.Prefix,
		})
	}

	// Convert prefix rules
	for text := range rules.PrefixTokens {
		rulesFile.Prefix = append(rulesFile.Prefix, tokenizer.PrefixRule{
			Text: text,
		})
	}

	// Convert start rules
	for text, data := range rules.StartTokens {
		rulesFile.Start = append(rulesFile.Start, tokenizer.StartRule{
			Text:      text,
			ClosedBy:  data.ClosedBy,
			Expecting: data.Expecting, // Include the expecting field as it exists in StartTokenData
		})
	}

	// Convert bridge rules
	for text, data := range rules.BridgeTokens {
		rulesFile.Bridge = append(rulesFile.Bridge, tokenizer.BridgeRule{
			Text:      text,
			Expecting: data.Expecting,
			In:        data.In,
		})
	}

	// Convert wildcard rules
	for text := range rules.WildcardTokens {
		rulesFile.Wildcard = append(rulesFile.Wildcard, tokenizer.WildcardRule{
			Text: text,
		})
	}

	// Convert operator rules
	for text, precedence := range rules.OperatorPrecedences {
		rulesFile.Operator = append(rulesFile.Operator, tokenizer.OperatorRule{
			Text:       text,
			Precedence: precedence,
		})
	}

	// Marshal to YAML and output to stdout
	yamlBytes, err := yaml.Marshal(rulesFile)
	if err != nil {
		return fmt.Errorf("failed to marshal rules to YAML: %w", err)
	}

	fmt.Print(string(yamlBytes))
	return nil
}
