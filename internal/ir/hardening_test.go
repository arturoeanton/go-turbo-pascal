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

// T2: money formatting — half-up rounding, carry, and negative values.
func TestFormatCurrency(t *testing.T) {
	cases := []struct {
		scaled int64 // value * CurrencyScale (4 decimals)
		want   string
	}{
		{50000, "5.00"},
		{-50000, "-5.00"},
		{12345, "1.23"},   // .2345 -> 23.45c -> 23 (half-up at the cent)
		{12350, "1.24"},   // .2350 -> 23.50c -> 24 (half-up)
		{-12350, "-1.24"}, // negative rounds by magnitude
		{19999, "2.00"},   // cents carry into units
		{0, "0.00"},
		{-1, "0.00"}, // -0.0001 rounds to 0.00 (no "-0.00")
	}
	for _, c := range cases {
		if got := formatCurrency(c.scaled); got != c.want {
			t.Errorf("formatCurrency(%d) = %q, want %q", c.scaled, got, c.want)
		}
	}
}

// T3: record field insert/overwrite/lookup via the association-slice helpers.
func TestRecordFieldHelpers(t *testing.T) {
	v := Value{Kind: VKRecord}
	c := v.PutField("x", &Value{Kind: VKInt, Int: 7})
	if v.Field("x") != c || v.Field("x").Int != 7 {
		t.Fatal("PutField/Field did not round-trip the cell")
	}
	v.PutField("x", &Value{Kind: VKInt, Int: 9}) // overwrite, must not append
	if v.Field("x").Int != 9 || len(v.Rec) != 1 {
		t.Fatalf("overwrite: Int=%d len=%d", v.Field("x").Int, len(v.Rec))
	}
	v.PutField("y", &Value{Kind: VKInt, Int: 1}) // new field appends
	if len(v.Rec) != 2 {
		t.Fatalf("append: len=%d, want 2", len(v.Rec))
	}
	if v.Field("missing") != nil {
		t.Fatal("Field on an absent name should be nil")
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
