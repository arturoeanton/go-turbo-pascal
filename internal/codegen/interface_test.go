package codegen

import "testing"

// A class implements an interface; an interface-typed variable dispatches
// dynamically to the concrete class method.
func TestInterfaceDynamicDispatch(t *testing.T) {
	check(t, `program P;
type
  IGreeter = interface
    function Greet: Integer;
  end;
  TBase = class
  end;
  TG = class(TBase, IGreeter)
    function Greet: Integer;
  end;
function TG.Greet: Integer;
begin
  Greet := 42;
end;
var g: IGreeter;
begin
  g := TG.Create;
  WriteLn(g.Greet);
end.`, "42\n")
}

// Polymorphism through an interface: two classes implement the same interface
// and the same interface variable dispatches to each concrete type.
func TestInterfacePolymorphism(t *testing.T) {
	check(t, `program P;
type
  IShape = interface
    function Area: Integer;
  end;
  TSquare = class(TBase, IShape)
    Side: Integer;
    function Area: Integer;
  end;
  TBox = class(TBase, IShape)
    W, H: Integer;
    function Area: Integer;
  end;
  TBase = class
  end;
function TSquare.Area: Integer;
begin
  Area := Side * Side;
end;
function TBox.Area: Integer;
begin
  Area := W * H;
end;
var
  shape: IShape;
  sq: TSquare;
  bx: TBox;
begin
  sq := TSquare.Create;
  sq.Side := 4;
  bx := TBox.Create;
  bx.W := 3;
  bx.H := 5;
  shape := sq;
  WriteLn(shape.Area);
  shape := bx;
  WriteLn(shape.Area);
end.`, "16\n15\n")
}
