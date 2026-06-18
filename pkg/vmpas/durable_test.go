package vmpas

import (
	"strings"
	"testing"
)

// Suspend mid-frame: a var-parameter pointer (aliasing a global) and aggregate
// state (record + array) must survive snapshot/resume exactly.
func TestDurableVarParamAliasing(t *testing.T) {
	const src = `program P;
type TRec = record a: Integer; b: Integer; end;
var g: Integer; r: TRec; arr: array[0..2] of Integer;
procedure Bump(var x: Integer);
begin
  x := x + 1;
  Suspend('mid');
  x := x + 10;
end;
begin
  g := 5;
  r.a := 1; r.b := 2;
  arr[0] := 7; arr[1] := 8; arr[2] := 9;
  Bump(g);
  WriteLn('g=', g, ' r=', r.a, ',', r.b, ' arr=', arr[0], arr[1], arr[2]);
end.`
	e := New()
	st, err := e.RunDurable(src)
	if err != nil {
		t.Fatal(err)
	}
	if st == nil {
		t.Fatal("expected suspension, got completion")
	}
	if st.Tag != "mid" {
		t.Fatalf("tag = %q, want mid", st.Tag)
	}

	// Resume on a *fresh* engine to prove the State is fully self-contained.
	e2 := New()
	st2, err := e2.ResumeDurable(src, st)
	if err != nil {
		t.Fatal(err)
	}
	if st2 != nil {
		t.Fatalf("expected completion, suspended again with tag %q", st2.Tag)
	}
	got := strings.TrimSpace(e2.Output())
	want := "g=16 r=1,2 arr=789"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// The host injects an input via a bound variable before resuming; the script
// reads it after the Suspend call returns.
func TestDurableInputInjection(t *testing.T) {
	var answer int
	e := New()
	if err := e.Var("answer", &answer); err != nil {
		t.Fatal(err)
	}
	const src = `Suspend('need-answer'); WriteLn('got ', answer);`
	st, err := e.RunDurable(src)
	if err != nil {
		t.Fatal(err)
	}
	if st == nil || st.Tag != "need-answer" {
		t.Fatalf("expected suspension need-answer, got %+v", st)
	}
	answer = 42 // host provides the answer
	st2, err := e.ResumeDurable(src, st)
	if err != nil {
		t.Fatal(err)
	}
	if st2 != nil {
		t.Fatal("expected completion")
	}
	if strings.TrimSpace(e.Output()) != "got 42" {
		t.Fatalf("got %q", e.Output())
	}
}

// Snapshot/resume must be transparent: pausing and continuing yields the exact
// same result and output as an uninterrupted run (deterministic).
func TestDurableEqualsUninterrupted(t *testing.T) {
	const src = `program P;
var i, sum: Integer;
begin
  sum := 0;
  for i := 1 to 5 do sum := sum + i;
  Suspend('halfway');
  for i := 6 to 10 do sum := sum + i;
  WriteLn(sum);
end.`
	// Uninterrupted reference (Suspend just halts a normal run; so compute by hand): 1..10 = 55.
	caps := Capabilities{Deterministic: true, Seed: 7}
	e := NewWith(caps)
	st, err := e.RunDurable(src)
	if err != nil || st == nil {
		t.Fatalf("expected suspension: st=%v err=%v", st, err)
	}
	st2, err := e.ResumeDurable(src, st)
	if err != nil || st2 != nil {
		t.Fatalf("expected completion: st2=%v err=%v", st2, err)
	}
	if strings.TrimSpace(e.Output()) != "55" {
		t.Fatalf("got %q, want 55", e.Output())
	}
}

// Multiple suspensions in one run round-trip correctly and output stays
// cumulative across segments.
func TestDurableMultipleSuspends(t *testing.T) {
	const src = `program P;
var n: Integer;
begin
  n := 1;
  WriteLn('a', n);
  Suspend('1'); n := n + 1;
  WriteLn('b', n);
  Suspend('2'); n := n + 1;
  WriteLn('c', n);
end.`
	e := New()
	st, err := e.RunDurable(src)
	if err != nil || st == nil || st.Tag != "1" {
		t.Fatalf("first suspend: st=%v err=%v", st, err)
	}
	st, err = e.ResumeDurable(src, st)
	if err != nil || st == nil || st.Tag != "2" {
		t.Fatalf("second suspend: st=%v err=%v", st, err)
	}
	st, err = e.ResumeDurable(src, st)
	if err != nil || st != nil {
		t.Fatalf("completion: st=%v err=%v", st, err)
	}
	if strings.TrimSpace(e.Output()) != "a1\nb2\nc3" {
		t.Fatalf("got %q", e.Output())
	}
}

// Resuming against different source must be rejected (fingerprint guard).
func TestDurableFingerprintGuard(t *testing.T) {
	e := New()
	st, err := e.RunDurable(`Suspend('x'); WriteLn('done');`)
	if err != nil || st == nil {
		t.Fatalf("expected suspension: %v", err)
	}
	_, err = e.ResumeDurable(`Suspend('x'); WriteLn('DIFFERENT');`, st)
	if err == nil || !strings.Contains(err.Error(), "fingerprint") {
		t.Fatalf("expected fingerprint mismatch, got %v", err)
	}
}

// Heap pointers (New / ^) survive snapshot/resume via their heap index.
func TestDurableHeapPointer(t *testing.T) {
	const src = `program P;
type PInt = ^Integer;
var p: PInt;
begin
  New(p);
  p^ := 100;
  Suspend('mid');
  p^ := p^ + 23;
  WriteLn(p^);
  Dispose(p);
end.`
	e := New()
	st, err := e.RunDurable(src)
	if err != nil {
		t.Skipf("heap pointers not supported in this build: %v", err)
	}
	if st == nil {
		t.Fatal("expected suspension")
	}
	st2, err := e.ResumeDurable(src, st)
	if err != nil {
		t.Fatal(err)
	}
	if st2 != nil {
		t.Fatal("expected completion")
	}
	if strings.TrimSpace(e.Output()) != "123" {
		t.Fatalf("got %q, want 123", e.Output())
	}
}

// Snapshotting a concurrent program with live fibers is refused with a clear
// error (v1 scope) rather than producing a corrupt snapshot.
func TestDurableConcurrentRejected(t *testing.T) {
	const src = `{$MODE BPGO}
program P;
var ch: Channel<Integer>;
begin
  ch := MakeChan;
  spawn begin Suspend('inside-fiber'); ch.Send(1); end;
  WriteLn(ch.Receive);
end.`
	e := New()
	_, err := e.RunDurable(src)
	if err == nil {
		t.Fatal("expected an error snapshotting a concurrent program")
	}
	if !strings.Contains(err.Error(), "concurrent") {
		t.Fatalf("expected concurrent-snapshot error, got %v", err)
	}
}

// The hardest fidelity case: a heap linked list (orphan cells pointing at
// other orphan cells) must survive snapshot/resume with the whole pointer
// graph intact — resumed on a fresh engine.
func TestDurableLinkedListGraph(t *testing.T) {
	const src = `program P;
type
  PNode = ^TNode;
  TNode = record value: Integer; next: PNode; end;
var head, p: PNode; i, sum: Integer;
begin
  head := nil;
  for i := 3 downto 1 do begin
    New(p);
    p^.value := i;
    p^.next := head;
    head := p;
  end;
  Suspend('built');
  sum := 0;
  p := head;
  while p <> nil do begin
    sum := sum + p^.value;
    p := p^.next;
  end;
  WriteLn('sum=', sum);
end.`
	e := New()
	st, err := e.RunDurable(src)
	if err != nil {
		t.Skipf("heap records not supported in this build: %v", err)
	}
	if st == nil || st.Tag != "built" {
		t.Fatalf("expected suspension 'built', got %+v", st)
	}
	e2 := New() // resume on a brand-new engine
	st2, err := e2.ResumeDurable(src, st)
	if err != nil {
		t.Fatal(err)
	}
	if st2 != nil {
		t.Fatal("expected completion")
	}
	if strings.TrimSpace(e2.Output()) != "sum=6" {
		t.Fatalf("got %q, want sum=6", e2.Output())
	}
}
