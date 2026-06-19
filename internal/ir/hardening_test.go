package ir

import "testing"

// runMain builds a single-function program from the given instructions and runs
// it, returning the VM so the test can inspect RuntimeError. A clean run must
// never panic the host, even on malformed bytecode (e.g. from a corrupt snapshot).
func runMain(code []Instr) *VM {
	fn := NewFunction("main")
	for _, in := range code {
		fn.Emit(in)
	}
	p := &Program{
		Modules: []*Module{{Name: "main", Funcs: map[string]*Function{"main": fn}}},
		Entry:   "main",
	}
	vm := NewVM(p)
	vm.Run()
	return vm
}

// B2: stack-consuming opcodes must report a runtime error on an empty stack
// instead of panicking the host.
func TestStackUnderflowGuards(t *testing.T) {
	cases := map[string][]Instr{
		"OPDup":      {{Op: OPDup}, {Op: OPHalt}},
		"OPInc":      {{Op: OPInc}, {Op: OPHalt}},
		"OPDec":      {{Op: OPDec}, {Op: OPHalt}},
		"OPCaseTest": {{Op: OPCaseTest, S: "1"}, {Op: OPHalt}},
	}
	for name, code := range cases {
		t.Run(name, func(t *testing.T) {
			vm := runMain(code) // must not panic
			if vm.RuntimeError != 204 {
				t.Fatalf("%s on empty stack: want RuntimeError 204, got %d", name, vm.RuntimeError)
			}
		})
	}
}

// B3: dereferencing a pointer whose heap index is out of range must be a clean
// runtime error, not an out-of-range panic.
func TestDerefHeapBounds(t *testing.T) {
	fn := NewFunction("main")
	fn.Emit(Instr{Op: OPLoadGlobal, S: "p"})
	fn.Emit(Instr{Op: OPDeref})
	fn.Emit(Instr{Op: OPHalt})
	p := &Program{
		Modules: []*Module{{Name: "main", Funcs: map[string]*Function{"main": fn}}},
		Entry:   "main",
	}
	vm := NewVM(p)
	vm.Globals["p"] = &Value{Kind: VKPtr, Ptr: 999} // index far past an empty heap
	vm.Run()
	if vm.RuntimeError != 204 {
		t.Fatalf("deref out-of-range pointer: want RuntimeError 204, got %d", vm.RuntimeError)
	}
}
