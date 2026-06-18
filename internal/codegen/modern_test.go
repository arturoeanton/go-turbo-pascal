package codegen

import (
	"strings"
	"testing"
)

// runErr compiles src and returns the error (or nil). Used to assert that
// modern syntax is rejected outside {$MODE BPGO} and that immutability holds.
func runErr(t *testing.T, src string) error {
	t.Helper()
	_, err := Compile(src, "test.pas")
	return err
}

func TestModernTypeInference(t *testing.T) {
	check(t, `{$MODE BPGO}
program P;
var
  n := 21 * 2;
  name := 'bpgo';
  ok := true;
begin
  WriteLn(n, ' ', name, ' ', ok);
end.`, "42 bpgo TRUE\n")
}

func TestModernInferenceInRoutine(t *testing.T) {
	check(t, `{$MODE BPGO}
program P;
function Compute: Integer;
var
  a := 10;
  b := a * 4;
begin
  Compute := a + b;
end;
begin
  WriteLn(Compute());
end.`, "50\n")
}

func TestModernLetBinding(t *testing.T) {
	check(t, `{$MODE BPGO}
program P;
let answer = 42;
let greeting := 'hola';
begin
  WriteLn(greeting, ' ', answer);
end.`, "hola 42\n")
}

func TestModernLetIsImmutable(t *testing.T) {
	err := runErr(t, `{$MODE BPGO}
program P;
let answer = 42;
begin
  answer := 7;
end.`)
	if err == nil || !strings.Contains(err.Error(), "immutable") {
		t.Fatalf("expected immutable-binding error, got %v", err)
	}
}

// Without {$MODE BPGO}, `let` and `:=` inference are not modern syntax: `let`
// stays a plain identifier, so a program may even use it as a variable name.
func TestLetIsPlainIdentifierWithoutMode(t *testing.T) {
	check(t, `program P;
var let: Integer;
begin
  let := 5;
  WriteLn(let);
end.`, "5\n")
}

// Inference is gated: `var x := expr` is only valid under modern mode.
func TestInferenceRejectedWithoutMode(t *testing.T) {
	if err := runErr(t, `program P;
var n := 5;
begin
  WriteLn(n);
end.`); err == nil {
		t.Fatal("expected `var x := expr` to be rejected outside {$MODE BPGO}")
	}
}

func TestRecordHelperMethod(t *testing.T) {
	check(t, `{$MODE BPGO}
program P;
type
  TPoint = record
    x, y: Integer;
  end;
  TPointHelper = record helper for TPoint
    function Sum: Integer;
    procedure Scale(f: Integer);
  end;
function TPointHelper.Sum: Integer;
begin
  Sum := x + y;     { bare field access resolves against TPoint via Self }
end;
procedure TPointHelper.Scale(f: Integer);
begin
  x := x * f;
  y := y * f;
end;
var p: TPoint;
begin
  p.x := 3; p.y := 4;
  p.Scale(2);          { extension method mutates p }
  WriteLn(p.Sum());    { extension method reads p }
end.`, "14\n")
}

func TestClassHelperMethod(t *testing.T) {
	check(t, `{$MODE BPGO}
program P;
type
  TBox = class
    w, h: Integer;
  end;
  TBoxHelper = class helper for TBox
    function Area: Integer;
  end;
function TBoxHelper.Area: Integer;
begin
  Area := w * h;
end;
var b: TBox;
begin
  b := TBox.Create;
  b.w := 5; b.h := 6;
  WriteLn(b.Area());
end.`, "30\n")
}

func TestIntegratedUnitTests(t *testing.T) {
	check(t, `{$MODE BPGO}
program P;
function Add(a, b: Integer): Integer;
begin
  Add := a + b;
end;
test 'suma correcta' begin
  AssertEqual(Add(2, 3), 5);
  AssertTrue(Add(0, 0) = 0);
end;
test 'falla a proposito' begin
  AssertEqual(Add(2, 2), 5);
end;
begin
end.`, "PASS: suma correcta\nFAIL: falla a proposito\n")
}

func TestAssertFalse(t *testing.T) {
	check(t, `{$MODE BPGO}
program P;
test 'assertfalse' begin
  AssertFalse(1 = 2);
end;
begin
end.`, "PASS: assertfalse\n")
}
