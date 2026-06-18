# FAQ & troubleshooting

Short answers to the questions that come up most when embedding Pascal with
`pkg/vmpas` or running `.pas` on the engine.

## Embedding

**Do I pass a fragment or a full program?**
Either. `Run` accepts a bare fragment (it is wrapped in an implicit
`begin … end.`) or a complete `program … end.` / `unit … end.`. A fragment is
convenient for one-liners and rules; a full program when you declare your own
types and routines.

**How do I read a value back out of the script?**
Bind a Go variable by **pointer** with `Var`; vmpas seeds it into the script and
copies it back after the run:

```go
total := 10
eng.Var("total", &total)
eng.Run(`total := total * 2`)
// total == 20
```

Pass a value (not a pointer) for read-only. Exported struct fields map to record
fields by name (case-insensitive); slices map to 0-based arrays.

**Why does my script fail with "unknown identifier" at compile time?**
That is the point — vmpas type-checks before running. The name is undeclared, or
it is a capability-gated builtin (e.g. `HttpGet`) and you did not grant the
capability, so it was never registered. Grant the capability or fix the name. Use
`Analyze` to see what a script needs.

**Can the same engine run many scripts?**
Yes. Each `Run` builds a fresh VM and resets transient state, so runs do not leak
into each other. For the same code executed repeatedly, `Compile` once into a
`Script` and call `Script.Run` — see [vmpas: compile once, run many](vmpas.md).

**Is vmpas safe for untrusted code?**
With the right configuration, yes — see [security](security.md). Start from
`Sandboxed()`, set `MaxDuration`/`MaxOutput`, and keep registered Go callbacks
narrow. It is an in-process language sandbox, not an OS jail.

**Does importing vmpas pull in extra dependencies?**
No. `pkg/vmpas` is zero-dependency, enforced by `TestVMPasHasNoExternalDeps`.
SQL support uses Go's stdlib `database/sql`; *you* bring the driver and inject the
handle with `UseDB`, so no driver ever enters vmpas's import tree.

## Language

**`Do`/`To` as identifiers don't parse.**
`do` and `to` are reserved keywords. To name a method or variable that collides,
use a different identifier (the lexer is case-insensitive, so `Do`/`To` are
reserved too).

**`x := inherited Foo + y` fails to parse.**
Known limitation: `inherited` works as a **statement** (`inherited Init(a)`) but
not yet inside an **expression**. Rewrite it in two steps:

```pascal
tmp := inherited Foo;   { statement-style call into a temporary }
x := tmp + y;
```

**Is OOP supported?**
Yes: `object` types, fields, inheritance, **virtual methods with dynamic
dispatch**, constructors/destructors and statement-form `inherited`. See the
[compatibility matrix](compatibility.md).

**What about `match`, `defer`, channels?**
Those are modern extensions on top of TP7: [match](match.md),
[defer/panic/recover](defer.md), [spawn/channels](concurrency.md).

## Tooling & errors

**Editor support?**
LSP (`pls`) and DAP (`pdap`) servers drive VSCode and Zed — diagnostics, hover,
completion, go-to-definition and debugging. See [editors](editors.md).

**A runtime error code like 200/202/203 — what is it?**
Those are limit breaches: 200 = step/time budget, 202 = call depth, 203 =
heap/output. Raise the corresponding `Capabilities` limit if legitimate, or treat
it as the sandbox doing its job. The error catalog is in [errors](errors.md).

**`bpgo test-compat` only shows 2 tests — is that the real coverage?**
No. That harness is a legacy stub. The authoritative signal is the engine test
suite: `go test ./...` (600+ tests). See the
[compatibility matrix](compatibility.md).

## Performance

**Is vmpas faster than goja?**
On **memory**, vmpas allocates far less. On raw **time**, goja is currently
~1.6–3.3× faster — it is a heavily optimized JS interpreter. vmpas focuses
elsewhere: ahead-of-time type checking, the capability sandbox, durable execution
and zero dependencies. Numbers and methodology are in [status](status.md).

**How do I make repeated execution fast?**
Use `Compile` → `Script.Run` (compile once, run many) and bind variables rather
than rebuilding the program string each time.
