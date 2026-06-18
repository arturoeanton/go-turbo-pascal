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

## Scripts multi-tenant aislados (`examples/multitenant/`)

```bash
go run ./examples/multitenant
```

Simula un SaaS donde cada tenant sube su propia regla de negocio: ejecuta cada
script en un engine fresco y acotado con `vmpas.RunSandboxed` + el preset
`vmpas.Sandboxed()` (default-deny, con techos de pasos/heap/salida/profundidad/
tiempo). Muestra el aislamiento *share-nothing* y cómo un script malicioso
(bucle infinito) se detiene sin colgar el host.

## Ejecución durable: pausar y reanudar (`examples/durable/`)

```bash
go run ./examples/durable
```

Una regla de aprobación de gastos se ejecuta hasta que necesita una decisión
humana: se pausa con `Suspend`, el host serializa el estado (`RunDurable` →
`*State`), inyecta la respuesta y reanuda en un engine nuevo (`ResumeDurable`),
continuando exactamente donde quedó. Ver [`../docs/durable.md`](../docs/durable.md).

Ver también [`../docs/inicio.md`](../docs/inicio.md) y la sección de
integración en [`../docs/vmpas.md`](../docs/vmpas.md).
