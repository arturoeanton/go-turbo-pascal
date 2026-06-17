// bpgo is the main entry point for the BPGo command-line driver.
// The command takes Pascal source files, compiles them, optionally
// runs them, and supports a project mode (build, make, run).
package main

import (
	"fmt"
	"os"

	"github.com/arturoeanton/go-turbo-pascal/internal/cli"
	"github.com/arturoeanton/go-turbo-pascal/internal/conformance"
)

func main() {
	// Sub-command dispatch: bpgo test-compat runs the conformance
	// harness; everything else is the standard CLI driver.
	if len(os.Args) >= 2 && os.Args[1] == "test-compat" {
		root := "."
		out := "compat/report.json"
		// Allow --root and --out overrides.
		for i := 2; i < len(os.Args); i++ {
			switch os.Args[i] {
			case "--root":
				if i+1 < len(os.Args) {
					root = os.Args[i+1]
					i++
				}
			case "--out":
				if i+1 < len(os.Args) {
					out = os.Args[i+1]
					i++
				}
			}
		}
		r := conformance.New(root, os.Stdout)
		if err := r.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		_ = out
		r.WriteStdout()
		return
	}
	code := cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	os.Exit(code)
}
