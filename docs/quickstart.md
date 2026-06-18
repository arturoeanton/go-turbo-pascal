# Quickstart

Tour rápido de **go-turbo-pascal**: ejecutar Pascal, embeberlo en Go, las
extensiones modernas (`{$MODE BPGO}`) y la integración con HTTP/JSON/SQL. Para
la guía breve de instalación/editor ver [`inicio.md`](inicio.md).

## 0. Requisitos y build

```bash
# Go 1.23+
go build ./...                 # compila todo
go test ./...                  # corre la suite
go build -o bin/pasrun ./cmd/pasrun   # el runner de programas .pas
```

## 1. Ejecutar un programa Pascal

```pascal
{ hola.pas }
program Hola;
begin
  WriteLn('Hola, mundo!');
end.
```

```bash
bin/pasrun hola.pas        # -> Hola, mundo!
```

Es Turbo Pascal 7 real: procedimientos/funciones, records, arrays (estáticos y
dinámicos), sets, punteros, `class`/`object` con herencia y métodos virtuales,
interfaces, genéricos, closures, excepciones, units, archivos. Ver la matriz en
[`compatibilidad.md`](compatibilidad.md).

## 2. Modo moderno: `{$MODE BPGO}`

Con el directivo `{$MODE BPGO}` al inicio se habilitan extensiones modernas. Sin
él, el compilador es **TP7 estricto** (compatibilidad total: `let`, `match`,
`spawn`, etc. siguen siendo identificadores normales).

```pascal
{$MODE BPGO}
program ModernoDemo;

type
  TShape = (Circle(Integer), Rect(Integer, Integer));   { tipos suma / ADTs }

function Area(s: TShape): Integer;
begin
  Area := match s of                       { match como expresión }
    Circle(r)  => r * r * 3;
    Rect(w, h) => w * h;
  end;
end;

let factor = 2;                            { binding inmutable }
var total := 0;                            { inferencia de tipo }
begin
  total := Area(Rect(3, 4)) * factor;      { 24 }
  WriteLn(total);

  match total of                           { guards y or-patterns }
    0          => WriteLn('cero');
    24, 48     => WriteLn('múltiplo esperado');
    _ when total > 0 => WriteLn('positivo');
    else       WriteLn('otro');   { en match-sentencia el else no lleva => }
  end;
end.
```

Las features modernas, en guías dedicadas:

- **[match / tipos suma / Option](match.md)** — `match`, ADTs, `Some`/`None`.
- **[defer / panic / recover](defer.md)** — limpieza garantizada y manejo de panics.
- **[spawn / channels](concurrency.md)** — concurrencia cooperativa.
- Otras (en [`compatibilidad.md`](compatibilidad.md)): inferencia local, `let`
  inmutable, *extension methods* (`record/class helper`), unit tests integrados
  (`test … AssertEqual …`).

### Concurrencia en 6 líneas

```pascal
{$MODE BPGO}
program ProdCons;
var ch: Channel<Integer>; i, j, total: Integer;
begin
  ch := MakeChan(64);
  spawn begin for j := 1 to 100 do ch.Send(j); end;
  total := 0;
  for i := 1 to 100 do total := total + ch.Receive;
  WriteLn(total);          { 5050 }
end.
```

## 3. Embeber Pascal en Go (`pkg/vmpas`)

El motor `vmpas` ejecuta Pascal dentro de un programa Go, con binding de
variables/funciones, **sandbox de capacidades** y **cero dependencias externas**.

```go
package main

import (
    "fmt"
    "github.com/arturoeanton/go-turbo-pascal/pkg/vmpas"
)

func main() {
    eng := vmpas.New() // sandbox restringido por defecto
    total := 10
    eng.Var("total", &total)
    eng.Function("Triple", func(x int) int { return x * 3 }) // función Go llamable desde Pascal
    eng.Run(`total := Triple(total) + 5`)
    fmt.Println(total) // 35
}
```

`vmpas` mapea structs de Go ↔ records de Pascal, expone funciones/métodos Go, y
permite *compile-once / run-many*. Detalle en [`vmpas.md`](vmpas.md); ejemplo
ejecutable en `examples/embed`.

### Sandbox de capacidades

Por defecto (`New()` / `Restricted()`) se **deniega** filesystem, red, exec,
entorno y base de datos. Concedés solo lo necesario, con límites:

```go
eng := vmpas.NewWith(vmpas.Capabilities{
    Network:     true,
    MaxSteps:    5_000_000,
    MaxDuration: 200 * time.Millisecond,
})
```

## 4. Integración: consumir APIs y SQL

Bajo las capacidades correspondientes, el código Pascal embebido puede consumir
HTTP y bases de datos (el host inyecta el driver; el motor sigue sin deps):

```pascal
body := HttpGet('https://api.example.com/users');   { Network }
name := JsonStr(body, 'user.name');                  { JSON: leer por path }
req  := JsonSetStr('{}', 'n', '1');                  { JSON: construir }
HttpPost(url, 'application/json', req);

if DbOpen('SELECT id, name FROM users') then         { Database (UseDB) }
  while not DbEof do
  begin
    WriteLn(DbFieldInt(0), ' ', DbFieldStr(1));
    DbNext;
  end;
DbClose;
```

Ejemplo autocontenido (server local + base en memoria, offline):

```bash
go run ./examples/integration
```

Detalle en la sección de integración de [`vmpas.md`](vmpas.md).

### Paralelismo a nivel host

Cada `vmpas.Engine` es independiente (share-nothing). Para cargas paralelas
reales, el host de Go lanza varios engines en goroutines:

```go
for i := 0; i < n; i++ {
    go func() {
        eng := vmpas.NewWith(caps)
        eng.Run(script)   // VM propio, sin estado compartido
    }()
}
```

## 5. Tooling de editor (LSP / DAP)

```bash
make tools && export PATH="$PWD/bin:$PATH"   # pls (LSP) y pdap (DAP)
```

- **`pls`** — diagnósticos, hover, ir-a-definición, símbolos, autocompletado.
- **`pdap`** — breakpoints, step, inspección de variables.
- Extensiones **VSCode** (LSP + depuración) y **Zed** (LSP) en `editors/`.

Ver [`editores.md`](editores.md).

## Siguiente

- Matriz de compatibilidad y extensiones: [`compatibilidad.md`](compatibilidad.md)
- Arquitectura del compilador/VM: [`arquitectura.md`](arquitectura.md)
- Plan y roadmap: [`plan.md`](plan.md)
