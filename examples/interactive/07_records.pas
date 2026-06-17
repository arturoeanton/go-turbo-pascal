program RecordInput;
type
  TPoint = record
    X, Y: Integer;
  end;
var
  P: TPoint;
begin
  Write('X: ');
  ReadLn(P.X);
  Write('Y: ');
  ReadLn(P.Y);
  WriteLn('Point entered.');
end.
