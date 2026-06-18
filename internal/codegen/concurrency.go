package codegen

import (
	"github.com/arturoeanton/go-turbo-pascal/internal/ast"
	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
)

// compileSpawn lowers `spawn Stmt`: the statement is wrapped in an anonymous
// procedure (a closure capturing the enclosing scope by reference) and started
// as a new cooperative fiber.
func (g *gen) compileSpawn(s *ast.SpawnStmt) {
	g.concurrent = true
	af := &ast.AnonFunc{
		Base: ast.Base{P: s.P},
		Body: &ast.BlockBody{Base: ast.Base{P: s.P}, Stmts: []ast.Stmt{s.Stmt}},
	}
	g.compileAnonFunc(af) // pushes a VKFunc closure
	g.fn.Emit(ir.Instr{Op: ir.OPSpawn})
}

// compileChanMethod lowers a channel operation `ch.Send(v)` / `ch.Receive` /
// `ch.Close`. It returns true if it handled the call. asStmt drops a produced
// value (Receive) when used as a statement.
func (g *gen) compileChanMethod(recv ast.Expr, method string, args []ast.Expr, asStmt bool) bool {
	bt := g.typeOf(recv)
	if bt == nil || bt.kind != ktChan {
		return false
	}
	g.concurrent = true
	switch lower(method) {
	case "send":
		g.compileExpr(recv)
		if len(args) > 0 {
			g.compileExpr(args[0])
		} else {
			g.fn.Emit(ir.Instr{Op: ir.OPPushNil})
		}
		g.fn.Emit(ir.Instr{Op: ir.OPChanSend})
		return true
	case "receive":
		g.compileExpr(recv)
		g.fn.Emit(ir.Instr{Op: ir.OPChanRecv})
		if asStmt {
			g.fn.Emit(ir.Instr{Op: ir.OPPop})
		}
		return true
	case "close":
		g.compileExpr(recv)
		g.fn.Emit(ir.Instr{Op: ir.OPChanClose})
		return true
	}
	return false
}

// compileMakeChan lowers MakeChan / MakeChan(n) into a new channel value.
func (g *gen) compileMakeChan(args []ast.Expr) {
	g.concurrent = true
	if len(args) > 0 {
		g.compileExpr(args[0])
		g.fn.Emit(ir.Instr{Op: ir.OPMakeChan, A: 1})
		return
	}
	g.fn.Emit(ir.Instr{Op: ir.OPMakeChan, A: 0})
}
