package crt

import (
	"fmt"
	"time"

	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
)

// tpToAnsi maps Turbo Pascal 7 color codes (0..15) to ANSI foreground codes.
var tpToAnsi = [16]int{
	30, 34, 32, 36, 31, 35, 33, 37, // 0..7  (black..light gray)
	90, 94, 92, 96, 91, 95, 93, 97, // 8..15 (dark gray..white)
}

// Register installs the Crt unit's builtins on the VM. Screen control is
// emitted as ANSI escape sequences to the VM output, so `uses Crt` programs
// work on any ANSI terminal.
func Register(vm *ir.VM) {
	out := func(vm *ir.VM, s string) { vm.Output.WriteString(s) }
	nilv := ir.Value{Kind: ir.VKNil}

	vm.Builtins["ClrScr"] = func(vm *ir.VM, _ []ir.Value) ir.Value {
		out(vm, "\x1b[2J\x1b[H")
		return nilv
	}
	vm.Builtins["ClrEol"] = func(vm *ir.VM, _ []ir.Value) ir.Value {
		out(vm, "\x1b[K")
		return nilv
	}
	vm.Builtins["GotoXY"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		if len(a) >= 2 {
			out(vm, fmt.Sprintf("\x1b[%d;%dH", int(a[1].Int), int(a[0].Int)))
		}
		return nilv
	}
	vm.Builtins["TextColor"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		if len(a) >= 1 {
			out(vm, fmt.Sprintf("\x1b[%dm", tpToAnsi[int(a[0].Int)&15]))
		}
		return nilv
	}
	vm.Builtins["TextBackground"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		if len(a) >= 1 {
			out(vm, fmt.Sprintf("\x1b[%dm", tpToAnsi[int(a[0].Int)&7]+10))
		}
		return nilv
	}
	vm.Builtins["NormVideo"] = func(vm *ir.VM, _ []ir.Value) ir.Value { out(vm, "\x1b[0m"); return nilv }
	vm.Builtins["HighVideo"] = func(vm *ir.VM, _ []ir.Value) ir.Value { out(vm, "\x1b[1m"); return nilv }
	vm.Builtins["LowVideo"] = func(vm *ir.VM, _ []ir.Value) ir.Value { out(vm, "\x1b[2m"); return nilv }
	vm.Builtins["TextMode"] = func(vm *ir.VM, _ []ir.Value) ir.Value { return nilv }
	vm.Builtins["Window"] = func(vm *ir.VM, _ []ir.Value) ir.Value { return nilv }
	vm.Builtins["Sound"] = func(vm *ir.VM, _ []ir.Value) ir.Value { return nilv }
	vm.Builtins["NoSound"] = func(vm *ir.VM, _ []ir.Value) ir.Value { return nilv }
	vm.Builtins["WhereX"] = func(vm *ir.VM, _ []ir.Value) ir.Value { return ir.Value{Kind: ir.VKInt, Int: 1} }
	vm.Builtins["WhereY"] = func(vm *ir.VM, _ []ir.Value) ir.Value { return ir.Value{Kind: ir.VKInt, Int: 1} }

	vm.Builtins["Delay"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		if len(a) >= 1 && a[0].Int > 0 {
			time.Sleep(time.Duration(a[0].Int) * time.Millisecond)
		}
		return nilv
	}

	// ReadKey returns the next input character (reads a line from the input and
	// yields its first character; a simplified, line-oriented model).
	vm.Builtins["ReadKey"] = func(vm *ir.VM, _ []ir.Value) ir.Value {
		line := vm.Input.ReadLine()
		if len(line) == 0 {
			return ir.Value{Kind: ir.VKChar, Ch: 13} // Enter
		}
		return ir.Value{Kind: ir.VKChar, Ch: line[0]}
	}
	// KeyPressed has no asynchronous key model here; report no key waiting.
	vm.Builtins["KeyPressed"] = func(vm *ir.VM, _ []ir.Value) ir.Value {
		return ir.Value{Kind: ir.VKBool, Bool: false}
	}
}

// CrtExports are the names the Crt unit makes available; codegen uses these to
// allow calls after `uses Crt`.
var CrtExports = []string{
	"ClrScr", "ClrEol", "GotoXY", "TextColor", "TextBackground",
	"NormVideo", "HighVideo", "LowVideo", "TextMode", "Window",
	"Sound", "NoSound", "WhereX", "WhereY", "Delay", "ReadKey", "KeyPressed",
}
