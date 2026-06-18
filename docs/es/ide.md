# Herramientas de línea de comandos y depurador

go-turbo-pascal incluye estas herramientas orientadas al usuario:

- `bpgo`: el compilador / ejecutor de línea de comandos. Use `bpgo --help`.
- `tpc`: un envoltorio compatible con TP7 alrededor de `bpgo` que selecciona
  el mapa y la información de depuración de forma predeterminada.
- `turbo`: el IDE de estilo TP7. Admite operación interactiva y sin interfaz
  (headless) para flujos de trabajo de archivo/proyecto, compilación/ejecución
  y depuración.
- `tdebug`: un depurador de nivel de fuente para línea de comandos.
- `bprun`: un ejecutor para archivos `.bpi` emitidos por el pipeline.

## Uso de `bpgo`

```text
Usage: bpgo [options] file [args]

  -h, --help           show help
  -V, --version        show version
  -v, --verbose        verbose output
  -q, --quiet          suppress info output
  -M, --map            generate map file
  -d, --debug-info     include debug info
  -B, --build          build all (project)
  -R, --run            compile and run
  -U<dir>              add unit search path
  -I<dir>              add include search path
  -O<dir>              add object file search path
  -D<name>             define a symbol
  -m<stack,heap>       memory sizes
  -do<file>            write standalone EXE
  -o<file>             output file
  -E<file>             read configuration file
```

Subcomando: `bpgo test-compat` ejecuta el arnés de conformidad y
escribe `compat/report.json`.

## IDE `turbo`

El IDE proporciona:

- Un `Buffer` con `SetText`, `InsertString`, `Backspace`, `Delete`,
  `InsertNewline`, `MoveCursor`, `GotoLine`, `WordLeft/Right`,
  `Find`, `ReplaceAll`. Cada búfer rastrea `Dirty`, `Filename`,
  `CursorX/Y` y una selección `Block`.
- Un `Menu` con cinco elementos de nivel superior: `File`, `Edit`, `Run`, `Debug`
  y `Search` (el último es un grupo lógico). Cada elemento lista su
  campo `Command`; `IDE.RunCommand(name, args...)` ejecuta el
  comando.
- Un `Project` con `Source`, `Output`, `Units`, `UnitDirs`,
  `IncludeDirs` y `ObjectDirs`.
- Comandos implementados: `New`, `Open`, `OpenProject`, `ProjectInfo`,
  `Save`, `SaveAs`, `Compile`, `Run`, `Build`, `Find`, `Replace`,
  `GotoLine`, `Cut`, `Copy`, `Paste`, `Undo`, `Redo`,
  `SetBreakpoint`, `DebugStep`, `DebugContinue`, `Watch`, `Mouse`,
  `Exit`.

`turbo --headless` ejecuta el IDE sin un TTY. Úselo desde pruebas
y desde el arnés de conformidad.

El `turbo` interactivo arranca con un banner de escritorio compatible con TP7 y
acepta comandos con los nombres de las teclas familiares: `f9`, `ctrl-f9`, `f7`,
`f8`, `ctrl-f8`, además de `project FILE`, `open FILE`, `save`, `compile`,
`run`, `break N`, `watch EXPR` y `mouse X Y BUTTON DOWN`.

## Comandos de `tdebug`

```text
tdebug - BPGo source-level debugger

commands:
  break FILE LINE    set a breakpoint
  watch EXPR         add a watch expression
  step               single-step
  continue           resume execution
  snapshot           print the current debugger state
  exit               exit the debugger
```

Internamente, el depurador rastrea `Breakpoints`, `Watches`, `Stack`,
`StepCount` y `StepLimit` (usado para detectar programas descontrolados). El
arnés de conformidad ejercita una secuencia completa: establecer punto de
interrupción, step, continue, snapshot, exit.

## Uso de `bprun`

`bprun program.bpi [args...]` carga un archivo `.bpi` y lo ejecuta en la
VM embebida. El ejecutor conecta los builtins de la unidad System y
reenvía `ParamStr(0)` al nombre del programa.

## Pruebas de extremo a extremo y de integración

Los paquetes `internal/e2e` e `internal/integration` ejercitan el
pipeline completo más el IDE y la CLI. Se pueden invocar con
`go test ./internal/e2e/...` y
`go test ./internal/integration/...`.

El paquete `internal/conformance` implementa el arnés que
produce `compat/report.json`. Ejecútelo mediante
`go run ./cmd/bpgo test-compat`.
