package codegen

import "testing"

// runIn compiles and runs a program with the given stdin, returning output.
func runIn(t *testing.T, src, input string) string {
	t.Helper()
	prog, err := Compile(src, "test.pas")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	out, _, err := Run(prog, nil, input)
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	return out
}

func TestReadLnInteger(t *testing.T) {
	got := runIn(t, `program R;
var n, sq: Integer;
begin
  ReadLn(n);
  sq := n * n;
  WriteLn('cuadrado: ', sq);
end.`, "9\n")
	if got != "cuadrado: 81\n" {
		t.Errorf("got %q", got)
	}
}

func TestReadLnTwoValues(t *testing.T) {
	got := runIn(t, `program R;
var a, b: Integer;
begin
  ReadLn(a, b);
  WriteLn(a + b);
end.`, "10 32\n")
	if got != "42\n" {
		t.Errorf("got %q", got)
	}
}

func TestReadLnString(t *testing.T) {
	got := runIn(t, `program R;
var s: String;
begin
  ReadLn(s);
  WriteLn('hola ', s);
end.`, "mundo\n")
	if got != "hola mundo\n" {
		t.Errorf("got %q", got)
	}
}

func TestWriteFieldFormatting(t *testing.T) {
	check(t, `program F;
var r: Real;
begin
  WriteLn(5:4);
  WriteLn('x', 42:6);
  r := 3.14159;
  WriteLn(r:8:2);
end.`, "   5\nx    42\n    3.14\n")
}
