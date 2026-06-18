package codegen

import (
	"errors"
	"strings"
	"testing"
)

func compileErr(t *testing.T, src string) *CompileError {
	t.Helper()
	_, err := Compile(src, "t.pas")
	if err == nil {
		t.Fatal("expected a compile error")
	}
	var ce *CompileError
	if !errors.As(err, &ce) {
		t.Fatalf("expected *CompileError, got %T: %v", err, err)
	}
	return ce
}

func TestSemArgCountMismatch(t *testing.T) {
	ce := compileErr(t, `program P;
function Add(a, b: Integer): Integer;
begin Add := a + b; end;
var x: Integer;
begin
  x := Add(1);        { faltan argumentos }
end.`)
	if !strings.Contains(ce.Error(), "argument") {
		t.Fatalf("expected an argument-count error, got: %s", ce.Error())
	}
}

func TestSemTooManyArgs(t *testing.T) {
	ce := compileErr(t, `program P;
procedure Greet(name: string);
begin WriteLn(name); end;
begin
  Greet('a', 'b');    { sobran argumentos }
end.`)
	if !strings.Contains(ce.Error(), "argument") {
		t.Fatalf("expected an argument-count error, got: %s", ce.Error())
	}
}

func TestSemUnknownType(t *testing.T) {
	ce := compileErr(t, `program P;
var x: TNoExiste;
begin
end.`)
	if !strings.Contains(ce.Error(), "unknown type") {
		t.Fatalf("expected an unknown-type error, got: %s", ce.Error())
	}
}

// Guard against false positives: a valid program with common numeric aliases
// must compile cleanly.
func TestSemNumericAliasesOK(t *testing.T) {
	check(t, `program P;
var a: Cardinal; b: Int64; c: SmallInt; d: Byte;
begin
  a := 1; b := 2; c := 3; d := 4;
  WriteLn(a + b + c + d);
end.`, "10\n")
}
