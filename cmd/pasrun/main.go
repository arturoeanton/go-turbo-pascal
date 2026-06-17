// pasrun compiles and runs a Turbo Pascal 7 source file on the real BPGo
// engine (internal/codegen + internal/ir). Unlike the legacy bpgo pipeline,
// pasrun uses the full procedural compiler.
//
// Usage:
//
//	pasrun programa.pas [args...]
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/arturoeanton/go-turbo-pascal/internal/codegen"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "uso: pasrun programa.pas [args...]")
		os.Exit(2)
	}
	path := os.Args[1]
	src, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "pasrun:", err)
		os.Exit(1)
	}
	prog, err := codegen.Compile(string(src), path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error de compilación:", err)
		os.Exit(1)
	}
	// Forward stdin to the program (for Read/ReadLn) when piped.
	var input string
	if fi, _ := os.Stdin.Stat(); fi != nil && fi.Mode()&os.ModeCharDevice == 0 {
		if data, err := io.ReadAll(os.Stdin); err == nil {
			input = string(data)
		}
	}
	out, code, err := codegen.Run(prog, os.Args[2:], input)
	fmt.Print(out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error en ejecución:", err)
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}
