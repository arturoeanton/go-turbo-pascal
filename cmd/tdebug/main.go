// tdebug is the BPGo source-level debugger. It is a thin CLI around
// the internal debug package: it accepts commands to set
// breakpoints, step, continue, evaluate expressions and print the
// call stack. The debugger can be driven by the conformance harness
// or interactively from a terminal.
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/arturoeanton/go-turbo-pascal/internal/debug"
)

func main() {
	dbg := debug.New()
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "-V", "--version":
		fmt.Println("tdebug 0.2.0 (BPGo debugger)")
		return
	case "-h", "--help", "help":
		usage()
		return
	}
	args := os.Args[1:]
	for len(args) > 0 {
		cmd := args[0]
		args = args[1:]
		switch cmd {
		case "break":
			if len(args) < 2 {
				fmt.Fprintln(os.Stderr, "usage: break FILE LINE")
				os.Exit(1)
			}
			line, err := strconv.Atoi(args[1])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			dbg.SetBreakpoint(args[0], line)
			args = args[2:]
		case "step":
			dbg.Step()
		case "continue":
			dbg.Continue()
		case "watch":
			if len(args) < 1 {
				fmt.Fprintln(os.Stderr, "usage: watch EXPR")
				os.Exit(1)
			}
			dbg.AddWatch(args[0], true)
			args = args[1:]
		case "snapshot":
			fmt.Println(dbg.Snapshot())
		case "help":
			usage()
		case "-h", "--help":
			usage()
		case "exit":
			return
		default:
			fmt.Fprintln(os.Stderr, "unknown command:", cmd)
			usage()
			os.Exit(1)
		}
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, strings.TrimSpace(`
tdebug - BPGo source-level debugger

commands:
  break FILE LINE    set a breakpoint
  watch EXPR         add a watch expression
  step               single-step
  continue           resume execution
  snapshot           print the current debugger state
  exit               exit the debugger
`))
	flag.PrintDefaults()
}
