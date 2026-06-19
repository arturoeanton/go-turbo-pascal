package vmpas

import (
	"strings"
	"testing"
)

func TestRunMutatesGoVariable(t *testing.T) {
	eng := New()
	v1 := 10
	if err := eng.Var("v1", &v1); err != nil {
		t.Fatal(err)
	}
	if err := eng.Run("v1 := v1 + 5"); err != nil {
		t.Fatal(err)
	}
	if v1 != 15 {
		t.Fatalf("v1 = %d", v1)
	}
}

func TestRunCallsFunctionAndProcess(t *testing.T) {
	eng := New()
	v1 := 0
	seen := 0
	f1 := func(x int) int { return x * 3 }
	p1 := func(x int) { seen = x }
	if err := eng.Var("v1", &v1); err != nil {
		t.Fatal(err)
	}
	if err := eng.Function("f1", f1); err != nil {
		t.Fatal(err)
	}
	if err := eng.Process("p1", p1); err != nil {
		t.Fatal(err)
	}
	if err := eng.Run("v1 := f1(7); p1(v1)"); err != nil {
		t.Fatal(err)
	}
	if v1 != 21 || seen != 21 {
		t.Fatalf("v1=%d seen=%d", v1, seen)
	}
}

func TestRunProgramAndOutput(t *testing.T) {
	eng := New()
	if err := eng.Run("program T; begin WriteLn('hello ', 7); end."); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(eng.Output(), "hello 7") {
		t.Fatalf("output = %q", eng.Output())
	}
}

func TestRunLoop(t *testing.T) {
	eng := New()
	sum := 0
	if err := eng.Var("sum", &sum); err != nil {
		t.Fatal(err)
	}
	if err := eng.Run("for i := 1 to 5 do sum := sum + i"); err != nil {
		t.Fatal(err)
	}
	if sum != 15 {
		t.Fatalf("sum = %d", sum)
	}
}

func TestVarBindingString(t *testing.T) {
	e := New()
	s := "abc"
	if err := e.Var("s", &s); err != nil {
		t.Fatal(err)
	}
	if err := e.Run(`s := s + 'def'`); err != nil {
		t.Fatalf("run: %v", err)
	}
	if s != "abcdef" {
		t.Errorf("s = %q", s)
	}
}

type point struct {
	X int
	Y int
}

func TestStructMapping(t *testing.T) {
	e := New()
	p := point{X: 3, Y: 4}
	if err := e.Var("p", &p); err != nil {
		t.Fatal(err)
	}
	if err := e.Run(`p.X := p.X + p.Y; p.Y := 0`); err != nil {
		t.Fatalf("run: %v", err)
	}
	if p.X != 7 || p.Y != 0 {
		t.Errorf("p = %+v, want {7 0}", p)
	}
}

func TestCompileErrorIsCaughtBeforeRun(t *testing.T) {
	e := New()
	// An unknown identifier must fail at compile time (strong typing), not be
	// silently auto-created.
	if err := e.Run(`undefined_variable := 5`); err == nil {
		t.Fatal("expected a compile error for an unknown identifier")
	}
}

func TestSandboxBlocksFileSystemByDefault(t *testing.T) {
	e := New() // Restricted by default
	err := e.Run(`program T; var f: Text; begin Assign(f, 'x.txt'); end.`)
	if err == nil {
		t.Fatal("expected file access to be blocked under the default sandbox")
	}
}

func TestSandboxFullAllowsFileSystem(t *testing.T) {
	e := NewWith(Full())
	if err := e.Run(`program T; var f: Text; begin Assign(f, 'x.txt'); end.`); err != nil {
		t.Fatalf("run under Full caps: %v", err)
	}
}

func TestSliceBinding(t *testing.T) {
	e := New()
	xs := []int{1, 2, 3, 4, 5}
	if err := e.Var("xs", &xs); err != nil {
		t.Fatal(err)
	}
	if err := e.Run(`for i := 0 to 4 do xs[i] := xs[i] * xs[i]`); err != nil {
		t.Fatalf("run: %v", err)
	}
	want := []int{1, 4, 9, 16, 25}
	for i := range want {
		if xs[i] != want[i] {
			t.Fatalf("xs = %v, want %v", xs, want)
		}
	}
}

func TestFunctionReturnsSlice(t *testing.T) {
	e := New()
	e.Function("Evens", func(n int) []int {
		out := make([]int, n)
		for i := range out {
			out[i] = i * 2
		}
		return out
	})
	if err := e.Run(`program P;
var a: array[0..3] of Integer;
begin
  a := Evens(4);
  WriteLn(a[0], ' ', a[3]);
end.`); err != nil {
		t.Fatalf("run: %v", err)
	}
	if got := e.Output(); got != "0 6\n" {
		t.Fatalf("output = %q", got)
	}
}

type rect struct{ W, H int }

func (r rect) Area() int { return r.W * r.H }

func TestBindGoMethodValue(t *testing.T) {
	e := New()
	r := rect{W: 4, H: 5}
	if err := e.Function("Area", r.Area); err != nil { // method value
		t.Fatal(err)
	}
	out := 0
	e.Var("out", &out)
	if err := e.Run(`out := Area()`); err != nil {
		t.Fatalf("run: %v", err)
	}
	if out != 20 {
		t.Fatalf("out = %d, want 20", out)
	}
}

func TestCompileOnceRunMany(t *testing.T) {
	e := New()
	x := 0
	if err := e.Var("x", &x); err != nil {
		t.Fatal(err)
	}
	sc, err := e.Compile(`x := x * x`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	for i := 1; i <= 5; i++ {
		x = i
		if err := sc.Run(); err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
		if x != i*i {
			t.Fatalf("run %d: x=%d, want %d", i, x, i*i)
		}
	}
}

const benchSrc = `begin
  s := 0;
  for i := 1 to 100 do s := s + i
end.`

func BenchmarkRunCompilesEach(b *testing.B) {
	e := New()
	s := 0
	e.Var("s", &s)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = e.Run(benchSrc)
	}
}

func BenchmarkCompileOnceRunMany(b *testing.B) {
	e := New()
	s := 0
	e.Var("s", &s)
	sc, err := e.Compile(benchSrc)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = sc.Run()
	}
}

// recordBenchSrc is record-heavy: it builds and copies records (by-value return)
// and reads their fields in a loop, exercising record allocation and field
// access — the path the slot/assoc-slice record representation optimizes.
const recordBenchSrc = `program R;
type TP = record a, b, c, d: Integer; end;
function mk(n: Integer): TP;
var r: TP;
begin r.a := n; r.b := n * 2; r.c := n * 3; r.d := n * 4; mk := r; end;
var i, s: Integer; p: TP;
begin
  s := 0;
  for i := 1 to 500 do
  begin
    p := mk(i);
    s := s + p.a + p.b + p.c + p.d;
  end;
end.`

// builtinBenchSrc calls a builtin (Write) every iteration — the path the
// OPCallBuiltin resolution cache optimizes.
const builtinBenchSrc = `program B;
var i: Integer;
begin
  for i := 1 to 500 do Write('x');
end.`

func BenchmarkBuiltinCallLoop(b *testing.B) {
	e := NewWith(Capabilities{MaxOutput: 1 << 20})
	sc, err := e.Compile(builtinBenchSrc)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = sc.Run()
	}
}

func BenchmarkRecordHeavy(b *testing.B) {
	e := New()
	sc, err := e.Compile(recordBenchSrc)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = sc.Run()
	}
}
