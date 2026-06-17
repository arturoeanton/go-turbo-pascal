program NilPointerDemo;
type
  PInt = ^Integer;
var
  P: PInt;
begin
  P := nil;
  if P = nil then
    WriteLn('Pointer is nil');
end.
