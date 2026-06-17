program ObjectPoly;
type
  TBase = object
    X: Integer;
    constructor Init(AX: Integer);
    function GetX: Integer; virtual;
  end;
  TDerived = object(TBase)
    Y: Integer;
    constructor Init(AX, AY: Integer);
    function GetX: Integer; virtual;
  end;
constructor TBase.Init(AX: Integer);
begin
  X := AX;
end;
function TBase.GetX: Integer;
begin
  GetX := X;
end;
constructor TDerived.Init(AX, AY: Integer);
begin
  inherited Init(AX);
  Y := AY;
end;
function TDerived.GetX: Integer;
begin
  GetX := inherited GetX + Y;
end;
var
  B: TBase;
  D: TDerived;
begin
  B.Init(1);
  D.Init(2, 3);
end.
