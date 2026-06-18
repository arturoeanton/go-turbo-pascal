# Matriz de compatibilidad con TP7

`go-turbo-pascal` es una implementación clean-room en Go de un front-end y motor
compatible con Turbo Pascal 7 / Borland Pascal 7. No se incluye ningún binario,
código fuente ni documentación de Borland. Este documento expone con honestidad
qué funciona hoy en el **motor real** (`internal/lexer` → `parser` → `sem` →
`codegen` → `ir` VM) y qué es legacy o queda fuera de alcance.

> **Cómo leer esto.** "Soportado" significa que compila y se ejecuta en el motor
> y está cubierto por pruebas en `internal/codegen`, `internal/ir`,
> `internal/e2e`, `internal/integration` o `pkg/vmpas`. Cuando hay una limitación
> conocida, se indica de forma explícita en lugar de ocultarse.

## Lenguaje — núcleo procedural

| Característica | Estado | Notas |
|---|---|---|
| Procedimientos y funciones | ✅ | frame propio, recursión, variables locales |
| Parámetros: valor / `var` / `const` | ✅ | `var` es verdadero paso por referencia (aliasing de celdas) |
| Resultado de función (asignación a `Result`/nombre) | ✅ | |
| Enteros, `Real`, `Boolean`, `Char` | ✅ | |
| `ShortString` (`string[N]`) | ✅ | byte de longitud + indexación basada en 1 |
| Registros, registros anidados | ✅ | semántica de copia por valor |
| Registros variantes (`case` en registro) | ✅ | `testdata/pas/variant.pas` |
| Arrays estáticos, multidimensionales | ✅ | arrays anidados, indexación con comprobación de rango |
| Enumeraciones y subrangos | ✅ | mapeo ordinal |
| Conjuntos (`+ - *`, `in`, comparaciones) | ✅ | `testdata/pas/sets.pas` |
| Punteros (`^T`, `@`, `New`/`Dispose`, `nil`) | ✅ | punteros a heap y a celdas |
| Referencias de tipo forward (`PNode = ^TNode`) | ✅ | `TNode` puede declararse después |
| Control de flujo `if`/`case`/`for`/`while`/`repeat` | ✅ | |
| `with` | ✅ | resolución de selectores |
| `break`/`continue`/`Exit`/`goto`/`label` | ✅ | |
| `Inc`/`Dec`, formato `:w:d` de `Write`/`WriteLn` | ✅ | formato tipado |

## Lenguaje — POO (modelo de objetos de TP7)

| Característica | Estado | Notas |
|---|---|---|
| Tipos `object` con campos | ✅ | |
| Herencia `object(Parent)` | ✅ | disposición de campos heredada |
| Métodos (procedure/function) | ✅ | |
| Métodos `virtual` + despacho dinámico | ✅ | polimorfismo real vía puntero base (VMT) |
| `constructor` / `destructor` | ✅ | |
| `inherited` — forma de **sentencia** | ✅ | `inherited Init(a)`, `inherited Draw` |
| `inherited` — forma de **expresión** | ❌ | `x := inherited Foo + y` aún no se parsea |

La única limitación de POO es `inherited` usado dentro de una expresión. Como
sentencia funciona; por eso `testdata/pas/objectpoly.pas` (que escribe
`GetX := inherited GetX + Y`) es el único programa del corpus que se omite en las
pruebas end-to-end de POO.

## Lenguaje — extensiones modernas

Estas son adiciones de go-turbo-pascal sobre TP7, útiles al embeber:

| Característica | Estado | Doc |
|---|---|---|
| `match` / tipos suma / `Option` | ✅ | [match.md](match.md) |
| `defer` / `panic` / `recover` | ✅ | [defer.md](defer.md) |
| `spawn` / canales | ✅ | [concurrency.md](concurrency.md) |

## Biblioteca de runtime (unidades)

Implementadas como paquetes Go conectados a la VM e importables vía `uses`. Ver
[units.md](units.md) para el mapa de símbolos por unidad.

| Unidad | Estado | Notas |
|---|---|---|
| `System` | ✅ | implícita; memoria, E/S, cadenas, matemáticas, ordinales, conjuntos |
| `Crt` | ✅ | `ClrScr`/`GotoXY`/`TextColor`/`KeyPressed`/`ReadKey`, pantalla virtual 80×25 |
| `Dos` | ✅ | fecha/hora, entorno, búsqueda de archivos, servicios en sandbox |
| `Strings` | ✅ | helpers de `PChar` (`StrCat`, `StrComp`, …) |
| `WinDos` | ✅ | variantes `PChar` de los servicios de Dos |
| `Printer` | ✅ | archivo `Lst` (en memoria o respaldado por archivo) |
| `Graph` / `Graph3` | ✅ | framebuffer por software, paleta, viewports, primitivas |
| `Turbo3` / `Overlay` | ✅ | variables de archivo de TP3; gestor de overlays (contadores) |
| Sistema de unidades `uses` | ✅ | interface/implementation/initialization, RTL ligado a la VM |
| E/S de archivos (texto y tipados) | ✅ | `Assign`/`Reset`/`Rewrite`/`Read`/`Write`/`Close`/`Eof`/`Append` |

## Embedding y herramientas

| Componente | Estado | Doc |
|---|---|---|
| `pkg/vmpas` — motor embebible | ✅ | [vmpas.md](vmpas.md) |
| Binding Go ↔ Pascal (vars, funcs, struct↔record) | ✅ | [vmpas.md](vmpas.md) |
| Sandbox de capacidades (FS/red/exec/env/db + límites) | ✅ | [seguridad.md](seguridad.md) |
| Ejecución durable (snapshot/resume determinista) | ✅ | [durable.md](durable.md) |
| Inferencia de capacidades (`Analyze`) + log de auditoría | ✅ | [vmpas.md](vmpas.md) |
| Servidor LSP (`cmd/pls`) | ✅ | [editores.md](editores.md) |
| Adaptador de depuración DAP (`cmd/pdap`) | ✅ | [editores.md](editores.md) |
| Plugin de VSCode (sintaxis + LSP + depuración) | ✅ | [editores.md](editores.md) |
| Plugin de Zed (LSP) | ✅ | [editores.md](editores.md) |

## Directivas de compilador

Las directivas se tokenizan y se parsean; su efecto en runtime varía — ver
[directives.md](directives.md) para la tabla por directiva. En resumen:
interruptores como `{$R+}`/`{$I+}` se aceptan y son en su mayoría no-ops en el
backend de bytecode; la inclusión `{$I file.inc}` y el enlazado `{$L file.obj}`
no están conectados al backend de la VM.

## Legacy / experimental (fuera del camino principal)

Estos compilan y tienen pruebas pero **no** forman parte del motor soportado. Se
conservan por historia y experimentación:

| Componente | Estado |
|---|---|
| `internal/compile` + `internal/conformance` | harness stub mínimo detrás de `bpgo test-compat` (2 smoke tests); la cobertura real es la suite de pruebas del motor |
| `internal/codegen8086` | emite ensamblador 8086 textual; no se ensambla a un programa ejecutable |
| `internal/mz` | escribe una cabecera MZ EXE sintácticamente válida; sin sección de código funcional |
| `internal/omf` | lee solo THEADR/LNAMES/SEGDEF/PUBDEF/EXTDEF |
| `internal/tv/*` (Turbo Vision) | stubs de vista no funcionales |
| `cmd/turbo` (IDE estilo TP7) | stub de shell interactivo/headless |
| `cmd/tdebug` (depurador CLI) | sustituido por `cmd/pdap` / `internal/ir.Debugger` |

## Fuera de alcance

El ensamblador en línea, los overlays en el sentido DOS, los punteros far, la
generación real de código MZ EXE y la compatibilidad binaria con DOS **no** son
objetivos.

## Medirlo por tu cuenta

```bash
go test ./...                 # suite completa de pruebas del motor + biblioteca
go run ./cmd/pasrun x.pas     # ejecuta un .pas real en el motor
go run ./cmd/bpgo test-compat # harness de conformidad legacy → compat/report.json
```

`compat/report.json` lo produce el harness legacy y reporta cobertura de símbolos
por unidad, directivas y diagnósticos; trata la suite de pruebas del motor como
la señal autoritativa.
