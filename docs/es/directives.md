# Directivas del compilador BPGo

BPGo reconoce las directivas de compilador estándar de Turbo Pascal 7 /
Borland Pascal 7. La lista completa está definida en
`compat/spec/directives.json`; este documento resume la
semántica implementada en el backend de la vm.

## Directivas de conmutación

| Directiva | Predeterminado | Descripción                          |
|-----------|---------|--------------------------------------|
| `{$A+}`   | `{$A-}` | Alinea los campos de registro en límite de palabra |
| `{$A-}`   |         | Empaqueta los campos de forma ajustada |
| `{$B+}`   | `{$B-}` | Evaluación booleana: completa (siempre en vm) |
| `{$B-}`   |         | Evaluación de cortocircuito          |
| `{$D+}`   | `{$D-}` | Emite información de depuración       |
| `{$D-}`   |         | Sin información de depuración         |
| `{$E+}`   | `{$E-}` | Usa emulación del 8087                |
| `{$E-}`   |         | Sin emulación                         |
| `{$F+}`   | `{$F-}` | Fuerza llamadas far                   |
| `{$F-}`   |         | Permite llamadas near                 |
| `{$G+}`   | `{$G-}` | Usa instrucciones del 80286           |
| `{$G-}`   |         | Solo instrucciones del 8086           |
| `{$I+}`   | `{$I-}` | Habilita la comprobación de E/S       |
| `{$I-}`   |         | Sin comprobación de E/S               |
| `{$N+}`   | `{$N-}` | Usa el coprocesador numérico          |
| `{$N-}`   |         | Sin coprocesador numérico             |
| `{$Q+}`   | `{$Q-}` | Habilita la comprobación de desbordamiento de enteros |
| `{$Q-}`   |         | Sin comprobación de desbordamiento    |
| `{$R+}`   | `{$R-}` | Habilita la comprobación de rango     |
| `{$R-}`   |         | Sin comprobación de rango             |
| `{$S+}`   | `{$S-}` | Habilita la comprobación de pila      |
| `{$S-}`   |         | Sin comprobación de pila              |
| `{$V+}`   | `{$V-}` | Comprobación estricta de var-string   |
| `{$V-}`   |         | Comprobación relajada de var-string   |
| `{$X+}`   | `{$X-}` | Habilita la sintaxis extendida (llamadas a funciones en expresiones) |
| `{$X-}`   |         | Deshabilita la sintaxis extendida     |

## Directivas paramétricas

| Directiva              | Descripción                       |
|------------------------|-----------------------------------|
| `{$I filename}`        | Incluye un archivo en línea       |
| `{$L filename.obj}`    | Enlaza un archivo .obj OMF        |
| `{$M stack,heapmin,heapmax}` | Establece los tamaños de memoria en párrafos |
| `{$O unitname}`        | Marca una unidad como overlay     |

## Compilación condicional

| Directiva            | Descripción                  |
|----------------------|------------------------------|
| `{$DEFINE name}`     | Define un símbolo            |
| `{$UNDEF name}`      | Anula la definición de un símbolo |
| `{$IFDEF name}`      | Compila el código siguiente si está definido |
| `{$IFNDEF name}`     | Compila el código siguiente si no está definido |
| `{$IFOPT switch}`    | Compila el código siguiente si la conmutación está activada |
| `{$ELSE}`            | Rama else del condicional   |
| `{$ENDIF}`           | Fin del bloque condicional  |

## Notas sobre la implementación de BPGo

- El lexer descarta silenciosamente las directivas `{$...}` y `(*$...*)`
  durante la tokenización. El arnés de conformidad verifica que los
  nombres de las directivas sean reconocidos en `compat/spec/directives.json`.
- El analizador sem y el backend de la vm todavía no respetan estas
  conmutaciones; las directivas se registran en el mapa de fuente para
  su visualización en el IDE, pero por lo demás se tratan como no-op.
- `{$I file.inc}` y `{$L file.obj}` todavía no están conectados a las
  rutas de búsqueda de include / OMF.
