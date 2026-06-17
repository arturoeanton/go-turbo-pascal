# Inicio rápido

Guía breve para usar go-turbo-pascal: ejecutar programas Pascal, embeber Pascal
en Go, y editar/depurar `.pas` desde el editor.

## Requisitos

- Go 1.23+.

## Construir las herramientas

```bash
make tools        # genera bin/pasrun, bin/pls, bin/pdap
# o, individualmente:
go build -o bin/pasrun ./cmd/pasrun
```

## 1) Ejecutar un programa Pascal

```bash
go run ./cmd/pasrun examples/pascal/factorial.pas
```

Programas interactivos (leen de stdin):

```bash
echo "12 5" | go run ./cmd/pasrun examples/pascal/calc.pas
```

`pasrun` usa el compilador real (`internal/codegen`) y la VM (`internal/ir`).

## 2) Embeber Pascal en Go (`pkg/vmpas`)

```go
package main

import (
    "fmt"
    "github.com/arturoeanton/go-turbo-pascal/pkg/vmpas"
)

func main() {
    eng := vmpas.New() // sandbox restringido
    total := 10
    eng.Var("total", &total)
    eng.Run(`for i := 1 to 5 do total := total + i`)
    fmt.Println(total) // 25
}
```

Más en [`vmpas.md`](vmpas.md): mapeo de structs ↔ records, llamar funciones Go
desde Pascal y el sandbox de capacidades. Ejemplo completo: `examples/embed`.

## 3) Editor: diagnósticos y depuración

Construí `pls` (language server) y `pdap` (debug adapter) y ponelos en el PATH:

```bash
make tools && export PATH="$PWD/bin:$PATH"
```

- **VSCode**: instalá la extensión de `editors/vscode` (F5 en modo dev). Tendrás
  resaltado, diagnósticos en vivo y depuración (breakpoints, step, variables).
- **Zed**: instalá la extensión de `editors/zed` (Install Dev Extension) para
  resaltado y diagnósticos.

Detalles en [`editores.md`](editores.md).

## Lenguaje soportado

Ver la [matriz de compatibilidad](compatibilidad.md). El núcleo procedural +
OOP + I/O + units está soportado; algunas características de TP7 siguen
pendientes (con `with`, archivos tipados, etc.).

## Arquitectura

Ver [`arquitectura.md`](arquitectura.md) para el pipeline
lexer → parser → codegen → VM y cómo encaja `pkg/vmpas`.
