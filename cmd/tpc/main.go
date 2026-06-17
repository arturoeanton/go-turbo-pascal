// tpc is a thin wrapper around bpgo that selects TP7-compatible
// defaults. It exists so users with a Borland-style muscle memory
// can type "tpc hello.pas" and get the same behaviour.
package main

import (
	"os"

	"github.com/arturoeanton/go-turbo-pascal/internal/cli"
)

func main() {
	argv := append([]string{"-M", "-d"}, os.Args[1:]...)
	code := cli.Run(argv, os.Stdin, os.Stdout, os.Stderr)
	os.Exit(code)
}
