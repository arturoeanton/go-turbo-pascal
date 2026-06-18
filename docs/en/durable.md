# Durable execution: determinism + snapshot/resume (Phase F)

`vmpas` can **pause** an execution, **serialize** its entire state to bytes and
**resume** it exactly where it left off — even in another process, minutes or days
later. Combined with the **deterministic** mode, this gives business logic that is
*pausable, reproducible and auditable*: workflows that wait for an approval, an
external event or a schedule, without blocking a host thread.

Few embedded scripting engines offer this: you not only run typed code, you can
also **suspend and continue** an execution like a durable state machine.

## Determinism

With `Capabilities.Deterministic`, execution is bit-for-bit reproducible: the
same source + the same inputs produce the same output and the same state.
`Randomize` seeds the RNG from `Seed` (not from the host's entropy).

```go
caps := vmpas.Capabilities{Deterministic: true, Seed: 42}
out1, _ := vmpas.RunSandboxed(src, caps)
out2, _ := vmpas.RunSandboxed(src, caps) // out1 == out2, guaranteed
```

Determinism is the foundation of snapshot/resume: to resume reliably
you need the continuation to be independent of the clock or of the entropy.

## Pause and resume

The script is paused by calling the builtin `Suspend(tag)`. The host gets a
serializable `*State`; it persists it and later passes it to `ResumeDurable` to
continue.

```pascal
program Aprobacion;
var monto: Currency; aprobado: Boolean; resultado: string;
begin
  if monto > 1000.00 then
  begin
    Suspend('aprobacion-requerida');   { pauses here }
    if aprobado then resultado := 'APROBADO' else resultado := 'RECHAZADO';
  end
  else resultado := 'APROBADO (automatico)';
  WriteLn(resultado);
end.
```

```go
eng := vmpas.NewWith(vmpas.Capabilities{Deterministic: true, Seed: 1})
eng.Var("monto", &monto)
eng.Var("aprobado", &aprobado)

state, err := eng.RunDurable(rule)   // runs until Suspend (or until it finishes)
// state == nil  -> finished; state != nil -> paused (state.Tag, state.Data)

// ... persist state.Data (DB, file, queue) and, later:

aprobado = true                       // the host injects the decision
final, err := eng.ResumeDurable(rule, state)  // continues after the Suspend
// final == nil -> finished; final != nil -> paused again
```

See the runnable example in [`../examples/durable`](../../examples/durable).

## What is captured

The snapshot captures **all execution state**: global and local
variables, the operand stack, the **call stack with its program counters**,
the heap (including pointers and object graphs, e.g. linked lists created with
`New`), the RNG state and the exception state. On resume, execution
continues at the instruction following the `Suspend`, with pointer aliasing and
`var`-parameters intact.

### Input/output contract

- **Script state** (locals, unbound globals, heap, stack): captured and
  restored from the snapshot.
- **Bound Go variables** (`Var`): they are the **I/O channel with the host**. They
  are re-seeded on every resume, so the host injects answers by
  updating them *before* `ResumeDurable`, and the script reads them *after*
  `Suspend` returns.
- **Output** (`Output`): it is cumulative across segments (`State.Output` carries
  everything produced up to the pause).

## API

| Symbol | What it does |
|---------|----------|
| `Capabilities.Deterministic` / `Seed` | enables reproducible execution |
| `(*Engine).RunDurable(code) (*State, error)` | runs; returns `*State` if paused, `nil` if finished |
| `(*Engine).ResumeDurable(code, *State) (*State, error)` | restores and continues (same source) |
| `Suspend(tag)` (Pascal builtin) | pauses durable execution with a tag |
| `State{Tag, Data, Output}` | opaque, serializable snapshot (`Data` is portable) |

`ResumeDurable` requires the **same source** that produced the `State`: a *fingerprint*
of the compiled program is stored in the snapshot and validated on resume (changing
the code —even a literal— invalidates an old state, preventing PCs and
slot indices from silently misaligning).

## Sandbox: capability inference and auditable trace

Two tools for running untrusted scripts with *least-privilege* and
traceability (useful in multi-tenant and compliance scenarios).

### Minimum-capability inference

`Engine.Analyze(code)` compiles the script and reports **which capabilities it needs**
by scanning the bytecode for calls to gated host builtins. It does not
execute anything and works even if the engine is restricted — so you can grant
exactly what the script uses, or reject it if it asks for too much.

```go
rep, _ := eng.Analyze(src)
// rep.Required -> e.g. [Env Network]
// rep.Needs(vmpas.CapFileSystem) -> false
// rep.Calls[vmpas.CapNetwork]    -> ["httpget"]
if rep.Needs(vmpas.CapExec) {
    return errors.New("este tenant no puede ejecutar procesos")
}
```

### Auditable trace

With `Capabilities.Audit`, every call to a gated builtin (file, network,
exec, env, database) is logged **in execution order** with its
arguments. The log is deterministic and combines with snapshot/resume for
forensic replay.

```go
eng := vmpas.NewWith(vmpas.Capabilities{Network: true, Audit: true})
_ = eng.Run(src)
for _, ev := range eng.AuditLog() {
    log.Printf("%s %s%v", ev.Capability, ev.Builtin, ev.Args)
}
```

## Scope (v1)

- **Non-concurrent programs**: the snapshot supports single-fiber
  execution (the business-rules case). Snapshotting a concurrent program with
  live fibers (`spawn`/channels) is rejected with a clear error rather than
  producing a corrupt snapshot.
- **No open files**: you cannot snapshot while a `File` is
  open (it is a host resource, not serializable). Close it before `Suspend`.
- `MaxDuration` is not enforced on durable runs (a pause can last
  arbitrarily long); bound the work with `MaxSteps`.
