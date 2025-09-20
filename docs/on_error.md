# Error Handling in the Nutmeg Tokeniser

The tokeniser will typically be used within a pipeline and simply crashing
the pipeline is not very user-friendly. The strategy is:

- When an error is found (e.g. invalid number token) an `X` (exception)
  token is generated.

- Processing is stopped.

- If `--exit0` is given as an option, we simply exit normally and do not
  print to stderr. Otherwise we exit with code 1 and print the error to stderr.

