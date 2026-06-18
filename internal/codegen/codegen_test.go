package codegen

import "testing"

// run compiles and executes a Pascal program, returning its output.
func run(t *testing.T, src string) string {
	t.Helper()
	prog, err := Compile(src, "test.pas")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	out, _, err := Run(prog, nil, "")
	if err != nil {
		t.Fatalf("run error: %v (output so far: %q)", err, out)
	}
	return out
}

func check(t *testing.T, src, want string) {
	t.Helper()
	got := run(t, src)
	if got != want {
		t.Errorf("output mismatch\n--- source ---\n%s\n--- want ---\n%q\n--- got ---\n%q", src, want, got)
	}
}

func TestHelloWorld(t *testing.T) {
	check(t, `program H; begin WriteLn('Hello, World!'); end.`, "Hello, World!\n")
}

func TestWriteMultiple(t *testing.T) {
	check(t, `program W;
var i: Integer;
begin
  i := 42;
  WriteLn('i = ', i);
end.`, "i = 42\n")
}

func TestRealDivision(t *testing.T) {
	// In TP7 '/' is always real division; 'div' is integer division.
	check(t, `program D;
begin
  WriteLn((7 / 2):0:2);
  WriteLn(7 div 2);
end.`, "3.50\n3\n")
}

func TestIntegerArithmetic(t *testing.T) {
	check(t, `program A;
var x: Integer;
begin
  x := 2 + 3 * 4;
  WriteLn(x);
  WriteLn(17 div 5);
  WriteLn(17 mod 5);
  WriteLn(10 - 20);
end.`, "14\n3\n2\n-10\n")
}

func TestForSum(t *testing.T) {
	check(t, `program F;
var i, s: Integer;
begin
  s := 0;
  for i := 1 to 10 do
    s := s + i;
  WriteLn(s);
end.`, "55\n")
}

func TestForDownto(t *testing.T) {
	check(t, `program FD;
var i: Integer;
begin
  for i := 3 downto 1 do
    WriteLn(i);
end.`, "3\n2\n1\n")
}

func TestWhile(t *testing.T) {
	check(t, `program WH;
var n: Integer;
begin
  n := 1;
  while n <= 16 do
  begin
    Write(n, ' ');
    n := n * 2;
  end;
  WriteLn;
end.`, "1 2 4 8 16 \n")
}

func TestRepeat(t *testing.T) {
	check(t, `program R;
var n: Integer;
begin
  n := 0;
  repeat
    n := n + 1;
    Write(n);
  until n >= 5;
  WriteLn;
end.`, "12345\n")
}

func TestIfElse(t *testing.T) {
	check(t, `program IfE;
var x: Integer;
begin
  x := 7;
  if x > 5 then WriteLn('big') else WriteLn('small');
  if x < 5 then WriteLn('big') else WriteLn('small');
end.`, "big\nsmall\n")
}

func TestCase(t *testing.T) {
	check(t, `program C;
var x: Integer;
begin
  for x := 1 to 4 do
    case x of
      1: WriteLn('one');
      2, 3: WriteLn('two or three');
    else
      WriteLn('other');
    end;
end.`, "one\ntwo or three\ntwo or three\nother\n")
}

func TestRecursionFactorial(t *testing.T) {
	check(t, `program Fact;
function Factorial(n: Integer): Integer;
begin
  if n <= 1 then
    Factorial := 1
  else
    Factorial := n * Factorial(n - 1);
end;
begin
  WriteLn(Factorial(5));
  WriteLn(Factorial(10));
end.`, "120\n3628800\n")
}

func TestFibonacci(t *testing.T) {
	check(t, `program Fib;
function Fib(n: Integer): Integer;
begin
  if n < 2 then Fib := n
  else Fib := Fib(n-1) + Fib(n-2);
end;
begin
  WriteLn(Fib(10));
end.`, "55\n")
}

func TestValueParam(t *testing.T) {
	check(t, `program VP;
function Add(a, b: Integer): Integer;
begin
  Add := a + b;
end;
begin
  WriteLn(Add(3, 4));
end.`, "7\n")
}

func TestVarParamSwap(t *testing.T) {
	check(t, `program SwapTest;
var a, b: Integer;
procedure Swap(var x, y: Integer);
var t: Integer;
begin
  t := x;
  x := y;
  y := t;
end;
begin
  a := 1; b := 2;
  Swap(a, b);
  WriteLn(a, ' ', b);
end.`, "2 1\n")
}

func TestVarParamAccumulate(t *testing.T) {
	check(t, `program Acc;
var total: Integer;
procedure AddTo(var sum: Integer; n: Integer);
begin
  sum := sum + n;
end;
var i: Integer;
begin
  total := 0;
  for i := 1 to 5 do
    AddTo(total, i);
  WriteLn(total);
end.`, "15\n")
}

func TestProcedureNoParams(t *testing.T) {
	check(t, `program P;
procedure Greet;
begin
  WriteLn('hi');
end;
begin
  Greet;
  Greet;
end.`, "hi\nhi\n")
}

func TestStringConcat(t *testing.T) {
	check(t, `program S;
var a, b: String;
begin
  a := 'Hello, ';
  b := a + 'World!';
  WriteLn(b);
end.`, "Hello, World!\n")
}

func TestBooleanShortCircuit(t *testing.T) {
	// If short-circuit works, Divides(0) is never evaluated (no div by zero).
	check(t, `program B;
var n: Integer;
function Positive(x: Integer): Boolean;
begin
  Positive := x > 0;
end;
begin
  n := 0;
  if (n <> 0) and (100 div n > 1) then
    WriteLn('yes')
  else
    WriteLn('no');
  if (n = 0) or (100 div n > 1) then
    WriteLn('ok');
  WriteLn(Positive(5));
  WriteLn(Positive(-1));
end.`, "no\nok\nTRUE\nFALSE\n")
}

func TestConst(t *testing.T) {
	check(t, `program K;
const
  Max = 100;
  Greeting = 'Pi is';
begin
  WriteLn(Greeting, ' ', Max div 3);
end.`, "Pi is 33\n")
}

func TestIncDec(t *testing.T) {
	check(t, `program ID;
var x: Integer;
begin
  x := 10;
  Inc(x);
  Inc(x, 5);
  Dec(x);
  WriteLn(x);
end.`, "15\n")
}

func TestNestedCalls(t *testing.T) {
	check(t, `program N;
function Twice(x: Integer): Integer;
begin
  Twice := x * 2;
end;
function Inc1(x: Integer): Integer;
begin
  Inc1 := x + 1;
end;
begin
  WriteLn(Twice(Inc1(4)));
end.`, "10\n")
}

func TestCharWrite(t *testing.T) {
	check(t, `program Ch;
var c: Char;
begin
  c := 'A';
  WriteLn(c);
  WriteLn(Chr(66));
  WriteLn(Ord('A'));
end.`, "A\nB\n65\n")
}

func TestBuiltinMath(t *testing.T) {
	check(t, `program M;
begin
  WriteLn(Abs(-7));
  WriteLn(Sqr(6));
  WriteLn(Trunc(Sqrt(16) + 0.5));
end.`, "7\n36\n4\n")
}

func TestRecord(t *testing.T) {
	check(t, `program R;
type TPoint = record x, y: Integer; end;
var p: TPoint;
begin
  p.x := 3;
  p.y := 4;
  WriteLn(p.x + p.y);
end.`, "7\n")
}

func TestWithStatement(t *testing.T) {
	check(t, `program W;
type TPoint = record x, y: Integer; end;
var p: TPoint;
begin
  with p do
  begin
    x := 3;
    y := 4;
  end;
  WriteLn(p.x + p.y);
end.`, "7\n")
}

func TestWithMultiple(t *testing.T) {
	check(t, `program W2;
type
  TA = record a: Integer; end;
  TB = record b: Integer; end;
var ra: TA; rb: TB;
begin
  with ra, rb do
  begin
    a := 10;
    b := 20;
  end;
  WriteLn(ra.a + rb.b);
end.`, "30\n")
}

func TestRecordValueSemantics(t *testing.T) {
	check(t, `program RV;
type TPoint = record x, y: Integer; end;
var p, q: TPoint;
begin
  p.x := 1; p.y := 2;
  q := p;
  q.x := 99;
  WriteLn(p.x, ' ', q.x);
end.`, "1 99\n")
}

func TestArray(t *testing.T) {
	check(t, `program A;
var a: array[1..5] of Integer;
    i, s: Integer;
begin
  for i := 1 to 5 do a[i] := i * i;
  s := 0;
  for i := 1 to 5 do s := s + a[i];
  WriteLn(s);
end.`, "55\n")
}

func TestArrayZeroBased(t *testing.T) {
	check(t, `program A0;
var a: array[0..2] of Integer;
begin
  a[0] := 10;
  a[2] := 20;
  WriteLn(a[0] + a[1] + a[2]);
end.`, "30\n")
}

func TestEnum(t *testing.T) {
	check(t, `program E;
type TColor = (Red, Green, Blue);
var c: TColor;
begin
  c := Green;
  WriteLn(Ord(c));
  if c = Green then WriteLn('green');
end.`, "1\ngreen\n")
}

func TestSetMembership(t *testing.T) {
	check(t, `program S;
var digits: set of 0..9;
    i: Integer;
begin
  digits := [1, 3, 5, 7, 9];
  for i := 0 to 9 do
    if i in digits then Write(i);
  WriteLn;
end.`, "13579\n")
}

func TestPointerLinkedList(t *testing.T) {
	check(t, `program L;
type
  PNode = ^TNode;
  TNode = record
    value: Integer;
    next: PNode;
  end;
var head, p: PNode;
    i: Integer;
begin
  head := nil;
  for i := 3 downto 1 do
  begin
    New(p);
    p^.value := i;
    p^.next := head;
    head := p;
  end;
  p := head;
  while p <> nil do
  begin
    Write(p^.value, ' ');
    p := p^.next;
  end;
  WriteLn;
end.`, "1 2 3 \n")
}

// TestUnsupportedReportsError ensures coverage gaps fail loudly instead of
// silently miscompiling. goto/label is not implemented in this phase.
func TestUnsupportedReportsError(t *testing.T) {
	// An unknown identifier must fail at compile time (strong typing).
	_, err := Compile(`program U;
begin
  nosuchvar := 5;
end.`, "u.pas")
	if err == nil {
		t.Fatal("expected a compile error for an unknown identifier")
	}
}

func TestGoto(t *testing.T) {
	check(t, `program G;
label 99;
var i: Integer;
begin
  i := 0;
  while i < 100 do
  begin
    i := i + 1;
    if i = 5 then goto 99;
  end;
99:
  WriteLn('i = ', i);
end.`, "i = 5\n")
}

func TestParameterlessCallWithoutParens(t *testing.T) {
	// TP7: a parameterless function used in an expression is called.
	check(t, `program P;
function Answer: Integer;
begin
  Answer := 42;
end;
var n: Integer;
begin
  n := Answer;          { no parens }
  WriteLn(Answer);      { no parens, as an argument }
  WriteLn(n);
end.`, "42\n42\n")
}

func TestParameterlessSelfMethodWithoutParens(t *testing.T) {
	check(t, `program P;
type
  TBox = object
    v: Integer;
    function Get: Integer;
    function Twice: Integer;
  end;
function TBox.Get: Integer;
begin
  Get := v;
end;
function TBox.Twice: Integer;
begin
  Twice := Get * 2;     { bare parameterless method on Self }
end;
var b: TBox;
begin
  b.v := 21;
  WriteLn(b.Twice);
end.`, "42\n")
}
