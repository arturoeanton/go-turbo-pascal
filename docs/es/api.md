# API pública y estabilidad (`pkg/vmpas`)

`pkg/vmpas` es la superficie soportada y embebible de este proyecto. Sigue
[Versionado Semántico](https://semver.org/lang/es/): dentro de una versión mayor
(`v1.x`), los símbolos listados abajo no se rompen — las firmas se mantienen
compatibles y el comportamiento sigue lo documentado. Se puede *agregar* API
nueva (bumps menores). Todo lo que no esté acá (en particular los paquetes
`internal/...`) es inestable y puede cambiar en cualquier momento.

Un test de contrato (`pkg/vmpas/api_contract_test.go`) fija esta superficie: un
cambio que la rompa hace fallar el build, así la rotura se detecta antes de
publicar.

## Superficie estable

### Ciclo de vida del Engine
- `func New() *Engine` — engine totalmente restringido.
- `func NewWith(caps Capabilities) *Engine`
- `type Engine` con métodos:
  - `Var(name string, ptr any) error`
  - `Function(name string, fn any) error` / `Func(...)`
  - `Process(name string, fn any) error` / `Procedure(...)`
  - `Run(code string) error`
  - `Compile(code string) (*Script, error)`
  - `Output() string`
  - `SetCapabilities(caps Capabilities)`
  - `UseDB(db SQLDB)`
  - `Analyze(code string) (*CapReport, error)`
  - `AuditLog() []AuditEntry`
  - `RunDurable(code string) (*State, error)`
  - `ResumeDurable(code string, st *State) (*State, error)`

### Compile-once / run-many
- `type Script` con `Run() error` y `Output() string`.

### Sandbox
- `type Capabilities struct { ... }` (campos: `FileSystem`, `Network`, `Exec`,
  `Env`, `Database`, `MaxSteps`, `MaxHeap`, `MaxOutput`, `MaxCallDepth`,
  `MaxDuration`, `Deterministic`, `Seed`, `Audit`).
- Presets: `Restricted()`, `Full()`, `Sandboxed()`.
- `func RunSandboxed(code string, caps Capabilities) (string, error)`

### Inferencia de capacidades y auditoría
- `type Capability string` con `CapFileSystem`, `CapNetwork`, `CapExec`,
  `CapEnv`, `CapDatabase`.
- `type CapReport struct { Required []Capability; Calls map[Capability][]string }`
  con `Needs(Capability) bool`.
- `type AuditEntry struct { Capability Capability; Builtin string; Args []string }`

### Ejecución durable
- `type State struct { Tag string; Data []byte; Output string }`

### Puente de base de datos
- `type SQLDB interface { ... }`, `type SQLRows interface { ... }`
- `func WrapSQLDB(db *sql.DB) SQLDB`

### Conveniencia a nivel de paquete (engine por defecto compartido)
- `Run`, `Var`, `Function`/`Func`, `Process`/`Procedure`, `Output`, `Reset`,
  `SetCapabilities`.

## Lo que NO es estable
- Todo bajo `internal/` (lexer, parser, codegen, ir, rtl). Los embebedores deben
  pasar por `pkg/vmpas`.
- La superficie del lenguaje Pascal evoluciona bajo su propia política de
  compatibilidad (ver [compatibility.md](compatibility.md)); las features
  modernas están detrás de `{$MODE BPGO}`.
