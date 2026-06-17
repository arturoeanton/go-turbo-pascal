program FactorialDemo;
var
  I, N, F: Integer;
begin
  Write('N: ');
  ReadLn(N);
  F := 1;
  for I := 1 to N do
    F := F * I;
  WriteLn('Factorial = ', F);
end.
