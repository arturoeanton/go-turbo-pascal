# Errores de BPGo

El catálogo de diagnósticos de BPGo refleja los números de error de
Turbo Pascal 7 / Borland Pascal 7. La lista completa está definida en
`internal/diagnostics/diagnostics.go`; este documento resume
las categorías.

## Errores de compilación (1-99)

| Código | Nombre                         | Notas |
|------|--------------------------------|-------|
| 1    | Símbolo inesperado             | Token inesperado en una declaración o sentencia |
| 2    | Se esperaba un identificador   | Palabra reservada usada donde se requiere un identificador |
| 3    | Identificador desconocido      | El nombre no está declarado en el ámbito actual |
| 4    | Identificador duplicado        | El nombre ya está declarado en este ámbito |
| 5    | Error de sintaxis              | Error de sintaxis genérico |
| 6    | Error en constante real        | El literal real está mal formado |
| 7    | Error en constante entera      | El literal entero está mal formado o fuera de rango |
| 8    | La constante de cadena excede la línea | Use concatenación con `+` para abarcar varias líneas |
| 9    | Cadena sin terminar            | Falta la `'` de cierre |
| 10   | Se esperaba comilla de cierre  | Igual que 9 en modo estricto |
| 11   | Se esperaba `=`                 | Se requiere `=` en declaraciones const o type |
| 12   | Se esperaba `:=`                | Se requiere `:=` para la asignación |
| 13   | Se esperaba un identificador de tipo | Se esperaba un nombre de tipo |
| 14   | Se esperaba `of`                | Se requiere `of` después de `array`/`case`/`set`/`file` |
| 15   | Se esperaba `.`                 | Se esperaba un `.` (p. ej. fin de unidad) |
| 16   | Demasiados procedimientos anidados | Reduzca el anidamiento o use unidades |
| 17   | Tipo incorrecto                | El tipo no es válido en este contexto |
| 18   | Se esperaba `END`              | El bloque actual debe cerrarse con `END` |
| 26   | Tipos incompatibles            | Los tipos de origen y destino no son compatibles |
| 27   | Tipo base de subrango inválido | El tipo base del subrango debe ser ordinal |
| 28   | Límite inferior > límite superior | Intercambie los límites |
| 29   | Se esperaba un ordinal         | Se requiere un tipo ordinal |
| 30   | Se esperaba una constante entera | Proporcione una constante entera |
| 31   | Se esperaba una constante      | Proporcione una expresión constante |
| 32   | Se esperaba un entero o real   | Proporcione una constante numérica |
| 33   | Se esperaba un tipo puntero    | Use `^T` |
| 34   | Tipo de resultado de función inválido | Use un tipo escalar, puntero o cadena |
| 35   | Se esperaba un identificador de etiqueta | Proporcione una etiqueta numérica |
| 36   | Se esperaba BEGIN              | Inicie el bloque con BEGIN |
| 37   | Parte de sentencias demasiado grande | Divídala en procedimientos más pequeños |
| 38   | Se esperaba DO                 | Agregue DO |
| 39   | Se esperaba THEN               | Agregue THEN |
| 40   | Demasiadas variables           | Reduzca el número de variables o divida la unidad |
| 41   | Tipo no definido               | Declare el tipo |
| 42   | Archivo no permitido aquí      | Los archivos tienen restricciones en este contexto |
| 43   | Discrepancia en longitud de cadena | Las cadenas de origen y destino difieren en la longitud declarada |
| 44   | Se esperaba una constante de cadena | Use un literal de cadena |
| 45   | Se esperaba una variable entera o real | Proporcione una variable numérica |
| 46   | Se esperaba una variable ordinal | Proporcione una variable ordinal |
| 47   | Se esperaba una expresión de carácter | Proporcione una expresión compatible con Char |
| 48   | Se esperaba una variable estructurada | Proporcione un record/array/file |
| 49   | Se esperaba una expresión constante | Use una constante |
| 50   | Se esperaba una expresión entera | Use una expresión entera |
| 51   | Se esperaba una expresión booleana | Use una expresión booleana |
| 52   | Los tipos de operandos no coinciden | El operador no está definido para estos tipos |
| 53   | Se esperaba un identificador de campo | Use un nombre de campo de record |
| 54   | Archivo de objeto demasiado grande | Reduzca el código o divida la unidad |
| 55   | Externo no definido            | Proporcione el símbolo o la biblioteca externos |
| 56   | Registro de archivo de objeto inválido | El registro OMF no es compatible |
| 57   | Segmento de código demasiado grande | El código no puede exceder 64KB sin overlays |
| 58   | Segmento de datos demasiado grande | Los datos no pueden exceder 64KB |
| 84   | Discrepancia en el nombre de la unidad | El identificador de la unidad no coincide con el nombre del archivo |
| 85   | Discrepancia en la versión de la unidad | Recompile la unidad |
| 86   | Nombre de unidad duplicado     | Una unidad aparece dos veces en uses |
| 87   | Ciclo de unidades detectado    | Elimine los uses circulares |
| 88   | Unidad no encontrada           | Agregue la ruta de la unidad o cree la unidad |

## Errores de tiempo de ejecución (1-255)

Los errores de tiempo de ejecución se reportan como códigos enteros por el
runtime de la unidad System (`RunError`) y el bucle de mensajes del IDE.

| Código | Nombre                            |
|-------|-----------------------------------|
| 1     | Número de función inválido        |
| 2     | Archivo no encontrado             |
| 3     | Ruta no encontrada                |
| 4     | Demasiados archivos abiertos      |
| 5     | Acceso al archivo denegado        |
| 6     | Identificador de archivo inválido |
| 12    | Código de acceso a archivo inválido |
| 15    | Número de unidad inválido         |
| 16    | No se puede eliminar el directorio actual |
| 17    | No es el mismo dispositivo        |
| 18    | No hay más archivos               |
| 100   | Error de lectura de disco         |
| 101   | Error de escritura de disco       |
| 102   | Archivo no asignado               |
| 103   | Archivo no abierto                |
| 104   | Archivo no abierto para entrada   |
| 105   | Archivo no abierto para salida    |
| 106   | Formato numérico inválido         |
| 150   | División por cero                 |
| 151   | Error de comprobación de rango    |
| 152   | Desbordamiento de pila            |
| 153   | Desbordamiento de heap            |
| 154   | Operación de puntero inválida     |
| 155   | Desbordamiento de punto flotante  |
| 156   | División por cero en punto flotante |
| 157   | Operación de punto flotante inválida |
| 158   | Subdesbordamiento de punto flotante |
| 159   | Desbordamiento de enteros         |
| 160   | Operación variant inválida        |
| 161   | Conversión de tipo variant inválida |
| 162   | Error de despacho                 |
| 200   | División por cero (bucle de retardo) |
| 201   | Comprobación de rango             |
| 202   | Desbordamiento de pila            |
| 203   | Desbordamiento de heap            |
| 204   | Puntero inválido                  |
| 205   | Desbordamiento de punto flotante  |
| 206   | Subdesbordamiento de punto flotante |
| 207   | Opcode del 8087 inválido          |

## Errores de E/S

| Código | Nombre                |
|-------|-----------------------|
| 2     | Archivo no encontrado |
| 3     | Ruta no encontrada    |
| 5     | Acceso denegado       |
| 32    | Violación de uso compartido |
| 100   | Error de lectura de disco |
| 101   | Error de escritura de disco |

## Errores de Graph

| Código | Nombre                            |
|-------|-----------------------------------|
| 0     | Sin error                         |
| -1    | Gráficos no inicializados         |
| -2    | Hardware de gráficos no detectado |
| -3    | Archivo de controlador no encontrado |
| -4    | Controlador inválido              |
| -5    | Memoria insuficiente para cargar el controlador |
| -6    | Memoria insuficiente para el relleno por barrido |
| -7    | Memoria insuficiente para el relleno por inundación |
| -8    | Archivo de fuente no encontrado   |
| -9    | Fuente inválida                   |
| -10   | Modo inválido                     |
| -11   | Relleno inválido                  |
| -12   | Índice de paleta fuera de rango   |
| -13   | Búfer de imagen inválido          |
| -14   | Sin memoria                       |
| -15   | Estilo de línea inválido          |
| -16   | Fuera del viewport                |
| -17   | Viewport inválido                 |

## Errores de overlay

| Código | Nombre                    |
|-------|---------------------------|
| 0     | OK                        |
| -1    | Error de overlay          |
| -2    | Archivo de overlay no encontrado |
| -3    | Sin memoria               |
| -4    | Error de lectura de overlay |

## Errores de depuración

| Código | Nombre                        |
|-------|-------------------------------|
| 1     | No hay fuente para la dirección |
| 2     | Punto de interrupción inválido |
| 3     | Símbolo no encontrado         |
| 4     | El proceso no se está ejecutando |
