# `defer`, `panic` y `recover` (modo moderno)

Estas son **extensiones modernas** del lenguaje (estilo Go), no parte de Turbo
Pascal 7. Se activan solo con `{$MODE BPGO}` al inicio del fuente. Sin él, el
compilador es TP7 estricto y `defer`, `panic`, `recover` siguen siendo
identificadores normales (compatibilidad total).

Se construyen sobre el mismo mecanismo de excepciones que `try/except/finally`.

## 1. `defer`: limpieza garantizada al salir

`defer Sentencia` agenda esa sentencia para que corra **cuando la rutina (o el
programa) termina**, sin importar por qué camino salió: retorno normal, `exit`,
o un `panic`. Es la forma idiomática de garantizar limpieza (cerrar archivos,
liberar recursos) cerca de donde se adquiere el recurso.

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

Salida:

```
cuerpo
3
2
1
```

Los `defer` corren en **orden inverso** (LIFO): el último agendado es el primero
en ejecutarse. Eso permite deshacer en el orden correcto lo que se hizo.

### Patrón típico: adquirir y diferir la liberación

```pascal
procedure Procesar;
begin
  Assign(f, 'datos.txt');
  Reset(f);
  defer Close(f);     { se cierra pase lo que pase }
  { ... trabajar con f; aunque algo falle, Close(f) corre ... }
end;
```

### `defer` condicional

Un `defer` solo se ejecuta si el control efectivamente pasó por él:

```pascal
procedure Run(abrir: Boolean);
begin
  if abrir then defer WriteLn('cerrar');
  WriteLn('trabajo');
end;
```

`Run(true)` imprime `trabajo` y luego `cerrar`; `Run(false)` solo `trabajo`.

## 2. `panic`: abortar con un valor

`panic(v)` lanza una excepción con el valor `v` y comienza a desenrollar la pila.
Mientras desenrolla, **los `defer` de cada rutina se ejecutan** (por eso la
limpieza sigue ocurriendo aunque haya panic).

```pascal
panic('algo salió muy mal');
```

Si nadie lo recupera (ver abajo) ni hay un `try/except` que lo atrape, el
programa termina con un error de runtime.

## 3. `recover`: recuperarse de un panic

`recover`, llamado **dentro de un `defer`**, detiene la propagación de un panic:
devuelve el valor con el que se llamó a `panic` y hace que la rutina que estaba
en panic **retorne normalmente**. Si no hay ningún panic activo, devuelve `nil`.

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

- `Safe(1)`: no hay panic; el `defer` llama `recover` (devuelve `nil`), el
  resultado queda en `'completado'`.
- `Safe(0)`: `panic('boom')` desenrolla; el `defer` corre, `recover` captura el
  panic (≠ `nil`), pone el resultado en `'recuperado'` y la función retorna
  normalmente con ese valor.

### `defer` corre también cuando el panic se propaga

Si no recuperás, el panic sigue subiendo, pero los `defer` igual corren:

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

Salida:

```
cleanup
atrapado
```

El `defer` de `Inner` corre durante el desenrollado, y luego el `try/except`
del programa atrapa el panic.

## Relación con `try/except/finally`

`defer`/`panic`/`recover` y `try/except/finally` conviven y se complementan:

| Quiero… | Uso |
|---|---|
| Limpieza ligada a un recurso, cerca de donde lo adquiero | `defer` |
| Abortar con un valor | `panic` (o `raise`) |
| Recuperarme y seguir | `recover` (en un `defer`) o `try/except` |
| Limpieza de un bloque acotado | `try/finally` |

`panic` es esencialmente `raise` con nombre de Go; `recover` es una forma cómoda
de atrapar sin escribir un `try/except` explícito.

## Limitaciones (honestas)

- **`defer` dentro de un bucle corre una sola vez** al salir, no una vez por
  iteración (a diferencia de Go). Para limpieza por iteración, usá `try/finally`
  dentro del bucle.
- La **sentencia diferida lee los valores actuales al momento de salir**, no una
  copia tomada al momento del `defer`.
- `recover` solo tiene efecto dentro de un `defer` que corre durante un panic;
  en cualquier otro lugar devuelve `nil`.

## Ver también

- [match / tipos suma / Option](match.md)
- [Compatibilidad TP7 y extensiones modernas](compatibilidad.md)
