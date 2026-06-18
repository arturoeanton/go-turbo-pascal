# Status, validation and viability

Report on the validation phase prior to the TUI: what was tested, what works, and
what is missing for the project to be viable as (A) a Pascal language embedded in
Go and (B) a TP7 alternative in the console.

## Validation (all green)

- `go build ./...` ✅
- `go test ./... -count=1` ✅ — **469 tests** PASS, 0 failures.
- `gofmt -l` ✅ empty (formatted).
- `go vet` ✅ except for warnings in the **legacy** `internal/tv` code (old
  Turbo Vision stubs, off the new path).
- Tooling (`pasrun`, `pls`, `pdap`) builds via `make tools`.
- Examples: `factorial`, `listas`, `figuras`, `calc` (interactive), `units/demo`
  and `examples/embed` (Pascal in Go) run correctly.
- LSP and DAP tested end-to-end (diagnostics; full debugging session).

### Performance (compile once, run many)

| Benchmark | Result |
|---|---|
| Loop sum 1..10000 | ~2.5 ms/op (~36M instr/s), 35 allocs |
| recursive `fib(20)` | ~4.8 ms/op, **88 allocs** (previously ~65k) |
| `vmpas` recompiles each Run | ~46 µs/op, 265 allocs |
| `vmpas` compile-once / run-many | ~34 µs/op, 94 allocs |

Optimizations applied (A1/A2/A4):
- **compile-once / run-many** in `vmpas` (`Engine.Compile` → `Script.Run`).
- **frame pool** + argument binding via stack slice in the VM
  (`fib(20)` dropped from ~65k to ~88 allocs).
- **builtins cache** per engine (the RTL is not re-registered on each run): the
  loop dropped to **12 allocs**.

### Direct benchmark vs goja (A4)

`internal/bench` is a **separate Go module** (so goja does not enter the main
`go.mod`, which stays zero-dependency). Run it with:
`cd internal/bench && go test ./... -bench . -benchmem`. It compiles once and uses a
fresh context per run:

| Benchmark | vmpas | goja |
|---|---|---|
| Sum 1..1000 | ~252 µs, **12 allocs** | ~157 µs, 3744 allocs |
| `fib(20)` | ~5.0 ms, 65 allocs | ~1.5 ms, 72 allocs |

Honest read: **vmpas wins on memory** (12 vs 3744 allocs in the loop) but
**goja is ~1.6–3.3× faster on time**. goja is a heavily optimized JS bytecode
interpreter; the vmpas VM uses a tagged union (`Value`) with boxing.
To beat goja on time would require optimizing the interpreter loop (dispatch,
avoiding `Value` boxing, possibly a register-based design) — a major
effort, not just wiring. The vmpas differentiators remain the
**strong typing before execution**, the **capability sandbox** and **zero
dependencies**.

## (A) Pascal embedded in Go — VIABLE, with perf/binding work

Ready:
- Compiles and type-checks **before** executing (strong vs. dynamic engines).
- Go↔Pascal binding: scalar variables and **structs ↔ records**, and Go functions
  callable from Pascal.
- **Capability sandbox** (FS/net/exec/limits) — a real differentiator against
  goja.
- **Zero dependencies** (guaranteed by a test).

To close out viability (to be clearly competitive vs goja):
1. **compile-once / run-many API**: today `Engine.Run` recompiles on every call;
   expose a reusable compiled program.
2. **Micro-optimize calls**: frame pool, avoid allocating slot slices
   per call (reduces the ~65k allocs of `fib(20)`).
3. **More type mapping**: Go↔Pascal slices/arrays and maps, pointers to struct,
   expose Go struct methods as object methods.
4. **Direct benchmark vs goja** to publish numbers.

## (B) TP7 alternative in the console — VIABLE for the console; RTL pieces missing

Ready:
- Procedural + **OOP** + control flow + **console and text-file
  I/O** + **units** (`uses`).
- `pasrun` runs real `.pas`; modern tooling (LSP + DAP debugging) in
  VSCode.

For a faithful TP7 console experience, what is missing (suggested order):
1. **Functional `Crt` unit**: `ClrScr`, `GotoXY`, colors, `KeyPressed`/`ReadKey`
   — it is what TP7 console apps use most.
2. **`with`**, **typed/binary files**, **variant records**.
3. **Strings**: `ShortString[N]` semantics and 1-based character indexing.
4. **Sets**: set operators `+ - *` (today: literals and `in`).
5. **`goto`/`label`**.
6. (deferred) the Turbo Pascal-style **TUI IDE**.

## Verdict

Both directions are **viable** and the core is solid and tested. For (A) the
remaining work is performance and binding breadth; for (B) it is RTL coverage
(above all `Crt`) and some language features. Neither requires redoing the
design: they are incremental extensions on top of what is already built. See the
[compatibility matrix](compatibility.md) for the per-feature detail.
