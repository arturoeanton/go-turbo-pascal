# `spawn` and `Channel<T>` (concurrency, modern mode)

Modern Go-style extensions, enabled with `{$MODE BPGO}`. Without the directive,
`spawn`, `MakeChan`, etc. remain ordinary identifiers (full
compatibility).

## Model

Concurrency is **cooperative**: a built-in *scheduler* interleaves **fibers**
(green threads) on a single thread. A fiber runs until it **blocks on a
channel** or finishes; at that point the scheduler switches to another. There is
no real parallelism nor instruction-level data races (only one fiber runs at a
time), but there is real **concurrency** for coordinating work (I/O, pipelines,
producer/consumer).

Because the engine owns all of the fibers' state (it is plain data), execution
is **serializable** — the basis of the deterministic snapshot/replay
(phase F).

> Performance: programs that do **not** use `spawn`/channels run through the
> fast single-fiber path, **with no scheduler overhead**. The scheduler only
> activates when there is concurrency.

## `spawn`: launch a fiber

`spawn Statement` runs that statement as a new fiber. The statement captures
the environment **by reference** (like a closure):

```pascal
spawn ch.Send(42);

spawn begin
  x := Trabajar;
  resultado.Send(x);
end;
```

When **`main` finishes, the program finishes** (as in Go), even if fibers
remain alive.

## `Channel<T>`: communication

A channel connects fibers. It is created with `MakeChan` (unbuffered) or
`MakeChan(n)` (buffer of size `n`):

```pascal
var ch: Channel<Integer>;
begin
  ch := MakeChan;       { unbuffered: Send waits for a Receive }
  ch := MakeChan(64);   { buffered: Send does not block until the buffer fills }
```

Operations (method syntax):

| Operation | What it does |
|---|---|
| `ch.Send(v)` | sends `v`; blocks if there is no buffer/receiver |
| `ch.Receive` | receives a value; blocks if there is nothing |
| `ch.Close` | closes the channel |

`Receive` on a **closed and empty** channel returns `nil` (it does not block),
which lets you detect the close:

```pascal
if ch.Receive = nil then WriteLn('canal cerrado');
```

## Example: producer / consumer

```pascal
{$MODE BPGO}
program ProdCons;
var ch: Channel<Integer>; i, j, total: Integer;
begin
  ch := MakeChan(64);
  spawn begin
    for j := 1 to 100 do ch.Send(j);
  end;
  total := 0;
  for i := 1 to 100 do
    total := total + ch.Receive;
  WriteLn(total);          { 5050 }
end.
```

## Example: request / response (two channels)

```pascal
{$MODE BPGO}
program PingPong;
var req, resp: Channel<Integer>; x: Integer;
begin
  req := MakeChan;
  resp := MakeChan;
  spawn begin
    x := req.Receive;
    resp.Send(x * x);
  end;
  req.Send(9);
  WriteLn(resp.Receive);   { 81 }
end.
```

## Deadlock

If **all** fibers become blocked (no one can make progress), the scheduler
detects it and terminates with a **runtime error** (it does not hang):

```pascal
ch := MakeChan;
x := ch.Receive;   { no one ever sends -> runtime error (deadlock) }
```

## `panic` in a fiber

A `panic` inside a fiber unwinds that fiber (running its `defer`s). If you
catch it inside the fiber (with `try/except` or `recover`), the rest of the
program continues:

```pascal
spawn begin
  try
    panic('boom');
  except
    ch.Send(-1);   { the fiber recovers and reports }
  end;
end;
```

## ⚠️ Caution: global variables are shared

All fibers **share the globals** (just like in Go with an unsynchronized
variable). Using the same global from two concurrent fibers — for example
the **same loop variable** — produces incorrect results:

```pascal
{ WRONG: 'i' is global and both fibers clobber it }
spawn begin for i := 1 to N do ch.Send(i); end;
for i := 1 to N do total := total + ch.Receive;

{ RIGHT: each fiber with its own variable }
spawn begin for j := 1 to N do ch.Send(j); end;
for i := 1 to N do total := total + ch.Receive;
```

Inside a **routine**, on the other hand, a spawned fiber captures the **locals
by reference** (closure), which is the expected and safe way to pass data to the
fiber.

## Limitations (honest)

- **Cooperative, no preemption**: a fiber that never touches a channel (a pure
  CPU loop) does not yield control until it finishes. Concurrency happens at
  channel points.
- **No real parallelism**: one fiber at a time (it is concurrency, not
  parallelism). For the embedded use case (coordinating I/O) it is the right fit.
- **`select`** (multiplexing several channels) is not available yet.
- Globals are shared without automatic synchronization (see above).

## See also

- [defer / panic / recover](defer.md)
- [TP7 compatibility and modern extensions](compatibility.md)
