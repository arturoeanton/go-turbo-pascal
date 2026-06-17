package ast

import "testing"

func TestDumpProgram(t *testing.T) {
	prog := &Program{
		Name: "Hello",
		Block: &Block{
			Vars: []Decl{
				&VarDecl{Names: []string{"X"}, Type: &OrdType{Kind: "Integer"}},
			},
		},
		Body: &BlockBody{Stmts: []Stmt{
			&CallStmt{Call: CallExpr{Func: &Ident{Name: "WriteLn"}}},
		}},
	}
	got := Dump(prog)
	if got == "<nil>" {
		t.Fatal("dump returned nil")
	}
}

func TestIdentLower(t *testing.T) {
	id := &Ident{Name: "MyVar", Lower: "myvar"}
	if id.String() != "MyVar" {
		t.Errorf("ident should keep original spelling: %q", id.String())
	}
}

func TestBinary(t *testing.T) {
	b := &BinaryExpr{Op: "+", Left: &IntLit{Value: 1}, Right: &IntLit{Value: 2}}
	if b.String() != "(1 + 2)" {
		t.Errorf("binary: %q", b.String())
	}
}
