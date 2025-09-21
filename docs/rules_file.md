# Rule Files

A rules-file is a YAML file that provides overriding rules for tokenisation.
The rules are in the following categories:

- bracket
- prefix
- start
- label
- compound
- wildcard
- operator

## Key ideas

- Token boundaries are baked into the algorithm but the classification is
  decided by matching against the rules. The boundaries are:
    - strings and numbers are specially recognised
    - alphanumerics + underbars bind together
    - sign characters bind together
    - everything else is a single character

- The end-tokens `E` category is implicitly defined by the `closed_by` field of the
  start tokens.

- The close-tokens `]` category is implicitly defined `closed_by` field of the
  open-tokens `[`.

- Categories can be included or omitted. Where a category is included it is
  the only way that a token can be put in that category.

- If a token is recognised by any of the rules, this takes precedence over
  the default rules.


## Bracket rules

Example:

```yaml
bracket:
  - text: "("
    closed_by: 
      - ")"
    infix: true
    prefix: true
  - text: "["
    closed_by: 
      - "]"
    infix: true
    prefix: true
  - text: "{"
    closed_by: 
      - "}"
    infix: true
    prefix: true
```

## Prefix-Form rules

Example:
```yaml
prefix:
  - text: return
  - text: yield
```

## Start rules

Example
```yaml
start:
  - text: "if"
    closed_by:
      - end
      - endif
    expecting: 
      - then
    single: true
  - text: "def"
    closed_by:
      - end
      - enddef
    expecting:
      - "=>>"
    single: true
```

## Label rules

Example:
```yaml
label:
  - text: "=>>"
    expecting:
      - do
    in:
      - def
  - text: "do"
    expecting: []
    in: 
      - def
      - for
```

## Compound-Label rules

Example:
```yaml
compound:
  - text: elseif
    expecting:
      - then
    in:
      - if
  - text: elseifnot
    expecting:
      - then
    in:
      - if
```

## Wildcard-Label rules

```yaml
wildcard:
  - text: ":"
```

## Operator rules

Example:

```yaml
operator:
  - text: "+"
    precedence: [0,150,0]
  - text: "*"
    precedence: [0, 100, 0]
```
