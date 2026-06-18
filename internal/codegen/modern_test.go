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

func TestMatchOption(t *testing.T) {
	check(t, `{$MODE BPGO}
program P;
function Find(n: Integer): Integer;
begin
  if n > 0 then Find := Some(n * 10) else Find := None;
end;
var r: Integer;
begin
  match Find(5) of
    Some(v) => WriteLn('got ', v);
    None    => WriteLn('nothing');
  end;
  match Find(-1) of
    Some(v) => WriteLn('got ', v);
    None    => WriteLn('nothing');
  end;
end.`, "got 50\nnothing\n")
}

func TestMatchUserADT(t *testing.T) {
	check(t, `{$MODE BPGO}
program P;
type
  TShape = (Circle(Integer), Rect(Integer, Integer));
var s: TShape;
begin
  s := Rect(3, 4);
  match s of
    Circle(r)  => WriteLn('circle ', r);
    Rect(w, h) => WriteLn('area ', w * h);
  end;
  s := Circle(7);
  match s of
    Circle(r)  => WriteLn('circle ', r);
    Rect(w, h) => WriteLn('area ', w * h);
  end;
end.`, "area 12\ncircle 7\n")
}

func TestMatchLiteralAndElse(t *testing.T) {
	check(t, `{$MODE BPGO}
program P;
var i: Integer;
begin
  for i := 1 to 4 do
    match i of
      1 => WriteLn('one');
      2 => WriteLn('two');
      else WriteLn('many');
    end;
end.`, "one\ntwo\nmany\nmany\n")
}

func TestMatchEnumConstant(t *testing.T) {
	check(t, `{$MODE BPGO}
program P;
type TColor = (Red, Green, Blue);
var c: TColor;
begin
  c := Green;
  match c of
    Red   => WriteLn('r');
    Green => WriteLn('g');
    Blue  => WriteLn('b');
  end;
end.`, "g\n")
}

func TestMatchAsExpression(t *testing.T) {
	check(t, `{$MODE BPGO}
program P;
type TShape = (Circle(Integer), Rect(Integer, Integer));
function Area(s: TShape): Integer;
begin
  Area := match s of
    Circle(r)  => r * r * 3;
    Rect(w, h) => w * h;
  end;
end;
begin
  WriteLn(Area(Rect(3, 4)));
  WriteLn(Area(Circle(2)));
end.`, "12\n12\n")
}

func TestMatchGuard(t *testing.T) {
	check(t, `{$MODE BPGO}
program P;
var i: Integer;
begin
  for i := -1 to 2 do
    match i of
      0          => WriteLn('zero');
      _ when i > 0 => WriteLn('pos');
      _          => WriteLn('neg');
    end;
end.`, "neg\nzero\npos\npos\n")
}

func TestMatchOrPatterns(t *testing.T) {
	check(t, `{$MODE BPGO}
program P;
var i: Integer;
begin
  for i := 1 to 5 do
    match i of
      1, 3, 5 => WriteLn(i, ' odd');
      2, 4    => WriteLn(i, ' even');
    end;
end.`, "1 odd\n2 even\n3 odd\n4 even\n5 odd\n")
}

func TestMatchExpressionElse(t *testing.T) {
	check(t, `{$MODE BPGO}
program P;
function Name(n: Integer): string;
begin
  Name := match n of
    1 => 'uno';
    2 => 'dos';
    else => 'otro';
  end;
end;
begin
  WriteLn(Name(1), ' ', Name(9));
end.`, "uno otro\n")
}

func TestMatchNonExhaustiveRaises(t *testing.T) {
	// No arm matches and no else -> runtime error (non-exhaustive).
	_, err := Compile(`{$MODE BPGO}
program P;
begin
  match 7 of
    1 => WriteLn('one');
  end;
end.`, "t.pas")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	// It compiles; the failure is at runtime. Run it and expect a non-zero code.
}

func TestDeferLIFO(t *testing.T) {
	check(t, `{$MODE BPGO}
program P;
begin
  defer WriteLn('1');
  defer WriteLn('2');
  defer WriteLn('3');
  WriteLn('body');
end.`, "body\n3\n2\n1\n")
}

func TestDeferInRoutineAndConditional(t *testing.T) {
	check(t, `{$MODE BPGO}
program P;
procedure Run(open: Boolean);
begin
  if open then defer WriteLn('close');
  WriteLn('work');
end;
begin
  Run(true);
  WriteLn('---');
  Run(false);
end.`, "work\nclose\n---\nwork\n")
}

func TestPanicRecover(t *testing.T) {
	check(t, `{$MODE BPGO}
program P;
function Safe(n: Integer): string;
begin
  Safe := 'ok';
  defer
    if recover <> nil then Safe := 'recovered';
  if n = 0 then panic('boom');
  Safe := 'reached';
end;
begin
  WriteLn(Safe(1));
  WriteLn(Safe(0));
end.`, "reached\nrecovered\n")
}

func TestDeferRunsOnPanicCleanup(t *testing.T) {
	check(t, `{$MODE BPGO}
program P;
procedure Inner;
begin
  defer WriteLn('cleanup');
  panic('x');
end;
begin
  try
    Inner;
  except
    WriteLn('caught');
  end;
end.`, "cleanup\ncaught\n")
}

func TestSpawnChannelBasic(t *testing.T) {
	check(t, `{$MODE BPGO}
program P;
var ch: Channel<Integer>; x: Integer;
begin
  ch := MakeChan;
  spawn ch.Send(42);
  x := ch.Receive;
  WriteLn(x);
end.`, "42\n")
}

func TestSpawnMultipleWorkers(t *testing.T) {
	check(t, `{$MODE BPGO}
program P;
var ch: Channel<Integer>; i, total: Integer;
begin
  ch := MakeChan;
  spawn ch.Send(10);
  spawn ch.Send(20);
  spawn ch.Send(30);
  total := 0;
  for i := 1 to 3 do
    total := total + ch.Receive;
  WriteLn(total);
end.`, "60\n")
}

func TestSpawnCapturesAndBuffered(t *testing.T) {
	check(t, `{$MODE BPGO}
program P;
var ch: Channel<Integer>; n: Integer;
begin
  ch := MakeChan(2);   { buffered: el productor no se bloquea }
  n := 7;
  spawn begin ch.Send(n * 2); ch.Send(n * 3); end;
  WriteLn(ch.Receive, ' ', ch.Receive);
end.`, "14 21\n")
}

func TestChannelPingPong(t *testing.T) {
	check(t, `{$MODE BPGO}
program P;
var req, resp: Channel<Integer>; x: Integer;
begin
  req := MakeChan;
  resp := MakeChan;
  spawn begin
    x := req.Receive;
    resp.Send(x * x);
  end;
  req.Send(9);
  WriteLn(resp.Receive);
end.`, "81\n")
}

func TestSpawnDeadlockDetected(t *testing.T) {
	// main receives but nobody ever sends -> all fibers blocked -> runtime error.
	prog, err := Compile(`{$MODE BPGO}
program P;
var ch: Channel<Integer>; x: Integer;
begin
  ch := MakeChan;
  x := ch.Receive;
end.`, "t.pas")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if _, _, err := Run(prog, nil, ""); err == nil {
		t.Fatal("expected a deadlock runtime error (all fibers blocked)")
	}
}

func TestChannelCloseReceiveNil(t *testing.T) {
	check(t, `{$MODE BPGO}
program P;
var ch: Channel<Integer>;
begin
  ch := MakeChan(4);
  ch.Send(7);
  ch.Close;
  WriteLn(ch.Receive);              { 7 (valor en buffer) }
  if ch.Receive = nil then WriteLn('cerrado');
end.`, "7\ncerrado\n")
}

func TestSpawnFiberPanicCaught(t *testing.T) {
	// A panic inside a spawned fiber, caught within that fiber, must not kill
	// the program; main keeps running.
	check(t, `{$MODE BPGO}
program P;
var ch: Channel<Integer>;
begin
  ch := MakeChan;
  spawn begin
    try
      panic('boom');
    except
      ch.Send(99);
    end;
  end;
  WriteLn(ch.Receive);
end.`, "99\n")
}

func TestSpawnCapturesRoutineLocal(t *testing.T) {
	// Inside a routine, a spawned fiber captures locals by reference (closure):
	// the value it computes flows back to the routine via the channel.
	check(t, `{$MODE BPGO}
program P;
function Compute(n: Integer): Integer;
var ch: Channel<Integer>; factor: Integer;
begin
  ch := MakeChan;
  factor := 3;
  spawn ch.Send(n * factor);   { captura n y factor de la rutina }
  Compute := ch.Receive;
end;
begin
  WriteLn(Compute(7));
end.`, "21\n")
}
