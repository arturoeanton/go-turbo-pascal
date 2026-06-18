# go-turbo-pascal

**An embeddable, strongly-typed Pascal engine for Go — plus a clean-room Turbo Pascal 7 toolchain.**

`go-turbo-pascal` lets you run real Pascal *inside* a Go program through the
[`pkg/vmpas`](pkg/vmpas) library, and ships modern editor tooling (LSP + DAP)
for `.pas` files. The compiler front-end and bytecode VM are a clean-room
implementation: no Borland binary, source, or documentation is embedded.

[![CI](https://github.com/arturoeanton/go-turbo-pascal/actions/workflows/ci.yml/badge.svg)](https://github.com/arturoeanton/go-turbo-pascal/actions/workflows/ci.yml)
&nbsp;·&nbsp; Go 1.23+ &nbsp;·&nbsp; MIT licensed &nbsp;·&nbsp; `pkg/vmpas` has **zero external dependencies**

📖 **Documentation:** [🇬🇧 English](docs/en/README.md) · [🇪🇸 Español](docs/es/README.md) · [Examples](examples/README.md)

---

## Why embed Pascal?

`pkg/vmpas` is an embeddable scripting engine in the spirit of the popular
JavaScript and Lua engines for Go — but for Pascal, and **strongly typed**. It
compiles and type-checks the whole program **before the first instruction runs**,
so a class of errors that dynamic engines only surface at runtime is caught up
front.

What you get:

- **Strong typing, checked ahead of time** — the program is compiled to bytecode
  and type-checked before execution, not interpreted off the AST.
- **Two-way Go ↔ Pascal binding** — expose Go variables (`Var`), call Go
  functions from Pascal (`Function`/`Process`), and map Go structs to Pascal
  records.
- **A capability sandbox (default-deny)** — file system, network, process
  execution, environment and database access are all off unless you opt in,
  with step/heap/output/time limits on top.
- **Durable execution** — a running program can be deterministically snapshotted,
  persisted, and resumed later (even on another machine). See
  [durable execution](docs/en/durable.md).
- **Capability inference & audit** — statically discover which capabilities a
  script needs before running it, and get an audit log of every gated call.
- **Zero dependencies** — embedding Pascal never pulls anything into your build;
  a test enforces it.

## When vmpas is the right fit

There are excellent embeddable engines in Go already, and for many projects one
of those is the right call. vmpas focuses on a specific combination that is
otherwise hard to assemble in one place:

> **Pascal + ahead-of-time type checking + a capability sandbox + deterministic,
> resumable execution + zero dependencies.**

Reach for it when you need to:

- **Pause, persist and resume** an execution deterministically and auditably —
  e.g. long-running workflows, step-through approvals, or migrating a computation
  between machines (see [durable execution](docs/en/durable.md)).
- Run **untrusted or per-tenant code** under a default-deny capability sandbox
  with hard resource limits (see [security](docs/en/security.md)).
- Embed a **strongly typed** language whose errors surface at compile time, not
  mid-run.
- Bring **Pascal** to a Go service — for legacy TP7 logic, domain rules authored
  in Pascal, or teaching.

If raw single-threaded throughput is your only metric, a mature JavaScript or Lua
engine will likely be faster; vmpas trades a little speed for typing, isolation
and durability. The honest numbers are in [status & benchmarks](docs/en/status.md).

## Quickstart

Embed Pascal in Go ([`examples/embed`](examples/embed/main.go)):

```go
import "github.com/arturoeanton/go-turbo-pascal/pkg/vmpas"

eng := vmpas.New()        // default: Restricted (no FS/net/exec)
total := 10
eng.Var("total", &total)
eng.Run(`for i := 1 to 5 do total := total + i`)
// total == 25
```

Run a `.pas` program on the engine:

```bash
go run ./cmd/pasrun examples/pascal/factorial.pas
go run ./examples/embed
```

Compile once, run many times:

```go
script, _ := eng.Compile(code)
for _, input := range inputs {
    script.Run()
}
```

More: [Quickstart](docs/en/quickstart.md) ·
[vmpas guide](docs/en/vmpas.md) ·
[Durable execution](docs/en/durable.md) ·
[Security & sandbox](docs/en/security.md) ·
[TP7 compatibility](docs/en/compatibility.md) ·
[FAQ](docs/en/faq.md).

## Build and test

```bash
go build ./...   # build everything
go test ./...    # run the full test suite
go vet ./...     # static checks
```

## What's in the box

### `pkg/vmpas` — the embeddable engine
The library facade over the real compiler and VM. Bind Go values, call across
the Go ↔ Pascal boundary, sandbox capabilities, run durably. Zero external deps.
The public API is frozen and covered by a contract test — see
[Public API and stability](docs/en/api.md).

### Editor tooling
| Command  | Description |
|----------|-------------|
| `pls`    | Pascal **language server** (LSP) — diagnostics, hover, completion, go-to-definition |
| `pdap`   | Pascal **debug adapter** (DAP) — breakpoints, stepping, call stack, variables |
| `bpgo`   | main CLI: compile & run (`bpgo --run x.pas`) plus `test-compat` |
| `pasrun` | minimal "compile & run a `.pas`" driver |
| `tpc`    | a TP7-compatible CLI wrapper around `bpgo` |
| `tdebug` | source-level debugger CLI |

Editor plugins live in [`editors/`](editors): **VSCode** (syntax + LSP +
debugging) and **Zed** (LSP). See [Editors](docs/en/editors.md).

### The Pascal language
TP7 procedural **and** OOP: procedures/functions (value/`var`/`const` params,
recursion), records, multidimensional arrays, sets, enums, subranges, pointers
(`New`/`Dispose`/`^`/`@`), real `ShortString` and `Char`, full control flow,
`object` types with constructors/destructors, inheritance and virtual methods
(VMT dispatch), a real `uses` unit system, and text/typed file I/O. Modern
extras: `match`/sum types/`Option`, `defer`/`panic`/`recover`, and
`spawn`/channels.

See the [TP7 compatibility matrix](docs/en/compatibility.md) for exactly what is
and isn't supported.

## Architecture

```
Pascal source
  → internal/lexer    TP7 tokenizer
  → internal/parser   recursive-descent parser → internal/ast
  → internal/sem      semantic analysis & type system
  → internal/codegen  bytecode generation
  → internal/ir       bytecode VM (frames, records, arrays, sets, pointers,
                       strings, OOP/VMT, file I/O, snapshot/resume)
  → internal/rtl      RTL units (System, Crt, Dos, Strings, …) bound to the VM

Consumers:
  pkg/vmpas           Go facade: Var/Function/Process binding, sandbox, durable run
  cmd/{bpgo,pasrun,tpc,tdebug}   CLIs over the same pipeline
  cmd/{pls,pdap}      LSP / DAP servers (may have their own deps; vmpas stays clean)
```

[Full architecture](docs/en/architecture.md).

## Quality

Four test layers run on every change and in CI:

- **Unit tests** per `internal/...` package, covering each language and runtime feature.
- **End-to-end tests** (`internal/e2e`) exercising the full lex → parse → sem → codegen → VM pipeline over the `testdata/pas` corpus and the CLIs.
- **Integration tests** (`internal/integration`) running RTL units through the full pipeline.
- **Conformance tests** (`internal/conformance`) producing `compat/report.json`, a per-unit/-directive/-diagnostic compatibility report.

## Scope and non-goals

To set honest expectations:

- **Targeted:** embedding Pascal in Go, the TP7 language (procedural + OOP),
  the RTL units listed above, deterministic durable execution, and editor tooling.
- **Not targeted:** inline assembler, overlays, far pointers, real MZ EXE
  generation, and DOS binary compatibility.
- **Legacy / experimental:** `internal/tv` (Turbo Vision stubs) and the
  `codegen8086`/`mz`/`omf` back-end are not on the main path. A nostalgic
  TP7 TUI/IDE is not currently planned.

## License

[MIT](LICENSE).
