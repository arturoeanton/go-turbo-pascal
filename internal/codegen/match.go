package codegen

import (
	"fmt"
	"strings"

	"github.com/arturoeanton/go-turbo-pascal/internal/ast"
	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
)

func lower(s string) string { return strings.ToLower(s) }

// compileMatch lowers `match Expr of Pattern => Body; ... [else ...] end`.
// The scrutinee is evaluated once into a temp; each arm tests its pattern and,
// on a match, binds any payloads and runs its body, then jumps past the rest.
func (g *gen) compileMatch(m *ast.MatchStmt) {
	scrut := g.declareBinding(g.tmpName("match"), g.inferType(m.Expr))
	scrutID := identFor(scrut.bindName)
	g.assignToVar(scrut.bindName, func() { g.compileExpr(m.Expr) }, m)

	var endJumps []int
	for _, arm := range m.Arms {
		if arm.Wildcard {
			g.compileStmt(arm.Body)
			endJumps = append(endJumps, g.fn.Emit(ir.Instr{Op: ir.OPJump}))
			break // a wildcard is terminal
		}
		var miss int // jump taken when this arm does not match
		switch {
		case arm.Ctor != "" && g.isCtor(arm.Ctor):
			// Constructor pattern: compare the value's tag, then bind payloads.
			g.compileExpr(fieldOf(scrutID, "__tag"))
			g.fn.Emit(ir.Instr{Op: ir.OPPushStr, S: lower(arm.Ctor)})
			g.fn.Emit(ir.Instr{Op: ir.OPCompare, S: "="})
			miss = g.fn.Emit(ir.Instr{Op: ir.OPJumpIfFalse})
			for i, b := range arm.Binds {
				g.declareBinding(b, &typeInfo{kind: ktScalar, scalar: tUnknown})
				field := fieldOf(scrutID, fmt.Sprintf("__%d", i))
				g.assignToVar(b, func() { g.compileExpr(field) }, arm.Body)
			}
		case arm.Ctor != "":
			// Ident pattern that is not a constructor: a constant/enum value.
			g.compileExpr(scrutID)
			g.compileExpr(identFor(arm.Ctor))
			g.fn.Emit(ir.Instr{Op: ir.OPCompare, S: "="})
			miss = g.fn.Emit(ir.Instr{Op: ir.OPJumpIfFalse})
		default:
			// Literal pattern: compare the value to the literal.
			g.compileExpr(scrutID)
			g.compileExpr(arm.Lit)
			g.fn.Emit(ir.Instr{Op: ir.OPCompare, S: "="})
			miss = g.fn.Emit(ir.Instr{Op: ir.OPJumpIfFalse})
		}
		g.compileStmt(arm.Body)
		endJumps = append(endJumps, g.fn.Emit(ir.Instr{Op: ir.OPJump}))
		g.fn.Patch(miss, len(g.fn.Code))
	}
	for _, s := range m.Else {
		g.compileStmt(s)
	}
	for _, j := range endJumps {
		g.fn.Patch(j, len(g.fn.Code))
	}
}

// bindName is the display name a binding's vinfo is registered under (set by
// declareBinding). It lets synthesized code reference the binding by name.
func identFor(name string) *ast.Ident {
	return &ast.Ident{Name: name, Lower: lower(name)}
}

func fieldOf(recv ast.Expr, field string) ast.Expr {
	return &ast.FieldExpr{Expr: recv, Field: field}
}

// isCtor reports whether name is a registered ADT constructor.
func (g *gen) isCtor(name string) bool {
	_, ok := g.adtCtors[lower(name)]
	return ok
}

func (g *gen) tmpName(prefix string) string {
	g.tmpSeq++
	return fmt.Sprintf("$%s%d", prefix, g.tmpSeq)
}

// declareBinding allocates fresh storage (a global in the program body, a local
// slot inside a routine) and registers it under displayName so synthesized and
// user code can reference it. Used for match scrutinees and pattern bindings.
func (g *gen) declareBinding(displayName string, ti *typeInfo) *vinfo {
	if ti == nil {
		ti = &typeInfo{kind: ktScalar, scalar: tUnknown}
	}
	storage := g.tmpName("b")
	var vi *vinfo
	if len(g.scope) <= 1 { // program body: a global
		g.mod.Globals = append(g.mod.Globals, ir.Global{Name: storage, Type: typeTag(ti.vt())})
		vi = &vinfo{kind: vGlobal, gname: storage, typ: ti.vt(), ti: ti, bindName: displayName}
	} else { // inside a routine: a fresh local slot
		slot := len(g.fn.Params) + len(g.fn.Locals)
		g.fn.Locals = append(g.fn.Locals, storage)
		vi = &vinfo{kind: vLocal, slot: slot, typ: ti.vt(), ti: ti, bindName: displayName}
	}
	g.define(displayName, vi)
	return vi
}
