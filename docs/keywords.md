# Keywords in Nutmeg

## Surround Forms (S and E)

Start tokens:

- def, expecting=[`=>>`], closed_by=[`end`, `enddef`], single=false
- if, expecting=[`then`], closed_by=[`end`, `endif`], single=true
- ifnot, expecting=[`then`], closed_by=[`end`, `endifnot`], single=true
- fn, expecting=[], closed_by=[`end`, `endfn`], single=true
- for, expecting=[`do`], closed_by=[`end`, `endfor`], single=true
- class, expecting=[], closed_by=[`end`, `endclass`], single=true
- interface, expecting=[], closed_by=[`end`, `endinterface`], single=true
- try, expecting=[`catch`, `else`], closed_by=[`end`, `endtry`], single=false
- transaction expecting=[`else`], closed_by=[`end`, `endtransaction`], single=false

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


