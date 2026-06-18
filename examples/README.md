# Ejemplos

## Programas Pascal (`examples/pascal/`)

Ejecutar con el motor real:

```bash
go run ./cmd/pasrun examples/pascal/<archivo>.pas
```

| Archivo | Muestra |
|---|---|
| `factorial.pas` | funciones recursivas |
| `listas.pas` | punteros y records (lista enlazada) |
| `figuras.pas` | OOP: objetos, herencia, `inherited` |
| `calc.pas` | `ReadLn` + formato de campo (`echo "12 5" \| pasrun ...`) |
| `crt_demo.pas` | unit `Crt`: ClrScr/GotoXY/TextColor (salida ANSI) |
| `units/demo.pas` | sistema de units (`uses`, interface/implementation/initialization) |

## Embeber Pascal en Go (`examples/embed/`)

```bash
go run ./examples/embed
```

Demuestra `pkg/vmpas`: ejecutar Pascal, enlazar variables Go, mapear un
`struct` de Go a un `record` de Pascal, llamar funciones Go desde Pascal y el
sandbox de capacidades.

## Consumir APIs y SQL (`examples/integration/`)

```bash
go run ./examples/integration
```

Autocontenido y offline: levanta un servidor HTTP local y una base SQL en
memoria, y desde Pascal consume la API (`HttpGet`/`HttpPost`/`HttpLastStatus`)
y recorre una consulta (`DbOpen`/`DbEof`/`DbNext`/`DbFieldInt`/`DbFieldStr`)
bajo las capacidades `Network` y `Database`.

Ver también [`../docs/inicio.md`](../docs/inicio.md) y la sección de
integración en [`../docs/vmpas.md`](../docs/vmpas.md).
