package codegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/arturoeanton/go-turbo-pascal/internal/ast"
	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
)

// registerOperator records a user operator overload, keyed by the operator
// symbol and its two operand type names, so compileBinary can dispatch to it.
func (g *gen) registerOperator(pd *ast.ProcDecl) {
	left := paramTypeName(pd.Params, 0)
	right := paramTypeName(pd.Params, 1)
	if left == "" || right == "" {
		return
	}
	key := strings.ToLower(pd.OperatorSym) + "|" + left + "|" + right
	g.operators[key] = strings.ToLower(pd.Name)
}

// operatorFor returns the IR function name of an operator overload matching the
// operator and the static types of its operands, or "" when none applies.
func (g *gen) operatorFor(op string, left, right ast.Expr) string {
	if len(g.operators) == 0 {
		return ""
	}
	lt := typeInfoName(g.typeOf(left))
	rt := typeInfoName(g.typeOf(right))
	if lt == "" || rt == "" {
		return ""
	}
	return g.operators[strings.ToLower(op)+"|"+lt+"|"+rt]
}

// paramTypeName returns the lowercase declared type name of the operator's
// i-th operand (grouped parameters share a type).
func paramTypeName(params []ast.Param, i int) string {
	if len(params) == 0 {
		return ""
	}
	if i >= len(params) {
		i = len(params) - 1
	}
	if tr, ok := params[i].Type.(*ast.TypeRef); ok {
		return strings.ToLower(tr.Name)
	}
	if params[i].Type != nil {
		return strings.ToLower(params[i].Type.String())
	}
	return ""
}

// typeInfoName returns the lowercase name of a named type, or "".
func typeInfoName(ti *typeInfo) string {
	if ti == nil || ti.name == "" {
		return ""
	}
	return strings.ToLower(ti.name)
}

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
