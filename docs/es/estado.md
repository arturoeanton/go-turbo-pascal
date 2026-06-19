# Estado, validación y viabilidad

go-turbo-pascal está en **v1.2.0**: el motor embebible y su API pública son
estables (congelada desde v1.0.0; ver [api.md](api.md)). Esta página registra cómo se
validó la versión, el panorama honesto de rendimiento (incluida una comparación
directa contra goja) y las limitaciones conocidas que permanecen.

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

Medido con `go test -bench -benchmem` (los números varían según la máquina; estos
son indicativos, Apple Silicon):

| Benchmark | Resultado |
|---|---|
| `vmpas` compile-once / run-many (suma 1..100) | ~24 µs/op, **12 allocs** |
| Suma 1..1000 | ~230 µs/op, **12 allocs**, ~1.9 KB |
| `fib(20)` recursivo | ~4.0 ms/op, **65 allocs** |
| loop intensivo en records (construir/copiar/leer) | ~840 µs/op, ~7.5k allocs |

Optimizaciones aplicadas a lo largo de la serie 1.x:
- **compile-once / run-many** en `vmpas` (`Engine.Compile` → `Script.Run`).
- **pool de frames** + binding de argumentos por slice del stack en la VM.
- **caché de builtins** por engine (no se re-registra la RTL en cada run): el
  loop corre a **12 allocs**.
- **v1.1.1** — marshalling O(n) de los argumentos de los builtins y un camino rápido
  para enteros en los operadores binarios (~5% en loops escalares).
- **v1.2.0** — los records usan un slice de asociación en lugar de un map, así el acceso
  a campos evita el hashing del map y una asignación por record: el código intensivo en
  records es ~7% más rápido con ~12% menos memoria y ~17% menos asignaciones.

### Benchmark directo vs goja

`internal/bench` es un **módulo Go aparte** (así goja no entra al `go.mod`
principal, que queda con cero dependencias). Correr con:
`cd internal/bench && go test ./... -bench . -benchmem`. Compila una vez y usa un
contexto fresco por run:

| Benchmark | vmpas | goja |
|---|---|---|
| Suma 1..1000 | ~230 µs, **12 allocs**, ~1.9 KB | ~157 µs, 3744 allocs, ~112 KB |
| `fib(20)` | ~4.0 ms, 65 allocs | ~1.5 ms, 72 allocs |

Lectura honesta: vmpas **asigna muchísima menos memoria** (12 vs 3744 allocs y
~59× menos bytes en el loop), mientras que **goja es ~1.5–2.6× más rápido en tiempo**.
goja es un intérprete de bytecode JS muy optimizado; la VM de vmpas usa una unión
etiquetada (`Value`) con boxing. Cerrar aún más la brecha de tiempo requeriría
optimizar el bucle del intérprete (despacho, evitar el boxing de `Value`, posible
diseño por registros) — un trabajo mayor, no de wiring, y un no-objetivo deliberado
por ahora. Donde vmpas aporta valor es en otro lado: **tipado fuerte previo a
ejecutar**, el **sandbox de capacidades**, ejecución durable y **cero
dependencias**.

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
  ~1.5–2.6× más rápido en tiempo bruto (ver el benchmark anterior). Cerrar aún más la
  brecha de tiempo supondría rehacer el despacho del intérprete — un no-objetivo
  deliberado por ahora.
- Una **IDE TUI** nostálgica al estilo Turbo Pascal no está planeada; `internal/tv` y
  `cmd/turbo` permanecen como stubs legacy.

## Veredicto

Ambos objetivos se cumplen y el núcleo está sólido y testeado. Los elementos restantes son la
estrecha brecha del lenguaje mencionada arriba y el rendimiento en tiempo bruto, ninguno de los cuales requiere
rediseño. Ver la [matriz de compatibilidad](compatibility.md) para el detalle por característica.
