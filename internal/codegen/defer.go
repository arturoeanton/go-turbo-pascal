package codegen

import (
	"github.com/arturoeanton/go-turbo-pascal/internal/ast"
	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
)

// compileBodyWithDefers compiles a routine/program body, handling `defer`. If
// the body contains defers, it is wrapped in a try/finally whose finally runs
// each reached defer in reverse order — so deferred cleanup runs on every exit
// path, normal or via a panic. Each defer gets a boolean "reached" flag set
// when control passes the `defer` statement; only reached defers run.
func (g *gen) compileBodyWithDefers(stmts []ast.Stmt) {
	defers := collectDefers(stmts)
	if len(defers) == 0 {
		for _, s := range stmts {
			g.compileStmt(s)
		}
		return
	}
	var finallyStmts []ast.Stmt
	for i := len(defers) - 1; i >= 0; i-- { // reverse: LIFO
		flag := g.declareBinding(g.tmpName("defer"), &typeInfo{kind: ktScalar, scalar: tBool})
		g.deferFlags[defers[i]] = flag.bindName
		// reset the flag to false up front (globals/locals start zero, but be explicit)
		g.assignToVar(flag.bindName, func() { g.fn.Emit(ir.Instr{Op: ir.OPPushBool, A: 0}) }, defers[i])
		finallyStmts = append(finallyStmts, &ast.IfStmt{
			Cond: identFor(flag.bindName),
			Then: defers[i].Stmt,
		})
	}
	g.compileTry(&ast.TryStmt{Body: stmts, Finally: finallyStmts})
}

// collectDefers returns every `defer` statement in stmts, recursing into the
// standard statement containers, in source order.
func collectDefers(stmts []ast.Stmt) []*ast.DeferStmt {
	var out []*ast.DeferStmt
	var walk func(s ast.Stmt)
	walkList := func(ss []ast.Stmt) {
		for _, s := range ss {
			walk(s)
		}
	}
	walk = func(s ast.Stmt) {
		switch v := s.(type) {
		case *ast.DeferStmt:
			out = append(out, v)
		case *ast.CompoundStmt:
			walkList(v.Stmts)
		case *ast.IfStmt:
			walk(v.Then)
			if v.Else != nil {
				walk(v.Else)
			}
		case *ast.WhileStmt:
			walk(v.Body)
		case *ast.ForStmt:
			walk(v.Body)
		case *ast.RepeatStmt:
			walk(v.Body)
		case *ast.WithStmt:
			walk(v.Body)
		case *ast.CaseStmt:
			for _, b := range v.Cases {
				if b.Body != nil {
					walk(b.Body)
				}
			}
			if v.Else != nil {
				walk(v.Else)
			}
		case *ast.TryStmt:
			walkList(v.Body)
			walkList(v.Except)
			walkList(v.Finally)
		case *ast.MatchStmt:
			for _, a := range v.Arms {
				if a.Body != nil {
					walk(a.Body)
				}
			}
			walkList(v.Else)
		}
	}
	walkList(stmts)
	return out
}

// compileDefer marks a defer statement's flag as reached; the statement itself
// runs later, in the synthesized finally.
func (g *gen) compileDefer(d *ast.DeferStmt) {
	flag, ok := g.deferFlags[d]
	if !ok {
		g.errf("defer is only allowed inside a routine or program body")
		return
	}
	g.assignToVar(flag, func() { g.fn.Emit(ir.Instr{Op: ir.OPPushBool, A: 1}) }, d)
}
