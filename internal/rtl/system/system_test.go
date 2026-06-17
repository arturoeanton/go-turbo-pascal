package system

import (
	"strings"
	"testing"

	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
)

func TestSystemWriteLn(t *testing.T) {
	vm := ir.NewVM(&ir.Program{Modules: []*ir.Module{{Name: "main", Funcs: map[string]*ir.Function{}}}, Entry: "main"})
	Register(vm)
	vm.Builtins["WriteLn"](vm, []ir.Value{ir.Value{Kind: ir.VKStr, Str: "Hello"}})
	if !strings.Contains(vm.Output.String(), "Hello") {
		t.Errorf("expected Hello, got %q", vm.Output.String())
	}
}

func TestSystemMath(t *testing.T) {
	vm := ir.NewVM(&ir.Program{})
	Register(vm)
	if v := vm.Builtins["Abs"](vm, []ir.Value{ir.Value{Kind: ir.VKInt, Int: -5}}); v.Int != 5 {
		t.Errorf("Abs(-5) = %d", v.Int)
	}
	if v := vm.Builtins["Sqr"](vm, []ir.Value{ir.Value{Kind: ir.VKInt, Int: 7}}); v.Int != 49 {
		t.Errorf("Sqr(7) = %d", v.Int)
	}
	if v := vm.Builtins["Sqrt"](vm, []ir.Value{ir.Value{Kind: ir.VKReal, Real: 9.0}}); v.Real != 3.0 {
		t.Errorf("Sqrt(9) = %g", v.Real)
	}
}

func TestSystemStrings(t *testing.T) {
	vm := ir.NewVM(&ir.Program{})
	Register(vm)
	if v := vm.Builtins["Length"](vm, []ir.Value{ir.Value{Kind: ir.VKStr, Str: "hello"}}); v.Int != 5 {
		t.Errorf("Length('hello') = %d", v.Int)
	}
	if v := vm.Builtins["Copy"](vm, []ir.Value{ir.Value{Kind: ir.VKStr, Str: "hello"}, ir.Value{Kind: ir.VKInt, Int: 2}, ir.Value{Kind: ir.VKInt, Int: 3}}); v.Str != "ell" {
		t.Errorf("Copy('hello',2,3) = %q", v.Str)
	}
	if v := vm.Builtins["Concat"](vm, []ir.Value{ir.Value{Kind: ir.VKStr, Str: "foo"}, ir.Value{Kind: ir.VKStr, Str: "bar"}}); v.Str != "foobar" {
		t.Errorf("Concat('foo','bar') = %q", v.Str)
	}
	if v := vm.Builtins["Pos"](vm, []ir.Value{ir.Value{Kind: ir.VKStr, Str: "lo"}, ir.Value{Kind: ir.VKStr, Str: "hello"}}); v.Int != 4 {
		t.Errorf("Pos('lo','hello') = %d", v.Int)
	}
}

func TestSystemOrdinal(t *testing.T) {
	vm := ir.NewVM(&ir.Program{})
	Register(vm)
	if v := vm.Builtins["Ord"](vm, []ir.Value{ir.Value{Kind: ir.VKChar, Ch: 'A'}}); v.Int != 65 {
		t.Errorf("Ord('A') = %d", v.Int)
	}
	if v := vm.Builtins["Chr"](vm, []ir.Value{ir.Value{Kind: ir.VKInt, Int: 66}}); v.Ch != 'B' {
		t.Errorf("Chr(66) = %d", v.Ch)
	}
	if v := vm.Builtins["Pred"](vm, []ir.Value{ir.Value{Kind: ir.VKInt, Int: 10}}); v.Int != 9 {
		t.Errorf("Pred(10) = %d", v.Int)
	}
	if v := vm.Builtins["Succ"](vm, []ir.Value{ir.Value{Kind: ir.VKInt, Int: 10}}); v.Int != 11 {
		t.Errorf("Succ(10) = %d", v.Int)
	}
	if v := vm.Builtins["Odd"](vm, []ir.Value{ir.Value{Kind: ir.VKInt, Int: 3}}); !v.Bool {
		t.Error("Odd(3) should be true")
	}
}

func TestSystemHalt(t *testing.T) {
	vm := ir.NewVM(&ir.Program{})
	Register(vm)
	vm.Builtins["Halt"](vm, []ir.Value{ir.Value{Kind: ir.VKInt, Int: 42}})
	if !vm.Halted {
		t.Error("expected halted")
	}
	if vm.ExitCode != 42 {
		t.Errorf("expected exit 42, got %d", vm.ExitCode)
	}
}

func TestSystemRandom(t *testing.T) {
	vm1 := ir.NewVM(&ir.Program{})
	vm2 := ir.NewVM(&ir.Program{})
	Register(vm1)
	Register(vm2)
	// Same initial seed should produce same sequence.
	v1 := vm1.Builtins["Random"](vm1, []ir.Value{ir.Value{Kind: ir.VKInt, Int: 1000}})
	v2 := vm2.Builtins["Random"](vm2, []ir.Value{ir.Value{Kind: ir.VKInt, Int: 1000}})
	if v1.Int != v2.Int {
		t.Errorf("deterministic sequence broken: %d vs %d", v1.Int, v2.Int)
	}
}

func TestSystemIncludeExclude(t *testing.T) {
	vm := ir.NewVM(&ir.Program{})
	Register(vm)
	set := ir.Value{Kind: ir.VKSet}
	res := vm.Builtins["Include"](vm, []ir.Value{set, ir.Value{Kind: ir.VKInt, Int: 5}})
	if res.Set[0]&(1<<5) == 0 {
		t.Error("Include should set bit 5")
	}
	res = vm.Builtins["Exclude"](vm, []ir.Value{res, ir.Value{Kind: ir.VKInt, Int: 5}})
	if res.Set[0]&(1<<5) != 0 {
		t.Error("Exclude should clear bit 5")
	}
}

func TestSystemSetAndGetArguments(t *testing.T) {
	vm := ir.NewVM(&ir.Program{})
	Register(vm)
	SetArguments(vm, []string{"a.exe", "hello", "world"})
	if v := vm.Builtins["ParamCount"](vm, nil); v.Int != 3 {
		t.Errorf("ParamCount = %d", v.Int)
	}
	if v := vm.Builtins["ParamStr"](vm, []ir.Value{ir.Value{Kind: ir.VKInt, Int: 0}}); v.Str != "a.exe" {
		t.Errorf("ParamStr(0) = %q", v.Str)
	}
	if v := vm.Builtins["ParamStr"](vm, []ir.Value{ir.Value{Kind: ir.VKInt, Int: 1}}); v.Str != "hello" {
		t.Errorf("ParamStr(1) = %q", v.Str)
	}
}

func TestSystemFileAssignAndEof(t *testing.T) {
	vm := ir.NewVM(&ir.Program{})
	Register(vm)
	vm.Builtins["Assign"](vm, []ir.Value{ir.Value{Kind: ir.VKInt, Int: 1}, ir.Value{Kind: ir.VKStr, Str: "test.txt"}})
	vm.Builtins["Rewrite"](vm, []ir.Value{ir.Value{Kind: ir.VKInt, Int: 1}})
	if v := vm.Builtins["Eof"](vm, []ir.Value{ir.Value{Kind: ir.VKInt, Int: 1}}); !v.Bool {
		t.Error("Eof should be true on empty file")
	}
}

func TestSystemIOResult(t *testing.T) {
	vm := ir.NewVM(&ir.Program{})
	Register(vm)
	if v := vm.Builtins["IOResult"](vm, nil); v.Int != 0 {
		t.Errorf("initial IOResult = %d", v.Int)
	}
}
