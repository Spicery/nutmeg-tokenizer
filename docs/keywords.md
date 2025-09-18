# Keywords in Nutmeg

## Surround Forms (S and E)

Start tokens:

- def, expecting=[`=>>`], closed_by=[`end`, `enddef`]
- if, expecting=[`then`], closed_by=[`end`, `endif`]
- ifnot, expecting=[`then`], closed_by=[`end`, `endifnot`]
- fn, expecting=[], closed_by=[`end`, `endfn`]
- for, expecting=[`do`], closed_by=[`end`, `endfor`]
- class, expecting=[], closed_by=[`end`, `endclass`]
- interface, expecting=[], closed_by=[`end`, `endinterface`]
- try, expecting=[`catch`, `else`], closed_by=[`end`, `endtry`]
- transaction expecting=[`else`], closed_by=[`end`, `endtransaction`]

The end tokens are always `end` or `end{START}`

## Simple Labels (L)

- `=>>`, expecting=[`do`]
- do, expecting=[], in=[`for`, `def`]
- then, expecting=[`else`, `elseif`, `elseifnot`, `catch`], in=[`try`, `if`]
- else, expecting=[], in=[`if`, `try`]

## Compound Labels (C)

- catch, expecting=[`then`], in=[`try`]
- elseif, expecting=[`then`], in=[`if`]
- elseifnot, expecting=[`then`], in=[`if`]

## Prefix Forms (P)

- return
- yield


