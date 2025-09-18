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

- then, which may occur after `if` or `elseif` or `ifnot` or `elseifnot` or `catch`
- else, which may occur after `then` or `try` or `catch`

## Compound Labels (C)

- catch, which may only appear after `try` or `then`
- elseif, which may only occur after `then`
- elseifnot, which may only occur after `then`

## Prefix Forms (P)

- return
- yield


