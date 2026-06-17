# BPGo Turbo Pascal 7 — Zed extension

Adds Turbo Pascal 7 support to [Zed](https://zed.dev): syntax highlighting and
the BPGo language server (`pls`) for live diagnostics.

## Requirements

Build the BPGo tools and put them on your `PATH`:

```bash
go build -o pls ./cmd/pls     # language server
go build -o pdap ./cmd/pdap   # debug adapter (used by the VSCode extension)
```

## Install (dev)

1. In Zed: **Extensions → Install Dev Extension** and pick this `editors/zed`
   folder. Zed compiles the Rust extension to WebAssembly (needs the
   `wasm32-wasip1` Rust target).
2. Open a `.pas` file. Diagnostics from `pls` appear inline.

## Before publishing

- **Pin the grammar.** `extension.toml`'s `[grammars.pascal]` points at a
  tree-sitter Pascal grammar with a placeholder `commit`. Set it to a real
  revision (and verify the node names in `languages/pascal/highlights.scm`).

## Status

- ✅ Language registration, config and syntax highlighting (with a pinned grammar).
- ✅ LSP client (`pls`) — diagnostics.
- ⏳ Debugging in Zed is pending Zed's evolving debug-adapter extension API; the
  `pdap` adapter already works in the VSCode extension.
