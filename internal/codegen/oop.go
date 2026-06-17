package codegen

import (
	"strings"

	"github.com/arturoeanton/go-turbo-pascal/internal/ast"
	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
)

// compileMethod compiles a top-level "TType.Method" body as an IR function
// "ttype.method" with an implicit Self reference parameter.
func (g *gen) compileMethod(pd *ast.ProcDecl) {
	dot := strings.Index(pd.Name, ".")
	typeName := strings.ToLower(pd.Name[:dot])
	methodOrig := pd.Name[dot+1:]
	objType := g.types[typeName]
	if objType == nil || objType.kind != ktObject {
		g.errf("method %q on unknown object type", pd.Name)
		return
	}
	fnName := typeName + "." + strings.ToLower(methodOrig)
	// resultName is the unqualified method name so `Method := value` works.
	g.compileRoutine(pd, fnName, objType, methodOrig)
}

// buildVtables computes, for every object type, the flattened method table
// (method -> most-derived implementing function) used for dynamic dispatch.
func (g *gen) buildVtables() {
	for name, ti := range g.types {
		if ti.kind != ktObject {
			continue
		}
		tbl := map[string]string{}
		for _, m := range ti.methods {
			if owner := g.methodOwner(ti, m.name); owner != "" {
				tbl[m.name] = owner + "." + m.name
			}
		}
		g.vtables[name] = tbl
	}
}

// methodOwner finds the most-derived type in ti's ancestry that has a compiled
// body for method, returning its lowercase type name.
func (g *gen) methodOwner(ti *typeInfo, method string) string {
	for t := ti; t != nil; t = t.parent {
		if _, ok := g.mod.Funcs[t.objName+"."+method]; ok {
			return t.objName
		}
	}
	return ""
}

// compileMethodCall lowers receiver.method(args) with dynamic dispatch.
func (g *gen) compileMethodCall(recv ast.Expr, method string, args []ast.Expr, asStmt bool) {
	g.compileAddr(recv) // Self reference
	for _, a := range args {
		g.compileExpr(a)
	}
	g.fn.Emit(ir.Instr{Op: ir.OPCallMethod, S: strings.ToLower(method), A: int64(len(args) + 1)})
	if asStmt {
		g.fn.Emit(ir.Instr{Op: ir.OPPop})
	}
}

// compileSelfMethodCall lowers a bare method call inside a method body (Self).
func (g *gen) compileSelfMethodCall(method string, args []ast.Expr, asStmt bool) {
	g.fn.Emit(ir.Instr{Op: ir.OPLoadLocal, A: 0}) // Self
	for _, a := range args {
		g.compileExpr(a)
	}
	g.fn.Emit(ir.Instr{Op: ir.OPCallMethod, S: strings.ToLower(method), A: int64(len(args) + 1)})
	if asStmt {
		g.fn.Emit(ir.Instr{Op: ir.OPPop})
	}
}

// compileInherited lowers `inherited Method(args)` as a static call to the
// parent type's implementation, passing the current Self.
func (g *gen) compileInherited(v *ast.InheritedStmt) {
	if g.curObject == nil || g.curObject.parent == nil {
		g.errf("inherited used outside a derived method")
		return
	}
	if v.Call == nil {
		return
	}
	id, ok := v.Call.Func.(*ast.Ident)
	if !ok {
		g.errf("invalid inherited call")
		return
	}
	method := strings.ToLower(id.Name)
	owner := g.methodOwner(g.curObject.parent, method)
	if owner == "" {
		g.errf("inherited method %q not found", id.Name)
		return
	}
	g.fn.Emit(ir.Instr{Op: ir.OPLoadLocal, A: 0}) // Self
	for _, a := range v.Call.Args {
		g.compileExpr(a)
	}
	g.fn.Emit(ir.Instr{Op: ir.OPCall, S: owner + "." + method, A: int64(len(v.Call.Args) + 1)})
	g.fn.Emit(ir.Instr{Op: ir.OPPop})
}

// selfFieldFallback resolves a bare identifier that is a field of the object
// currently being compiled (a method body), emitting code via Self. It returns
// true if it handled the name.
func (g *gen) selfFieldAddr(name string) bool {
	if g.curObject == nil || g.curObject.field(name) == nil {
		return false
	}
	g.fn.Emit(ir.Instr{Op: ir.OPLoadLocal, A: 0}) // Self
	g.fn.Emit(ir.Instr{Op: ir.OPFieldAddr, S: strings.ToLower(name)})
	return true
}
