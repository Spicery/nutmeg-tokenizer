# The Wildcard Label `:`

The token `:` has a special role. When it is encountered, the tokenizer asks
what label X was expected in this context. The attributes of X are then copied
onto this token!

For example, in this sequence:
```
if x:
   ecetera
endif
```
The `if` keyword establishes the expectation of a following `then`. So when the
`:` token is encountered the tokenizer emits this:

```json
{"text":":","span":[1,1,1,2],"type":"L","expecting":["else","elseif","elseifnot","catch"],"in":["try","if"],"value":"then"}
```

Note that the text and span info preserved, the attributes of `then` are copied
over and a new attribute `value` has been added. The value attribute dictates
the name of the form-part to the parser.

## Unhappy path

What happens when a wildcard-label is used outside of a context which establishes
the `expecting`? It falls-back to being unclassified.


