# Arquitectura

go-turbo-pascal lleva Turbo Pascal 7 a Go con dos objetivos:

1. **Embeber Pascal en Go** a través de la biblioteca `pkg/vmpas`.
2. Proporcionar herramientas modernas para `.pas` (LSP + DAP, plugins de editor).

## Pipeline de compilación

```
Pascal source (.pas)
   │
   ▼  internal/lexer      → tokens
   ▼  internal/parser     → AST (internal/ast)
   ▼  internal/sem        → semantic analysis / types
   ▼  internal/codegen    → bytecode IR  ← real compiler
   ▼  internal/ir         → bytecode VM that executes the IR
```

### Front-end (lexer / parser / ast)

Sólido y estable. Cubre la mayor parte de la sintaxis de TP7: procedimientos y
funciones (parámetros por valor, `var`, `const`), records (incluidas variantes),
arrays, strings, conjuntos, enums, subrangos, punteros, todo el control de flujo
(`if/case/for/while/repeat/with`), OOP (`object`, métodos, herencia) y units.

### internal/codegen — el compilador real

Recorre el AST y emite bytecode IR. Hoy soporta:

- Procedimientos y funciones con su propio *frame*, parámetros por valor y `var`
  (referencia), recursión y variables locales.
- Records (acceso a campos, semántica de copia por valor), arrays estáticos
  (incl. multidimensionales como arrays anidados), punteros (`New`/`Dispose`,
  `^`, `@`, comparación con `nil`), enums y conjuntos.
- Control de flujo completo, `Inc`/`Dec`, `Write`/`WriteLn`, y un conjunto de
  builtins de la RTL.
- Opciones para embebedores (`pkg/vmpas`): externals (funciones del host /
  procedimientos de la RTL), globales preseteados, y auto-declaración de
  variables de bucle para fragmentos.

El compilador "de juguete" anterior (`internal/compile`) se conserva para el
arnés de conformance mientras `internal/codegen` se convierte en el camino
principal.

### internal/ir — la VM de bytecode

Una VM de pila con:

- Un modelo de **referencias por celda** (`*Value`): globales, slots de frame,
  campos de record, elementos de array y celdas de heap son direccionables.
  Esto da semántica correcta para `var` params, punteros y mutación anidada.
- Una convención de llamada con frames (parámetros primero, luego locales) y un
  *slot* de resultado de función.
- Valores con tipo en runtime: entero, real, booleano, char, string, conjunto,
  array, record, puntero, archivo y `nil`.

## Embebido: pkg/vmpas

`pkg/vmpas` es una fachada sobre `codegen` + `ir`:

- Compila el código (con verificación de tipos previa) y lo ejecuta en la VM.
- Enlaza variables Go (sembrándolas en los globales y leyéndolas de vuelta),
  mapea structs Go ↔ records Pascal por reflexión, y registra funciones Go como
  builtins de la VM.
- Aplica el **sandbox de capacidades** decidiendo qué builtins se registran.

Regla arquitectónica: `pkg/vmpas` se mantiene **cero-dependencias** (lo verifica
`TestVMPasHasNoExternalDeps`). Cualquier dependencia externa (tcell, servidores
LSP/DAP) vive fuera de ese árbol de imports.

## RTL

`internal/rtl/*` implementa las units estándar (System, Crt, Dos, Strings...) como
funciones Go registradas como builtins de la VM, importables mediante un sistema real
de units `uses` (interface/implementation/initialization).

## Herramientas

- **LSP + DAP**: un servidor de lenguaje (`cmd/pls`: diagnósticos/hover/completion/
  ir-a-definición) sobre el front-end, y un adaptador de depuración (`cmd/pdap`:
  breakpoints/step/variables) sobre la VM.
- **Plugins Zed y VSCode**: clientes finos sobre LSP/DAP (ver [editores.md](editores.md)).
- **TUI + IDE TP7** (no planeado actualmente): una IDE nostálgica al estilo Turbo Pascal
  sobre tcell. Los stubs de `internal/tv` y `cmd/turbo` son legacy, no están en el
  camino soportado.

## Estructura del proyecto (camino real vs. legacy)

Para no confundir: estos son los componentes **activos** (el camino real) y los
**legacy/experimentales** que se conservan pero no están en ese camino.

**Activo / camino real:**
- `internal/{lexer,parser,ast}` → `internal/codegen` → `internal/ir` (compilador + VM).
- `internal/rtl/system`, `internal/rtl/crt` (RTL conectada a la VM).
- `internal/cli` + `cmd/bpgo` (CLI principal), `cmd/pasrun` (runner), `cmd/tpc`.
- `internal/lsp` + `cmd/pls` (LSP); `internal/dap` + `cmd/pdap` (depuración).
- `pkg/vmpas` (motor embebible) + `examples/`.

**Legacy / experimental (compila y testea, fuera del camino real):**
- `internal/compile` + `internal/conformance` — solo para `bpgo test-compat`.
- `internal/codegen8086`, `internal/mz`, `internal/omf` — backend 8086 / EXE MZ
  (experimental; no es objetivo actual).
- `internal/tv/*`, `internal/ide`, `cmd/turbo` — IDE estilo Turbo Vision antigua
  (la IDE TUI sobre tcell es una fase futura).
- `internal/debug`, `cmd/tdebug` — debugger CLI antiguo (la depuración moderna es
  `cmd/pdap` / `internal/ir.Debugger`).
- `internal/sem` — análisis semántico; el codegen actual resuelve tipos por su
  cuenta, así que `sem` está poco usado en el camino real.

La auditoría completa y el plan priorizado están en [`plan.md`](plan.md).

## Fuera de alcance (por ahora)

Ensamblador inline, overlays, punteros far, generación de EXE MZ real y
compatibilidad binaria con DOS.
