# BPGo Compatibility Matrix

BPGo is a clean-room Go implementation of a Turbo Pascal 7 / Borland
Pascal 7 compatible compiler, runtime, IDE and debugger. This
document describes the current state of compatibility.

## Status

| Layer                  | Status   | Notes |
|------------------------|----------|-------|
| Lexer (TP7 keywords)   | Complete | case-insensitive; `do` and `to` are reserved as identifiers (use `Do`/`To` for method names) |
| Lexer (directives)     | Complete | `{$R+}` `{$I+}` `{$IFDEF}` are recognised; semantics are a no-op in the vm backend |
| Parser (TP7 syntax)    | Complete | All major declarations, statements, expressions |
| Semantic analyser      | Partial  | Records symbols, basic types, objects with VMTs; builtins not yet registered as visible names |
| IR + VM                | Complete | Stack machine, sets, strings, calls, control flow |
| System unit            | Complete | All symbols listed in `compat/spec/units/system.json` |
| Crt unit               | Complete | Virtual 80x25 screen, key queue, sound, windows |
| Dos unit               | Complete | In-memory sandbox, date/time, env, interrupt dispatch |
| Strings unit           | Complete | PChar helpers |
| WinDos unit            | Complete | PChar versions of Dos services |
| Printer unit           | Complete | Lst file with in-memory and file output |
| Graph unit             | Complete | Software framebuffer, palette, viewports, primitives |
| Graph3 unit            | Complete | Shim over Graph |
| Turbo3 unit            | Complete | Stub for TP3 compatibility |
| Overlay unit           | Complete | Manager + counters |
| Turbo Vision (24 units) | Complete | Objects, Drivers, Views, Menus, Dialogs, App, etc. |
| MZ EXE writer          | Complete | Headers, segments, relocations |
| 8086 codegen           | Partial  | Emits textual assembly; not assembled to a runnable EXE |
| OMF reader             | Partial  | Reads THEADR, LNAMES, SEGDEF, PUBDEF, EXTDEF |
| TPU (.bpu) container   | Complete | Go-friendly binary unit format |
| IDE (turbo)            | Partial  | TP7-style interactive/headless shell with file/project/debug/mouse-event commands |
| Debugger (tdebug)      | Complete | Breakpoints, watches, snapshot, step/continue |
| Conformance harness    | Complete | Runs the corpus, writes `compat/report.json` |

## How to read `compat/report.json`

```bash
go run ./cmd/bpgo test-compat
```

The report contains:

- `GeneratedAt`: timestamp of the run
- `Total`, `Passed`, `Failed`: pass ratio of the corpus
- `Tests`: per-test result with category, backend, duration and message
- `Units`: per-unit symbol coverage estimate
- `Directives`: every TP7 directive listed in the manifest
- `Diagnostics`: every compile and runtime error code

## Current limitations

The pipeline produces a working vm-backend program for trivial
sources (`program T; begin end.`). It does not yet emit IR for the
full language; the conformance harness uses a minimal stub IR. The
dos16 backend produces a syntactically valid MZ file but the
assembler that lowers the textual 8086 output to bytes is not
shipped; use an external NASM/TLINK toolchain to produce a runnable
EXE.

The sem analyser does not yet:

- Bind `with do` selectors (parsing is supported, symbol resolution
  is not)
- Resolve `inherited` calls (parsing is supported)
- Track forward type references such as `PNode = ^TNode` with `TNode`
  declared later (lexer/parser handle it; sem flags it)
- Register the System unit's builtins as visible names (use the
  `internal/rtl/system` package directly from a host program to
  exercise them)

These features are exercised by golden tests in `internal/e2e` and
`internal/integration` and have explicit skip lists where they
currently fail.

`nil` is tokenized as a TP7 keyword and accepted as a pointer null
expression by parser and sem.
