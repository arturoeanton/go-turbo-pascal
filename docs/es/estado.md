# Estado, validación y viabilidad

Informe de la fase de validación previa a la TUI: qué se probó, qué funciona, y
qué falta para que el proyecto sea viable como (A) lenguaje Pascal embebido en
Go y (B) alternativa TP7 en consola.

## Validación (todo verde)

- `go build ./...` ✅
- `go test ./... -count=1` ✅ — **469 tests** PASS, 0 fallos.
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

Lectura honesta: **vmpas gana en memoria** (12 vs 3744 allocs en el loop) pero
**goja es ~1.6–3.3× más rápido en tiempo**. goja es un intérprete de bytecode
JS muy optimizado; la VM de vmpas usa una unión etiquetada (`Value`) con boxing.
Para superar a goja en tiempo haría falta optimizar el bucle del intérprete
(despacho, evitar el boxing de `Value`, posible diseño por registros) — un
trabajo mayor, no de wiring. Los diferenciadores de vmpas siguen siendo el
**tipado fuerte previo a ejecutar**, el **sandbox de capacidades** y **cero
dependencias**.

## (A) Pascal embebido en Go — VIABLE, con trabajo de perf/binding

Listo:
- Compila y chequea tipos **antes** de ejecutar (fuerte vs. motores dinámicos).
- Binding Go↔Pascal: variables escalares y **structs ↔ records**, y funciones Go
  llamables desde Pascal.
- **Sandbox de capacidades** (FS/red/exec/límites) — diferenciador real frente a
  goja.
- **Cero dependencias** (test lo garantiza).

Para cerrar la viabilidad (ser claramente competitivo vs goja):
1. **API compile-once / run-many**: hoy `Engine.Run` recompila en cada llamada;
   exponer un programa compilado reutilizable.
2. **Micro-optimizar llamadas**: pool de frames, evitar asignar slices de slots
   por llamada (reduce los ~65k allocs de `fib(20)`).
3. **Más mapeo de tipos**: slices/arrays y maps Go↔Pascal, punteros a struct,
   exponer métodos de structs Go como métodos de objeto.
4. **Benchmark directo vs goja** para publicar números.

## (B) Alternativa TP7 en consola — VIABLE para consola; faltan piezas de RTL

Listo:
- Procedural + **OOP** + control de flujo + **I/O de consola y archivos de
  texto** + **units** (`uses`).
- `pasrun` ejecuta `.pas` reales; tooling moderno (LSP + depuración DAP) en
  VSCode.

Para una experiencia TP7 de consola fiel, faltan (orden sugerido):
1. **Unit `Crt` funcional**: `ClrScr`, `GotoXY`, colores, `KeyPressed`/`ReadKey`
   — es lo más usado por apps de consola TP7.
2. **`with`**, **archivos tipados/binarios**, **records variantes**.
3. **Strings**: semántica `ShortString[N]` e indexado 1-based de caracteres.
4. **Sets**: operadores `+ - *` de conjunto (hoy: literales y `in`).
5. **`goto`/`label`**.
6. (diferida) la **IDE TUI** estilo Turbo Pascal.

## Veredicto

Ambas direcciones son **viables** y el núcleo está sólido y testeado. Para (A) el
trabajo restante es de rendimiento y amplitud de binding; para (B) es cobertura
de RTL (sobre todo `Crt`) y algunas características de lenguaje. Ninguno requiere
rehacer el diseño: son extensiones incrementales sobre lo ya construido. Ver la
[matriz de compatibilidad](compatibility.md) para el detalle por característica.
