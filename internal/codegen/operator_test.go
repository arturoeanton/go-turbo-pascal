package codegen

import "testing"

// Operator overloading: `+` on a record type dispatches to the user operator.
func TestOperatorOverloadAdd(t *testing.T) {
	check(t, `program P;
type
  TVec = record
    x, y: Integer;
  end;
operator + (a, b: TVec): TVec;
begin
  Result.x := a.x + b.x;
  Result.y := a.y + b.y;
end;
var p, q, r: TVec;
begin
  p.x := 1; p.y := 2;
  q.x := 10; q.y := 20;
  r := p + q;
  WriteLn(r.x, ' ', r.y);
end.`, "11 22\n")
}

// Two distinct operators on the same type, and built-in integer + is unaffected.
func TestOperatorOverloadMultiple(t *testing.T) {
	check(t, `program P;
type
  TMoney = record
    cents: Integer;
  end;
operator + (a, b: TMoney): TMoney;
begin
  Result.cents := a.cents + b.cents;
end;
operator - (a, b: TMoney): TMoney;
begin
  Result.cents := a.cents - b.cents;
end;
var a, b, c: TMoney; n: Integer;
begin
  a.cents := 500;
  b.cents := 175;
  c := a - b;
  n := 3 + 4;        { el + entero sigue siendo el built-in }
  WriteLn(c.cents, ' ', n);
end.`, "325 7\n")
}
