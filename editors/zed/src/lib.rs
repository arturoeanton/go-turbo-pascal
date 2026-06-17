//! Zed extension for BPGo Turbo Pascal 7.
//!
//! It wires Zed to the BPGo language server (`pls`) and debug adapter (`pdap`),
//! which must be on PATH (build them with `go build -o pls ./cmd/pls` and
//! `go build -o pdap ./cmd/pdap`).

use zed_extension_api::{self as zed, Command, LanguageServerId, Result, Worktree};

struct BpgoPascalExtension;

impl zed::Extension for BpgoPascalExtension {
    fn new() -> Self {
        BpgoPascalExtension
    }

    fn language_server_command(
        &mut self,
        _language_server_id: &LanguageServerId,
        worktree: &Worktree,
    ) -> Result<Command> {
        let path = worktree
            .which("pls")
            .ok_or_else(|| "could not find `pls` on PATH (build ./cmd/pls)".to_string())?;
        Ok(Command {
            command: path,
            args: vec![],
            env: worktree.shell_env(),
        })
    }
}

zed::register_extension!(BpgoPascalExtension);
