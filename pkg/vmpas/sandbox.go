package vmpas

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
)

// registerHostCaps installs the capability-gated host builtins. These are vmpas
// extensions (not part of the Turbo Pascal 7 RTL): each is registered only when
// its capability is granted, so under the default-deny sandbox calling one is a
// compile-time "unknown identifier" error rather than a silent no-op.
//
//	Env      GetEnv(name: string): string        -> os.Getenv
//	Exec     Exec(command: string): Integer       -> run via the system shell
//	Network  HttpGet(url: string): string         -> HTTP GET body (or empty)
//
// Names are registered in PascalCase; aliasLowercase mirrors them so codegen's
// lowercased externals resolve.
func (e *Engine) registerHostCaps(vm *ir.VM) {
	if e.caps.Env {
		vm.Builtins["GetEnv"] = func(_ *ir.VM, args []ir.Value) ir.Value {
			if len(args) == 0 {
				return ir.Value{Kind: ir.VKStr}
			}
			return ir.Value{Kind: ir.VKStr, Str: os.Getenv(irToStr(args[0]))}
		}
	}
	if e.caps.Exec {
		vm.Builtins["Exec"] = func(_ *ir.VM, args []ir.Value) ir.Value {
			if len(args) == 0 {
				return ir.Value{Kind: ir.VKInt, Int: -1}
			}
			var cmd *exec.Cmd
			if runtime.GOOS == "windows" {
				cmd = exec.Command("cmd", "/C", irToStr(args[0]))
			} else {
				cmd = exec.Command("/bin/sh", "-c", irToStr(args[0]))
			}
			code := 0
			if err := cmd.Run(); err != nil {
				if ee, ok := err.(*exec.ExitError); ok {
					code = ee.ExitCode()
				} else {
					code = -1
				}
			}
			return ir.Value{Kind: ir.VKInt, Int: int64(code)}
		}
	}
	if e.caps.Network {
		vm.Builtins["HttpGet"] = func(_ *ir.VM, args []ir.Value) ir.Value {
			if len(args) == 0 {
				return ir.Value{Kind: ir.VKStr}
			}
			resp, err := http.Get(irToStr(args[0]))
			if err != nil {
				return ir.Value{Kind: ir.VKStr}
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return ir.Value{Kind: ir.VKStr}
			}
			return ir.Value{Kind: ir.VKStr, Str: string(body)}
		}
	}
}

// irToStr extracts a string from an IR value (string or char).
func irToStr(v ir.Value) string {
	switch v.Kind {
	case ir.VKStr:
		return v.Str
	case ir.VKChar:
		return string(rune(v.Ch))
	}
	return ""
}
