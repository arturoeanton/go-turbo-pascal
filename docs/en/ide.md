# Command-line tools and debugger

go-turbo-pascal ships these user-facing commands:

- `bpgo`: the command-line compiler / runner. Use `bpgo --help`.
- `tpc`: a TP7-compatible wrapper around `bpgo` that selects map
  and debug-info by default.
- `turbo`: the TP7-style IDE. It supports interactive and headless
  operation for file/project, compile/run and debug workflows.
- `tdebug`: a command-line source-level debugger.
- `bprun`: a runner for `.bpi` files emitted by the pipeline.

## `bpgo` usage

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

Subcommand: `bpgo test-compat` runs the conformance harness and
writes `compat/report.json`.

## `turbo` IDE

The IDE provides:

- A `Buffer` with `SetText`, `InsertString`, `Backspace`, `Delete`,
  `InsertNewline`, `MoveCursor`, `GotoLine`, `WordLeft/Right`,
  `Find`, `ReplaceAll`. Each buffer tracks `Dirty`, `Filename`,
  `CursorX/Y` and a `Block` selection.
- A `Menu` with five top-level items: `File`, `Edit`, `Run`, `Debug`
  and `Search` (the last one is a logical group). Each item lists its
  `Command` field; `IDE.RunCommand(name, args...)` executes the
  command.
- A `Project` with `Source`, `Output`, `Units`, `UnitDirs`,
  `IncludeDirs` and `ObjectDirs`.
- Commands implemented: `New`, `Open`, `OpenProject`, `ProjectInfo`,
  `Save`, `SaveAs`, `Compile`, `Run`, `Build`, `Find`, `Replace`,
  `GotoLine`, `Cut`, `Copy`, `Paste`, `Undo`, `Redo`,
  `SetBreakpoint`, `DebugStep`, `DebugContinue`, `Watch`, `Mouse`,
  `Exit`.

`turbo --headless` runs the IDE without a TTY. Use it from tests
and the conformance harness.

Interactive `turbo` starts with a TP7-compatible desktop banner and
accepts commands named after familiar keys: `f9`, `ctrl-f9`, `f7`,
`f8`, `ctrl-f8`, plus `project FILE`, `open FILE`, `save`, `compile`,
`run`, `break N`, `watch EXPR` and `mouse X Y BUTTON DOWN`.

## `tdebug` commands

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

Internally the debugger tracks `Breakpoints`, `Watches`, `Stack`,
`StepCount` and `StepLimit` (used to detect runaway programs). The
conformance harness exercises a full sequence: set breakpoint, step,
continue, snapshot, exit.

## `bprun` usage

`bprun program.bpi [args...]` loads a `.bpi` file and runs it on the
embedded VM. The runner wires the System unit's builtins and
forwards `ParamStr(0)` to the program name.

## End-to-end and integration tests

The `internal/e2e` and `internal/integration` packages exercise the
full pipeline plus the IDE and CLI. They can be invoked with
`go test ./internal/e2e/...` and
`go test ./internal/integration/...`.

The `internal/conformance` package implements the harness that
produces `compat/report.json`. Run it via
`go run ./cmd/bpgo test-compat`.
