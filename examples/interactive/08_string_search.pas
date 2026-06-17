program StringSearch;
var
  Text, Needle: String;
begin
  Write('Text: ');
  ReadLn(Text);
  Write('Find: ');
  ReadLn(Needle);
  if Text = Needle then
    WriteLn('Exact match')
  else
    WriteLn('Different strings');
end.
