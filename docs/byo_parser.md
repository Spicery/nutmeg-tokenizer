# Bring Your Own Parser

## Motivation

- The first challenge in learning a programing language is the surface syntax.
- Often distinctive and quirky e.g. E1 ? E2 : E3
- Although there is a relationship between the language and its syntax, it is often very weak, so it is frustrating not to be able to replace it.
- We resolved to make it possible for any programmer to completely replace the usual syntax ("common syntax") by bringing their own parser.

## Requirements

- The internal abstract syntax tree has to be easy to understand, easy to create and easy to render
- UNIX filter: Writeable in any language/toolkit and invoked by a command
- AST in XML (Or JSON post-2010)



## Example

❯ echo 'x + 1' | nutmeg parse

```json
{
    "arguments": {
        "body": [
            {
                "kind": "id",
                "name": "x",
                "reftype": "get"
            },
            {
                "kind": "int",
                "value": "1"
            }
        ],
        "kind": "seq"
    },
    "kind": "syscall",
    "name": "+"
}
```

