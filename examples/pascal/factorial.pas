program Factorial;
{ Calcula factoriales de forma recursiva. Ejecutar:
    go run ./cmd/pasrun examples/pascal/factorial.pas }

function Fact(n: Integer): Integer;
begin
  if n <= 1 then
    Fact := 1
  else
    Fact := n * Fact(n - 1);
end;

var
  i: Integer;
begin
  for i := 1 to 7 do
    WriteLn('Fact(', i, ') = ', Fact(i));
end.
