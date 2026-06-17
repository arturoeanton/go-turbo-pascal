package system

import (
	"strconv"
	"strings"

	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
)

// registerSysUtils installs modern (SysUtils-style) string and conversion
// helpers as builtins: IntToStr/StrToInt, FloatToStr/StrToFloat,
// UpperCase/LowerCase, Trim and friends.
func registerSysUtils(vm *ir.VM) {
	str := func(s string) ir.Value { return ir.Value{Kind: ir.VKStr, Str: s} }

	vm.Builtins["IntToStr"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		return str(strconv.FormatInt(irToInt(a[0]), 10))
	}
	vm.Builtins["StrToInt"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		n, _ := strconv.ParseInt(strings.TrimSpace(valueToString(a[0])), 10, 64)
		return ir.Value{Kind: ir.VKInt, Int: n}
	}
	vm.Builtins["StrToIntDef"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		if n, err := strconv.ParseInt(strings.TrimSpace(valueToString(a[0])), 10, 64); err == nil {
			return ir.Value{Kind: ir.VKInt, Int: n}
		}
		if len(a) > 1 {
			return ir.Value{Kind: ir.VKInt, Int: irToInt(a[1])}
		}
		return ir.Value{Kind: ir.VKInt}
	}
	vm.Builtins["FloatToStr"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		return str(strconv.FormatFloat(irToReal(a[0]), 'g', -1, 64))
	}
	vm.Builtins["StrToFloat"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		f, _ := strconv.ParseFloat(strings.TrimSpace(valueToString(a[0])), 64)
		return ir.Value{Kind: ir.VKReal, Real: f}
	}
	vm.Builtins["UpperCase"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		return str(strings.ToUpper(valueToString(a[0])))
	}
	vm.Builtins["LowerCase"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		return str(strings.ToLower(valueToString(a[0])))
	}
	vm.Builtins["Trim"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		return str(strings.TrimSpace(valueToString(a[0])))
	}
	vm.Builtins["TrimLeft"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		return str(strings.TrimLeft(valueToString(a[0]), " \t\r\n"))
	}
	vm.Builtins["TrimRight"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		return str(strings.TrimRight(valueToString(a[0]), " \t\r\n"))
	}
	vm.Builtins["StringOfChar"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		if len(a) < 2 {
			return str("")
		}
		var ch byte
		switch a[0].Kind {
		case ir.VKChar:
			ch = a[0].Ch
		case ir.VKStr:
			if len(a[0].Str) > 0 {
				ch = a[0].Str[0]
			}
		default:
			ch = byte(irToInt(a[0]))
		}
		n := int(irToInt(a[1]))
		if n < 0 {
			n = 0
		}
		return str(strings.Repeat(string([]byte{ch}), n))
	}
}
