# Status, validation and viability

go-turbo-pascal is at **v1.0.0**: the embeddable engine and its API are stable.
This page records how the release was validated, the honest performance picture
(including a direct comparison against goja), and the known limitations that
remain.

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

Honest read: vmpas **allocates far less memory** (12 vs 3744 allocs in the loop),
while **goja is ~1.6–3.3× faster on time**. goja is a heavily optimized JS
bytecode interpreter; the vmpas VM uses a tagged union (`Value`) with boxing.
Closing the time gap would require optimizing the interpreter loop (dispatch,
avoiding `Value` boxing, possibly a register-based design) — a major effort, not
just wiring. Where vmpas adds value is elsewhere: **strong typing before
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
  ~1.6–3.3× faster in raw time (see the benchmark above). Closing the time gap
  would mean reworking the interpreter dispatch — a deliberate non-goal for v1.0.0.
- A nostalgic Turbo Pascal-style **TUI IDE** is not planned; `internal/tv` and
  `cmd/turbo` remain legacy stubs.

## Verdict

Both goals are met and the core is solid and tested. The remaining items are the
narrow language gap above and raw-time performance, neither of which requires
redesign. See the [compatibility matrix](compatibility.md) for per-feature detail.
