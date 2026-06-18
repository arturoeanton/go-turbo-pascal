# vmpas — Pascal embedded in Go

`pkg/vmpas` is an **embeddable dynamic-code engine** for Go. It lets you
run Turbo Pascal 7 code inside a Go program, binding host variables,
functions and structs. Unlike dynamic scripting engines
(e.g. JavaScript with goja), vmpas **compiles and type-checks before
the first execution**: compilation errors are caught instantly,
not in the middle of a run.

Features:

- **Strongly typed**: compiled to bytecode and validated before running.
- **Go ↔ Pascal mapping**: scalar variables and `struct` ↔ `record`.
- **Bidirectional calls**: Pascal can call Go functions/procedures.
- **Capability sandbox**: fine-grained control over what guest code can
  do (filesystem, etc.), with `Restricted` (default) and `Full`.
- **Zero external dependencies**: importing `vmpas` does not drag in tcell or the
  editor toolchain (guaranteed by a test).

## Installation

```go
import "github.com/arturoeanton/go-turbo-pascal/pkg/vmpas"
```

## Basic usage

```go
eng := vmpas.New() // restricted sandbox by default
if err := eng.Run(`WriteLn('Hola, mundo!')`); err != nil {
    log.Fatal(err)
}
fmt.Print(eng.Output()) // "Hola, mundo!\n"
```

You can pass a fragment (automatically wrapped in a program) or a
complete program (`program ... end.`).

## Compile once, run many

`Run` compiles on every call. When you execute the same code repeatedly (a hot
path, a rules engine evaluated per request), compile it once into a `Script` and
reuse it:

```go
script, err := eng.Compile(code) // lex + parse + sem + codegen, once
if err != nil {
    log.Fatal(err) // all compile/type errors surface here, up front
}
for _, row := range rows {
    eng.Var("row", &row)
    script.Run()              // executes the cached program
    fmt.Print(script.Output())
}
```

This is the recommended pattern for performance; see the
[benchmarks](status.md) (the loop drops to ~12 allocations/run).

## Binding Go variables

Pass a **pointer** for read/write; a value for read-only.

```go
total := 10
eng.Var("total", &total)
eng.Run(`for i := 1 to 5 do total := total + i`)
// total == 25  (the Go variable was modified by the script)
```

Supported scalar types: integers, `float32/64`, `string`, `bool`.

## Mapping a struct to a record

```go
type Punto struct{ X, Y int }

p := Punto{X: 3, Y: 4}
eng.Var("p", &p)
eng.Run(`p.X := p.X * p.X + p.Y * p.Y`)
// p.X == 25  (exported fields are mapped by name)
```

The exported fields of the struct are exposed as record fields (the name
is compared case-insensitively) and copied back after
execution.

### Field names with tags

By default the Pascal field name is the Go field name. Use a struct tag to choose
a different name, or to hide a field. `vmpas:"…"` takes precedence over `json:"…"`
(so existing JSON tags are reused automatically), and `vmpas:"-"` skips the field:

```go
type User struct {
    FullName string `vmpas:"name"`   // exposed as record field `name`
    Email    string `json:"email"`   // reuses the JSON tag -> `email`
    Internal int    `vmpas:"-"`       // not exposed to Pascal
}
```

## Nested structs and pointers

Nested structs map to nested records, and pointers are followed: a Go function
that takes or returns a `*T` works (a nil pointer maps to Pascal `nil`, a non-nil
one to the pointed-to record). Pointer arguments and results round-trip with their
fields.

```go
type Point struct{ X, Y int }
eng.Function("SumSq", func(p *Point) int { return p.X*p.X + p.Y*p.Y })
eng.Var("p", &Point{X: 3, Y: 4})
eng.Run(`out := SumSq(p)`) // out == 25
```

## Slices and arrays

A Go slice/array maps to a Pascal `array` (0-based index) and is copied
back after running:

```go
xs := []int{1, 2, 3, 4, 5}
eng.Var("xs", &xs)
eng.Run(`for i := 0 to 4 do xs[i] := xs[i] * xs[i]`)
// xs == [1 4 9 16 25]
```

Go functions that take or return slices also work (the result
is assigned to an `array` variable in Pascal).

## Methods of Go structs

A Go *method value* is a function, so it is bound just like any
function:

```go
r := Rect{W: 4, H: 5}
eng.Function("Area", r.Area) // Area() int
eng.Run(`out := Area()`)     // out == 20
```

## Calling Go functions from Pascal

```go
eng.Function("Duplicar", func(n int) int { return n * 2 })
eng.Process("Registrar", func(s string) { log.Println(s) })

eng.Run(`
  r := Duplicar(21);
  Registrar('listo')`)
```

`Function` registers a callable that returns a value; `Process` one that
does not (a procedure). Arguments and the result are converted automatically
between Go and Pascal.

### Errors: Go `error` → Pascal exception

If a bound Go function's last result is an `error`, returning a non-nil error
**raises a Pascal exception** instead of returning a value. The script can catch
it with `try/except`; if it doesn't, the run stops and `Run` returns the Go error
message.

```go
eng.Function("Charge", func(amount int) (int, error) {
    if amount <= 0 {
        return 0, errors.New("amount must be positive")
    }
    return amount, nil
})
```

```pascal
try
  total := Charge(-5);          { raises -> jumps to except }
except
  WriteLn('charge failed');
end;
```

A function whose only result is `error` behaves as a procedure that may raise.
This makes host failures first-class in the script instead of being silently
dropped.

## Capability sandbox

Each `Engine` runs under a sandbox. The default value (`New()` /
`Restricted()`) **denies** file access. To allow everything (trusted
code only, e.g. a TP7 IDE) use `Full()`:

```go
eng := vmpas.NewWith(vmpas.Full())
```

`Capabilities`:

| Field         | Effect                                                          |
|---------------|-----------------------------------------------------------------|
| `FileSystem`  | enables the file builtins (Assign/Reset/...)                    |
| `Network`     | enables the HTTP builtins (`HttpGet`/`HttpPost`/`HttpLastStatus`) |
| `Exec`        | enables the host builtin `Exec(command): Integer`               |
| `Env`         | enables the host builtin `GetEnv(name): string`                 |
| `Database`    | enables the SQL builtins (`Db*`); requires `UseDB`              |
| `MaxSteps`     | VM step limit (0 = default)                                    |
| `MaxHeap`      | maximum heap allocations, `New`/pointers (0 = no limit)        |
| `MaxOutput`    | maximum bytes of captured output (0 = no limit)                |
| `MaxCallDepth` | maximum call-stack depth (0 = no limit)                        |
| `MaxDuration`  | wall-clock execution time limit (0 = no limit)                 |
| `Deterministic` / `Seed` | reproducible execution (see [durable.md](durable.md)) |
| `Audit`        | logs every gated call (`Engine.AuditLog`); see [durable.md](durable.md) |
| `LiveBindings` | sync bound variables with the script around host calls (see below) |

Capabilities are enforced at the Go↔Pascal boundary: forbidden builtins are not
registered, so calling them is a **compilation error** (not a runtime
failure). `GetEnv`, `Exec`, the `Http*` and the `Db*` are
**vmpas host extensions** (not part of the TP7 RTL) and only
exist when their capability is granted.

## Live bindings

By default a bound variable is copied into the script when a run starts and
copied back when it ends, so changes a host callback makes to it mid-run are
overwritten by that final copy-back. Enable `LiveBindings` to keep the binding in
sync **around every call into a bound Go function/procedure**: the script's
current value is written back to Go before the call, and the host's mutation is
made visible to the script after it.

```go
counter := 10
eng := vmpas.NewWith(vmpas.Capabilities{LiveBindings: true})
eng.Var("counter", &counter)
eng.Process("Bump", func() { counter++ })   // mutates the Go variable
eng.Run(`Bump; seen := counter; Bump`)       // seen == 11, counter ends at 12
```

Without `LiveBindings` the same script leaves `counter` at 10 (the callback's
writes are discarded by the end-of-run copy-back). The option adds per-call
overhead proportional to the number of bound variables, so it is opt-in.

## Capability inference (`Analyze`)

Before granting anything, you can statically discover which capabilities a script
actually needs. `Analyze` compiles the code and scans the builtins it calls,
returning a `CapReport`:

```go
rep, err := eng.Analyze(tenantScript)
if err != nil {
    log.Fatal(err) // the code does not even compile
}
if rep.Needs(vmpas.CapNetwork) {
    // this script wants HTTP — decide whether to allow it
}
fmt.Println(rep.Required) // e.g. [filesystem network]
fmt.Println(rep.Calls)    // the gated builtins it calls
```

Use it to reject scripts that exceed a policy, to show an operator what a script
will touch before approving it, or to grant the minimal capability set. `Analyze`
never executes the code. For an after-the-fact record of what actually ran, see
the `Audit` capability below (`Engine.AuditLog`).

## Multi-tenant: running untrusted scripts

For a service where **each tenant provides its own script** (embedded business
rules engine), the pattern is **one engine per request/tenant** —
*share-nothing*: no state is shared between executions. The helper
`RunSandboxed` does this in one line, on a fresh, isolated engine:

```go
out, err := vmpas.RunSandboxed(tenantScript, vmpas.Sandboxed())
```

`Sandboxed()` is a *default-deny* preset with conservative ceilings designed for
untrusted code (no FS/network/exec/env, with limits on steps, heap, output,
call depth and time). Adjust the fields to taste:

```go
caps := vmpas.Sandboxed()
caps.MaxDuration = 500 * time.Millisecond
caps.MaxOutput   = 256 * 1024
caps.Network     = true            // allow HTTP if the tenant needs it
out, err := vmpas.RunSandboxed(tenantScript, caps)
```

Isolation guarantees:

- **No leaks between runs**: each `Run` creates a new VM (globals zeroed),
  and transient host state (SQL cursor, last HTTP error/status,
  headers) is reset at the start of each run. Reusing the same `Engine`
  for several tenants does not leak data between them.
- **Hard limits**: a script that floods output, recurses endlessly or enters
  an infinite loop is stopped with a runtime error (it does not exhaust memory or
  hang the host process).
- **Parallelism**: the host can run many engines in different goroutines
  in parallel (an engine is single-threaded per run; real concurrency
  is provided by the host with one engine per goroutine).

## Integration: HTTP and SQL (consuming APIs and databases)

Under the `Network` capability, Pascal code can consume APIs with all the
verbs, headers (e.g. authentication tokens) and JSON parsing:

```pascal
{ Verbs: GET, POST, PUT, PATCH, DELETE, and HttpRequest for any method }
HttpSetHeader('Authorization', 'Bearer ' + token);  { header on subsequent calls }
body   := HttpGet('https://api.example.com/users');
result := HttpPost('https://api.example.com/users', 'application/json', '{"n":1}');
HttpPut('https://api.example.com/users/1', 'application/json', '{"n":2}');
HttpDelete('https://api.example.com/users/1');
HttpRequest('OPTIONS', 'https://api.example.com', '', '');
status := HttpLastStatus();   { status code of the last call }

{ Read JSON (no capability: it is pure computation) }
name := JsonStr(body, 'user.name');     { dotted-path access }
id   := JsonInt(body, 'items.0.id');    { numeric segment = array index }
len  := JsonLen(body, 'items');         { array/object length }
if JsonValid(body) then ...             { JsonValid / JsonBool / JsonStr / JsonInt / JsonLen }

{ Build JSON (set by path; creates intermediate objects/arrays) }
req := JsonSetStr('{}', 'user.name', 'bob');
req := JsonSetInt(req, 'user.age', 25);
req := JsonSetBool(req, 'user.active', true);
HttpPost(url, 'application/json', req);  { -> {"user":{"active":true,"age":25,"name":"bob"}} }
s := JsonEscape('con "comillas"');       { -> "con \"comillas\"" (for manual assembly) }
```

Under the `Database` capability, the code talks to any database supported by
Go's `database/sql`. The host injects the handle (and brings the driver), so
`pkg/vmpas` stays **free of external dependencies**:

```go
import "database/sql"
// _ "github.com/mattn/go-sqlite3"  // the host brings the driver

db, _ := sql.Open("sqlite3", "app.db")
eng := vmpas.NewWith(vmpas.Capabilities{Database: true})
eng.UseDB(vmpas.WrapSQLDB(db))   // adapts *sql.DB (stdlib only)
```

The SQL API in Pascal is a Delphi dataset-style cursor:

```pascal
n := DbExec('INSERT INTO users(name) VALUES (?)', 'alice');  { affected rows }
if DbOpen('SELECT id, name FROM users') then
  while not DbEof() do
  begin
    WriteLn(DbFieldInt(0), ' ', DbFieldStr(1));
    DbNext;
  end;
DbClose;
if DbError() <> '' then WriteLn('error: ', DbError());
```

`DbExec(sql [, params...])` executes and returns affected rows; `DbOpen` runs
a query and positions the cursor; `DbEof`/`DbNext` iterate; `DbFieldStr(i)` /
`DbFieldInt(i)` read column `i` of the current row; `DbClose` closes; and
`DbError` returns the last error. Parameters are passed positionally
(placeholders `?`/`$1` depending on the driver). Functions/builtins with no parameters
can be called without parentheses (`DbEof`, `HttpLastStatus`); parentheses
are also valid. (A **procedural value** stored in a variable does
require `()` to invoke it, since the bare name is the value.)

The `MaxSteps`, `MaxHeap` and `MaxDuration` limits are enforced inside the VM and
stop the program with a runtime error (200 step/time, 203 heap) when
exceeded. Example of a strict configuration with time and memory caps:

```go
eng := vmpas.NewWith(vmpas.Capabilities{
    MaxSteps:    5_000_000,
    MaxHeap:     10_000,
    MaxDuration: 200 * time.Millisecond,
})
```

## Error checking before running

```go
err := eng.Run(`variable_inexistente := 5`)
// err != nil: "unknown identifier" caught at compile time
```

## Complete example

See `examples/embed/main.go`:

```bash
go run ./examples/embed
```

## Status and limitations

vmpas runs on the project's real compiler and VM. The procedural core and the
TP7 OOP object model are complete (procedures/functions with by-value, `var` and
`const` parameters, recursion, records, arrays, pointers, enums, sets, full
control flow, `object` types with inheritance, virtual dispatch, constructors and
`inherited`), as is the `uses` unit system and text/typed file I/O.

Known limitation: `inherited` is supported as a statement (`inherited Init(a)`)
but not yet inside an expression (`x := inherited Foo + y`). See the
[compatibility matrix](compatibility.md) for the full per-feature detail and
[architecture](architecture.md) for how the pieces fit together.
