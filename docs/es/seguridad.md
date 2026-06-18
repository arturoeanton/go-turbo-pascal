# Seguridad y el sandbox de capacidades

Cuando incrustas `pkg/vmpas` estás ejecutando código Pascal invitado dentro de tu
proceso Go. Esta página explica el modelo de seguridad, cómo ejecutar scripts no
confiables y —igual de importante— **contra qué no protege el sandbox**.

## El modelo en una frase

Cada `Engine` se ejecuta bajo un valor `Capabilities` que es **denegar por
defecto**: a menos que concedas explícitamente una capacidad, los builtins que
podrían alcanzar el mundo exterior **no se registran**, de modo que un script que
los invoque falla al **compilar** (no en tiempo de ejecución).

## Capacidades

| Capacidad  | Concede acceso a | Builtins |
|-------------|------------------|----------|
| `FileSystem` | E/S de archivos | `Assign`/`Reset`/`Rewrite`/`Read`/`Write`/`Close`/… |
| `Network`    | HTTP saliente | `HttpGet`/`HttpPost`/`HttpPut`/`HttpDelete`/`HttpRequest`/`HttpSetHeader`/`HttpLastStatus` |
| `Exec`       | lanzar procesos | `Exec(cmd): Integer` |
| `Env`        | variables de entorno | `GetEnv(name): string` |
| `Database`   | SQL vía `database/sql` | `Db*` (requiere `UseDB`) |

`GetEnv`, `Exec`, los builtins `Http*` y `Db*` son **extensiones del host de
vmpas**, no forman parte del RTL de TP7 — solo existen cuando se concede su
capacidad. El parseo/construcción de JSON (`Json*`) no necesita ninguna
capacidad: es computación pura.

## Límites de recursos

Las capacidades controlan *qué* puede alcanzar un script; los límites controlan
*cuánto* puede consumir. Todos están desactivados (0) por defecto y se aplican
dentro de la VM.

| Campo          | Efecto | Error al excederse |
|----------------|--------|-----------------|
| `MaxSteps`     | presupuesto de instrucciones de la VM | error de ejecución 200 |
| `MaxHeap`      | asignaciones en el heap (`New`/punteros) | error de ejecución 203 |
| `MaxOutput`    | bytes de salida capturada | error de ejecución 203 |
| `MaxCallDepth` | profundidad de la pila de llamadas | error de ejecución 202 |
| `MaxDuration`  | tiempo de reloj | error de ejecución 200 |

## Preajustes

```go
vmpas.New()                 // = Restricted(): sin FS/net/exec/env/db, sin límites
vmpas.NewWith(vmpas.Sandboxed()) // denegar por defecto + topes conservadores de steps/heap/output/depth/time
vmpas.NewWith(vmpas.Full())      // todo activado — solo código confiable
```

- **`Restricted`** (el valor por defecto): deniega todo acceso externo pero no
  establece topes de recursos. Bueno para código confiable que escribiste tú
  mismo.
- **`Sandboxed`**: el preajuste para código **no confiable** — denegar por
  defecto más límites conservadores de steps, heap, salida, profundidad de
  llamadas y tiempo.
- **`Full`**: concede todo. Úsalo solo para código en el que confías plenamente
  (por ejemplo, una herramienta de primera parte).

## Ejecutar scripts no confiables (multi-tenant)

El patrón es **un motor por petición/tenant, sin compartir nada**. El helper
`RunSandboxed` hace exactamente eso sobre un motor nuevo y aislado:

```go
out, err := vmpas.RunSandboxed(tenantScript, vmpas.Sandboxed())
```

Ajusta el preajuste a las necesidades del tenant:

```go
caps := vmpas.Sandboxed()
caps.MaxDuration = 500 * time.Millisecond
caps.MaxOutput   = 256 * 1024
caps.Network     = true      // permite HTTP si este tenant está autorizado
out, err := vmpas.RunSandboxed(tenantScript, caps)
```

Garantías de aislamiento:

- **Sin filtraciones entre ejecuciones**: cada `Run` construye una VM nueva
  (globales a cero) y reinicia el estado transitorio del host (cursor SQL, último
  estado/cabeceras HTTP). Reutilizar un `Engine` entre tenants no filtra datos
  entre ellos.
- **Paradas en firme**: un script que inunda la salida, recurre para siempre o
  hace un bucle interminable se detiene con un error de ejecución — no agota la
  memoria ni cuelga el host.
- **Paralelismo**: ejecuta muchos motores en goroutines separadas. Un motor es
  de un solo hilo por ejecución; la concurrencia viene de un motor por goroutine.

## Inspecciona antes de confiar: `Analyze` y `AuditLog`

- **Antes de ejecutar**, `Engine.Analyze(code)` informa estáticamente las
  capacidades que necesita un script (`CapReport.Needs`, `.Required`, `.Calls`)
  sin ejecutarlo. Úsalo para rechazar scripts que excedan la política o para
  conceder el conjunto mínimo.
- **Después de ejecutar**, la capacidad `Audit` registra cada llamada controlada
  para que tengas un rastro a posteriori (`Engine.AuditLog()`).

```go
rep, _ := eng.Analyze(script)
if rep.Needs(vmpas.CapExec) { return errors.New("exec not allowed") }
```

## Modelo de amenazas — qué es y qué no es

El sandbox es una frontera **en proceso, a nivel de lenguaje**. Es efectivo
contra las cosas que el Pascal invitado puede expresar, pero **no** es una
prisión a nivel de sistema operativo.

**Sí protege contra:**
- Código invitado que alcanza el sistema de archivos, la red, los procesos, el
  entorno o una base de datos sin una concesión explícita.
- Uso descontrolado de recursos (steps de CPU, memoria, salida, recursión,
  tiempo de reloj).
- Estado que se filtra entre ejecuciones/tenants.

**No protege contra:**
- **`Full()` o concesiones amplias** — estas son una declaración explícita de
  confianza.
- **Tus propios callbacks de Go.** Una función que registres con
  `Function`/`Process` se ejecuta con privilegios completos de Go; si toca el
  disco o la red, el script los alcanza sin importar las capacidades. Mantén los
  callbacks registrados tan acotados como la capacidad que de otro modo
  concederías.
- **Mal uso de los datos devueltos en el lado del host** (por ejemplo, pasar la
  salida del script a un shell).
- **Canales laterales y garantías absolutas** frente a un atacante determinado
  que explote un fallo en el motor. Para un aislamiento multi-tenant sólido de
  código genuinamente hostil, combina vmpas con sandboxing a nivel de sistema
  operativo (contenedores, seccomp, procesos/usuarios separados).

## Recomendaciones

- Parte de `Sandboxed()` para todo lo que no hayas escrito tú; amplía una
  capacidad a la vez.
- Establece siempre `MaxDuration` y `MaxOutput` para código no confiable.
- Prefiere `Analyze` para *decidir* y `AuditLog` para *registrar*.
- Registra la **menor cantidad de callbacks de Go, y los más acotados**, que
  puedas.

Véase también: [guía de vmpas](vmpas.md) · [ejecución durable](durable.md) ·
[matriz de compatibilidad](compatibility.md).
