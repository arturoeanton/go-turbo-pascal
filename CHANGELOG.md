# Changelog

Formato basado en [Keep a Changelog](https://keepachangelog.com/es/1.1.0/).
Este proyecto sigue [Versionado Semántico](https://semver.org/lang/es/): a partir
de la v1.0.0, la API pública de `pkg/vmpas` es estable dentro de cada serie mayor
(ver [docs/api.md](docs/api.md)).

## [No publicado]

### Agregado
- **Congelamiento de la API de `pkg/vmpas`** (N6): documento de estabilidad
  ([docs/api.md](docs/api.md)) y test de contrato que fija la superficie pública.
- `LICENSE` (MIT), `CHANGELOG.md` e integración continua (build + vet + test).

## [0.7.0] — Sandbox probable: inferencia de capacidades + traza auditable
### Agregado
- `Engine.Analyze`: infiere qué capacidades de host necesita un script
  escaneando el bytecode (G1).
- `Capabilities.Audit` + `Engine.AuditLog`: registra cada llamada gateada en
  orden de ejecución con sus argumentos (G2).

## [0.6.0] — Ejecución determinística + snapshot/resume (Fase F, núcleo)
### Agregado
- Modo determinista (`Capabilities.Deterministic`/`Seed`; `Randomize` seedeable).
- Snapshot/resume completo del estado del VM (globals, locals, pila de operandos,
  pila de llamadas con PCs, heap con grafos de punteros, RNG, excepciones).
- API durable: builtin `Suspend`, `Engine.RunDurable`/`ResumeDurable`, `State`.

## [0.5.0] — Robustez para reglas de negocio embebidas (serie N)
### Agregado
- Chequeo de tipos por categoría en asignaciones, sin falsos positivos (N5).
- Hardening multi-tenant: `Capabilities.MaxOutput`/`MaxCallDepth`, preset
  `Sandboxed()`, `RunSandboxed`, reset de estado entre ejecuciones (N7).
- Stdlib de gestión: IVA/redondeo/porcentajes con `Currency` exacto, días
  hábiles/edad/fin-de-mes, padding/máscaras/validaciones, `Split` (N8).

## [0.4.0] y anteriores
- Núcleo del lenguaje (TP7 procedural + OOP), VM de bytecode, RTL, units.
- Features modernas bajo `{$MODE BPGO}`: inferencia/`let`, helpers, `match` +
  Option/ADTs, `defer`/`panic`/`recover`, `spawn` + `Channel<T>`.
- `pkg/vmpas`: motor embebible, binding Go↔Pascal, sandbox de capacidades,
  integración HTTP/SQL/JSON, tipo `Currency`, stdlib de negocio (N1–N4).
- Tooling: diagnósticos IDE-grade; LSP/DAP y plugins de editor.

[No publicado]: https://github.com/arturoeanton/go-turbo-pascal/compare/v0.7.0...HEAD
[0.7.0]: https://github.com/arturoeanton/go-turbo-pascal/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/arturoeanton/go-turbo-pascal/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/arturoeanton/go-turbo-pascal/compare/v0.4.0...v0.5.0
