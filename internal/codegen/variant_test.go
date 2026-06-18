package codegen

import "testing"

func TestVariantRecordFieldsAccessible(t *testing.T) {
	// The selector field (Kind) and every variant case field (I, S) are
	// flattened in and individually addressable.
	check(t, `program V;
type
  TValue = record
    case Kind: Integer of
      0: (I: Integer);
      1: (S: string[15]);
  end;
var v: TValue;
begin
  v.Kind := 0;
  v.I := 42;
  WriteLn(v.Kind, ' ', v.I);
  v.Kind := 1;
  v.S := 'hola';
  WriteLn(v.Kind, ' ', v.S);
end.`, "0 42\n1 hola\n")
}

func TestVariantRecordWithFixedAndVariantFields(t *testing.T) {
	check(t, `program V;
type
  TShape = record
    Name: string[10];
    case Kind: Integer of
      0: (Radius: Integer);
      1: (W, H: Integer);
  end;
var s: TShape;
begin
  s.Name := 'rect';
  s.Kind := 1;
  s.W := 3;
  s.H := 4;
  WriteLn(s.Name, ' ', s.W * s.H);
end.`, "rect 12\n")
}
