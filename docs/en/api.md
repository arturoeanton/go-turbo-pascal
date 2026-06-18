# Public API and stability (`pkg/vmpas`)

`pkg/vmpas` is the supported, embeddable surface of this project. It follows
[Semantic Versioning](https://semver.org/): within a major version (`v1.x`),
the symbols listed below will not break — signatures stay compatible and
behavior stays as documented. New API may be added (minor bumps). Anything not
listed here (notably the `internal/...` packages) is unstable and may change at
any time.

A contract test (`pkg/vmpas/api_contract_test.go`) pins this surface: a
breaking change to it fails the build, so breakage is caught before release.

## Stable surface

### Engine lifecycle
- `func New() *Engine` — fully restricted engine.
- `func NewWith(caps Capabilities) *Engine`
- `type Engine` with methods:
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
- `type Script` with `Run() error` and `Output() string`.

### Sandbox
- `type Capabilities struct { ... }` (fields: `FileSystem`, `Network`, `Exec`,
  `Env`, `Database`, `MaxSteps`, `MaxHeap`, `MaxOutput`, `MaxCallDepth`,
  `MaxDuration`, `Deterministic`, `Seed`, `Audit`).
- Presets: `Restricted()`, `Full()`, `Sandboxed()`.
- `func RunSandboxed(code string, caps Capabilities) (string, error)`

### Capability inference and audit
- `type Capability string` with `CapFileSystem`, `CapNetwork`, `CapExec`,
  `CapEnv`, `CapDatabase`.
- `type CapReport struct { Required []Capability; Calls map[Capability][]string }`
  with `Needs(Capability) bool`.
- `type AuditEntry struct { Capability Capability; Builtin string; Args []string }`

### Durable execution
- `type State struct { Tag string; Data []byte; Output string }`

### Database bridge
- `type SQLDB interface { ... }`, `type SQLRows interface { ... }`
- `func WrapSQLDB(db *sql.DB) SQLDB`

### Package-level convenience (shared default engine)
- `Run`, `Var`, `Function`/`Func`, `Process`/`Procedure`, `Output`, `Reset`,
  `SetCapabilities`.

## What is NOT stable
- Everything under `internal/` (lexer, parser, codegen, ir, rtl). Embedders must
  go through `pkg/vmpas`.
- The Pascal language surface evolves under its own compatibility policy
  (see [compatibility.md](compatibility.md)); modern features are gated behind
  `{$MODE BPGO}`.
