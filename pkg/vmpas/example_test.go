package vmpas_test

import (
	"fmt"

	"github.com/arturoeanton/go-turbo-pascal/pkg/vmpas"
)

func Example() {
	eng := vmpas.New()
	v1 := 0
	f1 := func(x int) int { return x + 10 }
	p1 := func(x int) { fmt.Println("p1", x) }

	_ = eng.Var("v1", &v1)
	_ = eng.Function("f1", f1)
	_ = eng.Process("p1", p1)
	_ = eng.Run("v1 := f1(5); p1(v1)")

	fmt.Println(v1)
	// Output:
	// p1 15
	// 15
}

// Bind a Go struct as a Pascal record: exported fields map by name and are
// copied back after the run.
func ExampleEngine_Var_struct() {
	type Point struct{ X, Y int }
	eng := vmpas.New()
	p := Point{X: 3, Y: 4}
	_ = eng.Var("p", &p)
	_ = eng.Run(`p.X := p.X * p.X + p.Y * p.Y`)
	fmt.Println(p.X)
	// Output:
	// 25
}

// Run untrusted code on a fresh, share-nothing engine with conservative limits.
func ExampleRunSandboxed() {
	out, err := vmpas.RunSandboxed(`
program P;
var i, s: Integer;
begin
  s := 0;
  for i := 1 to 10 do s := s + i;
  WriteLn('sum=', s);
end.`, vmpas.Sandboxed())
	fmt.Print(out)
	fmt.Println("err:", err)
	// Output:
	// sum=55
	// err: <nil>
}

// Discover which host capabilities a script needs before granting anything.
func ExampleEngine_Analyze() {
	eng := vmpas.New()
	rep, _ := eng.Analyze(`begin WriteLn(GetEnv('HOME')) end`)
	fmt.Println("needs env:", rep.Needs(vmpas.CapEnv))
	fmt.Println("needs network:", rep.Needs(vmpas.CapNetwork))
	// Output:
	// needs env: true
	// needs network: false
}

// Pause a running program, persist it, and resume later — possibly in another
// process. The captured output is cumulative across segments.
func ExampleEngine_RunDurable() {
	code := `
program P;
begin
  WriteLn('started');
  Suspend('need-approval');
  WriteLn('resumed and finished');
end.`
	// First segment: runs until it suspends.
	eng := vmpas.NewWith(vmpas.Capabilities{Deterministic: true})
	st, _ := eng.RunDurable(code)
	fmt.Println("paused at:", st.Tag)

	// Later: restore from st (e.g. after loading st.Data from storage) and finish.
	eng2 := vmpas.NewWith(vmpas.Capabilities{Deterministic: true})
	_, _ = eng2.ResumeDurable(code, st)
	fmt.Print(eng2.Output())
	// Output:
	// paused at: need-approval
	// started
	// resumed and finished
}
