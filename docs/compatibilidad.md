# Matriz de compatibilidad TP7

Estado real del soporte respecto a Turbo Pascal 7. Leyenda: ✅ soportado ·
🚧 parcial · ❌ no soportado (todavía).

## Lenguaje

| Característica | Estado | Notas |
|---|---|---|
| Procedimientos y funciones | ✅ | parámetros por valor y `var`, recursión, locales, frames |
| Resultado de función | ✅ | por nombre de función o `Result` |
| Records | ✅ | campos, anidados, semántica de copia por valor |
| Records variantes (`case`) | ⚠️ | campo selector y campos de cada variante aplanados y accesibles; no se fuerza el layout de unión (cada campo ocupa su propio espacio) |
| Arrays estáticos | ✅ | rangos arbitrarios; multidimensionales como anidados (`a[i][j]`) |
| Arrays dinámicos | ✅ | `array of T`, `SetLength`, `Length`/`High`/`Low`, índice 0-based; crecer conserva datos |
| `for..in` | ✅ | sobre arrays y strings |
| Arrays abiertos (open array params) | ❌ | |
| Punteros | ✅ | `^T`, `@`, `New`/`Dispose`, deref `^`, comparación con `nil` |
| Enumerados y subrangos | ✅ | subrango tratado como ordinal |
| Conjuntos (sets) | ✅ | literales, `in`, operadores `+ - *`, comparación `= <> <= >=` |
| Strings | ✅ | dinámicos (UTF-8), concatenación, I/O, indexado 1-based `s[i]`; `AnsiString`/`WideString`/`UnicodeString`/`ShortString`/`PChar` son alias del String dinámico |
| Conversiones / SysUtils | ✅ | IntToStr, StrToInt(Def), FloatToStr, StrToFloat, UpperCase, LowerCase, Trim/TrimLeft/TrimRight, StringOfChar, Copy/Pos/Length/UpCase |
| Char | ✅ | |
| `if` / `case` / `for` / `while` / `repeat` | ✅ | |
| `break` / `continue` / `exit` | ✅ | |
| `with` (incl. `with a, b`) | ✅ | |
| `goto` / `label` | ✅ | |
| Excepciones (`try..except`, `try..finally`, `raise`) | ✅ | catch-all + propagación entre llamadas; `on E: T do` (handlers tipados) y binding del objeto: pendiente |
| OOP `object` | ✅ | campos, métodos, constructor/destructor |
| Herencia y métodos virtuales | ✅ | despacho dinámico (vtable + tag de tipo en runtime) |
| `inherited` | 🚧 | forma sentencia (`inherited Init(...)`); en expresión: ❌ |
| `Self` | ✅ | |
| `class` (estilo Delphi) | ✅ | tipo por referencia: `Create` (asigna), métodos, herencia, métodos virtuales (despacho dinámico), `Free`, nil por defecto |
| Propiedades (`property X read F write F`) | ✅ | campo de respaldo y métodos getter/setter (`read GetX write SetX`), incl. especificadores mixtos |
| Tipos procedurales y closures | ✅ | `type T = procedure/function(...)`, valor de rutina con `@R`, métodos anónimos `procedure/function(...) begin..end` con captura por referencia; llamar un valor en expresión requiere `()` |
| Interfaces | ✅ | `IFoo = interface ... end`, `class(TBase, IFoo)`, variable de tipo interfaz con despacho dinámico al tipo concreto; sin conteo de referencias ni verificación estricta de implementación (duck-typing por tag de runtime) |
| Genéricos (`TList<T>`, `Max<T>`) | ✅ | tipos y rutinas genéricas por **type erasure**: el parámetro de tipo se borra a "any" (el VM es dinámicamente tipado). Sin monomorfización ni chequeo de los argumentos de tipo; la instanciación `TList<Integer>` se erasa a `TList` |
| Sobrecarga de operadores | ✅ | forma FPC global `operator + (a, b: T): T; begin Result := ...; end;`; despacho por tipos de operandos para `+ - * /` y comparaciones; usar `Result` en el cuerpo. Operandos deben ser identificadores/campos (el tipo de un operando que es a su vez una expresión compuesta no se infiere) |
| Units (`uses`) | ✅ | interface / implementation / initialization de units de usuario |
| `finalization` | 🚧 | se parsea, no se ejecuta |
| Compilación separada `.tpu` | ❌ | las units se compilan desde fuente |
| Directivas `{$...}` | 🚧 | se parsean y se ignoran |
| Constantes tipadas | 🚧 | |

## RTL (runtime)

| Característica | Estado | Notas |
|---|---|---|
| `Write` / `WriteLn` | ✅ | formato de campo `x:ancho` y `x:ancho:dec` |
| `Read` / `ReadLn` (consola) | ✅ | por referencia, parseo por tipo |
| I/O de archivos de texto | ✅ | `Assign`/`Reset`/`Rewrite`/`Append`/`Close`/`Erase`, `Write`/`Read` a archivo, `Eof` |
| Archivos tipados / binarios | ✅ | `file of <escalar>` (Integer/Real/Char/Boolean), registros de 8 bytes, Read/Write/Seek/FilePos/FileSize/Eof; `file of record`: pendiente |
| Funciones de System | ✅ | `Ord` `Chr` `Abs` `Sqr` `Sqrt` `Sin` `Cos` `Ln` `Exp` `Trunc` `Round` `Length` `Copy` `Pos` `Inc` `Dec` … |
| División real `/` vs `div` | ✅ | `/` siempre Real, `div` entera |
| Unit `Crt` | ✅ | conectada vía `uses Crt`: ClrScr, ClrEol, GotoXY, TextColor/Background, NormVideo/HighVideo/LowVideo, Delay, ReadKey, KeyPressed, WhereX/Y, Window/Sound/NoSound (salida como ANSI) |
| Unit `Dos` | 🚧 | implementada en Go, aún sin conectar como builtins |

## Embebido y tooling

| Característica | Estado | Notas |
|---|---|---|
| Compilar y ejecutar (`pasrun`, `pkg/vmpas`) | ✅ | compilador real → bytecode → VM |
| `vmpas`: binding de variables Go | ✅ | escalares y `struct` ↔ `record` |
| `vmpas`: llamar funciones Go desde Pascal | ✅ | |
| `vmpas`: sandbox de capacidades | ✅ | `Restricted` / `Full` (FS, etc.) |
| LSP: diagnósticos | ✅ | `cmd/pls` |
| LSP: hover / completion / go-to-def | ❌ | |
| DAP: breakpoints / step / variables | ✅ | `cmd/pdap` |
| Plugin VSCode | ✅ | sintaxis + LSP + depuración |
| Plugin Zed | 🚧 | LSP listo; DAP pendiente de la API de Zed |

## Extensiones modernas (`{$MODE BPGO}`)

Las siguientes features **no son TP7**: se activan solo con el directivo
`{$MODE BPGO}` al inicio del fuente. Sin él, el compilador es TP7 estricto y las
palabras nuevas (`let`, `test`, `helper`) siguen siendo identificadores
normales, preservando la compatibilidad.

| Feature | Estado | Notas |
|---|---|---|
| `{$MODE BPGO}` | ✅ | gate de compatibilidad; sin él, TP7 puro |
| Inferencia local | ✅ | `var x := expr` infiere el tipo del inicializador (y `var x: T = expr`) |
| `let` inmutable | ✅ | `let x = expr` / `let x := expr`; reasignar es error de compilación |
| Extension methods (helpers) | ✅ | `record helper for T` / `class helper for T` (estilo Delphi); despacho estático por el tipo del receptor; `Self`/campos del tipo extendido accesibles |
| Unit tests integrados | ✅ | `test 'nombre' begin … end`; `AssertTrue`/`AssertFalse`/`AssertEqual` (una aserción fallida lanza y se reporta como FAIL); runner que imprime PASS/FAIL por test |

## Fuera de alcance (por ahora)

Ensamblador inline, overlays, punteros far, generación de EXE MZ real y
compatibilidad binaria con DOS.
