package codegen

import "testing"

func TestSetOperators(t *testing.T) {
	check(t, `program S;
var a, b, c: set of 0..15;
begin
  a := [1, 2, 3, 4];
  b := [3, 4, 5, 6];
  c := a + b;            { union: 1..6 }
  if 5 in c then WriteLn('union ok');
  c := a * b;            { interseccion: 3,4 }
  if (3 in c) and (4 in c) and not (1 in c) then WriteLn('inter ok');
  c := a - b;            { diferencia: 1,2 }
  if (1 in c) and not (3 in c) then WriteLn('diff ok');
  if [1, 2] <= a then WriteLn('subset ok');
  if a >= [2, 3] then WriteLn('superset ok');
  if [1,2,3,4] = a then WriteLn('eq ok');
end.`, "union ok\ninter ok\ndiff ok\nsubset ok\nsuperset ok\neq ok\n")
}
