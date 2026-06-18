# Estado, validación y viabilidad

go-turbo-pascal está en **v1.0.0**: el motor embebible y su API son estables.
Esta página registra cómo se validó la versión, el panorama honesto de rendimiento
(incluida una comparación directa contra goja) y las limitaciones conocidas que
permanecen.

## Validación (todo verde)

- `go build ./...` ✅
- `go test ./... -count=1` ✅ — **600+ tests** PASS, 0 fallos.
- `gofmt -l` ✅ vacío (formateado).
- `go vet` ✅ salvo avisos en el código **legacy** `internal/tv` (stubs viejos de
  Turbo Vision, fuera del camino nuevo).
- Herramientas (`pasrun`, `pls`, `pdap`) compilan vía `make tools`.
- Ejemplos: `factorial`, `listas`, `figuras`, `calc` (interactivo), `units/demo`
  y `examples/embed` (Pascal en Go) corren correctamente.
- LSP y DAP probados end-to-end (diagnósticos; sesión de depuración completa).

### Rendimiento (compilar una vez, ejecutar muchas)

| Benchmark | Resultado |
|---|---|
| Loop suma 1..10000 | ~2.5 ms/op (~36M instr/s), 35 allocs |
| `fib(20)` recursivo | ~4.8 ms/op, **88 allocs** (antes ~65k) |
| `vmpas` recompila cada Run | ~46 µs/op, 265 allocs |
| `vmpas` compile-once / run-many | ~34 µs/op, 94 allocs |

Optimizaciones aplicadas (A1/A2/A4):
- **compile-once / run-many** en `vmpas` (`Engine.Compile` → `Script.Run`).
- **pool de frames** + binding de argumentos por slice del stack en la VM
  (`fib(20)` bajó de ~65k a ~88 allocs).
- **caché de builtins** por engine (no se re-registra la RTL en cada run): el
  loop bajó a **12 allocs**.

### Benchmark directo vs goja (A4)

`internal/bench` es un **módulo Go aparte** (así goja no entra al `go.mod`
principal, que queda con cero dependencias). Correr con:
`cd internal/bench && go test ./... -bench . -benchmem`. Compila una vez y usa un
contexto fresco por run:

| Benchmark | vmpas | goja |
|---|---|---|
| Suma 1..1000 | ~252 µs, **12 allocs** | ~157 µs, 3744 allocs |
| `fib(20)` | ~5.0 ms, 65 allocs | ~1.5 ms, 72 allocs |

Lectura honesta: vmpas **asigna mucha menos memoria** (12 vs 3744 allocs en el
loop), mientras que **goja es ~1.6–3.3× más rápido en tiempo**. goja es un
intérprete de bytecode JS muy optimizado; la VM de vmpas usa una unión etiquetada
(`Value`) con boxing. Cerrar la brecha de tiempo requeriría optimizar el bucle
del intérprete (despacho, evitar el boxing de `Value`, posible diseño por
registros) — un trabajo mayor, no de wiring. Donde vmpas aporta valor es en otro
lado: **tipado fuerte previo a ejecutar**, el **sandbox de capacidades**,
ejecución durable y **cero dependencias**.

## (A) Pascal embebido en Go — entregado

- Compila y chequea tipos **antes** de ejecutar (tipado fuerte vs. motores dinámicos).
- Binding Go↔Pascal: variables escalares, **structs ↔ records**, slices ↔ arrays,
  y funciones/métodos Go llamables desde Pascal.
- **compile-once / run-many** (`Engine.Compile` → `Script.Run`).
- **Sandbox de capacidades** (FS/red/exec/env/db + límites de step/heap/output/time) e
  **inferencia de capacidades** (`Analyze`) — capacidades que la mayoría de los motores embebidos dejan al host.
- **Ejecución durable**: snapshot/resume determinista.
- **Cero dependencias** (test lo garantiza).

## (B) TP7 en consola — entregado

- Procedural + **OOP** + control de flujo completo + **I/O de consola y de archivos de texto/tipados**
  + un sistema real de **units** (`uses`).
- La unit `Crt` (`ClrScr`/`GotoXY`/colores/`KeyPressed`/`ReadKey`), `with`,
  records variantes, `ShortString[N]` con indexado 1-based, operadores de conjuntos y
  `goto`/`label` funcionan todos.
- `pasrun` ejecuta `.pas` reales; el tooling moderno (depuración LSP + DAP) funciona en VSCode
  y Zed.

## Limitaciones conocidas

- `inherited` funciona como sentencia pero todavía no dentro de una expresión
  (`x := inherited Foo + y`). Ver la [matriz de compatibilidad](compatibility.md).
- **Rendimiento vs goja**: vmpas asigna mucha menos memoria, mientras que goja es
  ~1.6–3.3× más rápido en tiempo bruto (ver el benchmark anterior). Cerrar la brecha
  de tiempo supondría rehacer el despacho del intérprete — un no-objetivo deliberado para v1.0.0.
- Una **IDE TUI** nostálgica al estilo Turbo Pascal no está planeada; `internal/tv` y
  `cmd/turbo` permanecen como stubs legacy.

## Veredicto

Ambos objetivos se cumplen y el núcleo está sólido y testeado. Los elementos restantes son la
estrecha brecha del lenguaje mencionada arriba y el rendimiento en tiempo bruto, ninguno de los cuales requiere
rediseño. Ver la [matriz de compatibilidad](compatibility.md) para el detalle por característica.
