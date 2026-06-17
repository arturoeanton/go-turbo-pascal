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
