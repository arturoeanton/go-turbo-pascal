# `defer`, `panic` and `recover` (modern mode)

These are **modern language extensions** (Go-style), not part of Turbo
Pascal 7. They are enabled only with `{$MODE BPGO}` at the start of the source.
Without it, the compiler is strict TP7 and `defer`, `panic`, `recover` remain
ordinary identifiers (full compatibility).

They are built on the same exception mechanism as `try/except/finally`.

## 1. `defer`: guaranteed cleanup on exit

`defer Statement` schedules that statement to run **when the routine (or the
program) finishes**, no matter which path it exited through: normal return,
`exit`, or a `panic`. It is the idiomatic way to guarantee cleanup (closing
files, releasing resources) near where the resource is acquired.

```pascal
{$MODE BPGO}
program P;
begin
  defer WriteLn('1');
  defer WriteLn('2');
  defer WriteLn('3');
  WriteLn('cuerpo');
end.
```

Output:

```
cuerpo
3
2
1
```

The `defer`s run in **reverse order** (LIFO): the last one scheduled is the
first to execute. This lets you undo, in the correct order, what was done.

### Typical pattern: acquire and defer the release

```pascal
procedure Procesar;
begin
  Assign(f, 'datos.txt');
  Reset(f);
  defer Close(f);     { it is closed no matter what }
  { ... work with f; even if something fails, Close(f) runs ... }
end;
```

### Conditional `defer`

A `defer` only executes if control actually passed through it:

```pascal
procedure Run(abrir: Boolean);
begin
  if abrir then defer WriteLn('cerrar');
  WriteLn('trabajo');
end;
```

`Run(true)` prints `trabajo` and then `cerrar`; `Run(false)` only `trabajo`.

## 2. `panic`: abort with a value

`panic(v)` raises an exception with the value `v` and begins unwinding the stack.
While unwinding, **each routine's `defer`s are executed** (which is why
cleanup still happens even when there is a panic).

```pascal
panic('algo salió muy mal');
```

If no one recovers it (see below) and there is no `try/except` to catch it, the
program terminates with a runtime error.

## 3. `recover`: recover from a panic

`recover`, called **inside a `defer`**, stops the propagation of a panic:
it returns the value `panic` was called with and makes the routine that was
panicking **return normally**. If there is no active panic, it returns `nil`.

```pascal
{$MODE BPGO}
program P;

function Safe(n: Integer): string;
begin
  Safe := 'ok';
  defer
    if recover <> nil then
      Safe := 'recuperado';
  if n = 0 then panic('boom');
  Safe := 'completado';
end;

begin
  WriteLn(Safe(1));   { completado }
  WriteLn(Safe(0));   { recuperado }
end.
```

- `Safe(1)`: there is no panic; the `defer` calls `recover` (returns `nil`), the
  result stays `'completado'`.
- `Safe(0)`: `panic('boom')` unwinds; the `defer` runs, `recover` captures the
  panic (≠ `nil`), sets the result to `'recuperado'` and the function returns
  normally with that value.

### `defer` also runs when the panic propagates

If you do not recover, the panic keeps rising, but the `defer`s still run:

```pascal
procedure Inner;
begin
  defer WriteLn('cleanup');
  panic('x');
end;

begin
  try
    Inner;
  except
    WriteLn('atrapado');
  end;
end.
```

Output:

```
cleanup
atrapado
```

`Inner`'s `defer` runs during the unwinding, and then the program's
`try/except` catches the panic.

## Relationship with `try/except/finally`

`defer`/`panic`/`recover` and `try/except/finally` coexist and complement each
other:

| I want to… | Use |
|---|---|
| Cleanup tied to a resource, near where I acquire it | `defer` |
| Abort with a value | `panic` (or `raise`) |
| Recover and continue | `recover` (in a `defer`) or `try/except` |
| Cleanup of a bounded block | `try/finally` |

`panic` is essentially `raise` with a Go name; `recover` is a convenient way
to catch without writing an explicit `try/except`.

## Limitations (honest)

- **A `defer` inside a loop runs only once** on exit, not once per
  iteration (unlike Go). For per-iteration cleanup, use `try/finally`
  inside the loop.
- The **deferred statement reads the current values at the moment of exit**, not
  a copy taken at the moment of the `defer`.
- `recover` only has effect inside a `defer` that runs during a panic;
  anywhere else it returns `nil`.

## See also

- [match / sum types / Option](match.md)
- [TP7 compatibility and modern extensions](compatibility.md)
