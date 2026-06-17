package codegen

import "testing"

func TestObjectMethodsAndFields(t *testing.T) {
	check(t, `program Obj;
type
  TCounter = object
    Value: Integer;
    procedure Reset;
    procedure Bump;
    function Get: Integer;
  end;
procedure TCounter.Reset; begin Value := 0; end;
procedure TCounter.Bump; begin Value := Value + 1; end;
function TCounter.Get: Integer; begin Get := Value; end;
var c: TCounter;
begin
  c.Reset;
  c.Bump; c.Bump; c.Bump;
  WriteLn(c.Get);
end.`, "3\n")
}

func TestObjectInheritanceDispatch(t *testing.T) {
	check(t, `program Poly;
type
  TAnimal = object
    function Sound: Integer;
  end;
  TDog = object(TAnimal)
    function Sound: Integer;
  end;
function TAnimal.Sound: Integer; begin Sound := 1; end;
function TDog.Sound: Integer; begin Sound := 2; end;
var d: TDog;
begin
  WriteLn(d.Sound);
end.`, "2\n")
}

func TestObjectInheritedConstructor(t *testing.T) {
	check(t, `program Ctor;
type
  TBase = object
    X: Integer;
    constructor Init(AX: Integer);
    function GetX: Integer;
  end;
  TDerived = object(TBase)
    Y: Integer;
    constructor Init(AX, AY: Integer);
  end;
constructor TBase.Init(AX: Integer); begin X := AX; end;
function TBase.GetX: Integer; begin GetX := X; end;
constructor TDerived.Init(AX, AY: Integer);
begin
  inherited Init(AX);
  Y := AY;
end;
var d: TDerived;
begin
  d.Init(5, 9);
  WriteLn(d.GetX);
  WriteLn(d.Y);
end.`, "5\n9\n")
}

func TestObjectPolymorphismViaPointer(t *testing.T) {
	check(t, `program Ptr;
type
  PAnimal = ^TAnimal;
  TAnimal = object
    F: Integer;
    function Speak: Integer;
  end;
  TCat = object(TAnimal)
    function Speak: Integer;
  end;
function TAnimal.Speak: Integer; begin Speak := 10; end;
function TCat.Speak: Integer; begin Speak := 20; end;
var
  a: PAnimal;
  c: TCat;
begin
  c.F := 0;
  a := @c;
  WriteLn(a^.Speak);
end.`, "20\n")
}
