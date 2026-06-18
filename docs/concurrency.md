# `spawn` y `Channel<T>` (concurrencia, modo moderno)

Extensiones modernas estilo Go, activadas con `{$MODE BPGO}`. Sin el directivo,
`spawn`, `MakeChan`, etc. siguen siendo identificadores normales (compatibilidad
total).

## Modelo

La concurrencia es **cooperativa**: un *scheduler* propio interleava **fibras**
(green threads) sobre un único hilo. Una fibra corre hasta que **se bloquea en
un canal** o termina; ahí el scheduler pasa a otra. No hay paralelismo real ni
data races a nivel de instrucción (solo una fibra ejecuta a la vez), pero sí
**concurrencia** real para coordinar trabajo (I/O, pipelines, productor/
consumidor).

Como el motor es dueño de todo el estado de las fibras (es dato plano), la
ejecución es **serializable** — la base del snapshot/replay determinístico
(fase F).

> Rendimiento: los programas que **no** usan `spawn`/canales corren por la ruta
> rápida de una sola fibra, **sin overhead de scheduler**. El scheduler solo se
> activa cuando hay concurrencia.

## `spawn`: lanzar una fibra

`spawn Sentencia` corre esa sentencia como una fibra nueva. La sentencia captura
el entorno **por referencia** (como una closure):

```pascal
spawn ch.Send(42);

spawn begin
  x := Trabajar;
  resultado.Send(x);
end;
```

Cuando **`main` termina, el programa termina** (como en Go), aunque queden
fibras vivas.

## `Channel<T>`: comunicación

Un canal comunica fibras. Se crea con `MakeChan` (sin buffer) o `MakeChan(n)`
(buffer de tamaño `n`):

```pascal
var ch: Channel<Integer>;
begin
  ch := MakeChan;       { sin buffer: Send espera a un Receive }
  ch := MakeChan(64);   { buffered: Send no bloquea hasta llenar el buffer }
```

Operaciones (sintaxis de método):

| Operación | Qué hace |
|---|---|
| `ch.Send(v)` | envía `v`; bloquea si no hay buffer/receptor |
| `ch.Receive` | recibe un valor; bloquea si no hay nada |
| `ch.Close` | cierra el canal |

`Receive` sobre un canal **cerrado y vacío** devuelve `nil` (no bloquea), lo que
permite detectar el cierre:

```pascal
if ch.Receive = nil then WriteLn('canal cerrado');
```

## Ejemplo: productor / consumidor

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

## Ejemplo: pedido / respuesta (dos canales)

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

Si **todas** las fibras quedan bloqueadas (nadie puede avanzar), el scheduler lo
detecta y termina con un **error de runtime** (no se cuelga):

```pascal
ch := MakeChan;
x := ch.Receive;   { nadie envía nunca -> error de runtime (deadlock) }
```

## `panic` en una fibra

Un `panic` dentro de una fibra desenrolla esa fibra (corriendo sus `defer`). Si
lo atrapás dentro de la fibra (con `try/except` o `recover`), el resto del
programa sigue:

```pascal
spawn begin
  try
    panic('boom');
  except
    ch.Send(-1);   { la fibra se recupera y avisa }
  end;
end;
```

## ⚠️ Cuidado: las variables globales se comparten

Todas las fibras **comparten las globales** (igual que en Go con una variable sin
sincronizar). Usar la misma global desde dos fibras concurrentes — por ejemplo
la **misma variable de bucle** — produce resultados incorrectos:

```pascal
{ MAL: 'i' es global y lo pisan ambas fibras }
spawn begin for i := 1 to N do ch.Send(i); end;
for i := 1 to N do total := total + ch.Receive;

{ BIEN: cada fibra con su propia variable }
spawn begin for j := 1 to N do ch.Send(j); end;
for i := 1 to N do total := total + ch.Receive;
```

Dentro de una **rutina**, en cambio, una fibra spawneada captura los **locales
por referencia** (closure), que es lo esperable y seguro para pasar datos a la
fibra.

## Limitaciones (honestas)

- **Cooperativa, sin preempción**: una fibra que nunca toca un canal (un bucle
  puro de CPU) no cede el control hasta terminar. La concurrencia ocurre en los
  puntos de canal.
- **Sin paralelismo real**: una fibra a la vez (es concurrencia, no paralelismo).
  Para el caso de uso embebido (coordinar I/O) es lo adecuado.
- **`select`** (multiplexar varios canales) todavía no está.
- Las globales se comparten sin sincronización automática (ver arriba).

## Ver también

- [defer / panic / recover](defer.md)
- [Compatibilidad TP7 y extensiones modernas](compatibilidad.md)
