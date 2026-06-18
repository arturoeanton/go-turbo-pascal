# vmpas — Pascal embebido en Go

`pkg/vmpas` es un **motor de código dinámico embebible** para Go. Permite
ejecutar código Turbo Pascal 7 dentro de un programa Go, enlazando variables,
funciones y structs del host. A diferencia de los motores de scripting
dinámicos (p. ej. JavaScript con goja), vmpas **compila y verifica tipos antes
de la primera ejecución**: los errores de compilación se detectan al instante,
no en medio de la ejecución.

Características:

- **Fuertemente tipado**: se compila a bytecode y se valida antes de correr.
- **Mapeo Go ↔ Pascal**: variables escalares y `struct` ↔ `record`.
- **Llamadas bidireccionales**: Pascal puede llamar funciones/procedimientos Go.
- **Sandbox de capacidades**: control fino de lo que el código invitado puede
  hacer (filesystem, etc.), con `Restricted` (por defecto) y `Full`.
- **Cero dependencias externas**: importar `vmpas` no arrastra tcell ni la
  cadena de herramientas del editor (lo garantiza un test).

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

Se puede pasar un fragmento (se envuelve automáticamente en un programa) o un
programa completo (`program ... end.`).

## Enlazar variables Go

Pasá un **puntero** para lectura/escritura; un valor para solo lectura.

```go
total := 10
eng.Var("total", &total)
eng.Run(`for i := 1 to 5 do total := total + i`)
// total == 25  (la variable Go fue modificada por el script)
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

## Slices y arrays

Un slice/array de Go se mapea a un `array` de Pascal (índice 0-based) y se copia
de vuelta tras ejecutar:

```go
xs := []int{1, 2, 3, 4, 5}
eng.Var("xs", &xs)
eng.Run(`for i := 0 to 4 do xs[i] := xs[i] * xs[i]`)
// xs == [1 4 9 16 25]
```

Las funciones Go que reciben o devuelven slices también funcionan (el resultado
se asigna a una variable `array` en Pascal).

## Métodos de structs Go

Un *method value* de Go es una función, así que se enlaza igual que cualquier
función:

```go
r := Rect{W: 4, H: 5}
eng.Function("Area", r.Area) // Area() int
eng.Run(`out := Area()`)     // out == 20
```

## Llamar funciones Go desde Pascal

```go
eng.Function("Duplicar", func(n int) int { return n * 2 })
eng.Process("Registrar", func(s string) { log.Println(s) })

eng.Run(`
  r := Duplicar(21);
  Registrar('listo')`)
```

`Function` registra un callable que devuelve un valor; `Process` uno que no
(procedimiento). Los argumentos y el resultado se convierten automáticamente
entre Go y Pascal.

## Sandbox de capacidades

Cada `Engine` corre bajo un sandbox. El valor por defecto (`New()` /
`Restricted()`) **deniega** el acceso a archivos. Para permitir todo (solo
código de confianza, p. ej. una IDE TP7) usá `Full()`:

```go
eng := vmpas.NewWith(vmpas.Full())
```

`Capabilities`:

| Campo         | Efecto                                                          |
|---------------|-----------------------------------------------------------------|
| `FileSystem`  | habilita los builtins de archivo (Assign/Reset/...)             |
| `Network`     | habilita los builtins HTTP (`HttpGet`/`HttpPost`/`HttpLastStatus`) |
| `Exec`        | habilita el builtin de host `Exec(comando): Integer`            |
| `Env`         | habilita el builtin de host `GetEnv(nombre): string`            |
| `Database`    | habilita los builtins SQL (`Db*`); requiere `UseDB`             |
| `MaxSteps`    | límite de pasos de la VM (0 = por defecto)                      |
| `MaxHeap`     | máximo de asignaciones de heap, `New`/punteros (0 = sin límite) |
| `MaxDuration` | límite de tiempo de pared de ejecución (0 = sin límite)         |

Las capacidades se aplican en el límite Go↔Pascal: los builtins prohibidos no
se registran, así que llamarlos es un **error de compilación** (no un fallo en
tiempo de ejecución). `GetEnv`, `Exec`, los `Http*` y los `Db*` son
**extensiones de host de vmpas** (no forman parte de la RTL de TP7) y solo
existen cuando su capacidad está concedida.

## Integración: HTTP y SQL (consumir APIs y bases de datos)

Bajo la capacidad `Network`, el código Pascal puede consumir APIs con todos los
verbos, headers (p. ej. tokens de autenticación) y parseo de JSON:

```pascal
{ Verbos: GET, POST, PUT, PATCH, DELETE, y HttpRequest para cualquier método }
HttpSetHeader('Authorization', 'Bearer ' + token);  { header en llamadas siguientes }
body   := HttpGet('https://api.example.com/users');
result := HttpPost('https://api.example.com/users', 'application/json', '{"n":1}');
HttpPut('https://api.example.com/users/1', 'application/json', '{"n":2}');
HttpDelete('https://api.example.com/users/1');
HttpRequest('OPTIONS', 'https://api.example.com', '', '');
status := HttpLastStatus();   { código de estado de la última llamada }

{ Parsear la respuesta JSON (sin capacidad: es computación pura) }
name := JsonStr(body, 'user.name');     { acceso por path con puntos }
id   := JsonInt(body, 'items.0.id');    { segmento numérico = índice de array }
len  := JsonLen(body, 'items');         { longitud de array/objeto }
if JsonValid(body) then ...             { JsonValid / JsonBool / JsonStr / JsonInt / JsonLen }
```

Bajo la capacidad `Database`, el código habla con cualquier base soportada por
`database/sql` de Go. El host inyecta el handle (y trae el driver), así
`pkg/vmpas` se mantiene **sin dependencias externas**:

```go
import "database/sql"
// _ "github.com/mattn/go-sqlite3"  // el driver lo trae el host

db, _ := sql.Open("sqlite3", "app.db")
eng := vmpas.NewWith(vmpas.Capabilities{Database: true})
eng.UseDB(vmpas.WrapSQLDB(db))   // adapta *sql.DB (solo stdlib)
```

La API SQL en Pascal es un cursor estilo dataset de Delphi:

```pascal
n := DbExec('INSERT INTO users(name) VALUES (?)', 'alice');  { filas afectadas }
if DbOpen('SELECT id, name FROM users') then
  while not DbEof() do
  begin
    WriteLn(DbFieldInt(0), ' ', DbFieldStr(1));
    DbNext;
  end;
DbClose;
if DbError() <> '' then WriteLn('error: ', DbError());
```

`DbExec(sql [, params...])` ejecuta y devuelve filas afectadas; `DbOpen` corre
una consulta y posiciona el cursor; `DbEof`/`DbNext` iteran; `DbFieldStr(i)` /
`DbFieldInt(i)` leen la columna `i` de la fila actual; `DbClose` cierra; y
`DbError()` devuelve el último error. Los parámetros se pasan posicionalmente
(placeholders `?`/`$1` según el driver). Un valor procedural sin argumentos en
una expresión requiere paréntesis: `DbEof()`, `HttpLastStatus()`.

Los límites `MaxSteps`, `MaxHeap` y `MaxDuration` se aplican dentro de la VM y
detienen el programa con un error de runtime (200 paso/tiempo, 203 heap) cuando
se exceden. Ejemplo de configuración estricta con tope de tiempo y memoria:

```go
eng := vmpas.NewWith(vmpas.Capabilities{
    MaxSteps:    5_000_000,
    MaxHeap:     10_000,
    MaxDuration: 200 * time.Millisecond,
})
```

## Verificación de errores antes de ejecutar

```go
err := eng.Run(`variable_inexistente := 5`)
// err != nil: "unknown identifier" detectado en compilación
```

## Ejemplo completo

Ver `examples/embed/main.go`:

```bash
go run ./examples/embed
```

## Estado y limitaciones

vmpas usa el compilador y la VM reales del proyecto. El núcleo procedural está
completo (procedimientos/funciones con parámetros por valor y `var`, recursión,
records, arrays, punteros, enums, conjuntos, control de flujo). En desarrollo:
el modelo de objetos OOP de TP7, el sistema de units y más de la RTL. Ver
[`docs/arquitectura.md`](arquitectura.md) y el roadmap del README.
