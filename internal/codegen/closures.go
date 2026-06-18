package codegen

import (
	"fmt"
	"sort"

	"github.com/arturoeanton/go-turbo-pascal/internal/ast"
	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
)

// capVar is an enclosing local captured by an anonymous method (by reference).
type capVar struct {
	name string
	vi   *vinfo
}

// compileAnonFunc compiles an anonymous method expression and leaves a callable
// VKFunc value on the stack. The closure captures the enclosing routine's
// locals and parameters by reference: they become leading var-parameters of the
// generated function. We capture every visible enclosing local (not just the
// free variables) — unused captures are harmless and this avoids a fragile
// free-variable analysis.
func (g *gen) compileAnonFunc(af *ast.AnonFunc) {
	caps := g.visibleLocals()

	// Build the closure environment: a reference for each captured variable.
	for _, c := range caps {
		if c.vi.kind == vVarParam {
			// Already a reference: forward the same cell.
			g.fn.Emit(ir.Instr{Op: ir.OPLoadLocal, A: int64(c.vi.slot)})
		} else {
			g.fn.Emit(ir.Instr{Op: ir.OPAddrLocal, A: int64(c.vi.slot)})
		}
	}
	name := fmt.Sprintf("$anon%d", g.anonSeq)
	g.anonSeq++
	g.fn.Emit(ir.Instr{Op: ir.OPMakeClosure, S: name, A: int64(len(caps))})

	g.compileAnonBody(af, name, caps)
}

// visibleLocals returns the enclosing locals/params/var-params currently in
// scope (excluding the global scope), innermost binding winning, in a
// deterministic order.
func (g *gen) visibleLocals() []capVar {
	seen := map[string]bool{}
	var out []capVar
	for i := len(g.scope) - 1; i >= 1; i-- { // scope[0] is globals
		for name, vi := range g.scope[i] {
			if vi.kind != vLocal && vi.kind != vParam && vi.kind != vVarParam {
				continue
			}
			if seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, capVar{name: name, vi: vi})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

// compileAnonBody emits the generated function for an anonymous method. It runs
// in an isolated scope (globals plus a fresh routine scope) so enclosing locals
// are reachable only through captures, which are bound as leading
// var-parameters, followed by the declared parameters.
func (g *gen) compileAnonBody(af *ast.AnonFunc, name string, caps []capVar) {
	fn := ir.NewFunction(name)
	prevFn := g.fn
	prevObj := g.curObject
	prevScope := g.scope
	prevLabels, prevGotos := g.labels, g.gotoFix

	g.fn = fn
	g.curObject = nil
	g.scope = []map[string]*vinfo{prevScope[0], {}}
	g.labels, g.gotoFix = map[int]int{}, nil

	slot := 0
	for _, c := range caps {
		fn.Params = append(fn.Params, c.name)
		g.define(c.name, &vinfo{kind: vVarParam, slot: slot, typ: c.vi.typ, ti: c.vi.ti})
		slot++
	}
	for _, par := range af.Params {
		pti := g.resolveType(par.Type)
		for _, n := range par.Names {
			fn.Params = append(fn.Params, n)
			kind := vParam
			if par.Var {
				kind = vVarParam
			}
			g.define(n, &vinfo{kind: kind, slot: slot, typ: pti.vt(), ti: pti})
			slot++
		}
	}
	if af.IsFunc {
		var rti *typeInfo
		if af.Result != nil {
			rti = g.resolveType(af.Result)
		}
		res := &vinfo{kind: vResult, typ: rti.vt(), ti: rti}
		g.define("Result", res)
	}
	if af.Body != nil {
		for _, s := range af.Body.Stmts {
			g.compileStmt(s)
		}
	}
	fn.Emit(ir.Instr{Op: ir.OPReturn})
	g.patchGotos(fn)

	g.scope = prevScope
	g.curObject = prevObj
	g.fn = prevFn
	g.labels, g.gotoFix = prevLabels, prevGotos
	g.mod.Funcs[name] = fn
}
