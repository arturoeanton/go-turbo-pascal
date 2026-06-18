# Editor tooling: LSP, VSCode and Zed

The project provides modern tooling for `.pas` instead of (or in addition to) a
TUI IDE. The intelligence lives in an **LSP server** (`cmd/pls`); the editors are
thin clients.

## Language server (`pls`)

`cmd/pls` is an LSP server that speaks JSON-RPC over stdio. It offers:

- **Live diagnostics** (lexer, parser and type/semantic errors via the
  real compiler).
- **Document symbols** (`documentSymbol`): program/unit, constants,
  types, variables and routines.
- **Hover**: shows the declaration/signature of the symbol under the cursor.
- **Go-to-definition** (`definition`): jumps to the identifier's declaration.
- **Autocompletion** (`completion`): document symbols plus Pascal keywords.

Build:

```bash
go build -o pls ./cmd/pls
```

Status: ✅ `initialize` handshake, `didOpen`/`didChange`/`didClose`, and
`publishDiagnostics`. Tested in `internal/lsp`.

## VSCode extension (`editors/vscode`)

Includes:

- registration of the Pascal language and syntax highlighting
  (`syntaxes/pascal.tmLanguage.json`),
- an LSP client that launches `pls` (`src/extension.ts`).

Install for development:

```bash
cd editors/vscode
npm install
npm run compile
# Open this folder in VSCode and press F5 (Extension Development Host).
```

Make sure `pls` is on the PATH, or configure `bpgoPascal.serverPath`.

## Zed extension (`editors/zed`)

A complete Rust/WASM extension: it registers the Pascal language, its
configuration and highlighting (`languages/pascal/`), and an **LSP client**
(`src/lib.rs` implements `zed_extension_api::Extension` and resolves `pls` from
the PATH).

Install (dev): in Zed, **Extensions → Install Dev Extension** and choose
`editors/zed` (Zed compiles the extension to WASM; it requires the
`wasm32-wasip1` target).

Before publishing: pin the `commit` of the tree-sitter grammar in
`extension.toml` (`[grammars.pascal]`) and verify the node names in
`highlights.scm`.

Debugging in Zed: pending Zed's debug-adapter API (still
evolving). The `pdap` adapter already works in VSCode.

## DAP (debugging) ✅

`cmd/pdap` is the debug adapter (Debug Adapter Protocol) over stdio,
built on top of the `ir.Debugger` engine. It supports:

- **launch** of a `.pas`, **line breakpoints**, **continue** and **step**
  (next/stepIn/stepOut), one thread and one stack frame, and inspection of
  **global variables**.
- The codegen populates the `SourceMap` (instruction → line) and the
  `ir.Debugger` runs the VM at line granularity.

Build:

```bash
go build -o pdap ./cmd/pdap
```

The VSCode extension registers the debug type `bpgo-pascal` and launches
`pdap`. Configurable with `bpgoPascal.debugAdapterPath`. Tested in
`internal/dap` (full session: launch → breakpoint → continue → terminated).
