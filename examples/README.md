# Examples

## Pascal programs (`examples/pascal/`)

Run them on the real engine:

```bash
go run ./cmd/pasrun examples/pascal/<file>.pas
```

| File | Shows |
|---|---|
| `factorial.pas` | recursive functions |
| `listas.pas` | pointers and records (linked list) |
| `figuras.pas` | OOP: objects, inheritance, `inherited` |
| `calc.pas` | `ReadLn` + field formatting (`echo "12 5" \| pasrun ...`) |
| `crt_demo.pas` | the `Crt` unit: ClrScr/GotoXY/TextColor (ANSI output) |
| `units/demo.pas` | the unit system (`uses`, interface/implementation/initialization) |

## Embedding Pascal in Go (`examples/embed/`)

```bash
go run ./examples/embed
```

Demonstrates `pkg/vmpas`: running Pascal, binding Go variables, mapping a Go
`struct` to a Pascal `record`, calling Go functions from Pascal, and the
capability sandbox.

## Consuming APIs and SQL (`examples/integration/`)

```bash
go run ./examples/integration
```

Self-contained and offline: it starts a local HTTP server and an in-memory SQL
database, and from Pascal consumes the API (`HttpGet`/`HttpPost`/`HttpLastStatus`)
and walks a query (`DbOpen`/`DbEof`/`DbNext`/`DbFieldInt`/`DbFieldStr`) under the
`Network` and `Database` capabilities.

## Isolated multi-tenant scripts (`examples/multitenant/`)

```bash
go run ./examples/multitenant
```

Simulates a SaaS where each tenant submits its own business rule: it runs every
script on a fresh, bounded engine with `vmpas.RunSandboxed` + the
`vmpas.Sandboxed()` preset (default-deny, with step/heap/output/depth/time
ceilings). It shows *share-nothing* isolation and how a malicious script (an
infinite loop) is stopped without hanging the host.

## Durable execution: pause and resume (`examples/durable/`)

```bash
go run ./examples/durable
```

An expense-approval rule runs until it needs a human decision: it pauses with
`Suspend`, the host serializes the state (`RunDurable` → `*State`), injects the
answer and resumes on a fresh engine (`ResumeDurable`), continuing exactly where
it left off. See [`../docs/en/durable.md`](../docs/en/durable.md).

See also [`../docs/en/getting-started.md`](../docs/en/getting-started.md) and the
embedding guide in [`../docs/en/vmpas.md`](../docs/en/vmpas.md).
