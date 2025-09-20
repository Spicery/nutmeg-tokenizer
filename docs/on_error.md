# Error Handling in the Nutmeg Tokenizer

## Approach

The tokenizer will typically be used within a pipeline and simply crashing
the pipeline is not very user-friendly. The strategy is:

- When an error is found (e.g. invalid number token) an `X` (exception)
  token is generated.

- Processing is stopped.

- If `--exit0` is given as an option, we simply exit normally and do not
  print to stderr. Otherwise we exit with code 1 and print the error to stderr.

## X tokens

- X Exception token (used for tokens that should never appear in valid code, e.g. invalid number literals)
- X tokens will also have:
- `reason`: A string explaining why this token is classified as an exception (e.g., "invalid number literal")
