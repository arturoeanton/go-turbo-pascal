package debug

import "testing"

func TestSetBreakpoint(t *testing.T) {
	d := New()
	d.SetBreakpoint("test.pas", 10)
	if !d.HitAt("test.pas", 10) {
		t.Error("HitAt should be true")
	}
}

func TestRemoveBreakpoint(t *testing.T) {
	d := New()
	d.SetBreakpoint("test.pas", 10)
	d.RemoveBreakpoint("test.pas", 10)
	if d.HitAt("test.pas", 10) {
		t.Error("HitAt should be false")
	}
}

func TestBreakpointHits(t *testing.T) {
	d := New()
	d.SetBreakpoint("test.pas", 10)
	d.SetBreakpoint("test.pas", 10)
	if len(d.Breakpoints) != 1 {
		t.Errorf("Breakpoints: %d", len(d.Breakpoints))
	}
	if d.Breakpoints[0].Hits != 2 {
		t.Errorf("Hits: %d", d.Breakpoints[0].Hits)
	}
}

func TestAddWatch(t *testing.T) {
	d := New()
	d.AddWatch("x", true)
	if len(d.Watches) != 1 {
		t.Errorf("Watches: %d", len(d.Watches))
	}
}

func TestPushPopFrame(t *testing.T) {
	d := New()
	d.PushFrame(Frame{Func: "main", Line: 10})
	d.PushFrame(Frame{Func: "helper", Line: 20})
	if len(d.Stack) != 2 {
		t.Errorf("Stack: %d", len(d.Stack))
	}
	d.PopFrame()
	if len(d.Stack) != 1 {
		t.Errorf("Stack: %d", len(d.Stack))
	}
}

func TestStep(t *testing.T) {
	d := New()
	if d.Step() {
		t.Error("should not halt")
	}
	if d.StepCount != 1 {
		t.Errorf("StepCount: %d", d.StepCount)
	}
}

func TestStepLimit(t *testing.T) {
	d := New()
	d.StepLimit = 2
	d.Step()
	d.Step()
	if !d.Step() {
		t.Error("should halt")
	}
}

func TestHaltContinue(t *testing.T) {
	d := New()
	d.Halt()
	if !d.Halted {
		t.Error("Halted")
	}
	d.Continue()
	if d.Halted {
		t.Error("should not be halted")
	}
}

func TestSnapshot(t *testing.T) {
	d := New()
	d.SetBreakpoint("t.pas", 5)
	s := d.Snapshot()
	if s == "" {
		t.Error("Snapshot empty")
	}
}

func TestHitAtMiss(t *testing.T) {
	d := New()
	if d.HitAt("t.pas", 100) {
		t.Error("HitAt should be false")
	}
}
