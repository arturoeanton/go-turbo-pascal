# vmpas — Pascal embebido en Go

`pkg/vmpas` es un **motor de código dinámico embebible** para Go. Permite
ejecutar código Turbo Pascal 7 dentro de un programa Go, enlazando variables,
funciones y structs del anfitrión. A diferencia de los motores de scripting
dinámicos (p. ej. JavaScript con goja), vmpas **compila y verifica tipos antes
de la primera ejecución**: los errores de compilación se detectan al instante,
no en medio de una ejecución.

Características:

- **Fuertemente tipado**: compilado a bytecode y validado antes de ejecutarse.
- **Mapeo Go ↔ Pascal**: variables escalares y `struct` ↔ `record`.
- **Llamadas bidireccionales**: Pascal puede llamar a funciones/procedimientos de Go.
- **Sandbox de capacidades**: control granular sobre lo que puede hacer el código
  invitado (sistema de archivos, etc.), con `Restricted` (por defecto) y `Full`.
- **Cero dependencias externas**: importar `vmpas` no arrastra tcell ni la
  cadena de herramientas del editor (garantizado por un test).

## Instalación

```go
import "github.com/arturoeanton/go-turbo-pascal/pkg/vmpas"
```

## Uso básico

```go
eng := vmpas.New() // sandbox restringido por defecto
if err := eng.Run(`WriteLn('Hola, mundo!')`); err != nil {
    log.Fatal(err)
}
fmt.Print(eng.Output()) // "Hola, mundo!\n"
```

Puedes pasar un fragmento (envuelto automáticamente en un programa) o un
programa completo (`program ... end.`).

## Compilar una vez, ejecutar muchas

`Run` compila en cada llamada. Cuando ejecutas el mismo código repetidamente (un
camino caliente, un motor de reglas evaluado por petición), compílalo una vez en
un `Script` y reutilízalo:

```go
script, err := eng.Compile(code) // lex + parse + sem + codegen, una sola vez
if err != nil {
    log.Fatal(err) // todos los errores de compilación/tipo afloran aquí, por adelantado
}
for _, row := range rows {
    eng.Var("row", &row)
    script.Run()              // ejecuta el programa cacheado
    fmt.Print(script.Output())
}
```

Este es el patrón recomendado para el rendimiento; consulta los
[benchmarks](estado.md) (el bucle baja a ~12 asignaciones/ejecución).

## Cancelación y lectura de resultados

Para ejecución acotada por petición, `RunContext` aborta la ejecución poco
después de que el contexto se cancela (o vence su deadline) y devuelve
`ctx.Err()`:

```go
ctx, cancel := context.WithTimeout(r.Context(), 200*time.Millisecond)
defer cancel()
if err := eng.RunContext(ctx, tenantScript); err != nil {
    // errors.Is(err, context.DeadlineExceeded) cuando expiró el tiempo
}
```

`Script.RunContext(ctx)` hace lo mismo para un script compilado. La cancelación
es cooperativa: la VM consulta el contexto con la misma cadencia regulada que
`MaxDuration`.

Para extraer un valor sin enlazar previamente una variable, `Get` lee una global
del script por nombre en un puntero de Go tipado después de una ejecución:

```go
eng.Run(`program P; var total: Currency; begin total := 19.99 + 5.01 end.`)
var total float64
eng.Get("total", &total) // total == 25
```

## Enlazar variables de Go

Pasa un **puntero** para lectura/escritura; un valor para solo lectura.

```go
total := 10
eng.Var("total", &total)
eng.Run(`for i := 1 to 5 do total := total + i`)
// total == 25  (la variable de Go fue modificada por el script)
```

Tipos escalares soportados: enteros, `float32/64`, `string`, `bool`.

## Mapear un struct a un record

```go
type Punto struct{ X, Y int }

p := Punto{X: 3, Y: 4}
eng.Var("p", &p)
eng.Run(`p.X := p.X * p.X + p.Y * p.Y`)
// p.X == 25  (los campos exportados se mapean por nombre)
```

Los campos exportados del struct se exponen como campos del record (el nombre
se compara sin distinguir mayúsculas/minúsculas) y se copian de vuelta tras la
ejecución.

### Nombres de campo con tags

Por defecto el nombre del campo en Pascal es el nombre del campo en Go. Usa un
struct tag para elegir un nombre diferente, o para ocultar un campo. `vmpas:"…"`
tiene prioridad sobre `json:"…"` (así que los tags JSON existentes se reutilizan
automáticamente), y `vmpas:"-"` omite el campo:

```go
type User struct {
    FullName string `vmpas:"name"`   // expuesto como campo de record `name`
    Email    string `json:"email"`   // reutiliza el tag JSON -> `email`
    Internal int    `vmpas:"-"`       // no expuesto a Pascal
}
```

## Structs anidados y punteros

Los structs anidados se mapean a records anidados, y los punteros se siguen: una
función de Go que recibe o devuelve un `*T` funciona (un puntero nil se mapea a
`nil` de Pascal, uno no nil al record apuntado). Los argumentos y resultados de
tipo puntero hacen el viaje de ida y vuelta con sus campos.

```go
type Point struct{ X, Y int }
eng.Function("SumSq", func(p *Point) int { return p.X*p.X + p.Y*p.Y })
eng.Var("p", &Point{X: 3, Y: 4})
eng.Run(`out := SumSq(p)`) // out == 25
```

## Slices y arrays

Un slice/array de Go se mapea a un `array` de Pascal (índice base 0) y se copia
de vuelta tras la ejecución:

```go
xs := []int{1, 2, 3, 4, 5}
eng.Var("xs", &xs)
eng.Run(`for i := 0 to 4 do xs[i] := xs[i] * xs[i]`)
// xs == [1 4 9 16 25]
```

Las funciones de Go que reciben o devuelven slices también funcionan (el
resultado se asigna a una variable `array` en Pascal).

## Métodos de structs de Go

Un *method value* de Go es una función, así que se enlaza igual que cualquier
función:

```go
r := Rect{W: 4, H: 5}
eng.Function("Area", r.Area) // Area() int
eng.Run(`out := Area()`)     // out == 20
```

## Llamar a funciones de Go desde Pascal

```go
eng.Function("Duplicar", func(n int) int { return n * 2 })
eng.Process("Registrar", func(s string) { log.Println(s) })

eng.Run(`
  r := Duplicar(21);
  Registrar('listo')`)
```

`Function` registra un invocable que devuelve un valor; `Process` uno que
no lo hace (un procedimiento). Los argumentos y el resultado se convierten
automáticamente entre Go y Pascal.

### Errores: `error` de Go → excepción Pascal

Si el último resultado de una función de Go enlazada es un `error`, devolver un
error no nil **lanza una excepción Pascal** en lugar de devolver un valor. El
script puede atraparla con `try/except`; si no lo hace, la ejecución se detiene y
`Run` devuelve el mensaje del error de Go.

```go
eng.Function("Charge", func(amount int) (int, error) {
    if amount <= 0 {
        return 0, errors.New("amount must be positive")
    }
    return amount, nil
})
```

```pascal
try
  total := Charge(-5);          { lanza -> salta a except }
except
  WriteLn('charge failed');
end;
```

Una función cuyo único resultado es `error` se comporta como un procedimiento que
puede lanzar. Esto hace que los fallos del anfitrión sean de primera clase en el
script, en lugar de descartarse silenciosamente.

## Sandbox de capacidades

Cada `Engine` se ejecuta bajo un sandbox. El valor por defecto (`New()` /
`Restricted()`) **deniega** el acceso a archivos. Para permitir todo (solo código
confiable, p. ej. un IDE de TP7) usa `Full()`:

```go
eng := vmpas.NewWith(vmpas.Full())
```

`Capabilities`:

| Campo         | Efecto                                                          |
|---------------|-----------------------------------------------------------------|
| `FileSystem`  | habilita los builtins de archivos (Assign/Reset/...)            |
| `Network`     | habilita los builtins HTTP (`HttpGet`/`HttpPost`/`HttpLastStatus`) |
| `Exec`        | habilita el builtin del anfitrión `Exec(command): Integer`      |
| `Env`         | habilita el builtin del anfitrión `GetEnv(name): string`        |
| `Database`    | habilita los builtins SQL (`Db*`); requiere `UseDB`             |
| `MaxSteps`     | límite de pasos de la VM (0 = por defecto)                     |
| `MaxHeap`      | máximo de asignaciones de heap, `New`/punteros (0 = sin límite) |
| `MaxOutput`    | máximo de bytes de salida capturada (0 = sin límite)           |
| `MaxCallDepth` | profundidad máxima de la pila de llamadas (0 = sin límite)     |
| `MaxDuration`  | límite de tiempo de ejecución de reloj (0 = sin límite)        |
| `Deterministic` / `Seed` | ejecución reproducible (ver [durable.md](durable.md)) |
| `Audit`        | registra cada llamada controlada (`Engine.AuditLog`); ver [durable.md](durable.md) |
| `LiveBindings` | sincroniza las variables enlazadas con el script alrededor de las llamadas al anfitrión (ver abajo) |

Las capacidades se aplican en la frontera Go↔Pascal: los builtins prohibidos no
se registran, así que llamarlos es un **error de compilación** (no un fallo en
tiempo de ejecución). `GetEnv`, `Exec`, los `Http*` y los `Db*` son
**extensiones de anfitrión de vmpas** (no forman parte del RTL de TP7) y solo
existen cuando se concede su capacidad.

## Referencias vivas

Por defecto, una variable enlazada se copia al script cuando arranca una
ejecución y se copia de vuelta cuando termina, de modo que los cambios que un
callback del anfitrión le hace a mitad de la ejecución son sobrescritos por esa
copia final. Habilita `LiveBindings` para mantener el enlace sincronizado
**alrededor de cada llamada a una función/procedimiento de Go enlazado**: el
valor actual del script se escribe de vuelta a Go antes de la llamada, y la
mutación del anfitrión se hace visible para el script después.

```go
counter := 10
eng := vmpas.NewWith(vmpas.Capabilities{LiveBindings: true})
eng.Var("counter", &counter)
eng.Process("Bump", func() { counter++ })   // muta la variable de Go
eng.Run(`Bump; seen := counter; Bump`)       // seen == 11, counter termina en 12
```

Sin `LiveBindings`, el mismo script deja `counter` en 10 (las escrituras del
callback se descartan con la copia de vuelta al final de la ejecución). La opción
añade una sobrecarga por llamada proporcional al número de variables enlazadas,
por lo que es opcional (opt-in).

## Inferencia de capacidades (`Analyze`)

Antes de conceder nada, puedes descubrir estáticamente qué capacidades necesita
realmente un script. `Analyze` compila el código y escanea los builtins que
llama, devolviendo un `CapReport`:

```go
rep, err := eng.Analyze(tenantScript)
if err != nil {
    log.Fatal(err) // el código ni siquiera compila
}
if rep.Needs(vmpas.CapNetwork) {
    // este script quiere HTTP — decide si permitirlo
}
fmt.Println(rep.Required) // p. ej. [filesystem network]
fmt.Println(rep.Calls)    // los builtins controlados que llama
```

Úsalo para rechazar scripts que excedan una política, para mostrar a un operador
qué tocará un script antes de aprobarlo, o para conceder el conjunto mínimo de
capacidades. `Analyze` nunca ejecuta el código. Para un registro a posteriori de
lo que realmente se ejecutó, consulta la capacidad `Audit` más abajo
(`Engine.AuditLog`).

## Multi-tenant: ejecutar scripts no confiables

Para un servicio donde **cada tenant provee su propio script** (motor de reglas
de negocio embebido), el patrón es **un motor por petición/tenant** —
*share-nothing*: no se comparte estado entre ejecuciones. El helper
`RunSandboxed` hace esto en una línea, sobre un motor nuevo y aislado:

```go
out, err := vmpas.RunSandboxed(tenantScript, vmpas.Sandboxed())
```

`Sandboxed()` es un preset *default-deny* con techos conservadores diseñado para
código no confiable (sin FS/red/exec/env, con límites en pasos, heap, salida,
profundidad de llamadas y tiempo). Ajusta los campos a tu gusto:

```go
caps := vmpas.Sandboxed()
caps.MaxDuration = 500 * time.Millisecond
caps.MaxOutput   = 256 * 1024
caps.Network     = true            // permite HTTP si el tenant lo necesita
out, err := vmpas.RunSandboxed(tenantScript, caps)
```

Garantías de aislamiento:

- **Sin fugas entre ejecuciones**: cada `Run` crea una nueva VM (globales a
  cero), y el estado transitorio del anfitrión (cursor SQL, último error/estado
  HTTP, cabeceras) se reinicia al inicio de cada ejecución. Reutilizar el mismo
  `Engine` para varios tenants no filtra datos entre ellos.
- **Límites duros**: un script que inunda la salida, recurre sin fin o entra en
  un bucle infinito se detiene con un error en tiempo de ejecución (no agota la
  memoria ni cuelga el proceso anfitrión).
- **Paralelismo**: el anfitrión puede ejecutar muchos motores en distintas
  goroutines en paralelo (un motor es de un solo hilo por ejecución; la
  concurrencia real la provee el anfitrión con un motor por goroutine).

## Integración: HTTP y SQL (consumir APIs y bases de datos)

Bajo la capacidad `Network`, el código Pascal puede consumir APIs con todos los
verbos, cabeceras (p. ej. tokens de autenticación) y parsing de JSON:

```pascal
{ Verbos: GET, POST, PUT, PATCH, DELETE, y HttpRequest para cualquier método }
HttpSetHeader('Authorization', 'Bearer ' + token);  { header on subsequent calls }
body   := HttpGet('https://api.example.com/users');
result := HttpPost('https://api.example.com/users', 'application/json', '{"n":1}');
HttpPut('https://api.example.com/users/1', 'application/json', '{"n":2}');
HttpDelete('https://api.example.com/users/1');
HttpRequest('OPTIONS', 'https://api.example.com', '', '');
status := HttpLastStatus();   { status code of the last call }

{ Read JSON (no capability: it is pure computation) }
name := JsonStr(body, 'user.name');     { dotted-path access }
id   := JsonInt(body, 'items.0.id');    { numeric segment = array index }
len  := JsonLen(body, 'items');         { array/object length }
if JsonValid(body) then ...             { JsonValid / JsonBool / JsonStr / JsonInt / JsonLen }

{ Build JSON (set by path; creates intermediate objects/arrays) }
req := JsonSetStr('{}', 'user.name', 'bob');
req := JsonSetInt(req, 'user.age', 25);
req := JsonSetBool(req, 'user.active', true);
HttpPost(url, 'application/json', req);  { -> {"user":{"active":true,"age":25,"name":"bob"}} }
s := JsonEscape('con "comillas"');       { -> "con \"comillas\"" (for manual assembly) }
```

Bajo la capacidad `Database`, el código habla con cualquier base de datos
soportada por el `database/sql` de Go. El anfitrión inyecta el handle (y aporta
el driver), de modo que `pkg/vmpas` se mantiene **libre de dependencias
externas**:

```go
import "database/sql"
// _ "github.com/mattn/go-sqlite3"  // el anfitrión aporta el driver

db, _ := sql.Open("sqlite3", "app.db")
eng := vmpas.NewWith(vmpas.Capabilities{Database: true})
eng.UseDB(vmpas.WrapSQLDB(db))   // adapta *sql.DB (solo stdlib)
```

La API SQL en Pascal es un cursor estilo dataset de Delphi:

```pascal
n := DbExec('INSERT INTO users(name) VALUES (?)', 'alice');  { affected rows }
if DbOpen('SELECT id, name FROM users') then
  while not DbEof() do
  begin
    WriteLn(DbFieldInt(0), ' ', DbFieldStr(1));
    DbNext;
  end;
DbClose;
if DbError() <> '' then WriteLn('error: ', DbError());
```

`DbExec(sql [, params...])` ejecuta y devuelve las filas afectadas; `DbOpen`
ejecuta una consulta y posiciona el cursor; `DbEof`/`DbNext` iteran;
`DbFieldStr(i)` / `DbFieldInt(i)` leen la columna `i` de la fila actual;
`DbClose` cierra; y `DbError` devuelve el último error. Los parámetros se pasan
posicionalmente (placeholders `?`/`$1` según el driver). Las funciones/builtins
sin parámetros pueden llamarse sin paréntesis (`DbEof`, `HttpLastStatus`); los
paréntesis también son válidos. (Un **valor procedural** almacenado en una
variable sí requiere `()` para invocarlo, ya que el nombre desnudo es el valor.)

Los límites `MaxSteps`, `MaxHeap` y `MaxDuration` se aplican dentro de la VM y
detienen el programa con un error en tiempo de ejecución (200 paso/tiempo, 203
heap) cuando se exceden. Ejemplo de una configuración estricta con techos de
tiempo y memoria:

```go
eng := vmpas.NewWith(vmpas.Capabilities{
    MaxSteps:    5_000_000,
    MaxHeap:     10_000,
    MaxDuration: 200 * time.Millisecond,
})
```

## Comprobación de errores antes de ejecutar

```go
err := eng.Run(`variable_inexistente := 5`)
// err != nil: "unknown identifier" detectado en tiempo de compilación
```

## Ejemplo completo

Consulta `examples/embed/main.go`:

```bash
go run ./examples/embed
```

## Estado y limitaciones

vmpas se ejecuta sobre el compilador y la VM reales del proyecto. El núcleo
procedural y el modelo de objetos OOP de TP7 están completos
(procedimientos/funciones con parámetros por valor, `var` y `const`, recursión,
records, arrays, punteros, enums, sets, control de flujo completo, tipos
`object` con herencia, despacho virtual, constructores e `inherited`), al igual
que el sistema de unidades `uses` y la E/S de archivos de texto/tipados.

Limitación conocida: `inherited` se soporta como sentencia (`inherited Init(a)`)
pero todavía no dentro de una expresión (`x := inherited Foo + y`). Consulta la
[matriz de compatibilidad](compatibility.md) para el detalle completo por
característica y la [arquitectura](arquitectura.md) para ver cómo encajan las
piezas.
