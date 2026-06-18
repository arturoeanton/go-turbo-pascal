# Plan y auditoría (qué falta y cómo priorizarlo)

Estado: el núcleo (compilador real → VM, OOP, I/O, units), el motor embebible
`vmpas` con sandbox, y el tooling (LSP + DAP + plugin VSCode) están completos y
testeados. Este documento audita el repo y prioriza lo que falta para los dos
casos de uso.

## Hoja de ruta D (orden de ejecución acordado)

Orden: **D1 → D11 → D10 → D7 → D2 → D4 → D3 → D5 → D6**. Primero correcciones de
trampas y la propuesta de valor del motor embebible (correctitud, velocidad,
seguridad, DX), luego el bloque de lenguaje Object Pascal moderno.

| ID | Título | Tipo | Descripción | Esfuerzo |
|----|--------|------|-------------|----------|
| D1 | Properties funcionales | Corrección | `property X read F write F` parsea pero se ignora en runtime; implementar read/write contra campo backing y métodos getter/setter. | M |
| D11 | Performance vs goja | Feature | Optimizar boxing de `Value` y despacho del intérprete para cerrar la brecha de tiempo (~43% en el loop); vmpas ya gana 58× en memoria. | L |
| D10 | Sandbox real (Net/Exec/Env + límites) | Feature | Flags reservados sin RTL que gobernar; hoy el sandbox solo cubre FS + MaxSteps. Añadir RTL sensible + enforcement + límite mem/tiempo. | M |
| D7 | LSP IDE-grade | Feature | `pls` hoy solo da diagnósticos; añadir hover, completion, goto-def y símbolos. | L |
| D2 | Variant records: implementar o error | Corrección | `flattenVariant()` es stub vacío; la parte `case` se descarta sin avisar. Mínimo: error claro; ideal: variantes reales. | S/L |
| D4 | Closures / métodos anónimos | Feature | No existe; `procedure`/`function` anónimos + captura de entorno. | XL |
| D3 | Interfaces | Feature | `class(TParent, IFoo)` se acepta pero no se modela; falta contrato + dispatch. | L |
| D5 | Genéricos `<T>` | Feature | No existe; instanciación/monomorfización. | XL |
| D6 | Operator overloading | Feature | Operadores hardcodeados por tipo; declarar `operator +` y resolver dispatch. | M |

Esfuerzo: S = horas · M = ~1 sesión · L = varias sesiones · XL = grande/multi-sesión.

> **Serie D: completa** (v0.1.2 → v0.1.3). Todas las features D1–D11 implementadas,
> testeadas y commiteadas. Además, **fase de integración** (v0.1.4 → v0.1.6):
> unit HTTP (todos los verbos + headers), JSON (leer + construir), y SQL sobre
> `database/sql` con driver inyectado por el host (cero-deps preservado).

## Hoja de ruta E (lenguaje moderno + determinismo)

Posicionamiento del producto: **el motor de scripting embebible para Go**. La
serie E suma features de lenguaje modernas y la Fase F agrega el diferenciador
estructural (ejecución determinística + snapshot). No se persigue paridad con
Delphi/Lazarus ni una IDE TUI (descartada).

**Regla transversal de compatibilidad:** toda sintaxis nueva se activa solo bajo
el directivo `{$MODE BPGO}` (E1); sin él, el compilador es TP7 puro. Las palabras
nuevas son **keywords contextuales** en ese modo. Esto blinda la compatibilidad.

Orden: **E1 → E2 → E3 → E4 → E5 → E6 → F**.

| ID | Título | Tipo | Descripción | Esfuerzo | Depende |
|----|--------|------|-------------|----------|---------|
| E1 | Gate de modo `{$MODE BPGO}` | Infra | Directivo de modo + keywords contextuales. Prerequisito de toda sintaxis nueva; sin él, TP7 puro. Barato y desbloquea todo. | S/M | — |
| E2 | Ergonomía: inferencia local + `let` | Feature | `var x := expr` infiere el tipo del inicializador; `let x = expr` binding inmutable (error al reasignar). Solo compile-time, runtime intacto. | M | E1 |
| E3 | Helpers + unit tests integrados | Feature | `record helper`/`class helper` de Delphi = extension methods (reusa el dispatch de métodos; permite extender tipos Go mapeados). Bloque `test "..." begin..end` + aserciones + runner en la CLI (reusa el VM + excepciones; corre en el sandbox). | M | E1 |
| E4 | `match` + `Option/Some/None` | Feature | Tipos suma (ADTs con payload) + `match expr of Patrón => …; else …; end` con binding por destructuring. Absorbe la null-safety honesta (Option, no Kotlin-style). Sub-fases: (a) ADTs+match; (b) exhaustividad (best-effort o diferida). No reusar `with` (ya es keyword). | L | E1 |
| E5 | `defer` / `panic` / `recover` | Feature | `panic`/`recover` mapean sobre `raise`/`except` existentes; `defer` = lista LIFO por frame ejecutada al retornar (reusa la maquinaria de `finally`). | M | E1 |
| E6 | `spawn` + `Channel<T>` | Feature | Concurrencia estilo Go con **scheduler cooperativo** (green threads en un VM single-thread): `spawn` crea una fibra, el VM interleava `Step()`, los canales bloquean/ceden. **Reescritura del loop del VM** (N pilas + scheduler + puntos de yield). La apuesta grande; decisión consciente de inversión. | XL | E1 |
| F (núcleo) | Ejecución determinística + snapshot/resume | Moat | Serializar todo el estado del VM (stack, frames, heap, globals, fibras) a bytes: pausar, reanudar, migrar, time-travel, replay determinístico. Único estructuralmente (goja/Lua no pueden). Habilita durable workflows / funciones reanudables. Requiere `Value`/heap 100% serializables (sin punteros Go crudos). | L/XL | E6 |
| F (opcional) | Sandbox probable | Moat | Gas determinístico (extiende `MaxSteps`) + **inferencia de capacidad mínima** (analizar el código para derivar qué capacidades necesita) + allowlists finas + traza auditable. Se evalúa al final, una vez F-núcleo firme. | M/L | F-núcleo, E4 |

Descartadas para enfocar (de las 5 ideas de moat): puente Go↔guest profundo,
verificación estática (taint/contratos), artefacto portable/WASM.

**Optimización de velocidad (diferida):** la reescritura grande del intérprete
(NaN-boxing / VM de registros) se hace **después** de E6 + F, sobre el diseño del
VM ya congelado — y solo la parte que una **medición real** justifique (en el
caso embebible, el cuello suele ser red/DB, no el bucle del VM). Las
optimizaciones quirúrgicas baratas (estilo D11) se hacen ad-hoc cuando una
medición lo pida.

> **Progreso reciente:** A1 (compile-once/run-many en `vmpas`) ✅, A2 (pool de
> frames + args por slice; `fib(20)` de ~65k a ~88 allocs) ✅, B1 (unit `Crt`
> conectada) ✅. Limpieza: `cmd/bprun` (muerto) y `internal/bgi` (vacío)
> eliminados; estructura real vs legacy documentada en `arquitectura.md`.

## Auditoría del repositorio

**Camino real (vivo):** `ast` → `lexer` → `parser` → `codegen` → `ir`; CLIs
`bpgo`/`pasrun`, `pls` (LSP), `pdap` (DAP); `pkg/vmpas`. RTL: **solo `system`**
está conectada. `compile`+`conformance` quedan solo para `bpgo test-compat`.

**Legacy / experimental (compila, fuera del camino real):**
- `internal/codegen8086`, `internal/mz`, `internal/omf` — backend 8086 / EXE MZ.
- `internal/tv/*` (14 paquetes) — stubs de Turbo Vision (para la futura IDE TUI).
- `internal/ide` + `cmd/turbo` — IDE ANSI vieja.
- `internal/debug` + `cmd/tdebug` — debugger viejo (reemplazado por `ir.Debugger`/DAP).
- `internal/rtl/{crt,dos,strings,graph,graph3,overlay,printer,turbo3,windos}` —
  implementadas en Go pero **sin `Register(vm)`**, no expuestas como builtins.

**Roto / a resolver:**
- `cmd/bprun` — espera `.bpi`; el motor nuevo no serializa IR → no funcional.
- `cmd/tdebug` — usa el debugger viejo, no el nuevo.
- `internal/bgi` — vacío.

## Caso A — Pascal embebido en Go (alternativa a goja)

| # | Tarea | Esfuerzo | Prioridad |
|---|---|---|---|
| A1 | **compile-once / run-many**: `vmpas.Compile(code)` → programa reutilizable; hoy `Run` recompila siempre | M | Alta |
| A2 | **Reducir allocs en llamadas**: pool de frames / slots reusables (bajar los ~65k allocs de `fib(20)`) | M | Alta |
| A3 | **Más mapeo de tipos Go↔Pascal**: slices↔arrays, maps, punteros a struct, métodos de structs Go como procs/métodos | M-L | Alta |
| A4 | **Benchmark directo vs goja** (suite publicable) | S | Media |
| A5 | **Sandbox + fuerte**: límites de memoria/tiempo configurables, allowlist de FS por ruta | S | Media |
| A6 | **Ergonomía de API**: errores con posición, timeout/contexto, `MustRun` | S | Baja |

Orden sugerido: **A1 → A2 → A4** (perf demostrable) → **A3** (cobertura) → A5/A6.

## Caso B — Alternativa TP7 en consola

| # | Tarea | Esfuerzo | Prioridad |
|---|---|---|---|
| B1 | **Conectar la RTL al motor** vía `uses`: dar `Register(vm)` a las units y registrarlas. Empezar por **`Crt`** (ClrScr, GotoXY, TextColor/Background, KeyPressed, ReadKey, Delay, Window) sobre una abstracción de terminal ANSI | M | Alta |
| B2 | **`with`** (resolver campos del record en el cuerpo) | S-M | Alta |
| B3 | **Strings TP7**: `ShortString[N]`, indexado 1-based `s[i]`, builtins (Copy/Pos/Delete/Insert/Length/Concat/UpCase) con semántica y por-referencia correctas | M | Alta |
| B4 | **Sets completos**: operadores `+ - *` y `<= >=` | S | Media |
| B5 | **Archivos tipados/binarios**: `file of T`, Read/Write/Seek/BlockRead/Write | M | Media |
| B6 | **`goto`/`label`** | S | Media |
| B7 | **Records variantes** (`case` en record) | S-M | Baja |
| B8 | **`case` con rangos y char** | S | Baja |
| B9 | **IDE TUI** sobre tcell (diferida) | L | Baja |

Orden sugerido: **B1 (Crt) → B2 (with) → B3 (strings)** cubren la mayoría de
programas de consola TP7; luego B4–B8; B9 al final.

## Fase C — evaluación de Object Pascal moderno

Para decidir el salto de "TP7 clásico" a "Pascal moderno". Esfuerzo: XS/S/M/L/XL.

| # | Feature | Esfuerzo | Qué toca | Riesgo | Valor |
|---|---|---|---|---|---|
| C3 | **Arrays dinámicos** (`array of T`, `SetLength`, `Length`/`High`) | M | parser + codegen + VM (VKArray ya es slice) | bajo | alto |
| C7 | **`for..in`** | S | parser + codegen (desugar a índice) | bajo | medio |
| C2 | **Excepciones** (`try..except..finally`, `raise`, `on E:`) | M-L | lexer + parser + VM (unwinding de frames; interactúa con el frame pool) | medio-alto | alto |
| C1 | **`class`** (propiedades, interfaces, `create`/`free`) | L | parser + sem + codegen + VM (semántica por referencia, VMT ya existe; interfaces = tablas por interfaz) | medio-alto | alto |
| C4 | **Strings modernos** (`AnsiString`/Unicode) | S–M | el `String` actual ya es dinámico (Go string, UTF-8); declarar `AnsiString` es barato; Unicode real (índice por code point) es medio | bajo–medio | medio |
| C6 | **Métodos anónimos / closures** | L | parser + VM (captura de entorno; hoy las funciones son top-level sin captura) | alto | medio |
| C5 | **Genéricos** | XL | parser + sem + codegen (monomorfización/especialización) | alto | alto (avanzado) |

Orden recomendado por valor/esfuerzo:
**C3 → C7 → C2 → C1 → C4 → C6 → C5.**
Notas: C3 es el de mejor relación (la VM ya tiene arrays como slices); C7 se hace
junto a C3; C1 se apoya en el modelo OOP `object`/VMT ya existente (class ≈ object
por referencia + create/free + properties + interfaces); C2 y C6 requieren tocar
el motor (unwinding / clausuras). La velocidad vs goja se ataca **después** de
todo esto (optimizar el bucle del intérprete: boxing de `Value` y despacho).

## Limpieza / deuda técnica

| # | Acción | Esfuerzo |
|---|---|---|
| C1 | Backend 8086/MZ (`codegen8086`/`mz`/`omf`): marcar experimental o retirar | S |
| C2 | `cmd/bprun`: serializar IR a `.bpi` desde codegen **o** retirar el comando | S |
| C3 | `cmd/tdebug`: reapuntar al nuevo `ir.Debugger` **o** retirar en favor de `pdap` | S |
| C4 | `internal/debug`, `internal/ide`, `internal/tv/*`: marcar "pre-TUI/legacy" | S |
| C5 | `internal/bgi`: quitar (vacío) | XS |
| C6 | `sem`: integrarlo en codegen para chequeos más fuertes **o** documentar que codegen resuelve por su cuenta | M |

## Recomendación de priorización

- **Si el objetivo inmediato es A (motor embebible competitivo):** A1, A2, A4.
- **Si el objetivo inmediato es B (apps de consola TP7):** B1 (Crt), B2, B3.
- **Transversal recomendado:** C5 (trivial), C2/C3 (coherencia de CLIs), y test
  de regresión que ejecute `examples/interactive/*.pas` con sus entradas.
