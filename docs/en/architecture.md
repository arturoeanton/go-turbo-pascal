# Architecture

go-turbo-pascal brings Turbo Pascal 7 to Go with two goals:

1. **Embed Pascal in Go** through the `pkg/vmpas` library.
2. Provide modern tooling for `.pas` (LSP + DAP, editor plugins).

## Compilation pipeline

```
Pascal source (.pas)
   │
   ▼  internal/lexer      → tokens
   ▼  internal/parser     → AST (internal/ast)
   ▼  internal/sem        → semantic analysis / types
   ▼  internal/codegen    → bytecode IR  ← real compiler
   ▼  internal/ir         → bytecode VM that executes the IR
```

### Front-end (lexer / parser / ast)

Solid and stable. It covers most of the TP7 syntax: procedures and
functions (value parameters, `var`, `const`), records (including variants),
arrays, strings, sets, enums, subranges, pointers, all the control flow
(`if/case/for/while/repeat/with`), OOP (`object`, methods, inheritance) and units.

### internal/codegen — the real compiler

Walks the AST and emits bytecode IR. Today it supports:

- Procedures and functions with their own *frame*, value and `var`
  (reference) parameters, recursion and local variables.
- Records (field access, value-copy semantics), static arrays
  (incl. multidimensional as nested arrays), pointers (`New`/`Dispose`,
  `^`, `@`, comparison with `nil`), enums and sets.
- Full control flow, `Inc`/`Dec`, `Write`/`WriteLn`, and a set of
  RTL builtins.
- Options for embedders (`pkg/vmpas`): externals (host functions /
  RTL procedures), preset globals, and auto-declaration of
  loop variables for fragments.

The previous "toy" compiler (`internal/compile`) is kept for the
conformance harness while `internal/codegen` becomes the main
path.

### internal/ir — the bytecode VM

A stack VM with:

- A **cell-reference** model (`*Value`): globals, frame slots,
  record fields, array elements and heap cells are addressable.
  This provides correct semantics for `var` params, pointers and nested mutation.
- A frame-based calling convention (parameters first, then locals) and a
  function-result *slot*.
- Runtime-typed values: integer, real, boolean, char, string, set,
  array, record, pointer, file and `nil`.

## Embedding: pkg/vmpas

`pkg/vmpas` is a facade over `codegen` + `ir`:

- Compiles the code (with prior type checking) and runs it on the VM.
- Binds Go variables (seeding them into globals and reading them back),
  maps Go structs ↔ Pascal records via reflection, and registers Go functions as
  VM builtins.
- Applies the **capability sandbox** by deciding which builtins are registered.

Architectural rule: `pkg/vmpas` stays **zero-dependency** (verified by
`TestVMPasHasNoExternalDeps`). Any external dependency (tcell, LSP/DAP
servers) lives outside that import tree.

## RTL

`internal/rtl/*` implements the standard units (System, Crt, Dos, Strings...) as
Go functions registered as VM builtins, importable through a real `uses` unit
system (interface/implementation/initialization).

## Tooling

- **LSP + DAP**: a language server (`cmd/pls`: diagnostics/hover/completion/
  go-to-definition) on top of the front-end, and a debug adapter (`cmd/pdap`:
  breakpoints/step/variables) on top of the VM.
- **Zed and VSCode plugins**: thin clients over LSP/DAP (see [editors.md](editors.md)).
- **TUI + TP7 IDE** (not currently planned): a nostalgic Turbo Pascal-style IDE
  on tcell. The `internal/tv` stubs and `cmd/turbo` are legacy, not on the
  supported path.

## Project structure (real path vs. legacy)

To avoid confusion: these are the **active** components (the real path) and the
**legacy/experimental** ones that are kept but not on that path.

**Active / real path:**
- `internal/{lexer,parser,ast}` → `internal/codegen` → `internal/ir` (compiler + VM).
- `internal/rtl/system`, `internal/rtl/crt` (RTL wired to the VM).
- `internal/cli` + `cmd/bpgo` (main CLI), `cmd/pasrun` (runner), `cmd/tpc`.
- `internal/lsp` + `cmd/pls` (LSP); `internal/dap` + `cmd/pdap` (debugging).
- `pkg/vmpas` (embeddable engine) + `examples/`.

**Legacy / experimental (builds and tests, off the real path):**
- `internal/compile` + `internal/conformance` — only for `bpgo test-compat`.
- `internal/codegen8086`, `internal/mz`, `internal/omf` — 8086 backend / MZ EXE
  (experimental; not a current goal).
- `internal/tv/*`, `internal/ide`, `cmd/turbo` — old Turbo Vision-style IDE
  (the TUI IDE on tcell is a future phase).
- `internal/debug`, `cmd/tdebug` — old CLI debugger (modern debugging is
  `cmd/pdap` / `internal/ir.Debugger`).
- `internal/sem` — semantic analysis; the current codegen resolves types on its
  own, so `sem` is barely used on the real path.

The full audit and the prioritized plan are in [`plan.md`](plan.md).

## Out of scope (for now)

Inline assembly, overlays, far pointers, real MZ EXE generation and
binary compatibility with DOS.
