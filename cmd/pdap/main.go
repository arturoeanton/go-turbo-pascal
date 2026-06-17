// pdap is the BPGo Pascal debug adapter. It speaks the Debug Adapter Protocol
// over stdio and drives the embedded VM debugger, so editors (VSCode, Zed) can
// set breakpoints and step through .pas programs.
package main

import (
	"os"

	"github.com/arturoeanton/go-turbo-pascal/internal/dap"
)

func main() {
	srv := dap.NewServer(os.Stdin, os.Stdout)
	if err := srv.Run(); err != nil {
		os.Exit(1)
	}
}
