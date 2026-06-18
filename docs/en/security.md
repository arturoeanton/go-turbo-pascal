# Security & the capability sandbox

When you embed `pkg/vmpas` you are running guest Pascal code inside your Go
process. This page explains the security model, how to run untrusted scripts,
and — just as important — **what the sandbox does not protect against**.

## The model in one sentence

Every `Engine` runs under a `Capabilities` value that is **default-deny**: unless
you explicitly grant a capability, the builtins that could reach the outside
world are **not registered**, so a script that calls them fails to **compile**
(not at runtime).

## Capabilities

| Capability  | Grants access to | Builtins |
|-------------|------------------|----------|
| `FileSystem` | file I/O | `Assign`/`Reset`/`Rewrite`/`Read`/`Write`/`Close`/… |
| `Network`    | outbound HTTP | `HttpGet`/`HttpPost`/`HttpPut`/`HttpDelete`/`HttpRequest`/`HttpSetHeader`/`HttpLastStatus` |
| `Exec`       | spawning processes | `Exec(cmd): Integer` |
| `Env`        | environment variables | `GetEnv(name): string` |
| `Database`   | SQL via `database/sql` | `Db*` (requires `UseDB`) |

`GetEnv`, `Exec`, the `Http*` and the `Db*` builtins are **vmpas host
extensions**, not part of the TP7 RTL — they exist only when their capability is
granted. JSON parsing/building (`Json*`) needs no capability: it is pure
computation.

## Resource limits

Capabilities gate *what* a script can reach; limits gate *how much* it can
consume. All are off (0) by default and enforced inside the VM.

| Field          | Effect | Error on breach |
|----------------|--------|-----------------|
| `MaxSteps`     | VM instruction budget | runtime error 200 |
| `MaxHeap`      | heap allocations (`New`/pointers) | runtime error 203 |
| `MaxOutput`    | bytes of captured output | runtime error 203 |
| `MaxCallDepth` | call-stack depth | runtime error 202 |
| `MaxDuration`  | wall-clock time | runtime error 200 |

## Presets

```go
vmpas.New()                 // = Restricted(): no FS/net/exec/env/db, no limits
vmpas.NewWith(vmpas.Sandboxed()) // default-deny + conservative step/heap/output/depth/time ceilings
vmpas.NewWith(vmpas.Full())      // everything on — trusted code only
```

- **`Restricted`** (the default): denies all outside access but sets no resource
  ceilings. Good for trusted code you wrote yourself.
- **`Sandboxed`**: the preset for **untrusted** code — default-deny plus
  conservative limits on steps, heap, output, call depth and time.
- **`Full`**: grants everything. Use only for code you fully trust (e.g. a
  first-party tool).

## Running untrusted scripts (multi-tenant)

The pattern is **one engine per request/tenant, share-nothing**. The helper
`RunSandboxed` does exactly that on a fresh, isolated engine:

```go
out, err := vmpas.RunSandboxed(tenantScript, vmpas.Sandboxed())
```

Adjust the preset to the tenant's needs:

```go
caps := vmpas.Sandboxed()
caps.MaxDuration = 500 * time.Millisecond
caps.MaxOutput   = 256 * 1024
caps.Network     = true      // allow HTTP if this tenant is permitted
out, err := vmpas.RunSandboxed(tenantScript, caps)
```

Isolation guarantees:

- **No leaks between runs**: each `Run` builds a fresh VM (globals zeroed) and
  resets transient host state (SQL cursor, last HTTP status/headers). Reusing one
  `Engine` across tenants does not leak data between them.
- **Hard stops**: a script that floods output, recurses forever or loops
  endlessly is stopped with a runtime error — it does not exhaust memory or hang
  the host.
- **Parallelism**: run many engines in separate goroutines. An engine is
  single-threaded per run; concurrency comes from one engine per goroutine.

## Inspect before you trust: `Analyze` and `AuditLog`

- **Before running**, `Engine.Analyze(code)` statically reports the capabilities a
  script needs (`CapReport.Needs`, `.Required`, `.Calls`) without executing it.
  Use it to reject scripts that exceed policy or to grant the minimal set.
- **After running**, the `Audit` capability records every gated call so you have
  an after-the-fact trail (`Engine.AuditLog()`).

```go
rep, _ := eng.Analyze(script)
if rep.Needs(vmpas.CapExec) { return errors.New("exec not allowed") }
```

## Threat model — what this is and isn't

The sandbox is an **in-process, language-level** boundary. It is effective
against the things guest Pascal can express, but it is **not** an OS-level jail.

**It does protect against:**
- Guest code reaching the filesystem, network, processes, env or a database
  without an explicit grant.
- Runaway resource use (CPU steps, memory, output, recursion, wall-clock).
- State bleeding between runs/tenants.

**It does not protect against:**
- **`Full()` or broad grants** — these are an explicit statement of trust.
- **Your own Go callbacks.** A function you register with `Function`/`Process`
  runs with full Go privileges; if it touches the disk or network, the script
  reaches them regardless of capabilities. Keep registered callbacks as narrow
  as the capability you would otherwise grant.
- **Host-side misuse of returned data** (e.g. passing script output into a shell).
- **Side channels and absolute guarantees** against a determined attacker
  exploiting a bug in the engine. For hard multi-tenant isolation of genuinely
  hostile code, combine vmpas with OS-level sandboxing (containers, seccomp,
  separate processes/users).

## Recommendations

- Start from `Sandboxed()` for anything you did not write; widen one capability
  at a time.
- Always set `MaxDuration` and `MaxOutput` for untrusted code.
- Prefer `Analyze` to *decide* and `AuditLog` to *record*.
- Register the **fewest, narrowest** Go callbacks you can.

See also: [vmpas guide](vmpas.md) · [durable execution](durable.md) ·
[compatibility matrix](compatibility.md).
