program DebugWalkthrough;
var
  I, Total: Integer;
begin
  Total := 0;
  for I := 1 to 5 do
    Total := Total + I;
  WriteLn('Set a breakpoint on the next line.');
  WriteLn('Total = ', Total);
end.
