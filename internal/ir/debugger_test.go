package ir

import "testing"

// Drive the line-level debugger over a hand-built program: a breakpoint stops at
// the mapped line, globals are inspectable at the stop, StepLine advances one
// source line, and Continue runs to completion.
func TestDebuggerBreakpointAndStep(t *testing.T) {
	fn := NewFunction("main")
	i1 := fn.Emit(Instr{Op: OPPushInt, A: 10})
	fn.SourceMap[i1] = SourceRef{Line: 1}
	i2 := fn.Emit(Instr{Op: OPStoreGlobal, S: "x"})
	fn.SourceMap[i2] = SourceRef{Line: 2}
	i3 := fn.Emit(Instr{Op: OPPushInt, A: 20})
	fn.SourceMap[i3] = SourceRef{Line: 3}
	i4 := fn.Emit(Instr{Op: OPStoreGlobal, S: "y"})
	fn.SourceMap[i4] = SourceRef{Line: 4}
	fn.Emit(Instr{Op: OPHalt})

	p := &Program{
		Modules: []*Module{{Name: "main", Funcs: map[string]*Function{"main": fn}}},
		Entry:   "main",
	}
	d := NewDebugger(NewVM(p))
	d.SetBreakpoints([]int{3})
	if !d.Start() {
		t.Fatal("Start failed")
	}

	hit, line := d.Continue()
	if !hit || line != 3 {
		t.Fatalf("Continue: hit=%v line=%d, want true/3", hit, line)
	}
	// x was assigned (line 2) before we stopped at line 3.
	if g := d.Globals(); g["x"].Int != 10 {
		t.Fatalf("globals at breakpoint: x=%d, want 10", g["x"].Int)
	}

	ok, next := d.StepLine()
	if !ok || next != 4 {
		t.Fatalf("StepLine: ok=%v line=%d, want true/4", ok, next)
	}

	// Running to the end halts cleanly with both globals set.
	d.Continue()
	if !d.Halted() {
		t.Fatal("program should have halted")
	}
	if g := d.Globals(); g["x"].Int != 10 || g["y"].Int != 20 {
		t.Fatalf("final globals: x=%d y=%d, want 10/20", g["x"].Int, g["y"].Int)
	}
}
