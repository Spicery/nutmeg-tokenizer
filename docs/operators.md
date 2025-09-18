# Operators in Nutmeg

Operators are a sequence of characters drawn from the following list, the
order corresponds to the precedence level, tightest first:

- `*`, precedence 10
- `/`, precedence 20
- `%`, precedence 30
- `+`, precedence 40
- `-`, precedence 50
- `<`, precedence 60
- `>`, precedence 70
- `~`, precedence 80
- `!`, precedence 90
- `&`, precedence 100
- `^`, precedence 110
- `|`, precedence 120
- `?`, precedence 130
- `=`, precedence 140
- `:`, precedence 150

The base precedence of an operator is determined by the precedence-value of the
first character (lower is tighter). If the first character is repeated then
subtract 1. e.g. the base precedence of `*` is 10 but `**` is 9, for example.

Role adjustments:
- If the operator is enabled in a prefix context then use the base precedence,
  otherwise 0.
- If the operator is enabled in a posfix context then add 1000 to the base
  precedence, otherwise 0.
- If the operator is used in an infix context then add 2000 to the base
  precedence.

By default the only operator that has a non-infix role enabled is `-`, which can be
used in prefix position as unary negation e.g. `-x`. However the necessity for
the separate prefix/infix/postfix precedences remains for when we generalise the
code to be configuration driven.

## Exceptions

- A single `:` is treated as a kind of wildcard simple-label.
- The token `=>>` is a simple-label.
