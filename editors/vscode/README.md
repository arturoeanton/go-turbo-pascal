# BPGo Turbo Pascal 7 — VSCode extension

Turbo Pascal 7 support for VSCode: syntax highlighting, live diagnostics (via
the `pls` language server) and debugging (via the `pdap` debug adapter).

## Requirements

Build the BPGo tools and put them on your `PATH` (or set the paths in settings):

```bash
go build -o pls ./cmd/pls     # language server
go build -o pdap ./cmd/pdap   # debug adapter
```

Settings: `bpgoPascal.serverPath` (default `pls`) and
`bpgoPascal.debugAdapterPath` (default `pdap`).

## Run (dev)

```bash
cd editors/vscode
npm install
npm run compile
```

Open this folder in VSCode and press **F5** (Extension Development Host).

## Features

- ✅ Syntax highlighting for `.pas`/`.pp`/`.inc`/`.dpr`.
- ✅ Live diagnostics (lex/parse/semantic errors) via LSP.
- ✅ Debugging: set breakpoints, continue and step, inspect global variables.
  Use the run configuration **"Depurar programa Pascal"** (debug type
  `bpgo-pascal`, `program` defaults to the current file).
