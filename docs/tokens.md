# Token JSON Format

This document describes the JSON format for tokens produced by the Nutmeg
tokenizer. It finishes with a JSON schema describing the format of a single
token. 


## Token Types

The tokenizer produces tokens with the following type codes:

- `n` - Numeric literals with radix support
- `s` - String literals with quotes and escapes
- `S` - Start tokens (form start tokens like `def`, `if`, `while`)
- `E` - End tokens (form end tokens like `end`, `endif`, `endwhile`)
- `B` - Bridge tokens (multi-part constructs)
- `P` - Prefix tokens (prefix operators like `return`, `yield`)
- `V` - Variable tokens (variable identifiers)
- `O` - Operator tokens (infix/postfix operators)
- `[` - Open delimiter tokens (opening brackets/braces/parentheses)
- `]` - Close delimiter tokens (closing brackets/braces/parentheses)
- `U` - Unclassified tokens
- `X` - Exception tokens (for invalid constructs)

## Common Fields

All tokens have these required fields:

```json
{
  "text": "string",     // The original text of the token
  "span": [1, 5, 1, 8], // [start_line, start_col, end_line, end_col]
  "type": "n"           // Token type code
}
```

### Span Format

The `span` field is serialized as a 4-element array `[start_line, start_col, end_line, end_col]` representing the token's position in the source file. Line and column numbers are 1-based.

## Token-Specific Fields

### String Tokens (`s`)

```json
{
  "text": "\"hello\"",
  "span": [1, 1, 1, 7],
  "type": "s",
  "value": "hello"      // Interpreted string value (unescaped)
}
```

### Numeric Tokens (`n`)

```json
{
  "text": "0x1A.5",
  "span": [1, 1, 1, 6],
  "type": "n",
  "radix": "0x",        // Textual radix prefix ("0x", "2r", "0t", "" for decimal)
  "base": 16,           // Numeric base (2-36)
  "mantissa": "1A",     // Mantissa part
  "fraction": "5",      // Fraction part (optional)
  "exponent": 3,        // Exponent part (optional, decimal integer)
  "balanced": true      // For balanced ternary numbers (optional)
}
```

### Start Tokens (`S`)

```json
{
  "text": "def",
  "span": [1, 1, 1, 3],
  "type": "S",
  "expecting": ["identifier"], // Immediate next expected tokens
  "closed_by": ["end"]         // Tokens that can close this start token
}
```

### Bridge Tokens (`B`)

```json
{
  "text": "else",
  "span": [1, 1, 1, 4],
  "type": "B",
  "expecting": ["then"],    // What tokens can follow this label
  "in": ["if", "unless"],   // What start tokens can contain this label
  "single": false
}
```

### Operator Tokens (`O`)

```json
{
  "text": "+",
  "span": [1, 1, 1, 1],
  "type": "O",
  "precedence": [0, 50, 0]  // [prefix, infix, postfix] precedence values
}
```

The `precedence` field is only included if at least one precedence value is non-zero.

### Open Delimiter Tokens (`[`)

```json
{
  "text": "(",
  "span": [1, 1, 1, 1],
  "type": "[",
  "closed_by": [")"],       // Corresponding closing delimiter
  "infix": false,           // Can be used as infix operator
  "prefix": true            // Can be used as prefix operator
}
```

### Exception Tokens (`X`)

```json
{
  "text": "0x",
  "span": [1, 1, 1, 2],
  "type": "X",
  "reason": "invalid number literal"  // Explanation of the error
}
```

### Wildcard Tokens

Wildcard tokens (labels, starts, ends) may include a `value` field containing the expected token text they represent:

```json
{
  "text": "*",
  "span": [1, 1, 1, 1],
  "type": "L",
  "value": "else",          // The expected token this wildcard represents
  "expecting": ["then"],
  "in": ["if"]
}
```

### Newline Tracking (Optional)

Some tokens may include newline tracking fields:

```json
{
  "text": "def",
  "span": [1, 1, 1, 3],
  "type": "S",
  "ln_before": true,        // Token was preceded by a newline
  "ln_after": false         // Token was followed by a newline
}
```

## Output Format

Each token is output as a single JSON object on its own line (JSONL format), not as a JSON array.

## JSON Schema

The following JSON schema defines the structure of all tokens:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["text", "span", "type"],
  "properties": {
    "text": {
      "type": "string",
      "description": "The original text of the token"
    },
    "span": {
      "type": "array",
      "items": { "type": "integer" },
      "minItems": 4,
      "maxItems": 4,
      "description": "Position as [start_line, start_col, end_line, end_col]"
    },
    "type": {
      "type": "string",
      "enum": ["n", "s", "S", "E", "C", "L", "P", "V", "O", "[", "]", "U", "X"],
      "description": "Token type code"
    },
    "value": {
      "type": "string",
      "description": "Interpreted string value (for string literals) or expected token text (for wildcards)"
    },
    "radix": {
      "type": "string",
      "description": "Textual radix prefix for numeric literals (e.g., '0x', '2r', '0t', '' for decimal)"
    },
    "base": {
      "type": "integer",
      "minimum": 2,
      "maximum": 36,
      "description": "Numeric base for numeric literals"
    },
    "mantissa": {
      "type": "string",
      "description": "Mantissa part of numeric literals"
    },
    "fraction": {
      "type": "string",
      "description": "Fraction part of numeric literals"
    },
    "exponent": {
      "type": "integer",
      "description": "Exponent part of numeric literals as a decimal integer"
    },
    "balanced": {
      "type": "boolean",
      "description": "True for balanced ternary numbers"
    },
    "expecting": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Expected next tokens (for start tokens) or tokens that can follow (for label/compound tokens)"
    },
    "in": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Start tokens that can contain this label or compound token"
    },
    "closed_by": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Tokens that can close this start token or delimiter"
    },
    "single": {
      "type": "boolean",
      "description": "True if token is followed by only a single expression"
    }
    "precedence": {
      "type": "array",
      "items": { "type": "integer", "minimum": 0 },
      "minItems": 3,
      "maxItems": 3,
      "description": "Precedence values as [prefix, infix, postfix]"
    },
    "infix": {
      "type": "boolean",
      "description": "True if delimiter can be used as infix operator"
    },
    "prefix": {
      "type": "boolean",
      "description": "True if delimiter can be used as prefix operator"
    },
    "reason": {
      "type": "string",
      "description": "Error explanation for exception tokens"
    },
    "ln_before": {
      "type": "boolean",
      "description": "True if token was preceded by a newline"
    },
    "ln_after": {
      "type": "boolean",
      "description": "True if token was followed by a newline"
    }
  },
  "additionalProperties": false
}
```
