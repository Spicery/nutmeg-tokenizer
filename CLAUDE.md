# Nutmeg Tokeniser - a standalone tokeniser for the Nutmeg project

## Tokens

We are collaborating on the development of a standalone tokeniser for the Nutmeg
programming language, implemented in the Go programming language. Given a source
code file, the tokeniser outputs a list of tokens in JSON format. It will handle
the following token types:

- Literal constants:
    - n - Numeric literals which have a complex regex supporting radixes 2-36, decimal points, and scientific notation
    - s - String literals enclosed in single, double or back quotes, supporting escape sequences and interpolation
- Identifier tokens matching the regex `[a-zA-Z_][a-zA-Z0-9_]*`
    -   S - Start token (form start, e.g., def, if, for)
    -   E - End token (form end, e.g., end, endif, endfor)
    -   C - Compound token (multi-part constructs e.g. `elseif`)
    -   L - Label token (identifiers used as labels e.g. `then`, `else`)
    -   P - Prefix token (operators that come before their operand) e.g. `return`, `yield`
    -   V - Variable token (identifiers used as variables)
-   O - Operator token (infix, postfix operators) matching the regex `[\*\/\%\+\-\<\>\~\!\&\^\|\?\:\=]+`
-   [ - Open delimiter i.e. bracket/brace/parenthesis matching the regex `[\(\[\{]`
-   ] - Close delimiter i.e. bracket/brace/parenthesis matching the regex `[\)\]\}]`
-   U - Unclassified

The tokeniser will ignore whitespace and comments, end of line comments starting
with `###`.

##  Output Format

Each token will be represented as a JSON object on a single line, with the following fields:

- All tokens will have:
    - `text`: The original text of the token
    - `span`: The start and one-past the end positions of the token in the source file as an array [line,col,line,col]
    - `type`: The token type (one of the above types)

- String tokens will also have:
    - `value`: The interpreted string value for string literals

- Numeric tokens will also have:
    - `radix`: The numeric base for numeric literals (e.g., 10 for decimal, 16 for hexadecimal) from 2-36
    - `mantissa`: The mantissa part of numeric literals as a string
    - `fraction`: The fraction part of numeric literals as a string (if any)
    - `exponent`: The exponent part of numeric literals as a string (if any) as a decimal integer

- Start tokens will also have:
    - `closed_by`: An array of token texts that can close this start token (e.g., `def` is closed by `end`)

- Operator tokens will also have:
    - prefix: a natural number indicating the prefix-precedence of the operator (0 if not a prefix operator)
    - infix: a natural number indicating the infix-precedence of the operator (0 if not an infix operator)
    - postfix: a natural number indicating the postfix-precedence of the operator (0 if not

- `[` delimiter (bracket/brace/parenthesis) tokens will also have:
    - `closed_by`: The corresponding possible closing delimiter token text (e.g., `(` is closed by [`)`])
    - `infix`: A boolean indicating if the delimiter can be used as an infix operator (e.g., `[` can be used as an infix operator for array indexing)
    - `prefix`: A boolean indicating if the delimiter can be used as a prefix operator (e.g., `(` can be used as a prefix operator for grouping expressions)


## Programming Guidelines

- Comments should be proper sentences, with correct grammar and punctuation,
  including the use of capitalization and periods.

- Where defensive checks are added, include a comment explaining why they are
  appropriate (not necessary, since defensive checks are not necessary).


## Collaboration Guidelines

When providing technical assistance:

- **Be objective and critical**: Focus on technical correctness over agreeability
- **Challenge assumptions**: If code has clear technical flaws, point them out directly
- **Prioritize correctness**: Don't compromise on proper implementation to avoid disagreement
- **Think through implications**: Consider how users will actually use features in practice
- **Be direct about problems**: If something is wrong or will cause user confusion, say so clearly

The goal is to build robust, well-designed software, not to avoid technical disagreements.