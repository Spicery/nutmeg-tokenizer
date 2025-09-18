# Keywords in Nutmeg

## Surround Forms (S and E)

Start tokens:

- def 
- if
- ifnot
- fn
- for
- class
- interface
- try
- transaction

The end tokens are always `end` or `end{START}`

## Simple Labels (L)

- then, expecting=[`else`, `elseif`, `elseifnot`, `catch`], in=[`try`, `if`]
- else, expecting=[], in=[`if`, `try`]

## Compound Labels (C)

- catch, expecting=[`then`, `:`], in=[`try`]
- elseif, expecting=[`then`, `:`], in=[`if`]
- elseifnot, expecting=[`then`, `:`], in=[`if`]

## Prefix Forms (P)

- return
- yield


