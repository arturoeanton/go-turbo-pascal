program Shapes;
type
  TShape = object
    X, Y: Integer;
    procedure Draw; virtual;
  end;
  TCircle = object(TShape)
    Radius: Integer;
    procedure Draw; virtual;
  end;
procedure TShape.Draw; begin end;
procedure TCircle.Draw; begin end;
var
  C: TCircle;
begin
  C.X := 1;
  C.Y := 2;
  C.Radius := 10;
  C.Draw;
end.
