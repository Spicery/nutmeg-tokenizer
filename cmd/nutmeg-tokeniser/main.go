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
  nutmeg-tokeniser [options] [file]

Options:
  -h, --help     Show this help message
  -v, --version  Show version information
  -             Read from stdin instead of a file

Examples:
  nutmeg-tokeniser source.nutmeg    # Tokenise a file
  nutmeg-tokeniser -                # Read from stdin
  echo "def foo end" | nutmeg-tokeniser -

The tokeniser outputs one JSON token object per line.
`
)

func main() {
	var showHelp, showVersion bool
	flag.BoolVar(&showHelp, "h", false, "Show help")
	flag.BoolVar(&showHelp, "help", false, "Show help")
	flag.BoolVar(&showVersion, "v", false, "Show version")
	flag.BoolVar(&showVersion, "version", false, "Show version")

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

	var input string
	var err error

	args := flag.Args()

	if len(args) == 0 || (len(args) == 1 && args[0] == "-") {
		// Read from stdin
		input, err = readFromStdin()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
			os.Exit(1)
		}
	} else if len(args) == 1 {
		// Read from file
		input, err = readFromFile(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file '%s': %v\n", args[0], err)
			os.Exit(1)
		}
	} else {
		fmt.Fprint(os.Stderr, "Error: Too many arguments\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Create tokeniser and process input
	t := tokeniser.New(input)
	tokens, err := t.Tokenise()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Tokenisation error: %v\n", err)
		os.Exit(1)
	}

	// Output tokens as JSON, one per line
	for _, token := range tokens {
		jsonBytes, err := json.Marshal(token)
		if err != nil {
			fmt.Fprintf(os.Stderr, "JSON encoding error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonBytes))
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