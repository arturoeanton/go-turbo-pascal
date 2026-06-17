// pls is the BPGo Pascal Language Server. It speaks LSP over stdio and
// provides live diagnostics for Turbo Pascal 7 source. It is the backend for
// the VSCode and Zed editor extensions.
package main

import (
	"os"

	"github.com/arturoeanton/go-turbo-pascal/internal/lsp"
)

func main() {
	srv := lsp.NewServer(os.Stdin, os.Stdout)
	if err := srv.Run(); err != nil {
		os.Exit(1)
	}
}
