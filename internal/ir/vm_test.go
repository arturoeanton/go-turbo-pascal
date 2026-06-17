package ir

import "testing"

func TestVMSimple(t *testing.T) {
	p := &Program{
		Modules: []*Module{{
			Name:    "main",
			Funcs:   map[string]*Function{},
			Globals: []Global{},
			Init:    []string{},
		}},
		Entry: "main",
	}
	fn := NewFunction("main")
	fn.Emit(Instr{Op: OPPushInt, A: 1})
	fn.Emit(Instr{Op: OPPushInt, A: 2})
	fn.Emit(Instr{Op: OPBinary, S: "+"})
	fn.Emit(Instr{Op: OPPop}) // pop result
	fn.Emit(Instr{Op: OPHalt})
	p.Modules[0].Funcs["main"] = fn
	vm := NewVM(p)
	vm.Run()
	if vm.RuntimeError != 0 {
		t.Errorf("unexpected runtime error: %d", vm.RuntimeError)
	}
}

func TestVMBoolShortCircuit(t *testing.T) {
	p := &Program{Entry: "main"}
	p.Modules = []*Module{{Name: "main", Funcs: map[string]*Function{}}}
	fn := NewFunction("main")
	fn.Emit(Instr{Op: OPPushBool, A: 0}) // false
	fn.Emit(Instr{Op: OPAndSC, A: 5})    // if false, jump to PC=5
	fn.Emit(Instr{Op: OPPushBool, A: 1}) // should be skipped
	fn.Emit(Instr{Op: OPPop})
	fn.Emit(Instr{Op: OPHalt}) // PC=5
	p.Modules[0].Funcs["main"] = fn
	vm := NewVM(p)
	vm.Run()
	if vm.RuntimeError != 0 {
		t.Errorf("unexpected runtime error: %d", vm.RuntimeError)
	}
	if len(vm.Stack) != 1 {
		t.Errorf("stack should have 1 item (the AND result), has %d items", len(vm.Stack))
	}
}

func TestVMCall(t *testing.T) {
	p := &Program{Entry: "main"}
	p.Modules = []*Module{{Name: "main", Funcs: map[string]*Function{}}}
	main := NewFunction("main")
	main.Emit(Instr{Op: OPPushInt, A: 7})
	main.Emit(Instr{Op: OPCall, S: "main.outer", A: 1})
	main.Emit(Instr{Op: OPPop})
	main.Emit(Instr{Op: OPHalt})
	outer := NewFunction("outer")
	outer.Params = []string{"x"}
	outer.Locals = []string{}
	outer.Emit(Instr{Op: OPLoadLocal, A: 0}) // load param
	outer.Emit(Instr{Op: OPPushInt, A: 1})
	outer.Emit(Instr{Op: OPBinary, S: "+"})
	outer.Emit(Instr{Op: OPReturn})
	p.Modules[0].Funcs["main"] = main
	p.Modules[0].Funcs["outer"] = outer
	vm := NewVM(p)
	vm.Run()
	if vm.RuntimeError != 0 {
		t.Errorf("unexpected runtime error: %d", vm.RuntimeError)
	}
}

func TestVMStrings(t *testing.T) {
	p := &Program{Entry: "main"}
	p.Modules = []*Module{{Name: "main", Funcs: map[string]*Function{}}}
	main := NewFunction("main")
	main.Emit(Instr{Op: OPPushStr, S: "hi "})
	main.Emit(Instr{Op: OPPushStr, S: "world"})
	main.Emit(Instr{Op: OPBinary, S: "+"})
	main.Emit(Instr{Op: OPPop})
	main.Emit(Instr{Op: OPHalt})
	p.Modules[0].Funcs["main"] = main
	vm := NewVM(p)
	vm.Run()
	if vm.RuntimeError != 0 {
		t.Errorf("unexpected runtime error: %d", vm.RuntimeError)
	}
}

func TestVMSets(t *testing.T) {
	p := &Program{Entry: "main"}
	p.Modules = []*Module{{Name: "main", Funcs: map[string]*Function{}}}
	main := NewFunction("main")
	main.Emit(Instr{Op: OPPushInt, A: 1})
	main.Emit(Instr{Op: OPPushInt, A: 3})
	main.Emit(Instr{Op: OPPushInt, A: 5})
	main.Emit(Instr{Op: OPMkSet, A: 3})
	main.Emit(Instr{Op: OPPushInt, A: 3})
	main.Emit(Instr{Op: OPIn})
	main.Emit(Instr{Op: OPPop})
	main.Emit(Instr{Op: OPHalt})
	p.Modules[0].Funcs["main"] = main
	vm := NewVM(p)
	vm.Run()
	if vm.RuntimeError != 0 {
		t.Errorf("unexpected runtime error: %d", vm.RuntimeError)
	}
}

func TestVMCompare(t *testing.T) {
	p := &Program{Entry: "main"}
	p.Modules = []*Module{{Name: "main", Funcs: map[string]*Function{}}}
	main := NewFunction("main")
	main.Emit(Instr{Op: OPPushInt, A: 5})
	main.Emit(Instr{Op: OPPushInt, A: 3})
	main.Emit(Instr{Op: OPCompare, S: ">"})
	main.Emit(Instr{Op: OPPop})
	main.Emit(Instr{Op: OPHalt})
	p.Modules[0].Funcs["main"] = main
	vm := NewVM(p)
	vm.Run()
	if vm.RuntimeError != 0 {
		t.Errorf("unexpected runtime error: %d", vm.RuntimeError)
	}
}

func TestVMCallBuiltin(t *testing.T) {
	p := &Program{Entry: "main"}
	p.Modules = []*Module{{Name: "main", Funcs: map[string]*Function{}}}
	main := NewFunction("main")
	main.Emit(Instr{Op: OPPushInt, A: 4})
	main.Emit(Instr{Op: OPCallBuiltin, S: "twice", A: 1})
	main.Emit(Instr{Op: OPPushInt, A: 7})
	main.Emit(Instr{Op: OPCallBuiltin, S: "twice", A: 1})
	main.Emit(Instr{Op: OPPop})
	main.Emit(Instr{Op: OPPop})
	main.Emit(Instr{Op: OPHalt})
	p.Modules[0].Funcs["main"] = main
	vm := NewVM(p)
	vm.Builtins["twice"] = func(vm *VM, args []Value) Value {
		return Value{Kind: VKInt, Int: toInt(args[0]) * 2}
	}
	vm.Run()
	if vm.RuntimeError != 0 {
		t.Errorf("unexpected runtime error: %d", vm.RuntimeError)
	}
}
