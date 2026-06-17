package sem

import (
	"testing"

	"github.com/arturoeanton/go-turbo-pascal/internal/ast"
	"github.com/arturoeanton/go-turbo-pascal/internal/lexer"
	"github.com/arturoeanton/go-turbo-pascal/internal/parser"
)

func analyze(t *testing.T, src string) *Analyzer {
	t.Helper()
	l := lexer.New(src)
	p := parser.New(l.Tokens())
	p.SetFile("test.pas")
	n := p.ParseUnit()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	a := New()
	a.Analyze(n)
	return a
}

func TestBasicTypes(t *testing.T) {
	a := analyze(t, "program T; begin end.")
	if _, ok := a.Program().Types["integer"]; !ok {
		t.Error("missing Integer type")
	}
	if _, ok := a.Program().Types["real"]; !ok {
		t.Error("missing Real type")
	}
}

func TestSizeOfBasic(t *testing.T) {
	cases := []struct {
		kind BasicKind
		want int64
	}{
		{BKInteger, 2}, {BKLongInt, 4}, {BKReal, 6}, {BKShortInt, 1},
		{BKByte, 1}, {BKWord, 2}, {BKBoolean, 1}, {BKChar, 1},
		{BKSingle, 4}, {BKDouble, 8}, {BKExtended, 10}, {BKComp, 8},
	}
	for _, c := range cases {
		if c.kind.Size() != c.want {
			t.Errorf("%s: want %d, got %d", c.kind, c.want, c.kind.Size())
		}
	}
}

func TestVarDecls(t *testing.T) {
	src := `program T;
var
  X: Integer;
  Y: Real;
  S: String;
begin
  X := 1;
  Y := 1.5;
  S := 'hi';
end.`
	a := analyze(t, src)
	if len(a.Errors()) > 0 {
		t.Errorf("unexpected errors: %v", a.Errors())
	}
}

func TestUnknownIdentifier(t *testing.T) {
	a := analyze(t, "program T; begin Foo := 1; end.")
	if len(a.Errors()) == 0 {
		t.Error("expected unknown identifier error")
	}
}

func TestTypeMismatch(t *testing.T) {
	src := `program T; var X: Integer; S: String; begin X := S; end.`
	a := analyze(t, src)
	if len(a.Errors()) == 0 {
		t.Error("expected type mismatch error")
	}
}

func TestAssignIntegerToInteger(t *testing.T) {
	src := `program T; var X: Integer; begin X := 5; end.`
	a := analyze(t, src)
	if len(a.Errors()) > 0 {
		t.Errorf("expected no errors: %v", a.Errors())
	}
}

func TestFunctionLookup(t *testing.T) {
	src := `program T;
function Add(A, B: Integer): Integer; begin Add := A + B end;
begin
  Add(1, 2);
end.`
	a := analyze(t, src)
	if len(a.Errors()) > 0 {
		t.Errorf("expected no errors: %v", a.Errors())
	}
}

func TestRecordLayout(t *testing.T) {
	src := `program T;
type
  R = record
    A: Integer;
    B: Byte;
  end;
var
  X: R;
begin
  X.A := 1;
  X.B := 2;
end.`
	a := analyze(t, src)
	if len(a.Errors()) > 0 {
		t.Errorf("expected no errors: %v", a.Errors())
	}
}

func TestObjectWithVMT(t *testing.T) {
	src := `program T;
type
  TFoo = object
    X: Integer;
    procedure DoIt; virtual;
  end;
procedure DoIt; begin end;
var
  F: TFoo;
begin
  F.X := 1;
  F.DoIt;
end.`
	a := analyze(t, src)
	if len(a.Errors()) > 0 {
		t.Errorf("expected no errors: %v", a.Errors())
	}
}

func TestEnum(t *testing.T) {
	src := `program T;
type
  TColor = (Red, Green, Blue);
var
  C: TColor;
begin
  C := Red;
end.`
	a := analyze(t, src)
	if len(a.Errors()) > 0 {
		t.Errorf("expected no errors: %v", a.Errors())
	}
}

func TestRangeType(t *testing.T) {
	src := `program T;
type
  TSub = 1..10;
var
  X: TSub;
begin
  X := 5;
end.`
	a := analyze(t, src)
	if len(a.Errors()) > 0 {
		t.Errorf("expected no errors: %v", a.Errors())
	}
}

func TestNilAssignToPointer(t *testing.T) {
	src := `program T;
type
  PInt = ^Integer;
var
  P: PInt;
begin
  P := nil;
end.`
	a := analyze(t, src)
	if len(a.Errors()) > 0 {
		t.Errorf("expected no errors: %v", a.Errors())
	}
}

func TestForwardProc(t *testing.T) {
	src := `program T;
procedure P; forward;
procedure P; begin end;
begin
  P;
end.`
	a := analyze(t, src)
	if len(a.Errors()) > 0 {
		t.Errorf("expected no errors: %v", a.Errors())
	}
}

func TestExternalProc(t *testing.T) {
	src := `program T;
procedure P; external 'foo';
begin
  P;
end.`
	a := analyze(t, src)
	if len(a.Errors()) > 0 {
		t.Errorf("expected no errors: %v", a.Errors())
	}
}

func TestInheritedCall(t *testing.T) {
	src := `program T;
type
  TBase = object
    procedure Do; virtual;
  end;
  TChild = object(TBase)
    procedure Do; virtual;
  end;
procedure TBase.Do; begin end;
procedure TChild.Do; begin end;
var
  C: TChild;
begin
  C.Do;
end.`
	a := analyze(t, src)
	if len(a.Errors()) > 0 {
		t.Errorf("expected no errors: %v", a.Errors())
	}
	_ = ast.IntLit{}
}
