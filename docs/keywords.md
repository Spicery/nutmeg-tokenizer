# Keywords in Nutmeg

## Surround Forms (S and E)

Start tokens:

- def, expecting=[`=>>`], closed_by=[`end`, `enddef`], arity=1
- if, expecting=[`then`], closed_by=[`end`, `endif`], arity=1
- ifnot, expecting=[`then`], closed_by=[`end`, `endifnot`], arity=1
- fn, expecting=[], closed_by=[`end`, `endfn`], arity=1
- for, expecting=[`do`], closed_by=[`end`, `endfor`], arity=1
- class, expecting=[], closed_by=[`end`, `endclass`], arity=1
- interface, expecting=[], closed_by=[`end`, `endinterface`], arity=1
- try, expecting=[`catch`, `else`], closed_by=[`end`, `endtry`], arity=2
- transaction expecting=[`else`], closed_by=[`end`, `endtransaction`], arity=2

The end tokens are always `end` or `end{START}`

## Bridges (B)

## single=false

- `=>>`, expecting=[`do`]
- do, expecting=[], in=[`for`, `def`]
- then, expecting=[`else`, `elseif`, `elseifnot`, `catch`], in=[`try`, `if`]
- else, expecting=[], in=[`if`, `try`]

## single=true

- catch, expecting=[`then`], in=[`try`]
- elseif, expecting=[`then`], in=[`if`]
- elseifnot, expecting=[`then`], in=[`if`]

## Prefix Forms (P)

- return
- yield

## Marks (M)

- `,` and `;`
