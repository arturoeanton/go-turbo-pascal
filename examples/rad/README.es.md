# Constructor RAD de workflows durables

*Un constructor visual de arrastrar‑y‑soltar para workflows durables, corriendo
sobre el motor Pascal embebible [`pkg/vmpas`](../../pkg/vmpas).*

🇬🇧 *English version: [README.md](README.md)*

![Constructor RAD de workflows durables](../../screenshot/rad-workflow.png)

```bash
cd examples/rad && go run .     # luego abrí http://localhost:8080
```

---

## Qué es

Una pequeña app web donde **componés un workflow arrastrando cajas a un lienzo**.
Cada caja es un fragmento de **Pascal editable**; las cajas, de arriba hacia
abajo, forman un único programa Pascal que corre en el motor vmpas. El objetivo
del demo es mostrar, de forma tangible, lo que hace a vmpas distinto de un motor
de scripting embebido típico:

- **Ejecución durable.** Una caja **Approval** llama a `Suspend`. La ejecución se
  pausa, su estado completo se serializa y se guarda, y la UI muestra
  **Aprobar / Rechazar**. Cuando elegís, el workflow **reanuda en un engine
  fresco** exactamente donde quedó — como lo haría un servicio real tras una
  decisión humana (o externa) que puede llegar en segundos o en días.
- **Traza de ejecución en vivo.** Cada caja reporta mientras corre, vía un
  callback Go bindeado (`Trace`): las cajas se marcan ✓ y la pausada late ⏸.
- **Binding Go ↔ Pascal.** Las entradas (`amount`, `approved`) son variables Go
  vinculadas; el resultado (`outcome`) se lee con `Get`. El backend expone
  `Trace`, SQL y HTTP al programa invitado.
- **Sandbox de capacidades.** Cada ejecución corre en un sandbox acotado (límites
  de pasos / tiempo). Al ser un demo local, se conceden las capacidades
  **Database** y **Network** para que funcionen las cajas de ejemplo (ver *Notas*).
- **Persistencia en SQLite.** Los programas guardados (cajas + posiciones +
  código), el historial de ejecuciones y los estados pausados viven en una base
  SQLite (`rad.db`).
- **Motor cero‑dependencias.** vmpas no arrastra nada. El driver de SQLite vive
  solo acá, en el **módulo Go propio** de este ejemplo, así que nunca entra al
  árbol de imports del motor.

## Cómo se usa

1. **Corrélo:** `cd examples/rad && go run .`, y abrí `http://localhost:8080`.
2. **Armá un flujo.** Arrastrá componentes de la paleta izquierda al lienzo. El
   **orden del flujo es la posición vertical** de las cajas (arriba → abajo); los
   conectores se dibujan solos. Arrastrá la cabecera de una caja para moverla.
3. **Editá cualquier caja.** Cada caja tiene su propio Pascal editable (CodeMirror,
   con modo Pascal). Editar una caja afecta **solo** a **esa** caja en **ese**
   programa — nunca al componente de la paleta ni a otros programas.
4. **Creá tu propio componente.** Clic en **➕ New component**, ponele nombre y
   código Pascal; se suma a la paleta y es reutilizable (se guarda en tu browser).
5. **Ejecutá.** Poné `amount` y apretá **▶ Run**. Con `amount > 1000` el flujo
   llega a la caja **Approval** y se pausa — clic en **Aprobar** o **Rechazar**
   para reanudar. Con un monto chico, la caja **Threshold** auto‑aprueba y corta.
6. **Guardá y recargá.** Escribí un nombre y **💾 Save** (queda en SQLite).
   Elegílo en **Load program…** para traerlo y volver a correrlo. El historial de
   ejecuciones se lista.
7. **Redimensioná** el editor del programa arrastrando el divisor horizontal de
   abajo.

## Componentes incluidos

| Componente | Qué hace su Pascal |
|---|---|
| **Start** | inicializa `outcome` |
| **Log** | `WriteLn` de un mensaje |
| **Threshold** | auto‑aprueba y hace `Halt` cuando `amount` está por debajo de un límite |
| **Approval** | `Suspend('approval')`; si se rechaza setea `outcome` y `Halt` |
| **Set approved** | setea `outcome := 'approved'` |
| **DB: list users** | consulta `Db*` sobre la SQLite local (`SELECT … FROM users`) |
| **HTTP: fetch** | `HttpGet` al endpoint incluido `/demo/api` + `JsonStr` |
| **End** | `WriteLn` del `outcome` final |
| **Custom Pascal** | (vía *New component*) lo que vos escribas |

## Cómo funciona

```
Browser (arrastrar‑y‑soltar, CodeMirror)
   │  compone las cajas en un único programa Pascal (+ Trace('id') por caja)
   ▼  POST /api/run  { program, amount }
Backend Go (net/http)
   │  vmpas.NewWith(sandbox) · UseDB(SQLite) · bind amount/approved/Trace
   ▼  RunDurable(program)
motor vmpas embebido
   ├─ corre hasta el final  → { done, output, outcome, trace }
   └─ llega a Suspend       → snapshot persistido en SQLite → { paused, id, trace }
                              Aprobar/Rechazar → POST /api/resume → ResumeDurable
```

El frontend compone el programa (e inyecta una llamada `Trace('<id de caja>')`
antes de cada caja para que el backend reporte el avance); el backend lo compila y
ejecuta en el motor, persistiendo el estado pausado y el historial en SQLite.

### API HTTP

| Endpoint | Para qué |
|---|---|
| `POST /api/run` | `{program, amount, flow}` → ejecuta; devuelve `done` / `paused` (+`id`) / `error`, con `output`, `outcome`, `trace` |
| `POST /api/resume` | `{id, approved}` → reanuda una ejecución pausada |
| `GET/POST /api/flows` | listar / guardar programas (SQLite) |
| `GET /api/flow?name=` | cargar un programa guardado |
| `GET /api/runs` | historial reciente de ejecuciones |
| `GET /demo/api` | un pequeño endpoint JSON incluido para el ejemplo HTTP |

## En qué se podría convertir

Es a propósito un demo compacto, pero la base es real. Extensiones naturales:

- **Ramas y paralelismo** — nodos de decisión con caminos verdadero/falso,
  fan‑out/join, en vez del pipeline lineal arriba‑abajo.
- **Un catálogo de componentes más rico** — timers/esperas, notificaciones por
  e‑mail/Slack, llamadas REST con headers de auth, escrituras a la DB,
  sub‑workflows.
- **Multi‑aprobación real** — varios `Suspend`, cada uno dirigido a un rol
  distinto, con el estado pausado en una base real indexada por un id de negocio.
- **Versionado y auditoría** — guardar versiones del programa; usar el modo
  determinista de vmpas + el log de auditoría para reproducir una ejecución exacta
  y mostrar quién/qué la tocó.
- **Exportar** — emitir el `.pas` compuesto para correrlo headless con
  `cmd/pasrun`, o embeber el programa guardado en otro servicio Go.
- **Un backend real engine‑por‑tenant** — el mismo modelo durable/sandbox detrás
  de una API autenticada: un servicio de workflows liviano y scriptable.

Como cada workflow es simplemente un programa vmpas, todo lo que el lenguaje y su
sandbox pueden hacer está disponible para una caja — esto es una UI sobre un motor
embebible real, no un juguete.

## Notas y advertencias

- **Solo demo local.** Para que funcionen las cajas de ejemplo SQL/HTTP, este
  servidor concede las capacidades `Database` y `Network` a cada ejecución. No lo
  expongas así a usuarios no confiables; un despliegue real acotaría las
  capacidades por workflow (ver [seguridad](../../docs/es/seguridad.md)).
- **CodeMirror carga por CDN.** Sin internet, los editores caen a textareas
  planos (igual editables).
- Los **componentes propios** se guardan en el browser (localStorage); los
  **programas** se guardan en el servidor en SQLite (`rad.db`, creada al iniciar).
- Las ejecuciones son **deterministas** (con seed), así que un estado pausado
  persistido reanuda de forma reproducible.

## Archivos

| Archivo | Rol |
|---|---|
| `main.go` | backend net/http: run/resume, SQLite, el endpoint `/demo/api` |
| `index.html` | toda la UI (JS vanilla + CodeMirror), embebida con `//go:embed` |
| `main_test.go` | tests del backend (run/pausa/resume durable, traza, SQLite) |
| `go.mod` | un **módulo aparte** para que el driver SQLite no toque el motor |
