# Nutmeg Tokenizer

A standalone tokenizer for the Nutmeg programming language, implemented in Go.

## Features

- Tokenizes Nutmeg source code into JSON format
- Supports all Nutmeg token types including:
  - Numeric literals with multiple radixes (2-36)
  - String literals with escape sequences
  - Identifiers and keywords
  - Operators with precedence information
  - Delimiters with usage context
- Command-line interface for file processing or stdin input
- Comprehensive test suite

## Installation

```bash
go build -o nutmeg-tokenizer ./cmd/nutmeg-tokenizer
```

## Usage

### Command Line

```bash
# Tokenize a file
./nutmeg-tokenizer examples/simple.nutmeg

# Read from stdin
echo "def hello end" | ./nutmeg-tokenizer -

# Show help
./nutmeg-tokenizer --help
```

### As a Library

```go
package main

import (
    "fmt"
    "github.com/spicery/nutmeg-tokenizer/pkg/tokenizer"
)

func main() {
    t := tokenizer.New("def hello(name) name end")
    tokens, err := t.Tokenize()
    if err != nil {
        panic(err)
    }

    for _, token := range tokens {
        fmt.Printf("%s: %s\n", token.Type, token.Text)
    }
}
```

## Token Types

- `n` - Numeric literals
- `s` - String literals
- `S` - Start tokens (def, if, while)
- `E` - End tokens (end, endif, endwhile)
- `C` - Compound tokens
- `L` - Label tokens
- `P` - Prefix tokens
- `V` - Variable tokens
- `O` - Operator tokens
- `[` - Open delimiters
- `]` - Close delimiters
- `U` - Unclassified tokens

## Output Format

Each token is output as a JSON object with the following structure:

```json
{
  "text": "def",
  "span": [1, 1, 1, 4],
  "type": "S",
  "closed_by": ["end"]
}
```

## Testing

```bash
go test ./pkg/tokenizer
```

## Examples

See the `examples/` directory for sample Nutmeg code that demonstrates various token types.