# Status, validation and viability

go-turbo-pascal is at **v1.4.0**: the embeddable engine and its public API are
stable (frozen since v1.0.0; see [api.md](api.md)). This page records how the
release was validated, the honest performance picture (including a direct
comparison against goja), and the known limitations that remain.

## Validation (all green)

- `go build ./...` ✅
- `go test ./... -count=1` ✅ — **600+ tests** PASS, 0 failures.
- `gofmt -l` ✅ empty (formatted).
- `go vet` ✅ except for warnings in the **legacy** `internal/tv` code (old
  Turbo Vision stubs, off the new path).
- Tooling (`pasrun`, `pls`, `pdap`) builds via `make tools`.
- Examples: `factorial`, `listas`, `figuras`, `calc` (interactive), `units/demo`
  and `examples/embed` (Pascal in Go) run correctly.
- LSP and DAP tested end-to-end (diagnostics; full debugging session).

### Performance (compile once, run many)

Measured with `go test -bench -benchmem` (numbers vary by machine; these are
indicative, Apple Silicon):

| Benchmark | Result |
|---|---|
| `vmpas` compile-once / run-many (sum 1..100) | ~24 µs/op, **12 allocs** |
| Sum 1..1000 | ~230 µs/op, **12 allocs**, ~1.9 KB |
| recursive `fib(20)` | ~4.0 ms/op, **65 allocs** |
| record-heavy loop (build/copy/read) | ~840 µs/op, ~7.5k allocs |

Optimizations applied over the 1.x series:
- **compile-once / run-many** in `vmpas` (`Engine.Compile` → `Script.Run`).
- **frame pool** + argument binding via a stack slice in the VM.
- **builtins cache** per engine (the RTL is not re-registered on each run): the
  loop runs at **12 allocs**.
- **v1.1.1** — O(n) builtin-argument marshalling and an integer fast path in the
  binary operators (~5% on scalar loops).
- **v1.2.0** — records use an association slice instead of a map, so field access
  avoids map hashing and a per-record allocation: record-heavy code is ~7%
  faster with ~12% less memory and ~17% fewer allocations.

### Direct benchmark vs goja

`internal/bench` is a **separate Go module** (so goja does not enter the main
`go.mod`, which stays zero-dependency). Run it with:
`cd internal/bench && go test ./... -bench . -benchmem`. It compiles once and uses a
fresh context per run:

| Benchmark | vmpas | goja |
|---|---|---|
| Sum 1..1000 | ~230 µs, **12 allocs**, ~1.9 KB | ~157 µs, 3744 allocs, ~112 KB |
| `fib(20)` | ~4.0 ms, 65 allocs | ~1.5 ms, 72 allocs |

Honest read: vmpas **allocates dramatically less memory** (12 vs 3744 allocs and
~59× fewer bytes in the loop), while **goja is ~1.5–2.6× faster on raw time**.
goja is a heavily optimized JS bytecode interpreter; the vmpas VM uses a tagged
union (`Value`) with boxing. Closing the time gap further would mean optimizing
the interpreter loop (dispatch, avoiding `Value` boxing, possibly a
register-based design) — a major effort, not just wiring, and a deliberate
non-goal for now. Where vmpas adds value is elsewhere: **strong typing before
execution**, the **capability sandbox**, durable execution and **zero
dependencies**.

## (A) Pascal embedded in Go — delivered

- Compiles and type-checks **before** executing (strong typing vs. dynamic engines).
- Go↔Pascal binding: scalar variables, **structs ↔ records**, slices ↔ arrays,
  and Go functions/methods callable from Pascal.
- **compile-once / run-many** (`Engine.Compile` → `Script.Run`).
- **Capability sandbox** (FS/net/exec/env/db + step/heap/output/time limits) and
  **capability inference** (`Analyze`) — capabilities most embedded engines leave to the host.
- **Durable execution**: deterministic snapshot/resume.
- **Zero dependencies** (guaranteed by a test).

## (B) TP7 in the console — delivered

- Procedural + **OOP** + full control flow + **console and text/typed-file I/O**
  + a real **units** system (`uses`).
- The `Crt` unit (`ClrScr`/`GotoXY`/colors/`KeyPressed`/`ReadKey`), `with`,
  variant records, `ShortString[N]` with 1-based indexing, set operators and
  `goto`/`label` all work.
- `pasrun` runs real `.pas`; modern tooling (LSP + DAP debugging) works in VSCode
  and Zed.

## Known limitations

- `inherited` works as a statement but not yet inside an expression
  (`x := inherited Foo + y`). See the [compatibility matrix](compatibility.md).
- **Performance vs goja**: vmpas allocates far less memory, while goja is
  ~1.5–2.6× faster in raw time (see the benchmark above). Closing the time gap
  further would mean reworking the interpreter dispatch — a deliberate non-goal
  for now.
- A nostalgic Turbo Pascal-style **TUI IDE** is not planned; `internal/tv` and
  `cmd/turbo` remain legacy stubs.

## Deferred ideas (to revisit another day)

These were considered and intentionally not done; recorded here so the decision
is explicit:

- **`inherited` inside an expression** — a codegen change (medium effort), the
  one real TP7 OOP gap.
- **Positional record/global slots** — would shave more memory/time but needs a
  shared type-layout registry across codegen/VM/snapshot; risk of incomplete
  offset coverage.
- **Smaller `Value` (hot/cold split)** — rejected: it adds an allocation per
  record/array value, regressing the memory profile that is the project's edge.
- **Operator-dispatch tag / register VM / closing the raw-time gap with goja** —
  large effort; measurements show operator decode is not the bottleneck.

## Verdict

Both goals are met and the core is solid and tested. The remaining items are the
narrow language gap above and raw-time performance, neither of which requires
redesign. See the [compatibility matrix](compatibility.md) for per-feature detail.
