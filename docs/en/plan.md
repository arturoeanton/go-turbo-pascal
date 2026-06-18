# Plan and audit (what is missing and how to prioritize it)

Status: the core (real compiler → VM, OOP, I/O, units), the embeddable engine
`vmpas` with sandbox, and the tooling (LSP + DAP + VSCode plugin) are complete and
tested. This document audits the repo and prioritizes what is missing for the two
use cases.

## Roadmap D (agreed execution order)

Order: **D1 → D11 → D10 → D7 → D2 → D4 → D3 → D5 → D6**. First the correctness
fixes and the value proposition of the embeddable engine (correctness, speed,
security, DX), then the modern Object Pascal language block.

| ID | Title | Type | Description | Effort |
|----|--------|------|-------------|----------|
| D1 | Functional properties | Fix | `property X read F write F` parses but is ignored at runtime; implement read/write against a backing field and getter/setter methods. | M |
| D11 | Performance vs goja | Feature | Optimize `Value` boxing and interpreter dispatch to close the time gap (~43% in the loop); vmpas already wins 58× on memory. | L |
| D10 | Real sandbox (Net/Exec/Env + limits) | Feature | Reserved flags with no RTL to govern; today the sandbox only covers FS + MaxSteps. Add sensitive RTL + enforcement + mem/time limit. | M |
| D7 | IDE-grade LSP | Feature | `pls` today only provides diagnostics; add hover, completion, goto-def and symbols. | L |
| D2 | Variant records: implement or error | Fix | `flattenVariant()` is an empty stub; the `case` part is discarded without warning. Minimum: a clear error; ideal: real variants. | S/L |
| D4 | Closures / anonymous methods | Feature | Does not exist; anonymous `procedure`/`function` + environment capture. | XL |
| D3 | Interfaces | Feature | `class(TParent, IFoo)` is accepted but not modeled; the contract + dispatch are missing. | L |
| D5 | Generics `<T>` | Feature | Does not exist; instantiation/monomorphization. | XL |
| D6 | Operator overloading | Feature | Operators hardcoded per type; declare `operator +` and resolve dispatch. | M |

Effort: S = hours · M = ~1 session · L = several sessions · XL = large/multi-session.

> **Series D: complete** (v0.1.2 → v0.1.3). All features D1–D11 implemented,
> tested and committed. In addition, an **integration phase** (v0.1.4 → v0.1.6):
> HTTP unit (all verbs + headers), JSON (read + build), and SQL over
> `database/sql` with a driver injected by the host (zero-deps preserved).

## Roadmap N (business rules engine) — before F

Decided positioning: the engine is the **base for proprietary business-scripting
products** (the engine is not licensed; it is an internal component). In that
framework, the "moat" problem disappears (the moat is the business app) and what
serves rule authors is prioritized. Series N comes **before Phase F**.

| ID | Title | What | Effort | Depends |
|----|--------|-----|----------|---------|
| N1 | IDE-grade diagnostics | Reporting infrastructure: line/col position on all errors, multiple errors per compilation, clear messages, surfaced via LSP. | M | — |
| N2 | `Currency`/decimal type | Exact arithmetic for financial rules. Recommended design: Delphi-style `Currency` (int64 ×10000, 4 fixed decimals) — idiomatic, exact, fast. | M | — |
| N3 | Business stdlib | Dates, money (uses N2), validations, decision tables. | M/L | N2 |
| N4 | Semantic/type-checking pass | Semantic pass (wire `internal/sem`) that catches types/identifiers/arity before runtime; feeds N1. Tackles the "loose typing". `match` exhaustiveness and strict null-safety (`T?`): later extensions. | L | N1 |

N1 (reporting) is the base; N4 (semantic) is the brain that produces rich
diagnostics on top of that base. On the speed and concurrency weaknesses: speed
is **not pursued** (irrelevant for business rules; if touched,
records-as-slices is the best ROI and helps F); concurrency is addressed with
`select` and reframed (host-level parallelism with multiple Engines).

### Robustness work (post-N, before/alongside F)

| ID | Title | What | Priority |
|----|--------|-----|-----------|
| N5 ✅ | Improve N4 | **Done.** Deeper, safer type checks (assignment compatibility by category, no false positives on valid TP7). | High |
| N6 ✅ | **Freeze/version the `vmpas` API** | **Done.** Stable public surface documented ([api.md](api.md)) with a semver policy; a contract test (`api_contract_test.go`) pins signatures and struct fields at compile time; `LICENSE` (MIT), `CHANGELOG.md` and CI (build+vet+test). | High |
| N7 ✅ | Multi-tenant SaaS hardening | **Done.** `Capabilities.MaxOutput`/`MaxCallDepth` (hard limits in the VM), reset of transient host state between executions, `Sandboxed()` preset and `RunSandboxed` API; the one-Engine-per-tenant pattern documented (docs/vmpas.md) + `examples/multitenant` example. | High |
| N8 ✅ | Expand stdlib (ERP/accounting/stock/HR/CMS/DMS) | **Done.** Rounding/VAT/percentages, business days/age/end-of-month/day-of-week, padding/replace/digit masks, numeric validations, split. Verticals as separate packages. | Medium |

> **Bus factor:** the foundation of proprietary products depends on a single
> maintainer. Recommended mitigation: document the internal architecture,
> embedding API tests (N6), and consider open-sourcing the *core* (not the
> product layer). The authoring/audit layer is where the business value lies.

## Roadmap E (modern language + determinism)

Product positioning: **the embeddable scripting engine for Go**. Series E adds
modern language features and Phase F adds the structural differentiator
(deterministic execution + snapshot). Parity with Delphi/Lazarus is not pursued,
nor is a TUI IDE (discarded).

**Cross-cutting compatibility rule:** all new syntax is enabled only under the
`{$MODE BPGO}` directive (E1); without it, the compiler is pure TP7. The new
words are **contextual keywords** in that mode. This shields compatibility.

Order: **E1 → E2 → E3 → E4 → E5 → E6 → F**.

| ID | Title | Type | Description | Effort | Depends |
|----|--------|------|-------------|----------|---------|
| E1 | `{$MODE BPGO}` mode gate | Infra | Mode directive + contextual keywords. Prerequisite for all new syntax; without it, pure TP7. Cheap and unblocks everything. | S/M | — |
| E2 | Ergonomics: local inference + `let` | Feature | `var x := expr` infers the type of the initializer; `let x = expr` immutable binding (error on reassignment). Compile-time only, runtime untouched. | M | E1 |
| E3 | Helpers + integrated unit tests | Feature | Delphi's `record helper`/`class helper` = extension methods (reuses method dispatch; allows extending mapped Go types). `test "..." begin..end` block + assertions + runner in the CLI (reuses the VM + exceptions; runs in the sandbox). | M | E1 |
| E4 | `match` + `Option/Some/None` | Feature | Sum types (ADTs with payload) + `match expr of Pattern => …; else …; end` with destructuring binding. Absorbs honest null-safety (Option, not Kotlin-style). Sub-phases: (a) ADTs+match; (b) exhaustiveness (best-effort or deferred). Do not reuse `with` (already a keyword). | L | E1 |
| E5 | `defer` / `panic` / `recover` | Feature | `panic`/`recover` map onto the existing `raise`/`except`; `defer` = per-frame LIFO list executed on return (reuses the `finally` machinery). | M | E1 |
| E6 | `spawn` + `Channel<T>` | Feature | Go-style concurrency with a **cooperative scheduler** (green threads in a single-thread VM): `spawn` creates a fiber, the VM interleaves `Step()`, channels block/yield. **Rewrite of the VM loop** (N stacks + scheduler + yield points). The big bet; a conscious investment decision. | XL | E1 |
| F (core) ✅ | Deterministic execution + snapshot/resume | Moat | **Done.** Deterministic mode (`Capabilities.Deterministic`/`Seed`; seedable `Randomize`). Full snapshot/resume of VM state (globals, locals, operand stack, call stack with PCs, heap with pointer graphs/linked lists, RNG, exceptions) to portable bytes, with a cell-identity table to preserve aliasing and discover orphan cells (`New`). Durable API in vmpas (`Suspend`, `RunDurable`/`ResumeDurable`, `State`) with a *fingerprint* guard. Docs (docs/durable.md) + example (examples/durable). v1 scope: non-concurrent, no open files. | L/XL | E6 |
| F (optional) ◐ | Provable sandbox | Moat | **Partial.** Deterministic gas: already covered by `MaxSteps`/`MaxHeap`/`MaxOutput`/`MaxCallDepth` (deterministic in `Deterministic` mode). **G1 ✅ minimum-capability inference** (`Engine.Analyze` scans the bytecode and reports which capabilities it uses). **G2 ✅ auditable trace** (`Capabilities.Audit` + `Engine.AuditLog`: records each gated call in order). Future notes: **G3** fine-grained allowlists (HTTP by domain, FS by path, SQL by verb) and **G4** formal "provable" (verifiable certificate of maximum capability) — deferred unless there is a concrete need. | M/L | F-core, E4 |

Discarded to stay focused (out of the 5 moat ideas): deep Go↔guest bridge,
static verification (taint/contracts), portable/WASM artifact.

**Speed optimization (deferred):** the big interpreter rewrite
(NaN-boxing / register VM) is done **after** E6 + F, on top of the already-frozen
VM design — and only the part that a **real measurement** justifies (in the
embeddable case, the bottleneck is usually network/DB, not the VM loop). The
cheap surgical optimizations (D11-style) are done ad-hoc when a
measurement calls for it.

> **Recent progress:** A1 (compile-once/run-many in `vmpas`) ✅, A2 (frame
> pool + args by slice; `fib(20)` from ~65k to ~88 allocs) ✅, B1 (`Crt` unit
> wired) ✅. Cleanup: `cmd/bprun` (dead) and `internal/bgi` (empty)
> removed; real vs legacy structure documented in `arquitectura.md`.

## Repository audit

**Real path (live):** `ast` → `lexer` → `parser` → `codegen` → `ir`; the CLIs
`bpgo`/`pasrun`, `pls` (LSP), `pdap` (DAP); `pkg/vmpas`. RTL: **only `system`**
is wired. `compile`+`conformance` remain only for `bpgo test-compat`.

**Legacy / experimental (builds, off the real path):**
- `internal/codegen8086`, `internal/mz`, `internal/omf` — 8086 backend / MZ EXE.
- `internal/tv/*` (14 packages) — Turbo Vision stubs (for the future TUI IDE).
- `internal/ide` + `cmd/turbo` — old ANSI IDE.
- `internal/debug` + `cmd/tdebug` — old debugger (replaced by `ir.Debugger`/DAP).
- `internal/rtl/{crt,dos,strings,graph,graph3,overlay,printer,turbo3,windos}` —
  implemented in Go but **without `Register(vm)`**, not exposed as builtins.

**Broken / to resolve:**
- `cmd/bprun` — expects `.bpi`; the new engine does not serialize IR → not functional.
- `cmd/tdebug` — uses the old debugger, not the new one.
- `internal/bgi` — empty.

## Case A — Pascal embedded in Go (alternative to goja)

| # | Task | Effort | Priority |
|---|---|---|---|
| A1 | **compile-once / run-many**: `vmpas.Compile(code)` → reusable program; today `Run` always recompiles | M | High |
| A2 | **Reduce allocs in calls**: frame pool / reusable slots (lower the ~65k allocs of `fib(20)`) | M | High |
| A3 | **More Go↔Pascal type mapping**: slices↔arrays, maps, pointers to struct, Go struct methods as procs/methods | M-L | High |
| A4 | **Direct benchmark vs goja** (publishable suite) | S | Medium |
| A5 | **Sandbox + strong**: configurable memory/time limits, per-path FS allowlist | S | Medium |
| A6 | **API ergonomics**: errors with position, timeout/context, `MustRun` | S | Low |

Suggested order: **A1 → A2 → A4** (demonstrable perf) → **A3** (coverage) → A5/A6.

## Case B — TP7 alternative in the console

| # | Task | Effort | Priority |
|---|---|---|---|
| B1 | **Wire the RTL to the engine** via `uses`: give the units `Register(vm)` and register them. Start with **`Crt`** (ClrScr, GotoXY, TextColor/Background, KeyPressed, ReadKey, Delay, Window) over an ANSI terminal abstraction | M | High |
| B2 | **`with`** (resolve record fields in the body) | S-M | High |
| B3 | **TP7 strings**: `ShortString[N]`, 1-based `s[i]` indexing, builtins (Copy/Pos/Delete/Insert/Length/Concat/UpCase) with correct semantics and by-reference behavior | M | High |
| B4 | **Complete sets**: operators `+ - *` and `<= >=` | S | Medium |
| B5 | **Typed/binary files**: `file of T`, Read/Write/Seek/BlockRead/Write | M | Medium |
| B6 | **`goto`/`label`** | S | Medium |
| B7 | **Variant records** (`case` in a record) | S-M | Low |
| B8 | **`case` with ranges and char** | S | Low |
| B9 | **TUI IDE** over tcell (deferred) | L | Low |

Suggested order: **B1 (Crt) → B2 (with) → B3 (strings)** cover the majority of
TP7 console programs; then B4–B8; B9 last.

## Phase C — modern Object Pascal evaluation

To decide the jump from "classic TP7" to "modern Pascal". Effort: XS/S/M/L/XL.

| # | Feature | Effort | What it touches | Risk | Value |
|---|---|---|---|---|---|
| C3 | **Dynamic arrays** (`array of T`, `SetLength`, `Length`/`High`) | M | parser + codegen + VM (VKArray is already a slice) | low | high |
| C7 | **`for..in`** | S | parser + codegen (desugar to index) | low | medium |
| C2 | **Exceptions** (`try..except..finally`, `raise`, `on E:`) | M-L | lexer + parser + VM (frame unwinding; interacts with the frame pool) | medium-high | high |
| C1 | **`class`** (properties, interfaces, `create`/`free`) | L | parser + sem + codegen + VM (by-reference semantics, VMT already exists; interfaces = per-interface tables) | medium-high | high |
| C4 | **Modern strings** (`AnsiString`/Unicode) | S–M | the current `String` is already dynamic (Go string, UTF-8); declaring `AnsiString` is cheap; real Unicode (code-point indexing) is medium | low–medium | medium |
| C6 | **Anonymous methods / closures** | L | parser + VM (environment capture; today functions are top-level without capture) | high | medium |
| C5 | **Generics** | XL | parser + sem + codegen (monomorphization/specialization) | high | high (advanced) |

Recommended order by value/effort:
**C3 → C7 → C2 → C1 → C4 → C6 → C5.**
Notes: C3 has the best ratio (the VM already has arrays as slices); C7 is done
alongside C3; C1 builds on the existing `object`/VMT OOP model (class ≈ object
by reference + create/free + properties + interfaces); C2 and C6 require touching
the engine (unwinding / closures). Speed vs goja is tackled **after** all of
this (optimizing the interpreter loop: `Value` boxing and dispatch).

## Cleanup / technical debt

| # | Action | Effort |
|---|---|---|
| C1 | 8086/MZ backend (`codegen8086`/`mz`/`omf`): mark experimental or retire | S |
| C2 | `cmd/bprun`: serialize IR to `.bpi` from codegen **or** retire the command | S |
| C3 | `cmd/tdebug`: repoint to the new `ir.Debugger` **or** retire in favor of `pdap` | S |
| C4 | `internal/debug`, `internal/ide`, `internal/tv/*`: mark "pre-TUI/legacy" | S |
| C5 | `internal/bgi`: remove (empty) | XS |
| C6 | `sem`: integrate it into codegen for stronger checks **or** document that codegen resolves on its own | M |

## Prioritization recommendation

- **If the immediate goal is A (competitive embeddable engine):** A1, A2, A4.
- **If the immediate goal is B (TP7 console apps):** B1 (Crt), B2, B3.
- **Recommended cross-cutting:** C5 (trivial), C2/C3 (CLI coherence), and a
  regression test that runs `examples/interactive/*.pas` with its inputs.
