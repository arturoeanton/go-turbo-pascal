program TextMenu;
var
  Choice: Integer;
begin
  WriteLn('1. Compile');
  WriteLn('2. Run');
  WriteLn('3. Debug');
  Write('Choice: ');
  ReadLn(Choice);
  if Choice = 1 then
    WriteLn('Compile selected')
  else
    if Choice = 2 then
      WriteLn('Run selected')
    else
      WriteLn('Debug selected');
end.
