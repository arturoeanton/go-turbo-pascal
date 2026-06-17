package codegen

import "testing"

func TestModernStringConversions(t *testing.T) {
	check(t, `program S;
var
  s: AnsiString;
  n: Integer;
begin
  s := IntToStr(123);
  WriteLn(s);
  n := StrToInt('  45 ');
  WriteLn(n + 5);
  WriteLn(UpperCase('hola'));
  WriteLn(LowerCase('MUNDO'));
  WriteLn('[' + Trim('  x  ') + ']');
  WriteLn(StringOfChar('-', 5));
end.`, "123\n50\nHOLA\nmundo\n[x]\n-----\n")
}

func TestAnsiStringIsDynamic(t *testing.T) {
	// AnsiString behaves like the dynamic String (concatenation, no 255 cap).
	check(t, `program S;
var s: AnsiString;
    i: Integer;
begin
  s := '';
  for i := 1 to 300 do s := s + 'x';
  WriteLn(Length(s));
end.`, "300\n")
}
