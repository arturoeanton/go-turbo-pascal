package codegen

import (
	"fmt"
	"strings"

	"github.com/arturoeanton/go-turbo-pascal/internal/ast"
	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
)

func lower(s string) string { return strings.ToLower(s) }

// compileMatch lowers a match, in statement or expression form (m.IsExpr). The
// scrutinee is evaluated once into a temp; each arm tests its pattern (and
// optional guard) and, on a match, binds payloads and runs its body, then jumps
// past the rest. In expression form each arm's Result is left on the stack. If
// no arm matches and there is no else, a non-exhaustive match raises at runtime.
func (g *gen) compileMatch(m *ast.MatchStmt) {
	scrut := g.declareBinding(g.tmpName("match"), g.inferType(m.Expr))
	scrutID := identFor(scrut.bindName)
	g.assignToVar(scrut.bindName, func() { g.compileExpr(m.Expr) }, m)

	var ends []int
	for _, arm := range m.Arms {
		ends = append(ends, g.emitArm(m, arm, scrutID))
		if arm.Wildcard && arm.Guard == nil {
			break // an unguarded wildcard is terminal
		}
	}
	// Fall-through: no arm matched.
	switch {
	case m.IsExpr && m.ElseExpr != nil:
		g.compileExpr(m.ElseExpr)
	case !m.IsExpr && m.Else != nil:
		for _, s := range m.Else {
			g.compileStmt(s)
		}
	default:
		g.raiseStr("match: no matching arm")
	}
	here := len(g.fn.Code)
	for _, e := range ends {
		g.fn.Patch(e, here)
	}
}

// emitArm emits one arm's test, bindings, guard and body, returning the index
// of the jump that skips to the end of the match (to patch once the end is
// known). Misses (failed pattern or guard) fall through to the next arm.
func (g *gen) emitArm(m *ast.MatchStmt, arm ast.MatchArm, scrutID ast.Expr) int {
	var misses []int
	switch {
	case arm.Wildcard:
		// matches unconditionally
	case arm.Ctor != "":
		// Constructor pattern: tag check, then bind payloads.
		g.compileExpr(fieldOf(scrutID, "__tag"))
		g.fn.Emit(ir.Instr{Op: ir.OPPushStr, S: lower(arm.Ctor)})
		g.fn.Emit(ir.Instr{Op: ir.OPCompare, S: "="})
		misses = append(misses, g.fn.Emit(ir.Instr{Op: ir.OPJumpIfFalse}))
		for i, b := range arm.Binds {
			g.declareBinding(b, &typeInfo{kind: ktScalar, scalar: tUnknown})
			field := fieldOf(scrutID, fmt.Sprintf("__%d", i))
			g.assignToVar(b, func() { g.compileExpr(field) }, m)
		}
	default:
		// Or-patterns: any alternative matching proceeds; none jumps to next arm.
		var hits []int
		for _, val := range arm.Values {
			g.emitValueTest(scrutID, val)
			hits = append(hits, g.fn.Emit(ir.Instr{Op: ir.OPJumpIfTrue}))
		}
		misses = append(misses, g.fn.Emit(ir.Instr{Op: ir.OPJump}))
		here := len(g.fn.Code)
		for _, h := range hits {
			g.fn.Patch(h, here)
		}
	}
	if arm.Guard != nil {
		g.compileExpr(arm.Guard)
		misses = append(misses, g.fn.Emit(ir.Instr{Op: ir.OPJumpIfFalse}))
	}
	if m.IsExpr {
		g.compileExpr(arm.Result)
	} else {
		g.compileStmt(arm.Body)
	}
	end := g.fn.Emit(ir.Instr{Op: ir.OPJump})
	next := len(g.fn.Code)
	for _, mi := range misses {
		g.fn.Patch(mi, next)
	}
	return end
}

// emitValueTest leaves a boolean: does the scrutinee match this value pattern?
// A bare identifier naming a constructor matches on the tag; anything else is an
// equality comparison (literal, constant or enum value).
func (g *gen) emitValueTest(scrutID, val ast.Expr) {
	if id, ok := val.(*ast.Ident); ok && g.isCtor(id.Name) {
		g.compileExpr(fieldOf(scrutID, "__tag"))
		g.fn.Emit(ir.Instr{Op: ir.OPPushStr, S: lower(id.Name)})
		g.fn.Emit(ir.Instr{Op: ir.OPCompare, S: "="})
		return
	}
	g.compileExpr(scrutID)
	g.compileExpr(val)
	g.fn.Emit(ir.Instr{Op: ir.OPCompare, S: "="})
}

// isCtor reports whether name is a registered ADT constructor.
func (g *gen) isCtor(name string) bool {
	_, ok := g.adtCtors[lower(name)]
	return ok
}

func identFor(name string) *ast.Ident {
	return &ast.Ident{Name: name, Lower: lower(name)}
}

func fieldOf(recv ast.Expr, field string) ast.Expr {
	return &ast.FieldExpr{Expr: recv, Field: field}
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
