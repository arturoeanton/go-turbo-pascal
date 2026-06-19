# Changelog

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project follows [Semantic Versioning](https://semver.org/): from v1.0.0 on,
the public API of `pkg/vmpas` is stable within each major series (see
[docs/en/api.md](docs/en/api.md)).

## [1.4.0] — Context cancellation and typed results
### Added
- `Engine.RunContext(ctx, code)` / `Script.RunContext(ctx)`: cooperative
  cancellation — the run aborts shortly after the context is cancelled or its
  deadline passes and returns `ctx.Err()`.
- `Engine.Get(name, out)`: read a script global by name into a typed Go pointer
  after a run, without pre-binding a variable.
### Tests
- WrapSQLDB / the *sql.DB adapter (via a stdlib-only fake driver) and the
  internal line debugger.
### Notes
- Evaluated an operator-dispatch tag (P3) but reverted it: measured no gain on
  the arithmetic benchmarks (the hot path is dominated by dispatch and Value
  copying, not operator decode).

## [1.3.1] — Robustness, micro-perf and cleanup
### Changed
- Snapshot restore validates frame PCs and bounds-checks frame indices; OPMk*
  clamp negative element counts — corrupt input is a clean error, not a panic.
- Micro-perf: cache `Deadline.IsZero()` per run; ADT field names from a table
  (no `Sprintf`); `selfTypeName` lowercases without allocating on the common path.
- Removed the dead `RandomState` field; RTL `New`/`GetMem` allocate via a new
  `AllocHeap` (count against `MaxHeap`, return a valid heap index).

## [1.3.0] — Typed errors, builtin cache, more guards
### Added
- `vmpas.RuntimeError` (Code + human-readable message, `errors.As`-friendly).
### Changed
- `OPCallBuiltin` caches its resolution per VM (skips the map lookup on repeats).
- `formatCurrency` no longer emits `-0.00`.
### Tests
- Money-format edges, record field helpers, set-copy isolation.

## [1.2.2] — Engine hardening + adoption polish
### Changed
- Output cap enforced at write time (a single oversized write can no longer
  exceed `MaxOutput`); stack-consuming opcodes and out-of-range pointer deref
  return a clean runtime error instead of panicking; `Http*` builtins are
  time-boxed (honor `MaxDuration`, else 30s).
### Added
- README install line + pkg.go.dev badge; runnable Example functions for struct
  binding, RunSandboxed, Analyze and durable run/resume; examples translated to
  English.

## [1.2.1] — Documentation current through v1.2.0
### Changed
- Rewrote the CHANGELOG and the status/benchmark pages with measured numbers
  (EN + ES).

## [1.2.0] — Record representation: association slice
### Changed
- Records use a `[]RecField` association slice instead of `map[string]*Value`.
  For the small field counts records typically have, a linear scan over an
  interned-name slice beats a map: it drops the map's allocation and the
  per-access string hashing. Field cells stay `*Value`, so `var`-param / `@field`
  aliasing is unchanged.
### Performance
- Record-heavy workloads: ~7% faster, ~12% less memory, ~17% fewer allocations.
  Scalar code keeps the same allocation count (memory profile preserved).

## [1.1.1] — Interpreter performance tuning
### Performance
- `OPCallBuiltin` marshals its arguments in O(n) with a single allocation (was
  O(n²) with a per-argument allocation).
- Integer fast path in the binary/comparison operators, skipping the
  currency/set/real/string cascade for the common int-int case.
- ~5% faster on the sum/fib benchmarks with no change in allocations.

## [1.1.0] — Richer Go ↔ Pascal interop
### Added
- Struct field tags: `vmpas:"name"` / `json:"name"` to rename a field, `vmpas:"-"`
  to hide it.
- Nested structs and pointers round-trip; a nil Go pointer maps to Pascal `nil`.
- A bound Go function whose last result is an `error` raises a catchable Pascal
  exception (`try/except`); uncaught, `Run` returns the Go error message.
- `Capabilities.LiveBindings` (opt-in): keeps bound Go variables in sync with the
  script around host calls.

## [1.0.2] — Docs: honest positioning
### Changed
- README gains a "When vmpas is the right fit" section framing the niche without
  naming other engines or claiming superiority; comparative wording softened to a
  factual, modest tone (the goja benchmark is kept as measured data).

## [1.0.1] — Documentation overhaul
### Changed
- Rewrote the TP7 compatibility matrix to reflect the real codegen+VM (including
  the precise `inherited`-in-expression limitation) and a clear legacy list.
- Removed stale "in development / roadmap" notes for shipped features.
- Documented `Engine.Compile`/`Script` and `Engine.Analyze`; added security and
  FAQ guides. English doc filenames under `docs/en`. Unified naming to
  go-turbo-pascal.

## [1.0.0] — Stable, reusable embeddable engine
### Added
- **Public API freeze** for `pkg/vmpas` (N6): stability document
  ([docs/en/api.md](docs/en/api.md)) and a contract test pinning the public surface.
- `LICENSE` (MIT), `CHANGELOG.md` and CI (build + vet + test).
- Bilingual documentation (`docs/en`, `docs/es`).

## [0.7.0] — Provable sandbox: capability inference + auditable trace
### Added
- `Engine.Analyze`: infers which host capabilities a script needs by scanning the
  bytecode (G1).
- `Capabilities.Audit` + `Engine.AuditLog`: records every gated call in execution
  order with its arguments (G2).

## [0.6.0] — Deterministic execution + snapshot/resume (Phase F, core)
### Added
- Deterministic mode (`Capabilities.Deterministic`/`Seed`; seedable `Randomize`).
- Full VM-state snapshot/resume (globals, locals, operand stack, call stack with
  PCs, heap with pointer graphs, RNG, exceptions).
- Durable API: the `Suspend` builtin, `Engine.RunDurable`/`ResumeDurable`, `State`.

## [0.5.0] — Robustness for embedded business rules (N series)
### Added
- Category-based type checking on assignments, without false positives (N5).
- Multi-tenant hardening: `Capabilities.MaxOutput`/`MaxCallDepth`, the
  `Sandboxed()` preset, `RunSandboxed`, per-run state reset (N7).
- Management stdlib: VAT/rounding/percentages with exact `Currency`, business
  days/age/end-of-month, padding/masks/validation, `Split` (N8).

## [0.4.0] and earlier
- Language core (TP7 procedural + OOP), bytecode VM, RTL, units.
- Modern features under `{$MODE BPGO}`: inference/`let`, helpers, `match` +
  Option/ADTs, `defer`/`panic`/`recover`, `spawn` + `Channel<T>`.
- `pkg/vmpas`: embeddable engine, Go↔Pascal binding, capability sandbox,
  HTTP/SQL/JSON integration, `Currency` type, business stdlib (N1–N4).
- Tooling: IDE-grade diagnostics; LSP/DAP and editor plugins.

[1.4.0]: https://github.com/arturoeanton/go-turbo-pascal/compare/v1.3.1...v1.4.0
[1.3.1]: https://github.com/arturoeanton/go-turbo-pascal/compare/v1.3.0...v1.3.1
[1.3.0]: https://github.com/arturoeanton/go-turbo-pascal/compare/v1.2.2...v1.3.0
[1.2.2]: https://github.com/arturoeanton/go-turbo-pascal/compare/v1.2.1...v1.2.2
[1.2.1]: https://github.com/arturoeanton/go-turbo-pascal/compare/v1.2.0...v1.2.1
[1.2.0]: https://github.com/arturoeanton/go-turbo-pascal/compare/v1.1.1...v1.2.0
[1.1.1]: https://github.com/arturoeanton/go-turbo-pascal/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/arturoeanton/go-turbo-pascal/compare/v1.0.2...v1.1.0
[1.0.2]: https://github.com/arturoeanton/go-turbo-pascal/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/arturoeanton/go-turbo-pascal/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/arturoeanton/go-turbo-pascal/compare/v0.7.0...v1.0.0
[0.7.0]: https://github.com/arturoeanton/go-turbo-pascal/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/arturoeanton/go-turbo-pascal/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/arturoeanton/go-turbo-pascal/compare/v0.4.0...v0.5.0
