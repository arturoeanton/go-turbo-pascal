package parser

import (
	"strings"
	"testing"

	"github.com/arturoeanton/go-turbo-pascal/internal/ast"
	"github.com/arturoeanton/go-turbo-pascal/internal/lexer"
)

func parse(t *testing.T, src string) ast.Node {
	t.Helper()
	l := lexer.New(src)
	if len(l.Errors()) > 0 {
		t.Fatalf("lex errors: %v", l.Errors())
	}
	p := New(l.Tokens())
	p.SetFile("test.pas")
	n := p.ParseUnit()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	return n
}

func TestProgramHello(t *testing.T) {
	n := parse(t, "program Hello; begin end.")
	prog, ok := n.(*ast.Program)
	if !ok {
		t.Fatalf("expected Program, got %T", n)
	}
	if prog.Name != "Hello" {
		t.Errorf("expected Hello, got %q", prog.Name)
	}
}

func TestVarAndAssign(t *testing.T) {
	src := `program T;
var
  X: Integer;
begin
  X := 1 + 2;
end.`
	n := parse(t, src)
	prog := n.(*ast.Program)
	if len(prog.Block.Vars) != 1 {
		t.Fatalf("expected 1 var")
	}
	as := prog.Body.Stmts[0].(*ast.AssignStmt)
	if as.Expr.String() != "(1 + 2)" {
		t.Errorf("assign: %v", as.Expr.String())
	}
}

func TestIfElse(t *testing.T) {
	src := `program T; var X: Integer; begin if X > 0 then X := 1 else X := 2; end.`
	n := parse(t, src)
	prog := n.(*ast.Program)
	as := prog.Body.Stmts[0].(*ast.IfStmt)
	if as.Else == nil {
		t.Error("expected else")
	}
}

func TestCase(t *testing.T) {
	src := `program T; var X: Integer; begin case X of 1: X := 1; 2: X := 2 else X := 0 end; end.`
	n := parse(t, src)
	prog := n.(*ast.Program)
	cs := prog.Body.Stmts[0].(*ast.CaseStmt)
	if len(cs.Cases) != 2 {
		t.Errorf("expected 2 cases")
	}
	if cs.Else == nil {
		t.Error("expected else")
	}
}

func TestForTo(t *testing.T) {
	src := `program T; var I, S: Integer; begin S := 0; for I := 1 to 10 do S := S + I; end.`
	n := parse(t, src)
	prog := n.(*ast.Program)
	if _, ok := prog.Body.Stmts[1].(*ast.ForStmt); !ok {
		t.Errorf("expected ForStmt")
	}
}

func TestWhileRepeat(t *testing.T) {
	src := `program T; var X: Integer; begin while X < 10 do X := X + 1; repeat X := X - 1 until X = 0; end.`
	n := parse(t, src)
	prog := n.(*ast.Program)
	if _, ok := prog.Body.Stmts[0].(*ast.WhileStmt); !ok {
		t.Errorf("expected WhileStmt")
	}
	if _, ok := prog.Body.Stmts[1].(*ast.RepeatStmt); !ok {
		t.Errorf("expected RepeatStmt")
	}
}

func TestUnitInterface(t *testing.T) {
	src := `unit Foo;
interface
type
  TFoo = object
    X: Integer;
    procedure DoIt;
  end;
implementation
procedure DoIt; begin end;
end.`
	n := parse(t, src)
	u, ok := n.(*ast.Unit)
	if !ok {
		t.Fatalf("expected Unit, got %T", n)
	}
	if u.Name != "Foo" {
		t.Errorf("expected unit Foo, got %q", u.Name)
	}
	if len(u.Interface.Decls) == 0 {
		t.Error("interface should have decls")
	}
}

func TestProcedureAndFunction(t *testing.T) {
	src := `program T;
function Add(A, B: Integer): Integer; begin Add := A + B end;
procedure P; begin end;
begin end.`
	n := parse(t, src)
	prog := n.(*ast.Program)
	if len(prog.Block.Procs) != 2 {
		t.Fatalf("expected 2 procs, got %d", len(prog.Block.Procs))
	}
	add := prog.Block.Procs[0].(*ast.ProcDecl)
	if !add.IsFunc {
		t.Error("first should be function")
	}
}

func TestNestedProc(t *testing.T) {
	src := `program T;
procedure Outer;
  procedure Inner; begin end;
begin end;
begin end.`
	n := parse(t, src)
	prog := n.(*ast.Program)
	outer := prog.Block.Procs[0].(*ast.ProcDecl)
	if outer.Nested == nil || len(outer.Nested.Procs) != 1 {
		t.Errorf("expected nested inner proc, got %d", len(outer.Nested.Procs))
	}
}

func TestRecords(t *testing.T) {
	src := `program T;
type
  P = record
    X, Y: Integer;
    case Tag: Byte of
      0: (A: Integer);
      1: (B: Integer);
  end;
begin end.`
	n := parse(t, src)
	prog := n.(*ast.Program)
	td := prog.Block.Types[0].(*ast.TypeDecl)
	rt := td.Type.(*ast.RecordType)
	if rt.Variant == nil {
		t.Error("expected variant part")
	}
}

func TestSetAndArrays(t *testing.T) {
	src := `program T;
type
  TSet = set of Char;
  TArr = array[0..9] of Integer;
var
  S: TSet;
  A: TArr;
begin
  S := ['a'..'z'];
  A[0] := 1;
end.`
	prog := parse(t, src).(*ast.Program)
	if prog.Name != "T" {
		t.Errorf("expected program T, got %q", prog.Name)
	}
}

func TestStringConcat(t *testing.T) {
	src := `program T; var S: String; begin S := 'a' + 'b'; end.`
	n := parse(t, src)
	prog := n.(*ast.Program)
	as := prog.Body.Stmts[0].(*ast.AssignStmt)
	be := as.Expr.(*ast.BinaryExpr)
	if be.Op != "+" {
		t.Errorf("expected +, got %s", be.Op)
	}
}

func TestGotoAndLabels(t *testing.T) {
	src := `program T;
label 1;
begin
  goto 1;
  1:
end.`
	n := parse(t, src)
	prog := n.(*ast.Program)
	if len(prog.Block.Labels) != 1 {
		t.Errorf("expected 1 label decl, got %d", len(prog.Block.Labels))
	}
	if _, ok := prog.Body.Stmts[0].(*ast.GotoStmt); !ok {
		t.Errorf("expected goto")
	}
}

func TestHaltAndExit(t *testing.T) {
	src := `program T; begin halt(1); end.`
	n := parse(t, src)
	prog := n.(*ast.Program)
	h, ok := prog.Body.Stmts[0].(*ast.HaltStmt)
	if !ok {
		t.Fatalf("expected HaltStmt, got %T", prog.Body.Stmts[0])
	}
	if h.Code == nil {
		t.Error("expected code")
	}
}

func TestMultipleErrors(t *testing.T) {
	l := lexer.New("program T; var : Integer; begin end.")
	p := New(l.Tokens())
	p.SetFile("x.pas")
	_ = p.ParseUnit()
	if len(p.Errors()) < 1 {
		t.Errorf("expected parse errors, got 0")
	}
}

func TestTypecast(t *testing.T) {
	src := `program T; var X: Integer; P: Pointer; begin X := Integer(P); end.`
	n := parse(t, src)
	prog := n.(*ast.Program)
	as := prog.Body.Stmts[0].(*ast.AssignStmt)
	tc, ok := as.Expr.(*ast.TypeCastExpr)
	if !ok {
		t.Fatalf("expected typecast, got %T", as.Expr)
	}
	if tc.Type.String() != "Integer" {
		t.Errorf("expected Integer cast, got %q", tc.Type.String())
	}
}

func TestRangeType(t *testing.T) {
	src := `program T; type TSub = 1..10; begin end.`
	n := parse(t, src)
	prog := n.(*ast.Program)
	td := prog.Block.Types[0].(*ast.TypeDecl)
	rt, ok := td.Type.(*ast.RangeType)
	if !ok {
		t.Fatalf("expected RangeType, got %T", td.Type)
	}
	if rt.Lo.String() != "1" || rt.Hi.String() != "10" {
		t.Errorf("range: %v..%v", rt.Lo, rt.Hi)
	}
}

func TestPointerType(t *testing.T) {
	src := `program T; type P = ^Integer; begin end.`
	n := parse(t, src)
	prog := n.(*ast.Program)
	td := prog.Block.Types[0].(*ast.TypeDecl)
	pt, ok := td.Type.(*ast.PointerType)
	if !ok || pt.Target.String() != "Integer" {
		t.Errorf("expected ^Integer, got %v", td.Type)
	}
}

func TestNilKeywordExpression(t *testing.T) {
	src := `program T; var P: ^Integer; begin P := nil; end.`
	n := parse(t, src)
	prog := n.(*ast.Program)
	as := prog.Body.Stmts[0].(*ast.AssignStmt)
	id, ok := as.Expr.(*ast.Ident)
	if !ok {
		t.Fatalf("expected nil ident expression, got %T", as.Expr)
	}
	if id.Lower != "nil" {
		t.Errorf("expected nil, got %q", id.Lower)
	}
}

func TestFileAndText(t *testing.T) {
	src := `program T; type TF = file of Integer; begin end.`
	n := parse(t, src)
	prog := n.(*ast.Program)
	td := prog.Block.Types[0].(*ast.TypeDecl)
	ft, ok := td.Type.(*ast.FileType)
	if !ok || ft.Text {
		t.Errorf("expected File of Integer")
	}
}

func TestEnums(t *testing.T) {
	src := `program T; type TColor = (Red, Green, Blue); begin end.`
	n := parse(t, src)
	prog := n.(*ast.Program)
	td := prog.Block.Types[0].(*ast.TypeDecl)
	en, ok := td.Type.(*ast.EnumType)
	if !ok || len(en.Names) != 3 {
		t.Errorf("expected 3 enums")
	}
}

func TestParseErrorRecovery(t *testing.T) {
	l := lexer.New("program T; var : Integer; begin x := 1; end.")
	p := New(l.Tokens())
	_ = p.ParseUnit()
	if !strings.Contains(p.Errors()[0], "line") {
		t.Errorf("errors should have line: %v", p.Errors())
	}
}
