# TP7 compatibility matrix

`go-turbo-pascal` is a clean-room Go implementation of a Turbo Pascal 7 /
Borland Pascal 7 compatible front-end and engine. No Borland binary, source or
documentation is embedded. This document states honestly what works today on the
**real engine** (`internal/lexer` → `parser` → `sem` → `codegen` → `ir` VM), and
what is legacy or out of scope.

> **How to read this.** "Supported" means it compiles and runs on the engine and
> is covered by tests in `internal/codegen`, `internal/ir`, `internal/e2e`,
> `internal/integration` or `pkg/vmpas`. Where there is a known gap, it is listed
> explicitly rather than hidden.

## Language — procedural core

| Feature | Status | Notes |
|---|---|---|
| Procedures & functions | ✅ | own frame, recursion, local variables |
| Parameters: value / `var` / `const` | ✅ | `var` is true by-reference (cell aliasing) |
| Function result (`Result`/name assignment) | ✅ | |
| Integers, `Real`, `Boolean`, `Char` | ✅ | |
| `ShortString` (`string[N]`) | ✅ | length byte + 1-based indexing |
| Records, nested records | ✅ | value-copy semantics |
| Variant records (`case` in record) | ✅ | `testdata/pas/variant.pas` |
| Static arrays, multidimensional | ✅ | nested arrays, range-checked indexing |
| Enumerations & subranges | ✅ | ordinal mapping |
| Sets (`+ - *`, `in`, comparisons) | ✅ | `testdata/pas/sets.pas` |
| Pointers (`^T`, `@`, `New`/`Dispose`, `nil`) | ✅ | heap and cell pointers |
| Forward type references (`PNode = ^TNode`) | ✅ | `TNode` may be declared later |
| Control flow `if`/`case`/`for`/`while`/`repeat` | ✅ | |
| `with` | ✅ | selector resolution |
| `break`/`continue`/`Exit`/`goto`/`label` | ✅ | |
| `Inc`/`Dec`, `Write`/`WriteLn` `:w:d` formatting | ✅ | typed formatting |

## Language — OOP (TP7 object model)

| Feature | Status | Notes |
|---|---|---|
| `object` types with fields | ✅ | |
| Inheritance `object(Parent)` | ✅ | field layout inherited |
| Methods (procedure/function) | ✅ | |
| `virtual` methods + dynamic dispatch | ✅ | real polymorphism via base pointer (VMT) |
| `constructor` / `destructor` | ✅ | |
| `inherited` — **statement** form | ✅ | `inherited Init(a)`, `inherited Draw` |
| `inherited` — **expression** form | ❌ | `x := inherited Foo + y` is not parsed yet |

The only OOP gap is `inherited` used inside an expression. As a statement it
works; this is why `testdata/pas/objectpoly.pas` (which writes
`GetX := inherited GetX + Y`) is the one corpus program skipped in the OOP
end-to-end tests.

## Language — modern extensions

These are go-turbo-pascal additions on top of TP7, useful when embedding:

| Feature | Status | Doc |
|---|---|---|
| `match` / sum types / `Option` | ✅ | [match.md](match.md) |
| `defer` / `panic` / `recover` | ✅ | [defer.md](defer.md) |
| `spawn` / channels | ✅ | [concurrency.md](concurrency.md) |

## Runtime library (units)

Implemented as Go packages wired into the VM and importable via `uses`. See
[units.md](units.md) for the per-unit symbol map.

| Unit | Status | Notes |
|---|---|---|
| `System` | ✅ | implicit; memory, I/O, strings, math, ordinals, sets |
| `Crt` | ✅ | `ClrScr`/`GotoXY`/`TextColor`/`KeyPressed`/`ReadKey`, virtual 80×25 screen |
| `Dos` | ✅ | date/time, env, file search, sandboxed services |
| `Strings` | ✅ | `PChar` helpers (`StrCat`, `StrComp`, …) |
| `WinDos` | ✅ | `PChar` flavours of Dos services |
| `Printer` | ✅ | `Lst` file (in-memory or file-backed) |
| `Graph` / `Graph3` | ✅ | software framebuffer, palette, viewports, primitives |
| `Turbo3` / `Overlay` | ✅ | TP3 file variables; overlay manager (counters) |
| `uses` unit system | ✅ | interface/implementation/initialization, RTL bound to the VM |
| File I/O (text & typed) | ✅ | `Assign`/`Reset`/`Rewrite`/`Read`/`Write`/`Close`/`Eof`/`Append` |

## Embedding & tooling

| Component | Status | Doc |
|---|---|---|
| `pkg/vmpas` — embeddable engine | ✅ | [vmpas.md](vmpas.md) |
| Go ↔ Pascal binding (vars, funcs, struct↔record) | ✅ | [vmpas.md](vmpas.md) |
| Capability sandbox (FS/net/exec/env/db + limits) | ✅ | [security.md](security.md) |
| Durable execution (deterministic snapshot/resume) | ✅ | [durable.md](durable.md) |
| Capability inference (`Analyze`) + audit log | ✅ | [vmpas.md](vmpas.md) |
| LSP server (`cmd/pls`) | ✅ | [editors.md](editors.md) |
| DAP debug adapter (`cmd/pdap`) | ✅ | [editors.md](editors.md) |
| VSCode plugin (syntax + LSP + debug) | ✅ | [editors.md](editors.md) |
| Zed plugin (LSP) | ✅ | [editors.md](editors.md) |

## Compiler directives

Directives are tokenized and parsed; their runtime effect varies — see
[directives.md](directives.md) for the per-directive table. In short: switches
such as `{$R+}`/`{$I+}` are accepted and are mostly no-ops on the bytecode
backend; `{$I file.inc}` include and `{$L file.obj}` linking are not wired to the
VM backend.

## Legacy / experimental (not on the main path)

These build and have tests but are **not** part of the supported engine. They are
kept for history and experimentation:

| Component | State |
|---|---|
| `internal/compile` + `internal/conformance` | minimal stub harness behind `bpgo test-compat` (2 smoke tests); the real coverage is the engine test suite |
| `internal/codegen8086` | emits textual 8086 assembly; not assembled to a runnable program |
| `internal/mz` | writes a syntactically valid MZ EXE header; no working code section |
| `internal/omf` | reads THEADR/LNAMES/SEGDEF/PUBDEF/EXTDEF only |
| `internal/tv/*` (Turbo Vision) | non-functional view stubs |
| `cmd/turbo` (TP7-style IDE) | interactive/headless shell stub |
| `cmd/tdebug` (CLI debugger) | superseded by `cmd/pdap` / `internal/ir.Debugger` |

## Out of scope

Inline assembler, overlays in the DOS sense, far pointers, real MZ EXE code
generation, and DOS binary compatibility are **not** targeted.

## Measuring it yourself

```bash
go test ./...                 # full engine + library test suite
go run ./cmd/pasrun x.pas     # run a real .pas on the engine
go run ./cmd/bpgo test-compat # legacy conformance harness → compat/report.json
```

`compat/report.json` is produced by the legacy harness and reports per-unit
symbol coverage, directives and diagnostics; treat the engine test suite as the
authoritative signal.
