package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/spicery/nutmeg-tokeniser/pkg/tokeniser"
)

const (
	version = "0.1.0"
	usage   = `nutmeg-tokeniser - A tokeniser for the Nutmeg programming language

Usage:
  nutmeg-tokeniser [options]

Options:
  -h, --help            Show this help message
  -v, --version         Show version information
  --input <file>        Input file (defaults to stdin)
  --output <file>       Output file (defaults to stdout)
  --rules <file>        YAML rules file for custom tokenisation rules (optional)
  --exit0               Exit with code 0 even on tokenisation errors (suppress stderr)

Examples:
  nutmeg-tokeniser                                    # Read from stdin, write to stdout
  nutmeg-tokeniser --input source.nutmeg             # Read from file, write to stdout
  nutmeg-tokeniser --output tokens.json              # Read from stdin, write to file
  nutmeg-tokeniser --input source.nutmeg --output tokens.json  # Read from file, write to file
  nutmeg-tokeniser --rules custom.yaml --input source.nutmeg   # Use custom rules
  echo "def foo end" | nutmeg-tokeniser              # Read from stdin, write to stdout

The tokeniser outputs one JSON token object per line.
See docs/rules_file.md for information about custom rules files.
`
)

func main() {
	var showHelp, showVersion, exit0 bool
	var inputFile, outputFile, rulesFile string

	flag.BoolVar(&showHelp, "h", false, "Show help")
	flag.BoolVar(&showHelp, "help", false, "Show help")
	flag.BoolVar(&showVersion, "v", false, "Show version")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&exit0, "exit0", false, "Exit with code 0 even on errors")
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
		fmt.Printf("nutmeg-tokeniser version %s\n", version)
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
	var t *tokeniser.Tokeniser
	if rulesFile != "" {
		rules, err := tokeniser.LoadRulesFile(rulesFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading rules file '%s': %v\n", rulesFile, err)
			os.Exit(1)
		}

		tokeniserRules, err := tokeniser.ApplyRulesToDefaults(rules)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error applying rules: %v\n", err)
			os.Exit(1)
		}
		t = tokeniser.NewWithRules(input, tokeniserRules)
	} else {
		t = tokeniser.New(input)
	}

	// Process input
	tokens, tokeniseErr := t.Tokenise()

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
	if tokeniseErr != nil {
		if exit0 {
			// With --exit0, exit normally despite error
			os.Exit(0)
		} else {
			// Without --exit0, print error to stderr and exit with error code
			fmt.Fprintf(os.Stderr, "Tokenisation error: %v\n", tokeniseErr)
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
