package codegen

import (
	"testing"

	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
)

func TestDebuggerBreakpointStepAndInspect(t *testing.T) {
	src := `program D;
var x, y: Integer;
begin
  x := 1;
  y := 2;
  x := x + y;
  WriteLn(x);
end.`
	prog, err := Compile(src, "d.pas")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	vm := NewVM(prog, nil, "")
	dbg := ir.NewDebugger(vm)
	if !dbg.Start() {
		t.Fatal("start failed")
	}
	dbg.SetBreakpoints([]int{6}) // x := x + y

	stopped, line := dbg.Continue()
	if !stopped || line != 6 {
		t.Fatalf("expected stop at line 6, got stopped=%v line=%d", stopped, line)
	}
	// Line 6 has not executed yet: x=1, y=2.
	g := dbg.Globals()
	if g["x"].Int != 1 || g["y"].Int != 2 {
		t.Fatalf("globals before line 6: x=%d y=%d", g["x"].Int, g["y"].Int)
	}

	// Step over line 6 -> now at line 7, x updated to 3.
	if ok, ln := dbg.StepLine(); !ok || ln != 7 {
		t.Fatalf("expected step to line 7, got ok=%v line=%d", ok, ln)
	}
	if g := dbg.Globals(); g["x"].Int != 3 {
		t.Fatalf("x after line 6 = %d, want 3", g["x"].Int)
	}

	// Continue to completion.
	dbg.Continue()
	if !dbg.Halted() {
		t.Fatal("program should have halted")
	}
	if out := dbg.Output(); out != "3\n" {
		t.Fatalf("output = %q", out)
	}
}
