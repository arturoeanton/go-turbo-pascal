# Tooling de editor: LSP, VSCode y Zed

El proyecto provee tooling moderno para `.pas` en lugar de (o además de) una IDE
TUI. La inteligencia vive en un **servidor LSP** (`cmd/pls`); los editores son
clientes finos.

## Servidor de lenguaje (`pls`)

`cmd/pls` es un servidor LSP que habla JSON-RPC por stdio. Hoy ofrece
**diagnósticos en vivo** (errores de lexer, parser y de tipo/semántica vía el
compilador real). Hover, autocompletado e ir-a-definición están planificados.

Compilar:

```bash
go build -o pls ./cmd/pls
```

Estado: ✅ handshake `initialize`, `didOpen`/`didChange`/`didClose`, y
`publishDiagnostics`. Probado en `internal/lsp`.

## Extensión de VSCode (`editors/vscode`)

Incluye:

- registro del lenguaje Pascal y resaltado de sintaxis
  (`syntaxes/pascal.tmLanguage.json`),
- cliente LSP que lanza `pls` (`src/extension.ts`).

Instalar para desarrollo:

```bash
cd editors/vscode
npm install
npm run compile
# Abrir esta carpeta en VSCode y pulsar F5 (Extension Development Host).
```

Asegurate de que `pls` esté en el PATH, o configurá `bpgoPascal.serverPath`.

## Extensión de Zed (`editors/zed`)

Extensión Rust/WASM completa: registra el lenguaje Pascal, su configuración y
resaltado (`languages/pascal/`), y un **cliente LSP** (`src/lib.rs` implementa
`zed_extension_api::Extension` y resuelve `pls` desde el PATH).

Instalar (dev): en Zed, **Extensions → Install Dev Extension** y elegir
`editors/zed` (Zed compila la extensión a WASM; requiere el target
`wasm32-wasip1`).

Antes de publicar: fijar el `commit` de la gramática tree-sitter en
`extension.toml` (`[grammars.pascal]`) y verificar los nombres de nodos en
`highlights.scm`.

Depuración en Zed: pendiente de la API de debug-adapter de Zed (aún en
evolución). El adaptador `pdap` ya funciona en VSCode.

## DAP (depuración) ✅

`cmd/pdap` es el adaptador de depuración (Debug Adapter Protocol) por stdio,
montado sobre el motor `ir.Debugger`. Soporta:

- **launch** de un `.pas`, **breakpoints por línea**, **continue** y **step**
  (next/stepIn/stepOut), un hilo y un frame de pila, e inspección de
  **variables globales**.
- El codegen puebla el `SourceMap` (instrucción → línea) y el `ir.Debugger`
  ejecuta la VM con granularidad de línea.

Compilar:

```bash
go build -o pdap ./cmd/pdap
```

La extensión de VSCode registra el tipo de depuración `bpgo-pascal` y lanza
`pdap`. Configurable con `bpgoPascal.debugAdapterPath`. Probado en
`internal/dap` (sesión completa: launch → breakpoint → continue → terminated).
