package codegen

import "testing"

// A named routine assigned to a procedural variable via @ and then called.
func TestProcVariableCallback(t *testing.T) {
	check(t, `program P;
type TIntFn = function(x: Integer): Integer;
function Double(x: Integer): Integer;
begin
  Double := x * 2;
end;
var f: TIntFn;
begin
  f := @Double;
  WriteLn(f(21));
end.`, "42\n")
}

// A procedure value passed as an argument (higher-order procedure).
func TestProcValueAsArgument(t *testing.T) {
	check(t, `program P;
type TIntFn = function(x: Integer): Integer;
function Apply(g: TIntFn; v: Integer): Integer;
begin
  Apply := g(v) + 1;
end;
function Square(x: Integer): Integer;
begin
  Square := x * x;
end;
begin
  WriteLn(Apply(@Square, 6));
end.`, "37\n")
}

// An anonymous function called immediately.
func TestAnonymousFunction(t *testing.T) {
	check(t, `program P;
type TIntFn = function(x: Integer): Integer;
var f: TIntFn;
begin
  f := function(x: Integer): Integer
       begin
         Result := x + 10;
       end;
  WriteLn(f(5));
end.`, "15\n")
}

// A closure capturing an enclosing local by reference: the captured variable is
// read and mutated through the closure, and the change is visible to the caller.
func TestClosureCapturesByReference(t *testing.T) {
	check(t, `program P;
type TProc = procedure;
var
  total: Integer;
  bump: TProc;
begin
  total := 0;
  bump := procedure
          begin
            total := total + 5;
          end;
  bump;
  bump;
  WriteLn(total);
end.`, "10\n")
}

// A closure reading a captured value in a function result.
func TestClosureCapturesValue(t *testing.T) {
	check(t, `program P;
type TIntFn = function: Integer;
var
  base: Integer;
  get: TIntFn;
begin
  base := 100;
  get := function: Integer
         begin
           Result := base + 1;
         end;
  base := 200;
  WriteLn(get());   { () llama al valor procedural; 'get' a secas es el valor }
end.`, "201\n")
}
