package codegen

import "testing"

// A generic container class instantiated with a concrete type argument.
func TestGenericClassContainer(t *testing.T) {
	check(t, `program P;
type
  TBox<T> = class
    value: T;
    procedure Put(x: T);
    function Get: T;
  end;
procedure TBox.Put(x: Integer);
begin
  value := x;
end;
function TBox.Get: Integer;
begin
  Get := value;
end;
var b: TBox<Integer>;
begin
  b := TBox.Create;
  b.Put(42);
  WriteLn(b.Get);
end.`, "42\n")
}

// A generic free function; the call site needs no explicit type argument.
func TestGenericFunction(t *testing.T) {
	check(t, `program P;
function Max<T>(a, b: T): T;
begin
  if a > b then Max := a else Max := b;
end;
begin
  WriteLn(Max(3, 9));
end.`, "9\n")
}

// A generic type using a dynamic array of the type parameter.
func TestGenericDynamicArray(t *testing.T) {
	check(t, `program P;
type
  TStack<T> = class
    data: array of T;
    count: Integer;
    procedure Push(x: T);
    function Pop: T;
  end;
procedure TStack.Push(x: Integer);
begin
  SetLength(data, count + 1);
  data[count] := x;
  count := count + 1;
end;
function TStack.Pop: Integer;
begin
  count := count - 1;
  Pop := data[count];
end;
var s: TStack<Integer>;
begin
  s := TStack.Create;
  s.Push(10);
  s.Push(20);
  WriteLn(s.Pop + s.Pop);
end.`, "30\n")
}
