# Quickstart

A quick tour of **go-turbo-pascal**: running Pascal, embedding it in Go, the
modern extensions (`{$MODE BPGO}`) and the integration with HTTP/JSON/SQL. For
the short install/editor guide see [`inicio.md`](inicio.md).

## 0. Requirements and build

```bash
# Go 1.23+
go build ./...                 # compiles everything
go test ./...                  # runs the suite
go build -o bin/pasrun ./cmd/pasrun   # the .pas program runner
```

## 1. Running a Pascal program

```pascal
{ hola.pas }
program Hola;
begin
  WriteLn('Hola, mundo!');
end.
```

```bash
bin/pasrun hola.pas        # -> Hola, mundo!
```

It is real Turbo Pascal 7: procedures/functions, records, arrays (static and
dynamic), sets, pointers, `class`/`object` with inheritance and virtual methods,
interfaces, generics, closures, exceptions, units, files. See the matrix in
[`compatibilidad.md`](compatibility.md).

## 2. Modern mode: `{$MODE BPGO}`

With the `{$MODE BPGO}` directive at the start, modern extensions are enabled. Without
it, the compiler is **strict TP7** (full compatibility: `let`, `match`,
`spawn`, etc. remain normal identifiers).

```pascal
{$MODE BPGO}
program ModernoDemo;

type
  TShape = (Circle(Integer), Rect(Integer, Integer));   { sum types / ADTs }

function Area(s: TShape): Integer;
begin
  Area := match s of                       { match as an expression }
    Circle(r)  => r * r * 3;
    Rect(w, h) => w * h;
  end;
end;

let factor = 2;                            { immutable binding }
var total := 0;                            { type inference }
begin
  total := Area(Rect(3, 4)) * factor;      { 24 }
  WriteLn(total);

  match total of                           { guards and or-patterns }
    0          => WriteLn('cero');
    24, 48     => WriteLn('múltiplo esperado');
    _ when total > 0 => WriteLn('positivo');
    else       WriteLn('otro');   { in a match statement the else has no => }
  end;
end.
```

The modern features, in dedicated guides:

- **[match / sum types / Option](match.md)** — `match`, ADTs, `Some`/`None`.
- **[defer / panic / recover](defer.md)** — guaranteed cleanup and panic handling.
- **[spawn / channels](concurrency.md)** — cooperative concurrency.
- Others (in [`compatibilidad.md`](compatibility.md)): local inference, `let`
  immutable, *extension methods* (`record/class helper`), integrated unit tests
  (`test … AssertEqual …`).

### Concurrency in 6 lines

```pascal
{$MODE BPGO}
program ProdCons;
var ch: Channel<Integer>; i, j, total: Integer;
begin
  ch := MakeChan(64);
  spawn begin for j := 1 to 100 do ch.Send(j); end;
  total := 0;
  for i := 1 to 100 do total := total + ch.Receive;
  WriteLn(total);          { 5050 }
end.
```

## 3. Embedding Pascal in Go (`pkg/vmpas`)

The `vmpas` engine runs Pascal inside a Go program, with binding of
variables/functions, a **capability sandbox** and **zero external dependencies**.

```go
package main

import (
    "fmt"
    "github.com/arturoeanton/go-turbo-pascal/pkg/vmpas"
)

func main() {
    eng := vmpas.New() // restricted sandbox by default
    total := 10
    eng.Var("total", &total)
    eng.Function("Triple", func(x int) int { return x * 3 }) // Go function callable from Pascal
    eng.Run(`total := Triple(total) + 5`)
    fmt.Println(total) // 35
}
```

`vmpas` maps Go structs ↔ Pascal records, exposes Go functions/methods, and
allows *compile-once / run-many*. Details in [`vmpas.md`](vmpas.md); runnable
example in `examples/embed`.

### Capability sandbox

By default (`New()` / `Restricted()`) filesystem, network, exec,
environment and database are **denied**. You grant only what is needed, with limits:

```go
eng := vmpas.NewWith(vmpas.Capabilities{
    Network:     true,
    MaxSteps:    5_000_000,
    MaxDuration: 200 * time.Millisecond,
})
```

## 4. Integration: consuming APIs and SQL

Under the corresponding capabilities, the embedded Pascal code can consume
HTTP and databases (the host injects the driver; the engine remains dependency-free):

```pascal
body := HttpGet('https://api.example.com/users');   { Network }
name := JsonStr(body, 'user.name');                  { JSON: read by path }
req  := JsonSetStr('{}', 'n', '1');                  { JSON: build }
HttpPost(url, 'application/json', req);

if DbOpen('SELECT id, name FROM users') then         { Database (UseDB) }
  while not DbEof do
  begin
    WriteLn(DbFieldInt(0), ' ', DbFieldStr(1));
    DbNext;
  end;
DbClose;
```

Self-contained example (local server + in-memory database, offline):

```bash
go run ./examples/integration
```

Details in the integration section of [`vmpas.md`](vmpas.md).

### Host-level parallelism

Each `vmpas.Engine` is independent (share-nothing). For real parallel
workloads, the Go host launches several engines in goroutines:

```go
for i := 0; i < n; i++ {
    go func() {
        eng := vmpas.NewWith(caps)
        eng.Run(script)   // its own VM, no shared state
    }()
}
```

## 5. Editor tooling (LSP / DAP)

```bash
make tools && export PATH="$PWD/bin:$PATH"   # pls (LSP) and pdap (DAP)
```

- **`pls`** — diagnostics, hover, go-to-definition, symbols, autocompletion.
- **`pdap`** — breakpoints, step, variable inspection.
- **VSCode** (LSP + debugging) and **Zed** (LSP) extensions in `editors/`.

See [`editores.md`](editores.md).

## Next

- Compatibility matrix and extensions: [`compatibilidad.md`](compatibility.md)
- Compiler/VM architecture: [`arquitectura.md`](arquitectura.md)
- Plan and roadmap: [`plan.md`](plan.md)
