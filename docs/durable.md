# Ejecución durable: determinismo + snapshot/resume (Fase F)

`vmpas` puede **pausar** una ejecución, **serializar** todo su estado a bytes y
**reanudarla** exactamente donde quedó — incluso en otro proceso, minutos o días
después. Combinado con el modo **determinista**, esto da lógica de negocio
*pausable, reproducible y auditable*: workflows que esperan una aprobación, un
evento externo o un horario, sin bloquear un hilo del host.

Es el diferencial frente a motores de scripting embebidos típicos (goja y
similares): no solo ejecutás código dinámico tipado, sino que podés **suspender
y continuar** una ejecución como una máquina de estados durable.

## Determinismo

Con `Capabilities.Deterministic` la ejecución es reproducible bit a bit: el
mismo fuente + las mismas entradas producen la misma salida y el mismo estado.
`Randomize` siembra el RNG desde `Seed` (no desde la entropía del host).

```go
caps := vmpas.Capabilities{Deterministic: true, Seed: 42}
out1, _ := vmpas.RunSandboxed(src, caps)
out2, _ := vmpas.RunSandboxed(src, caps) // out1 == out2, garantizado
```

El determinismo es la base del snapshot/resume: para reanudar de forma confiable
necesitás que la continuación sea independiente del reloj o de la entropía.

## Pausar y reanudar

El script se pausa llamando al builtin `Suspend(tag)`. El host obtiene un
`*State` serializable; lo persiste y luego lo pasa a `ResumeDurable` para
continuar.

```pascal
program Aprobacion;
var monto: Currency; aprobado: Boolean; resultado: string;
begin
  if monto > 1000.00 then
  begin
    Suspend('aprobacion-requerida');   { se pausa acá }
    if aprobado then resultado := 'APROBADO' else resultado := 'RECHAZADO';
  end
  else resultado := 'APROBADO (automatico)';
  WriteLn(resultado);
end.
```

```go
eng := vmpas.NewWith(vmpas.Capabilities{Deterministic: true, Seed: 1})
eng.Var("monto", &monto)
eng.Var("aprobado", &aprobado)

state, err := eng.RunDurable(rule)   // corre hasta Suspend (o hasta terminar)
// state == nil  -> terminó; state != nil -> se pausó (state.Tag, state.Data)

// ... persistir state.Data (DB, archivo, cola) y, más tarde:

aprobado = true                       // el host inyecta la decisión
final, err := eng.ResumeDurable(rule, state)  // continúa tras el Suspend
// final == nil -> terminó; final != nil -> se volvió a pausar
```

Ver el ejemplo ejecutable en [`../examples/durable`](../examples/durable).

## Qué se captura

El snapshot captura **todo el estado de ejecución**: variables globales y
locales, la pila de operandos, la **pila de llamadas con sus program counters**,
el heap (incluyendo punteros y grafos de objetos, p. ej. listas enlazadas con
`New`), el estado del RNG y el de excepciones. Al reanudar, la ejecución
continúa en la instrucción siguiente al `Suspend`, con el aliasing de punteros y
`var`-parámetros intacto.

### Contrato de entrada/salida

- **Estado del script** (locals, globals no enlazados, heap, pila): se captura y
  restaura del snapshot.
- **Variables Go enlazadas** (`Var`): son el **canal de E/S con el host**. Se
  re-siembran en cada reanudación, así que el host inyecta respuestas
  actualizándolas *antes* de `ResumeDurable`, y el script las lee *después* de
  que `Suspend` retorna.
- **Salida** (`Output`): es acumulativa entre segmentos (`State.Output` arrastra
  lo producido hasta la pausa).

## API

| Símbolo | Qué hace |
|---------|----------|
| `Capabilities.Deterministic` / `Seed` | activa ejecución reproducible |
| `(*Engine).RunDurable(code) (*State, error)` | corre; devuelve `*State` si se pausó, `nil` si terminó |
| `(*Engine).ResumeDurable(code, *State) (*State, error)` | restaura y continúa (mismo fuente) |
| `Suspend(tag)` (builtin Pascal) | pausa la ejecución durable con una etiqueta |
| `State{Tag, Data, Output}` | snapshot opaco y serializable (`Data` es portable) |

`ResumeDurable` exige el **mismo fuente** que produjo el `State`: un *fingerprint*
del programa compilado se guarda en el snapshot y se valida al reanudar (cambiar
el código —incluso un literal— invalida un estado viejo, evitando que PCs e
índices de slots se desalineen en silencio).

## Sandbox: inferencia de capacidades y traza auditable

Dos herramientas para correr scripts no confiables con *least-privilege* y
trazabilidad (útiles en multi-tenant y cumplimiento).

### Inferencia de capacidad mínima

`Engine.Analyze(code)` compila el script y reporta **qué capacidades necesita**
escaneando el bytecode en busca de llamadas a builtins de host gateados. No
ejecuta nada y funciona aunque el engine esté restringido — así podés conceder
exactamente lo que el script usa, o rechazarlo si pide de más.

```go
rep, _ := eng.Analyze(src)
// rep.Required -> p.ej. [Env Network]
// rep.Needs(vmpas.CapFileSystem) -> false
// rep.Calls[vmpas.CapNetwork]    -> ["httpget"]
if rep.Needs(vmpas.CapExec) {
    return errors.New("este tenant no puede ejecutar procesos")
}
```

### Traza auditable

Con `Capabilities.Audit`, cada llamada a un builtin gateado (archivo, red,
exec, env, base de datos) se registra **en orden de ejecución** con sus
argumentos. El log es determinista y se combina con el snapshot/resume para
replay forense.

```go
eng := vmpas.NewWith(vmpas.Capabilities{Network: true, Audit: true})
_ = eng.Run(src)
for _, ev := range eng.AuditLog() {
    log.Printf("%s %s%v", ev.Capability, ev.Builtin, ev.Args)
}
```

## Alcance (v1)

- **Programas no concurrentes**: el snapshot soporta la ejecución de una sola
  fibra (el caso de reglas de negocio). Snapshotear un programa concurrente con
  fibras vivas (`spawn`/canales) se rechaza con un error claro en vez de
  producir un snapshot corrupto.
- **Sin archivos abiertos**: no se puede snapshotear mientras hay un `File`
  abierto (es un recurso del host, no serializable). Cerralo antes de `Suspend`.
- `MaxDuration` no se aplica en runs durables (una pausa puede durar
  arbitrariamente); acotá el trabajo con `MaxSteps`.
