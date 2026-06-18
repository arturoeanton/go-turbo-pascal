package vmpas

import "testing"

const randProg = `program P;
var i, n: Integer;
begin
  Randomize;
  for i := 1 to 5 do begin
    n := Random(1000);
    WriteLn(n);
  end;
end.`

// With Deterministic+Seed, Randomize is reproducible: two independent runs
// produce byte-identical output (vs classic TP7 where Randomize uses entropy).
func TestDeterministicRandomReproducible(t *testing.T) {
	caps := Capabilities{Deterministic: true, Seed: 42}
	out1, err := RunSandboxed(randProg, caps)
	if err != nil {
		t.Fatal(err)
	}
	out2, err := RunSandboxed(randProg, caps)
	if err != nil {
		t.Fatal(err)
	}
	if out1 != out2 {
		t.Fatalf("not reproducible:\n--run1--\n%s\n--run2--\n%s", out1, out2)
	}
	if out1 == "" {
		t.Fatal("expected some output")
	}
}

// A different seed yields a different sequence (the seed actually matters).
func TestDeterministicSeedMatters(t *testing.T) {
	a, _ := RunSandboxed(randProg, Capabilities{Deterministic: true, Seed: 1})
	b, _ := RunSandboxed(randProg, Capabilities{Deterministic: true, Seed: 2})
	if a == b {
		t.Fatalf("different seeds gave identical output: %q", a)
	}
}
