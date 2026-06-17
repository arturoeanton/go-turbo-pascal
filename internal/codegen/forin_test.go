package codegen

import "testing"

func TestForInArray(t *testing.T) {
	check(t, `program FI;
var
  a: array of Integer;
  x, s: Integer;
begin
  SetLength(a, 4);
  a[0] := 10; a[1] := 20; a[2] := 30; a[3] := 40;
  s := 0;
  for x in a do s := s + x;
  WriteLn(s);
end.`, "100\n")
}

func TestForInString(t *testing.T) {
	check(t, `program FI;
var
  c: Char;
  s: String;
begin
  s := 'abc';
  for c in s do WriteLn(c);
end.`, "a\nb\nc\n")
}
