package codegen

import "testing"

func TestClassCreateMethods(t *testing.T) {
	check(t, `program C;
type
  TCounter = class
    Value: Integer;
    constructor Create(start: Integer);
    procedure Bump;
    function Get: Integer;
  end;
constructor TCounter.Create(start: Integer);
begin
  Value := start;
end;
procedure TCounter.Bump;
begin
  Value := Value + 1;
end;
function TCounter.Get: Integer;
begin
  Get := Value;
end;
var c: TCounter;
begin
  c := TCounter.Create(10);
  c.Bump;
  c.Bump;
  WriteLn(c.Get);
  c.Free;
end.`, "12\n")
}

func TestClassProperty(t *testing.T) {
	check(t, `program C;
type
  TPoint = class
    FX, FY: Integer;
    property X: Integer read FX write FX;
    property Y: Integer read FY write FY;
  end;
var p: TPoint;
begin
  p := TPoint.Create;   { sin constructor: solo asigna memoria }
  p.X := 30;
  p.Y := 12;
  WriteLn(p.X + p.Y);
end.`, "42\n")
}

func TestClassPropertyGetterSetter(t *testing.T) {
	check(t, `program C;
type
  TBox = class
    FValue: Integer;
    function GetValue: Integer;
    procedure SetValue(v: Integer);
    property Value: Integer read GetValue write SetValue;
  end;
function TBox.GetValue: Integer;
begin
  GetValue := FValue * 2;   { el getter transforma el valor }
end;
procedure TBox.SetValue(v: Integer);
begin
  FValue := v + 1;          { el setter transforma el valor }
end;
var b: TBox;
begin
  b := TBox.Create;
  b.Value := 10;            { SetValue(10) -> FValue = 11 }
  WriteLn(b.Value);         { GetValue -> 22 }
end.`, "22\n")
}

func TestClassPropertyGetterInExpr(t *testing.T) {
	check(t, `program C;
type
  TCell = class
    FN: Integer;
    function GetN: Integer;
    property N: Integer read GetN write FN;
  end;
function TCell.GetN: Integer;
begin
  GetN := FN;
end;
var c: TCell;
begin
  c := TCell.Create;
  c.N := 20;                { backing field directo }
  WriteLn(c.N + 22);        { getter usado en una expresión }
end.`, "42\n")
}

func TestClassInheritanceVirtual(t *testing.T) {
	check(t, `program C;
type
  TAnimal = class
    function Speak: Integer; virtual;
  end;
  TDog = class(TAnimal)
    function Speak: Integer; virtual;
  end;
function TAnimal.Speak: Integer; begin Speak := 1; end;
function TDog.Speak: Integer; begin Speak := 2; end;
var a: TAnimal;
begin
  a := TDog.Create;       { referencia base a una instancia derivada }
  WriteLn(a.Speak);       { despacho dinámico -> TDog.Speak }
end.`, "2\n")
}

func TestClassNilByDefault(t *testing.T) {
	check(t, `program C;
type TFoo = class x: Integer; end;
var f: TFoo;
begin
  if f = nil then WriteLn('nil');
  f := TFoo.Create;
  if f <> nil then WriteLn('assigned');
end.`, "nil\nassigned\n")
}
