package codegen

import "testing"

func TestStringCharAccess(t *testing.T) {
	// 1-based read of string characters.
	check(t, `program S;
var s: String;
  i: Integer;
begin
  s := 'ABC';
  for i := 1 to 3 do
    WriteLn(s[i]);
end.`, "A\nB\nC\n")
}

func TestStringCharAssign(t *testing.T) {
	// 1-based write of a string character.
	check(t, `program S;
var s: String;
begin
  s := 'cat';
  s[1] := 'b';
  WriteLn(s);     { bat }
  s[3] := 'd';
  WriteLn(s);     { bad }
end.`, "bat\nbad\n")
}

func TestStringBuiltins(t *testing.T) {
	check(t, `program S;
var s: String;
begin
  s := 'Hello, World';
  WriteLn(Length(s));
  WriteLn(Copy(s, 1, 5));
  WriteLn(Pos('World', s));
  WriteLn(UpCase('a'));
end.`, "12\nHello\n8\nA\n")
}
