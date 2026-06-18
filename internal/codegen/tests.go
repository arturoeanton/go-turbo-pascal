package codegen

import (
	"github.com/arturoeanton/go-turbo-pascal/internal/ast"
	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
)

// compileTestBlock lowers an integrated `test` block: the body runs inside a
// try/except so a failed assertion (which raises) is caught and reported.
// On success it prints "PASS: name", on failure "FAIL: name".
func (g *gen) compileTestBlock(tb *ast.TestBlock) {
	body := append([]ast.Stmt{}, tb.Body.Stmts...)
	body = append(body, writeLnStmt("PASS: "+tb.Name))
	g.compileStmt(&ast.TryStmt{
		Body:   body,
		Except: []ast.Stmt{writeLnStmt("FAIL: " + tb.Name)},
	})
}

// writeLnStmt builds a `WriteLn('s')` statement.
func writeLnStmt(s string) ast.Stmt {
	return &ast.CallStmt{Call: ast.CallExpr{
		Func: &ast.Ident{Name: "WriteLn", Lower: "writeln"},
		Args: []ast.Expr{&ast.StringLit{Value: s}},
	}}
}

// isAssertion reports whether name is a built-in assertion (and not shadowed by
// a user-defined routine of the same name).
func (g *gen) isAssertion(name string) bool {
	switch name {
	case "asserttrue", "assertfalse", "assertequal":
		return g.funcs[name] == nil
	}
	return false
}

// compileAssertion lowers AssertTrue/AssertFalse/AssertEqual. On failure it
// raises a string exception, which a surrounding `test` block (or try/except)
// catches; uncaught, it aborts the program (a failed assertion).
func (g *gen) compileAssertion(name string, args []ast.Expr) {
	switch name {
	case "asserttrue", "assertfalse":
		if len(args) < 1 {
			return
		}
		g.compileExpr(args[0])
		op := ir.OPJumpIfTrue
		if name == "assertfalse" {
			op = ir.OPJumpIfFalse
		}
		skip := g.fn.Emit(ir.Instr{Op: op}) // expectation met -> skip the failure
		g.raiseStr("assertion failed")
		g.fn.Patch(skip, len(g.fn.Code))
	case "assertequal":
		if len(args) < 2 {
			return
		}
		g.compileExpr(args[0])
		g.compileExpr(args[1])
		g.fn.Emit(ir.Instr{Op: ir.OPCompare, S: "="})
		skip := g.fn.Emit(ir.Instr{Op: ir.OPJumpIfTrue})
		g.raiseStr("assertion failed: values not equal")
		g.fn.Patch(skip, len(g.fn.Code))
	}
}

// raiseStr emits a raise of a string exception value.
func (g *gen) raiseStr(msg string) {
	g.fn.Emit(ir.Instr{Op: ir.OPPushStr, S: msg})
	g.fn.Emit(ir.Instr{Op: ir.OPRaise, A: 1})
}
