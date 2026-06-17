program FileIO;
var
  F: Text;
begin
  Assign(F, 'test.txt');
  Rewrite(F);
  WriteLn(F, 'hello');
  Close(F);
end.
