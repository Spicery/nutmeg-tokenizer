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

Token boundaries are baked into the algorithm but the classification is decided
by matching against the below rules.

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
  - text: "def"
    closed_by:
      - end
      - enddef
    expecting:
      - "=>>"
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
