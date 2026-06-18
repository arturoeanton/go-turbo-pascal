# Getting started

A short guide to using go-turbo-pascal: running Pascal programs, embedding Pascal
in Go, and editing/debugging `.pas` from the editor.

## Requirements

- Go 1.23+.

## Building the tools

```bash
make tools        # generates bin/pasrun, bin/pls, bin/pdap
# or, individually:
go build -o bin/pasrun ./cmd/pasrun
```

## 1) Running a Pascal program

```bash
go run ./cmd/pasrun examples/pascal/factorial.pas
```

Interactive programs (read from stdin):

```bash
echo "12 5" | go run ./cmd/pasrun examples/pascal/calc.pas
```

`pasrun` uses the real compiler (`internal/codegen`) and the VM (`internal/ir`).

## 2) Embedding Pascal in Go (`pkg/vmpas`)

```go
package main

import (
    "fmt"
    "github.com/arturoeanton/go-turbo-pascal/pkg/vmpas"
)

func main() {
    eng := vmpas.New() // restricted sandbox
    total := 10
    eng.Var("total", &total)
    eng.Run(`for i := 1 to 5 do total := total + i`)
    fmt.Println(total) // 25
}
```

More in [`vmpas.md`](vmpas.md): mapping structs ↔ records, calling Go functions
from Pascal and the capability sandbox. Complete example: `examples/embed`.

## 3) Editor: diagnostics and debugging

Build `pls` (language server) and `pdap` (debug adapter) and put them on the PATH:

```bash
make tools && export PATH="$PWD/bin:$PATH"
```

- **VSCode**: install the `editors/vscode` extension (F5 in dev mode). You will get
  highlighting, live diagnostics and debugging (breakpoints, step, variables).
- **Zed**: install the `editors/zed` extension (Install Dev Extension) for
  highlighting and diagnostics.

Details in [`editors.md`](editors.md).

## Supported language

See the [compatibility matrix](compatibility.md). The procedural +
OOP + I/O + units core is supported; some TP7 features are still
pending (`with`, typed files, etc.).

## Architecture

See [`architecture.md`](architecture.md) for the
lexer → parser → codegen → VM pipeline and how `pkg/vmpas` fits in.
