package vmpas

import (
	"errors"
	"strings"
	"testing"
)

// --- #5 struct tags ---

func TestStructTagsFieldNames(t *testing.T) {
	type rec struct {
		A      int `vmpas:"alpha"`
		B      int `json:"beta"`
		C      int // plain field name
		D      int `vmpas:"-"` // skipped
		hidden int //nolint:unused // unexported -> skipped
	}
	r := rec{A: 1, B: 2, C: 3, D: 4}
	e := New()
	if err := e.Var("r", &r); err != nil {
		t.Fatal(err)
	}
	// Access by the Pascal names produced from the tags.
	if err := e.Run(`r.alpha := r.alpha + 10; r.beta := r.beta + 20; r.C := r.C + 30`); err != nil {
		t.Fatalf("run: %v", err)
	}
	if r.A != 11 || r.B != 22 || r.C != 33 {
		t.Fatalf("tag fields: got A=%d B=%d C=%d", r.A, r.B, r.C)
	}
	if r.D != 4 {
		t.Fatalf("skipped field D should be untouched, got %d", r.D)
	}
}

// --- #3a nested struct pointers ---

type xypoint struct{ X, Y int }

func TestPointerParamRoundTrip(t *testing.T) {
	// A bound Go function taking *xypoint exercises irToGo's pointer case
	// (allocate + fill from the Pascal record passed in).
	e := New()
	if err := e.Function("SumSq", func(p *xypoint) int { return p.X*p.X + p.Y*p.Y }); err != nil {
		t.Fatal(err)
	}
	p := xypoint{X: 3, Y: 4}
	var out int
	if err := e.Var("p", &p); err != nil {
		t.Fatal(err)
	}
	if err := e.Var("out", &out); err != nil {
		t.Fatal(err)
	}
	if err := e.Run(`out := SumSq(p)`); err != nil {
		t.Fatalf("run: %v", err)
	}
	if out != 25 {
		t.Fatalf("pointer param: want 25, got %d", out)
	}
}

func TestPointerNilRoundTrip(t *testing.T) {
	// A Go function returning a nil pointer maps to Pascal nil (goToIR), and a
	// non-nil one round-trips with its fields through irToGo's pointer case.
	e := New()
	var gotNil, gotVal bool
	if err := e.Function("MakeNil", func() *xypoint { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := e.Function("MakeVal", func() *xypoint { return &xypoint{X: 7, Y: 8} }); err != nil {
		t.Fatal(err)
	}
	if err := e.Process("Inspect", func(p *xypoint) {
		if p == nil {
			gotNil = true
		} else if p.X == 7 && p.Y == 8 {
			gotVal = true
		}
	}); err != nil {
		t.Fatal(err)
	}
	if err := e.Run(`Inspect(MakeNil); Inspect(MakeVal)`); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !gotNil {
		t.Fatalf("nil Go pointer return should arrive as a nil pointer param")
	}
	if !gotVal {
		t.Fatalf("non-nil pointer should round-trip with its fields")
	}
}

// --- #4 error -> Pascal exception ---

func TestGoErrorRaisesException(t *testing.T) {
	e := New()
	if err := e.Function("MustPos", func(n int) (int, error) {
		if n < 0 {
			return 0, errors.New("negative input")
		}
		return n * 2, nil
	}); err != nil {
		t.Fatal(err)
	}
	var out int
	_ = e.Var("out", &out)

	// Happy path: no error -> value returned.
	if err := e.Run(`out := MustPos(5)`); err != nil {
		t.Fatalf("run ok: %v", err)
	}
	if out != 10 {
		t.Fatalf("want 10, got %d", out)
	}

	// Uncaught error -> Run fails, surfacing the Go error message.
	err := e.Run(`out := MustPos(-1)`)
	if err == nil || !strings.Contains(err.Error(), "negative input") {
		t.Fatalf("want uncaught error with message, got %v", err)
	}
}

func TestGoErrorCaughtByTryExcept(t *testing.T) {
	e := New()
	if err := e.Function("MustPos", func(n int) (int, error) {
		if n < 0 {
			return 0, errors.New("boom")
		}
		return n, nil
	}); err != nil {
		t.Fatal(err)
	}
	var out int
	_ = e.Var("out", &out)
	if err := e.Run(`out := 0; try out := MustPos(-1); except out := 99; end`); err != nil {
		t.Fatalf("try/except should catch the host error, got %v", err)
	}
	if out != 99 {
		t.Fatalf("want 99 (except branch), got %d", out)
	}
}

func TestErrorOnlyProcedure(t *testing.T) {
	// A Go function whose only result is error behaves as a procedure that may raise.
	e := New()
	calls := 0
	if err := e.Process("Step", func() error {
		calls++
		if calls >= 2 {
			return errors.New("stop")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	err := e.Run(`Step; Step; Step`)
	if err == nil || !strings.Contains(err.Error(), "stop") {
		t.Fatalf("want error 'stop', got %v", err)
	}
	if calls != 2 {
		t.Fatalf("want 2 calls before raise, got %d", calls)
	}
}

// --- typed runtime errors (A3) ---

func TestRuntimeErrorTyped(t *testing.T) {
	// A sandbox-limit breach surfaces as a *RuntimeError with the TP7 code and a
	// human-readable message, inspectable with errors.As.
	e := NewWith(Capabilities{MaxSteps: 1000})
	err := e.Run(`program P; var i: Integer; begin i := 0; while true do i := i + 1; end.`)
	var re *RuntimeError
	if !errors.As(err, &re) {
		t.Fatalf("want *RuntimeError, got %T: %v", err, err)
	}
	if re.Code != 200 {
		t.Fatalf("want code 200 (step limit), got %d", re.Code)
	}
	if !strings.Contains(re.Error(), "limit") {
		t.Fatalf("want a descriptive message, got %q", re.Error())
	}
}

// --- output cap on a single oversized write (B1) ---

func TestMaxOutputSingleWrite(t *testing.T) {
	// Build a large string in memory, then emit it in ONE Write. The byte cap is
	// enforced at write time, so even a single oversized write is bounded — not
	// only a loop of small writes.
	e := NewWith(Capabilities{MaxOutput: 64})
	err := e.Run(`program P;
var i: Integer; s: string;
begin
  s := 'x';
  for i := 1 to 8 do s := s + s;   { saturates ShortString well past 64 }
  Write(s);
end.`)
	if err == nil {
		t.Fatal("expected an output-limit error on a single oversized write")
	}
	if got := len(e.Output()); got > 64 {
		t.Fatalf("output not bounded by MaxOutput: %d bytes (want <= 64)", got)
	}
}

// --- #1 live bindings ---

func TestLiveBindingsHostMutationVisible(t *testing.T) {
	counter := 10
	e := NewWith(Capabilities{LiveBindings: true})
	if err := e.Var("counter", &counter); err != nil {
		t.Fatal(err)
	}
	if err := e.Process("Bump", func() { counter++ }); err != nil {
		t.Fatal(err)
	}
	var seen int
	_ = e.Var("seen", &seen)
	// Bump mutates the Go var; with live bindings Pascal sees it immediately.
	if err := e.Run(`Bump; seen := counter; Bump`); err != nil {
		t.Fatalf("run: %v", err)
	}
	if seen != 11 {
		t.Fatalf("live: Pascal should see host mutation (11), got %d", seen)
	}
	if counter != 12 {
		t.Fatalf("live: Go var should end at 12, got %d", counter)
	}
}

func TestLiveBindingsOffLosesHostMutation(t *testing.T) {
	// Without LiveBindings the host callback's writes to a bound var are
	// overwritten by the end-of-run readback (documented copy-out behavior).
	counter := 10
	e := New() // LiveBindings off
	if err := e.Var("counter", &counter); err != nil {
		t.Fatal(err)
	}
	if err := e.Process("Bump", func() { counter++ }); err != nil {
		t.Fatal(err)
	}
	if err := e.Run(`Bump; Bump`); err != nil {
		t.Fatalf("run: %v", err)
	}
	if counter != 10 {
		t.Fatalf("non-live: end readback should restore 10, got %d", counter)
	}
}
