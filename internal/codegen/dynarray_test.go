package codegen

import "testing"

func TestDynamicArray(t *testing.T) {
	check(t, `program D;
var
  a: array of Integer;
  i, s: Integer;
begin
  SetLength(a, 5);
  WriteLn('len=', Length(a));
  WriteLn('high=', High(a));
  for i := 0 to High(a) do a[i] := (i + 1) * 10;
  s := 0;
  for i := 0 to Length(a) - 1 do s := s + a[i];
  WriteLn('sum=', s);
end.`, "len=5\nhigh=4\nsum=150\n")
}

func TestDynamicArrayGrow(t *testing.T) {
	check(t, `program D;
var
  a: array of Integer;
begin
  SetLength(a, 2);
  a[0] := 1;
  a[1] := 2;
  SetLength(a, 4);     { conserva 1,2; nuevos en 0 }
  WriteLn(a[0], ' ', a[1], ' ', a[2], ' ', a[3]);
  WriteLn('len=', Length(a));
end.`, "1 2 0 0\nlen=4\n")
}
