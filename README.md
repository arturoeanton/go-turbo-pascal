# BPGo — Turbo Pascal 7 / Borland Pascal 7 compiler in Go

BPGo is a clean-room Go implementation of a Turbo Pascal 7 / Borland
Pascal 7 compatible compiler, runtime, IDE and debugger. Its two goals
are to **embed Pascal inside Go** (the `pkg/vmpas` library) and to bring
an **authentic Turbo Pascal 7 experience to the Linux/macOS terminal**.

BPGo is free software: it does not embed any Borland binary, source
or documentation. The compatibility manifests in `compat/spec/`
describe the target symbols and their status.

> **Project status (in active reconstruction).** The front-end
> (lexer + parser + AST) is solid. The compiler back-end (codegen + VM),
> the embeddable `pkg/vmpas` API, and the TP7 IDE are being rebuilt to a
> professional, fully-tested standard following the roadmap below. Some
> areas described here are still partial — see **Status** and the roadmap
> for what actually works today versus what is planned.

## Quickstart

Run a real Pascal program on the new engine:

```bash
go run ./cmd/pasrun examples/pascal/factorial.pas
go run ./cmd/pasrun examples/pascal/listas.pas
```

Embed Pascal inside Go (see `examples/embed/main.go`):

```bash
go run ./examples/embed
```

```go
eng := vmpas.New()
total := 10
eng.Var("total", &total)
eng.Run(`for i := 1 to 5 do total := total + i`)
// total == 25
```

Documentación / Documentation: **🇪🇸 [español](docs/es/README.md)** · **🇬🇧 [English](docs/en/README.md)** · [Ejemplos / Examples](examples/README.md).

Atajos (ES): [Quickstart](docs/es/quickstart.md) ·
[vmpas (embeber Pascal)](docs/es/vmpas.md) ·
[Ejecución durable (snapshot/resume)](docs/es/durable.md) ·
[match / Option](docs/es/match.md) ·
[spawn / channels](docs/es/concurrency.md) ·
[Compatibilidad TP7](docs/es/compatibility.md).

## Build and test

```bash
go build ./...   # build everything
go test ./...    # run the full test suite
```

## Commands

| Command  | Description |
|----------|-------------|
| `bpgo`   | main CLI: compile & run on the real engine (`bpgo --run x.pas`), plus `test-compat` |
| `pasrun` | minimal "compile & run a `.pas`" driver (same engine) |
| `pls`    | Pascal language server (LSP) — diagnostics for editors |
| `pdap`   | Pascal debug adapter (DAP) — breakpoints/step for editors |
| `tpc`    | a TP7-compatible wrapper around `bpgo` |
| `tdebug` | the source-level debugger CLI |
| `turbo`  | the legacy TP7-style IDE shell (the tcell IDE is a future phase) |

For embedding Pascal in Go, use the `pkg/vmpas` library (no external deps).

## Architecture

```
testdata/pas/     corpus of Pascal programs that exercise the pipeline
compat/spec/      unit / directive / diagnostic manifests
compat/report.json  produced by `bpgo test-compat`

cmd/              the five user-facing commands
  bpgo/            main CLI + test-compat subcommand
  tpc/             TP7-compatible wrapper
  bprun/           IR binary runner
  tdebug/          debugger CLI
  turbo/           IDE

internal/         the implementation
  lexer/           TP7 tokenizer
  parser/          recursive-descent parser
  ast/             AST node definitions
  sem/             semantic analyser and type system
  ir/              IR and the embedded VM
  codegen8086/     8086 textual assembly emitter
  mz/              DOS MZ EXE writer
  omf/             OMF reader for {$L file.obj}
  tpu/             BPU unit container (TP7-compatible unit file)
  cli/             CLI driver (used by bpgo and tpc)
  compile/         pipeline: lex -> parse -> IR
  conformance/     the compatibility harness
  e2e/             end-to-end tests
  integration/     cross-component integration tests
  debug/           the debugger
  ide/             the IDE
  diagnostics/     the diagnostic catalog
  rtl/             unit implementations
    system/         System unit
    crt/            Crt unit
    dos/            Dos unit
    strings/        Strings unit
    windos/         WinDos unit
    printer/        Printer unit
    graph/          Graph/BGI unit
    graph3/         Graph3 unit
    turbo3/         Turbo3 unit
    overlay/        Overlay unit
  tv/              Turbo Vision framework (24 units)
  testutil/        small helper package used by tests
```

## Test layers

BPGo ships four test layers to ensure stability:

- **Unit tests** under each `internal/...` package, covering the
  language and runtime features each package implements.
- **End-to-end tests** in `internal/e2e`. They exercise the full
  pipeline (lex -> parse -> sem -> IR -> VM) on the programs in
  `testdata/pas`, run the CLI driver, build the binaries and verify
  the binary's `--help` and `-V` output.
- **Integration tests** in `internal/integration`. They run System
  unit builtins and representative members of the standard units
  through the full pipeline.
- **Conformance tests** in `internal/conformance`. They produce
  `compat/report.json` with a per-test, per-unit, per-directive and
  per-diagnostic report.

The four layers together exercise the system end-to-end. The
conformance harness is the canonical entry point: it produces the
report that the user can review to gauge compatibility.

## Status

See `docs/en/compatibility.md` for the current compatibility matrix and
`compat/report.json` for the machine-generated report. The
compatibility harness reports the pass ratio of the corpus and the
percentage of each unit's symbols that are implemented.

## Roadmap (reconstruction)

The system is being rebuilt in phases; each phase ends with `go build`
and `go test` green:

1. ✅ **VM procedural** — real codegen (`internal/codegen`) + VM: procedures/
   functions (frames, value/`var` params, recursion), records, arrays
   (incl. multidim), pointers (`New`/`Dispose`/`^`/`@`/`nil`), enums, sets,
   full control flow.
2. ✅ **RTL + I/O + units** — `ReadLn`/`Read`, `Write` `:w:d` formatting,
   real division, text file I/O, and a real `uses` unit system.
3. ✅ **OOP TP7** — `object`, constructor/destructor, inheritance, dynamic
   method dispatch (vtables + runtime type tag), `inherited`, `Self`.
4. ✅ **`pkg/vmpas`** — facade over the real compiler/VM: bind Go vars,
   map Go structs ↔ Pascal records, call Go funcs from Pascal, capability
   sandbox (`Restricted`/`Full`), zero external deps.
5. ✅ **Tooling** — LSP server (`cmd/pls`, live diagnostics) and DAP debug
   adapter (`cmd/pdap`, breakpoints/step/variables).
6. ✅ **Editor plugins** — VSCode (syntax + LSP + debugging); Zed (LSP).
7. ✅ **Docs & examples (Spanish)** — `docs/`, `examples/`, `cmd/pasrun`.
8. ✅ **Host integration** — HTTP client (all verbs, headers), JSON
   accessors, and SQL over Go's `database/sql` (host supplies the driver),
   all capability-gated.

The product focus is the embeddable engine and editor tooling. A nostalgic
TP7 TUI/IDE is **not** on the roadmap.

## Known limitations / out of scope

- Inline assembler, overlays, far pointers, real MZ EXE generation and
  DOS binary compatibility are **not** targeted.
- `internal/tv` (non-functional Turbo Vision stubs) and the
  `codegen8086`/`mz`/`omf` back-end are legacy/experimental and are not on
  the main path; the nostalgic TUI/IDE is not planned.
- `pkg/vmpas` is guaranteed dependency-free (a test enforces that tcell
  never enters its import closure), so embedding Pascal never pulls in the
  IDE.
