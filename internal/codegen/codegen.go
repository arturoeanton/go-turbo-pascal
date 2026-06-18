// Package codegen is the real BPGo compiler back-end: it lowers a parsed
// Turbo Pascal 7 AST into IR for the embedded VM (internal/ir). Unlike the
// minimal on-the-fly translator in internal/compile, this package builds
// proper procedures and functions with their own frames, value and var
// parameters, local variables, recursion and full control flow.
//
// The procedural core lives here; records, arrays, pointers, sets, the unit
// system and the TP7 object model are layered on in later phases. Any
// language construct not yet supported produces a clear compile error rather
// than silently doing nothing, so coverage gaps are always visible.
package codegen

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/arturoeanton/go-turbo-pascal/internal/ast"
	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
	"github.com/arturoeanton/go-turbo-pascal/internal/lexer"
	"github.com/arturoeanton/go-turbo-pascal/internal/parser"
)

// vtype is a coarse value-type tag used for write formatting, zero
// initialization and boolean-vs-bitwise operator selection. The full type
// system lives in internal/sem; codegen only needs these distinctions.
type vtype int

const (
	tUnknown vtype = iota
	tInt
	tReal
	tStr
	tChar
	tBool
)

// varKind classifies how a name is stored.
type varKind int

const (
	vGlobal varKind = iota
	vLocal
	vParam
	vVarParam
	vConst
	vResult
)

type vinfo struct {
	kind     varKind
	slot     int       // frame slot for local/param/varparam
	gname    string    // global name for vGlobal
	constVal ir.Value  // value for vConst
	typ      vtype     // coarse type (scalar formatting / operator choice)
	ti       *typeInfo // resolved type (records, arrays, pointers, enums)
}

type paramInfo struct {
	varParam bool
	typ      vtype
}

type fnEntry struct {
	irName     string
	params     []paramInfo
	isFunc     bool
	resultType vtype
}

type loopCtx struct {
	continueTarget  int
	breakPatches    []int
	continuePatches []int
}

// withCtx is an active `with rec do` scope: bare field names of rec's type
// resolve to rec.field. The record address is recomputed per access (correct
// for the common case of a simple record variable).
type withCtx struct {
	expr ast.Expr
	ti   *typeInfo
}

type gen struct {
	mod       *ir.Module
	fn        *ir.Function
	scope     []map[string]*vinfo
	funcs     map[string]*fnEntry
	types     map[string]*typeInfo
	externals map[string]bool
	presets   map[string]bool
	autoLoop  bool
	loops     []*loopCtx
	curObject *typeInfo                    // object type when compiling a method
	withs     []withCtx                    // active `with` records (innermost last)
	labels    map[int]int                  // label number -> code index (per function)
	gotoFix   []gotoRef                    // pending goto jumps to patch (per function)
	vtables   map[string]map[string]string // type -> method -> ir func name
	operators map[string]string            // "op|leftType|rightType" -> ir func name
	dir       string                       // directory of the main source (unit lookup)
	loaded    map[string]bool              // units already loaded
	initFuncs []string                     // unit initialization functions to run first
	anonSeq   int                          // counter for anonymous-method function names
	errs      []string
}

// gotoRef is an emitted goto awaiting resolution to its label's code index.
type gotoRef struct {
	idx   int
	label int
}

// Options tunes code generation.
type Options struct {
	// Externals are lowercase names that should be lowered to host/RTL builtin
	// calls (OPCallBuiltin) instead of being reported as unknown identifiers.
	// Embedders (pkg/vmpas) use this to expose Go functions and RTL procedures
	// to Pascal code.
	Externals map[string]bool
	// PresetGlobals are lowercase global names that the embedder seeds into the
	// VM before running. Codegen skips their automatic zero-initialization so
	// the seeded value is preserved.
	PresetGlobals map[string]bool
	// AutoDeclareLoopVars allows a for-loop control variable that was not
	// declared to be auto-declared as an Integer global. This eases REPL-style
	// snippets (used by pkg/vmpas); strict programs leave it off.
	AutoDeclareLoopVars bool
}

// Compile lexes, parses and generates an IR program from Pascal source.
func Compile(src, file string) (*ir.Program, error) {
	return CompileWithOptions(src, file, Options{})
}

// CompileWithOptions is Compile with explicit options.
func CompileWithOptions(src, file string, opts Options) (*ir.Program, error) {
	l := lexer.New(src)
	if errs := l.Errors(); len(errs) > 0 {
		return nil, fmt.Errorf("lex errors: %v", errs)
	}
	p := parser.New(l.Tokens())
	p.SetFile(file)
	node := p.ParseUnit()
	if errs := p.Errors(); len(errs) > 0 {
		return nil, fmt.Errorf("parse errors: %v", errs)
	}
	prog, ok := node.(*ast.Program)
	if !ok {
		return nil, fmt.Errorf("codegen: only program units are supported in this phase")
	}
	ext := opts.Externals
	if ext == nil {
		ext = map[string]bool{}
	}
	presets := opts.PresetGlobals
	if presets == nil {
		presets = map[string]bool{}
	}
	g := &gen{
		mod:       &ir.Module{Name: "main", Funcs: map[string]*ir.Function{}, Init: []string{}},
		funcs:     map[string]*fnEntry{},
		types:     map[string]*typeInfo{},
		externals: ext,
		presets:   presets,
		autoLoop:  opts.AutoDeclareLoopVars,
		vtables:   map[string]map[string]string{},
		operators: map[string]string{},
		dir:       filepath.Dir(file),
		loaded:    map[string]bool{},
	}
	g.compileProgram(prog)
	if len(g.errs) > 0 {
		return nil, fmt.Errorf("codegen errors: %s", strings.Join(g.errs, "; "))
	}
	return &ir.Program{Modules: []*ir.Module{g.mod}, Entry: "main", Vtables: g.vtables}, nil
}

func (g *gen) errf(format string, args ...any) {
	g.errs = append(g.errs, fmt.Sprintf(format, args...))
}

// --- scope helpers ---

func (g *gen) push() { g.scope = append(g.scope, map[string]*vinfo{}) }
func (g *gen) pop()  { g.scope = g.scope[:len(g.scope)-1] }
func (g *gen) define(name string, v *vinfo) {
	g.scope[len(g.scope)-1][strings.ToLower(name)] = v
}

func (g *gen) lookup(name string) *vinfo {
	low := strings.ToLower(name)
	for i := len(g.scope) - 1; i >= 0; i-- {
		if v, ok := g.scope[i][low]; ok {
			return v
		}
	}
	return nil
}

// --- program / declarations ---

func (g *gen) compileProgram(p *ast.Program) {
	g.push() // global scope
	main := ir.NewFunction("main")
	g.fn = main

	// Load units named in the program's uses clause first.
	if p.Uses != nil {
		for _, it := range p.Uses.Items {
			g.useUnit(it.Name)
		}
	}

	if p.Block != nil {
		g.registerTypes(p.Block.Types)
		g.declareConsts(p.Block.Consts, true)
		// Register function signatures first so calls (incl. recursion and
		// forward references) resolve regardless of declaration order.
		for _, d := range p.Block.Procs {
			if pd, ok := d.(*ast.ProcDecl); ok {
				for _, tp := range pd.TypeParams { // erase generic routine params
					g.registerTypeParam(tp)
				}
				if pd.OperatorSym != "" {
					g.registerOperator(pd)
				}
				if !strings.Contains(pd.Name, ".") {
					g.registerSignature(pd)
				}
			}
		}
		g.declareGlobals(p.Block.Vars)
		// Generate each procedure/function body. Qualified names (Type.Method)
		// are object methods and compile with an implicit Self.
		for _, d := range p.Block.Procs {
			if pd, ok := d.(*ast.ProcDecl); ok && pd.Body != nil {
				if strings.Contains(pd.Name, ".") {
					g.compileMethod(pd)
				} else {
					g.compileProc(pd)
				}
			}
		}
		g.buildVtables()
	}

	// Main body.
	g.fn = main
	g.labels, g.gotoFix = map[int]int{}, nil
	// Run unit initialization sections before the program body.
	for _, initName := range g.initFuncs {
		main.Emit(ir.Instr{Op: ir.OPCall, S: initName})
		main.Emit(ir.Instr{Op: ir.OPPop})
	}
	if p.Body != nil {
		for _, s := range p.Body.Stmts {
			g.compileStmt(s)
		}
	}
	g.patchGotos(main)
	main.Emit(ir.Instr{Op: ir.OPHalt})
	g.mod.Funcs["main"] = main
	g.pop()
}

func (g *gen) declareConsts(decls []ast.Decl, global bool) {
	for _, d := range decls {
		cd, ok := d.(*ast.ConstDecl)
		if !ok {
			continue
		}
		val, typ, ok := g.constValue(cd.Value)
		if !ok {
			g.errf("const %s: unsupported constant expression", cd.Name)
			continue
		}
		g.define(cd.Name, &vinfo{kind: vConst, constVal: val, typ: typ})
	}
}

func (g *gen) declareGlobals(decls []ast.Decl) {
	for _, d := range decls {
		vd, ok := d.(*ast.VarDecl)
		if !ok {
			continue
		}
		ti := g.resolveType(vd.Type)
		t := ti.vt()
		for _, name := range vd.Names {
			g.mod.Globals = append(g.mod.Globals, ir.Global{Name: name, Type: typeTag(t)})
			g.define(name, &vinfo{kind: vGlobal, gname: name, typ: t, ti: ti})
			// Aggregates and non-int scalars need an explicit zero value, unless
			// the embedder seeds this global before running (preset).
			if needsInit(ti) && !g.presets[strings.ToLower(name)] {
				g.pushZero(ti)
				g.fn.Emit(ir.Instr{Op: ir.OPStoreGlobal, S: name})
			}
		}
	}
}

// pushZero emits an OPPushZero that pushes a deep copy of ti's zero value.
func (g *gen) pushZero(ti *typeInfo) {
	idx := g.fn.AddTemplate(g.zeroTemplate(ti))
	g.fn.Emit(ir.Instr{Op: ir.OPPushZero, A: int64(idx)})
}

// needsInit reports whether a variable of this type needs explicit zero
// initialization (frame slots and fresh globals default to integer 0).
func needsInit(ti *typeInfo) bool {
	if ti == nil {
		return false
	}
	switch ti.kind {
	case ktScalar:
		return ti.scalar != tInt && ti.scalar != tUnknown
	case ktEnum:
		return false
	}
	return true
}

func (g *gen) registerSignature(pd *ast.ProcDecl) {
	e := &fnEntry{irName: strings.ToLower(pd.Name), isFunc: pd.IsFunc}
	for _, par := range pd.Params {
		pt := vtypeOf(par.Type)
		for range par.Names {
			e.params = append(e.params, paramInfo{varParam: par.Var, typ: pt})
		}
	}
	if pd.Result != nil {
		e.resultType = vtypeOfRef(pd.Result)
	}
	g.funcs[strings.ToLower(pd.Name)] = e
}

func (g *gen) compileProc(pd *ast.ProcDecl) {
	g.compileRoutine(pd, strings.ToLower(pd.Name), nil, pd.Name)
}

// compileRoutine compiles a procedure, function or method body. When selfType
// is non-nil, slot 0 is an implicit Self reference and the routine is a method.
// resultName is the identifier that, when assigned, sets the function result.
func (g *gen) compileRoutine(pd *ast.ProcDecl, fnName string, selfType *typeInfo, resultName string) {
	fn := ir.NewFunction(fnName)
	prevFn := g.fn
	prevObj := g.curObject
	prevLabels, prevGotos := g.labels, g.gotoFix
	g.fn = fn
	g.curObject = selfType
	g.labels, g.gotoFix = map[int]int{}, nil
	g.push()

	slot := 0
	if selfType != nil {
		fn.Params = append(fn.Params, "Self")
		g.define("Self", &vinfo{kind: vVarParam, slot: 0, typ: tUnknown, ti: selfType})
		slot = 1
	}
	for _, par := range pd.Params {
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

	// Function result: assignable via the function name or "Result".
	if pd.IsFunc {
		var rti *typeInfo
		if pd.Result != nil {
			rti = g.resolveType(pd.Result)
		}
		res := &vinfo{kind: vResult, typ: rti.vt(), ti: rti}
		g.define(resultName, res)
		g.define("Result", res)
		// Initialize a structured result (record/array) so fields/elements of
		// the result are addressable (`Result.field := ...`).
		if needsInit(rti) {
			g.pushZero(rti)
			g.fn.Emit(ir.Instr{Op: ir.OPSetResult})
		}
	}

	// Local consts/types/vars live in the nested block.
	var locals []ast.Decl
	var consts []ast.Decl
	if pd.Nested != nil {
		consts = pd.Nested.Consts
		locals = pd.Nested.Vars
		g.registerTypes(pd.Nested.Types)
	}
	g.declareConsts(consts, false)
	var localInits []*vinfo
	for _, d := range locals {
		vd, ok := d.(*ast.VarDecl)
		if !ok {
			continue
		}
		lti := g.resolveType(vd.Type)
		for _, n := range vd.Names {
			vi := &vinfo{kind: vLocal, slot: slot, typ: lti.vt(), ti: lti}
			g.define(n, vi)
			fn.Locals = append(fn.Locals, n)
			localInits = append(localInits, vi)
			slot++
		}
	}

	// Zero-initialize locals whose zero value is not integer 0.
	for _, vi := range localInits {
		if needsInit(vi.ti) {
			g.pushZero(vi.ti)
			fn.Emit(ir.Instr{Op: ir.OPStoreLocal, A: int64(vi.slot)})
		}
	}

	if pd.Body != nil {
		for _, s := range pd.Body.Stmts {
			g.compileStmt(s)
		}
	}
	fn.Emit(ir.Instr{Op: ir.OPReturn})
	g.patchGotos(fn)
	g.pop()
	g.curObject = prevObj
	g.fn = prevFn
	g.labels, g.gotoFix = prevLabels, prevGotos
	g.mod.Funcs[fnName] = fn
}

// patchGotos resolves the function's goto jumps to their label code indices.
func (g *gen) patchGotos(fn *ir.Function) {
	for _, gf := range g.gotoFix {
		if target, ok := g.labels[gf.label]; ok {
			fn.Patch(gf.idx, target)
		} else {
			g.errf("undefined label %d", gf.label)
		}
	}
}

// --- statements ---

func (g *gen) compileStmt(s ast.Stmt) {
	// Record a source line for the statement's first instruction (used by the
	// debugger for line breakpoints and stepping).
	if s != nil {
		if pos := s.Pos(); pos.Line > 0 {
			g.fn.SourceMap[len(g.fn.Code)] = ir.SourceRef{File: pos.File, Line: pos.Line}
		}
	}
	switch v := s.(type) {
	case nil:
		// empty statement
	case *ast.CompoundStmt:
		for _, ss := range v.Stmts {
			g.compileStmt(ss)
		}
	case *ast.AssignStmt:
		g.compileAssign(v)
	case *ast.CallStmt:
		g.compileCallStmt(&v.Call)
	case *ast.IfStmt:
		g.compileIf(v)
	case *ast.WhileStmt:
		g.compileWhile(v)
	case *ast.RepeatStmt:
		g.compileRepeat(v)
	case *ast.ForStmt:
		g.compileFor(v)
	case *ast.ForInStmt:
		g.compileForIn(v)
	case *ast.CaseStmt:
		g.compileCase(v)
	case *ast.InheritedStmt:
		g.compileInherited(v)
	case *ast.WithStmt:
		g.compileWith(v)
	case *ast.TryStmt:
		g.compileTry(v)
	case *ast.RaiseStmt:
		if v.Expr != nil {
			g.compileExpr(v.Expr)
			g.fn.Emit(ir.Instr{Op: ir.OPRaise, A: 1})
		} else {
			g.fn.Emit(ir.Instr{Op: ir.OPRaise, A: 0})
		}
	case *ast.LabelStmt:
		g.labels[v.Label] = len(g.fn.Code)
	case *ast.GotoStmt:
		idx := g.fn.Emit(ir.Instr{Op: ir.OPJump})
		g.gotoFix = append(g.gotoFix, gotoRef{idx: idx, label: v.Label})
	case *ast.ExitStmt:
		g.fn.Emit(ir.Instr{Op: ir.OPReturn})
	case *ast.HaltStmt:
		if v.Code != nil {
			g.compileExpr(v.Code)
		} else {
			g.fn.Emit(ir.Instr{Op: ir.OPPushInt, A: 0})
		}
		g.fn.Emit(ir.Instr{Op: ir.OPSetResult})
		g.fn.Emit(ir.Instr{Op: ir.OPReturn})
	case *ast.BreakStmt:
		if l := g.curLoop(); l != nil {
			l.breakPatches = append(l.breakPatches, g.fn.Emit(ir.Instr{Op: ir.OPJump}))
		} else {
			g.errf("break outside loop")
		}
	case *ast.ContinueStmt:
		if l := g.curLoop(); l != nil {
			l.continuePatches = append(l.continuePatches, g.fn.Emit(ir.Instr{Op: ir.OPJump}))
		} else {
			g.errf("continue outside loop")
		}
	default:
		g.errf("unsupported statement %T (not yet implemented in this phase)", s)
	}
}

func (g *gen) curLoop() *loopCtx {
	if len(g.loops) == 0 {
		return nil
	}
	return g.loops[len(g.loops)-1]
}

func (g *gen) compileAssign(a *ast.AssignStmt) {
	if id, ok := a.Dest.(*ast.Ident); ok {
		g.assignToVar(id.Name, func() { g.compileExpr(a.Expr) }, id)
		return
	}
	// obj.Prop := v resolves through the property's `write` specifier: a setter
	// method becomes obj.SetProp(v); a backing field becomes a field store.
	if fe, ok := a.Dest.(*ast.FieldExpr); ok {
		if bt := g.typeOf(fe.Expr); bt != nil && bt.kind == ktObject {
			if pr, ok := bt.prop(fe.Field); ok && pr.write != "" {
				if bt.hasMethod(pr.write) {
					g.compileMethodCall(fe.Expr, pr.write, []ast.Expr{a.Expr}, true)
				} else {
					g.compileFieldAddr(fe.Expr, pr.write)
					g.compileExpr(a.Expr)
					g.fn.Emit(ir.Instr{Op: ir.OPStoreRef})
				}
				return
			}
		}
	}
	// s[i] := c : 1-based character assignment into a string.
	if ix, ok := a.Dest.(*ast.IndexExpr); ok {
		if bt := g.typeOf(ix.Expr); bt != nil && bt.kind == ktString {
			g.compileAddr(ix.Expr)
			g.compileExpr(ix.Index)
			g.compileExpr(a.Expr)
			g.fn.Emit(ir.Instr{Op: ir.OPStrCharStore})
			return
		}
	}
	// Field / array element / pointer target: compute the lvalue reference,
	// then store the value through it.
	g.compileAddr(a.Dest)
	g.compileExpr(a.Expr)
	g.fn.Emit(ir.Instr{Op: ir.OPStoreRef})
}

// assignToVar emits a store to the named variable, calling genVal to place
// the value on the stack at the right moment (var parameters need their
// reference pushed first).
func (g *gen) assignToVar(name string, genVal func(), at ast.Node) {
	vi := g.lookup(name)
	if vi == nil {
		if g.withFieldAddr(name) {
			genVal()
			g.fn.Emit(ir.Instr{Op: ir.OPStoreRef})
			return
		}
		if g.selfFieldAddr(name) {
			genVal()
			g.fn.Emit(ir.Instr{Op: ir.OPStoreRef})
			return
		}
		g.errf("unknown identifier %q", name)
		return
	}
	switch vi.kind {
	case vGlobal:
		genVal()
		g.fn.Emit(ir.Instr{Op: ir.OPStoreGlobal, S: vi.gname})
	case vLocal, vParam:
		genVal()
		g.fn.Emit(ir.Instr{Op: ir.OPStoreLocal, A: int64(vi.slot)})
	case vVarParam:
		g.fn.Emit(ir.Instr{Op: ir.OPLoadLocal, A: int64(vi.slot)})
		genVal()
		g.fn.Emit(ir.Instr{Op: ir.OPStoreRef})
	case vResult:
		genVal()
		g.fn.Emit(ir.Instr{Op: ir.OPSetResult})
	case vConst:
		g.errf("cannot assign to constant %q", name)
	}
}

func (g *gen) compileIf(v *ast.IfStmt) {
	g.compileExpr(v.Cond)
	jf := g.fn.Emit(ir.Instr{Op: ir.OPJumpIfFalse})
	g.compileStmt(v.Then)
	if v.Else != nil {
		jend := g.fn.Emit(ir.Instr{Op: ir.OPJump})
		g.fn.Patch(jf, len(g.fn.Code))
		g.compileStmt(v.Else)
		g.fn.Patch(jend, len(g.fn.Code))
	} else {
		g.fn.Patch(jf, len(g.fn.Code))
	}
}

func (g *gen) compileWhile(v *ast.WhileStmt) {
	top := len(g.fn.Code)
	g.compileExpr(v.Cond)
	jf := g.fn.Emit(ir.Instr{Op: ir.OPJumpIfFalse})
	l := &loopCtx{continueTarget: top}
	g.loops = append(g.loops, l)
	g.compileStmt(v.Body)
	g.fn.Emit(ir.Instr{Op: ir.OPJump, A: int64(top)})
	end := len(g.fn.Code)
	g.fn.Patch(jf, end)
	g.finishLoop(l, top, end)
}

func (g *gen) compileRepeat(v *ast.RepeatStmt) {
	top := len(g.fn.Code)
	l := &loopCtx{}
	g.loops = append(g.loops, l)
	g.compileStmt(v.Body)
	cont := len(g.fn.Code)
	g.compileExpr(v.Cond)
	// repeat..until cond: loop again while cond is false.
	g.fn.Emit(ir.Instr{Op: ir.OPJumpIfFalse, A: int64(top)})
	end := len(g.fn.Code)
	g.finishLoop(l, cont, end)
}

func (g *gen) compileFor(v *ast.ForStmt) {
	// Auto-declare an undeclared loop variable for snippet convenience.
	if g.autoLoop && g.lookup(v.Var) == nil {
		g.mod.Globals = append(g.mod.Globals, ir.Global{Name: v.Var, Type: "integer"})
		g.scope[0][strings.ToLower(v.Var)] = &vinfo{
			kind: vGlobal, gname: v.Var, typ: tInt,
			ti: &typeInfo{kind: ktScalar, scalar: tInt},
		}
	}
	// var := lo
	g.assignToVar(v.Var, func() { g.compileExpr(v.Lo) }, v)
	top := len(g.fn.Code)
	// while var <= hi (or >= hi for downto)
	g.loadVar(v.Var)
	g.compileExpr(v.Hi)
	op := "<="
	if v.Down {
		op = ">="
	}
	g.fn.Emit(ir.Instr{Op: ir.OPCompare, S: op})
	jf := g.fn.Emit(ir.Instr{Op: ir.OPJumpIfFalse})
	l := &loopCtx{}
	g.loops = append(g.loops, l)
	g.compileStmt(v.Body)
	cont := len(g.fn.Code)
	// var := var +/- 1
	bin := "+"
	if v.Down {
		bin = "-"
	}
	g.assignToVar(v.Var, func() {
		g.loadVar(v.Var)
		g.fn.Emit(ir.Instr{Op: ir.OPPushInt, A: 1})
		g.fn.Emit(ir.Instr{Op: ir.OPBinary, S: bin})
	}, v)
	g.fn.Emit(ir.Instr{Op: ir.OPJump, A: int64(top)})
	end := len(g.fn.Code)
	g.fn.Patch(jf, end)
	g.finishLoop(l, cont, end)
}

// compileForIn lowers `for x in coll do body` to an indexed for loop:
// arrays iterate 0..High(coll), strings iterate 1..Length(coll).
func (g *gen) compileForIn(v *ast.ForInStmt) {
	ct := g.typeOf(v.Coll)
	idxName := g.freshGlobal()
	idx := &ast.Ident{Name: idxName, Lower: strings.ToLower(idxName)}
	elemAssign := &ast.AssignStmt{
		Dest: &ast.Ident{Name: v.Var, Lower: strings.ToLower(v.Var)},
		Expr: &ast.IndexExpr{Expr: v.Coll, Index: idx},
	}
	body := &ast.CompoundStmt{Stmts: []ast.Stmt{elemAssign, v.Body}}

	var lo, hi ast.Expr
	if ct != nil && ct.kind == ktString {
		lo = &ast.IntLit{Value: 1}
		hi = &ast.CallExpr{Func: &ast.Ident{Name: "Length", Lower: "length"}, Args: []ast.Expr{v.Coll}}
	} else {
		lo = &ast.IntLit{Value: 0}
		hi = &ast.CallExpr{Func: &ast.Ident{Name: "High", Lower: "high"}, Args: []ast.Expr{v.Coll}}
	}
	g.compileFor(&ast.ForStmt{Var: idxName, Lo: lo, Hi: hi, Body: body})
}

// freshGlobal declares a synthetic Integer global (for desugaring) and returns
// its name.
func (g *gen) freshGlobal() string {
	name := fmt.Sprintf("$tmp%d", len(g.mod.Globals))
	g.mod.Globals = append(g.mod.Globals, ir.Global{Name: name, Type: "integer"})
	g.scope[0][strings.ToLower(name)] = &vinfo{
		kind: vGlobal, gname: name, typ: tInt,
		ti: &typeInfo{kind: ktScalar, scalar: tInt},
	}
	return name
}

func (g *gen) compileCase(v *ast.CaseStmt) {
	g.compileExpr(v.Expr) // selector stays on the stack as we test branches
	var endPatches []int
	for _, br := range v.Cases {
		var matchPatches []int
		for _, val := range br.Values {
			g.fn.Emit(ir.Instr{Op: ir.OPDup})
			g.compileExpr(val)
			g.fn.Emit(ir.Instr{Op: ir.OPCompare, S: "="})
			matchPatches = append(matchPatches, g.fn.Emit(ir.Instr{Op: ir.OPJumpIfTrue}))
		}
		// none matched -> skip this branch body
		skip := g.fn.Emit(ir.Instr{Op: ir.OPJump})
		here := len(g.fn.Code)
		for _, mp := range matchPatches {
			g.fn.Patch(mp, here)
		}
		g.fn.Emit(ir.Instr{Op: ir.OPPop}) // drop selector before running body
		g.compileStmt(br.Body)
		endPatches = append(endPatches, g.fn.Emit(ir.Instr{Op: ir.OPJump}))
		g.fn.Patch(skip, len(g.fn.Code))
	}
	// else / default: selector still on stack
	g.fn.Emit(ir.Instr{Op: ir.OPPop})
	if v.Else != nil {
		g.compileStmt(v.Else)
	}
	for _, ep := range endPatches {
		g.fn.Patch(ep, len(g.fn.Code))
	}
}

func (g *gen) finishLoop(l *loopCtx, continueTarget, end int) {
	for _, p := range l.continuePatches {
		g.fn.Patch(p, continueTarget)
	}
	for _, p := range l.breakPatches {
		g.fn.Patch(p, end)
	}
	g.loops = g.loops[:len(g.loops)-1]
}

// --- calls ---

func (g *gen) compileCallStmt(c *ast.CallExpr) {
	g.compileCall(c, true)
}

// compileCall lowers a call. asStmt controls whether the (always pushed)
// result is discarded.
func (g *gen) compileCall(c *ast.CallExpr, asStmt bool) {
	// Method call: receiver.method(args) dispatches dynamically.
	if fe, ok := c.Func.(*ast.FieldExpr); ok {
		// Constructor call on a class type name: TClass.Create(...).
		if id, ok2 := fe.Expr.(*ast.Ident); ok2 {
			if ct := g.types[strings.ToLower(id.Name)]; ct != nil && ct.kind == ktObject && ct.isClass && g.lookup(id.Name) == nil {
				g.compileClassCreate(ct, fe.Field, c.Args, asStmt)
				return
			}
		}
		g.compileMethodCall(fe.Expr, fe.Field, c.Args, asStmt)
		return
	}
	id, ok := c.Func.(*ast.Ident)
	if !ok {
		g.errf("unsupported call target %T", c.Func)
		return
	}
	name := strings.ToLower(id.Name)

	// Bare method call inside a method body resolves against Self.
	if g.curObject != nil && g.curObject.hasMethod(name) && g.lookup(id.Name) == nil {
		g.compileSelfMethodCall(name, c.Args, asStmt)
		return
	}

	// Call through a procedural-type variable / closure value.
	if vi := g.lookup(id.Name); vi != nil && vi.ti != nil && vi.ti.kind == ktFunc {
		g.loadVar(id.Name)
		for _, a := range c.Args {
			g.compileExpr(a)
		}
		g.fn.Emit(ir.Instr{Op: ir.OPCallValue, A: int64(len(c.Args))})
		if asStmt {
			g.fn.Emit(ir.Instr{Op: ir.OPPop})
		}
		return
	}

	switch name {
	case "assign", "reset", "rewrite", "append", "close", "erase", "rename":
		// File operations take the file variable by reference.
		if len(c.Args) > 0 {
			g.compileAddr(c.Args[0])
		}
		for _, a := range c.Args[1:] {
			g.compileExpr(a)
		}
		g.fn.Emit(ir.Instr{Op: ir.OPCallBuiltin, S: name, A: int64(len(c.Args))})
		if asStmt {
			g.fn.Emit(ir.Instr{Op: ir.OPPop})
		}
		return
	case "write", "writeln":
		// Typed file: Write(f, v1, v2, ...) writes each value as a binary record.
		if len(c.Args) > 0 && g.isTypedFileExpr(c.Args[0]) {
			for _, a := range c.Args[1:] {
				g.compileAddr(c.Args[0])
				g.compileExpr(a)
				g.fn.Emit(ir.Instr{Op: ir.OPCallBuiltin, S: "__fwritetyped", A: 2})
				g.fn.Emit(ir.Instr{Op: ir.OPPop})
			}
			return
		}
		writeArgs := c.Args
		bn := name
		if len(c.Args) > 0 && g.isFileExpr(c.Args[0]) {
			g.compileAddr(c.Args[0]) // file reference
			writeArgs = c.Args[1:]
			bn = "__f" + name
		}
		for _, a := range writeArgs {
			if wa, ok := a.(*ast.WriteArg); ok {
				g.compileExpr(wa.Value)
				if wa.Width != nil {
					g.compileExpr(wa.Width)
				} else {
					g.fn.Emit(ir.Instr{Op: ir.OPPushInt, A: -1})
				}
				if wa.Decimals != nil {
					g.compileExpr(wa.Decimals)
				} else {
					g.fn.Emit(ir.Instr{Op: ir.OPPushInt, A: -1})
				}
				g.fn.Emit(ir.Instr{Op: ir.OPCallBuiltin, S: "__writefmt", A: 3})
			} else {
				g.compileExpr(a)
			}
		}
		g.fn.Emit(ir.Instr{Op: ir.OPCallBuiltin, S: bn, A: int64(len(c.Args))})
		if asStmt {
			g.fn.Emit(ir.Instr{Op: ir.OPPop})
		}
		return
	case "read", "readln":
		// Typed file: Read(f, v1, ...) reads each value from a binary record.
		if len(c.Args) > 0 && g.isTypedFileExpr(c.Args[0]) {
			for _, a := range c.Args[1:] {
				g.compileAddr(c.Args[0])
				g.compileAddr(a)
				g.fn.Emit(ir.Instr{Op: ir.OPCallBuiltin, S: "__freadtyped", A: 2})
				g.fn.Emit(ir.Instr{Op: ir.OPPop})
			}
			return
		}
		// Read targets are passed by reference so the builtin can write them.
		// A leading file variable routes the read to that file.
		rbn := name
		if len(c.Args) > 0 && g.isFileExpr(c.Args[0]) {
			rbn = "__f" + name
		}
		for _, a := range c.Args {
			g.compileAddr(a)
		}
		g.fn.Emit(ir.Instr{Op: ir.OPCallBuiltin, S: rbn, A: int64(len(c.Args))})
		if asStmt {
			g.fn.Emit(ir.Instr{Op: ir.OPPop})
		}
		return
	case "inc", "dec":
		g.compileIncDec(c, name)
		return
	case "new":
		if len(c.Args) == 1 {
			g.compileNew(c.Args[0])
		} else {
			g.errf("New(p, constructor) is not supported yet")
		}
		return
	case "dispose":
		if len(c.Args) >= 1 {
			g.compileAddr(c.Args[0])
			g.fn.Emit(ir.Instr{Op: ir.OPPushNil})
			g.fn.Emit(ir.Instr{Op: ir.OPStoreRef})
		}
		return
	case "setlength":
		if len(c.Args) >= 2 {
			g.compileSetLength(c.Args[0], c.Args[1])
		} else {
			g.errf("SetLength requires a dynamic array and a length")
		}
		return
	}

	if fe, ok := g.funcs[name]; ok {
		// User-defined procedure/function: honour var parameters.
		for i, a := range c.Args {
			if i < len(fe.params) && fe.params[i].varParam {
				g.compileAddr(a)
			} else {
				g.compileExpr(a)
			}
		}
		g.fn.Emit(ir.Instr{Op: ir.OPCall, S: "main." + fe.irName, A: int64(len(c.Args))})
		if asStmt {
			g.fn.Emit(ir.Instr{Op: ir.OPPop})
		}
		return
	}

	if isBuiltinFunc(name) {
		for _, a := range c.Args {
			g.compileExpr(a)
		}
		g.fn.Emit(ir.Instr{Op: ir.OPCallBuiltin, S: canonicalBuiltin(name), A: int64(len(c.Args))})
		if asStmt {
			g.fn.Emit(ir.Instr{Op: ir.OPPop})
		}
		return
	}

	// Host/RTL externals (registered by the embedder) dispatch to a builtin.
	if g.externals[name] {
		for _, a := range c.Args {
			g.compileExpr(a)
		}
		g.fn.Emit(ir.Instr{Op: ir.OPCallBuiltin, S: name, A: int64(len(c.Args))})
		if asStmt {
			g.fn.Emit(ir.Instr{Op: ir.OPPop})
		}
		return
	}

	g.errf("unknown procedure or function %q", id.Name)
}

// compileNew implements New(p): allocate a heap cell of the pointer's
// element type and store its reference into p.
func (g *gen) compileNew(arg ast.Expr) {
	elem := g.pointerElem(g.typeOf(arg))
	g.compileAddr(arg)
	g.pushZero(elem)
	g.fn.Emit(ir.Instr{Op: ir.OPHeapAlloc})
	g.fn.Emit(ir.Instr{Op: ir.OPStoreRef})
}

// compileSetLength implements SetLength(arr, n) for a dynamic array.
func (g *gen) compileSetLength(arg, n ast.Expr) {
	var elem *typeInfo
	if ti := g.typeOf(arg); ti != nil && ti.kind == ktArray {
		elem = ti.elem
	}
	g.compileAddr(arg) // array reference
	g.compileExpr(n)   // new length
	g.pushZero(elem)   // element zero template
	g.fn.Emit(ir.Instr{Op: ir.OPSetLength})
}

// compileSet builds a set value from its elements. Ranges (a..b) must be
// constant in this phase.
func (g *gen) compileSet(s *ast.SetExpr) {
	count := 0
	for _, el := range s.Elements {
		if el.Hi == nil {
			g.compileExpr(el.Lo)
			count++
			continue
		}
		lo, _, ok1 := g.constValue(el.Lo)
		hi, _, ok2 := g.constValue(el.Hi)
		if !ok1 || !ok2 {
			g.errf("set ranges must be constant in this phase")
			continue
		}
		for v := lo.Int; v <= hi.Int; v++ {
			g.fn.Emit(ir.Instr{Op: ir.OPPushInt, A: v})
			count++
		}
	}
	g.fn.Emit(ir.Instr{Op: ir.OPMkSet, A: int64(count)})
}

// compileIncDec implements Inc(x)/Dec(x) and Inc(x,n)/Dec(x,n) as an
// assignment so they correctly modify the variable (var semantics).
func (g *gen) compileIncDec(c *ast.CallExpr, name string) {
	if len(c.Args) < 1 {
		g.errf("%s requires an argument", name)
		return
	}
	target, ok := c.Args[0].(*ast.Ident)
	if !ok {
		g.errf("%s requires a variable", name)
		return
	}
	op := "+"
	if name == "dec" {
		op = "-"
	}
	g.assignToVar(target.Name, func() {
		g.loadVar(target.Name)
		if len(c.Args) >= 2 {
			g.compileExpr(c.Args[1])
		} else {
			g.fn.Emit(ir.Instr{Op: ir.OPPushInt, A: 1})
		}
		g.fn.Emit(ir.Instr{Op: ir.OPBinary, S: op})
	}, c)
}

// --- expressions ---

func (g *gen) compileExpr(e ast.Expr) {
	switch v := e.(type) {
	case *ast.IntLit:
		g.fn.Emit(ir.Instr{Op: ir.OPPushInt, A: v.Value})
	case *ast.RealLit:
		g.fn.Emit(ir.Instr{Op: ir.OPPushReal, R: v.Value})
	case *ast.StringLit:
		// A single-character literal is a Char in TP7 context, but treating
		// it as a string keeps Write semantics correct; assignment to Char
		// vars is handled at the value level by the VM.
		g.fn.Emit(ir.Instr{Op: ir.OPPushStr, S: v.Value})
	case *ast.CharLit:
		g.fn.Emit(ir.Instr{Op: ir.OPPushChar, A: int64(v.Value)})
	case *ast.Ident:
		g.compileIdent(v)
	case *ast.BinaryExpr:
		g.compileBinary(v)
	case *ast.UnaryExpr:
		g.compileExpr(v.Expr)
		g.fn.Emit(ir.Instr{Op: ir.OPUnary, S: strings.ToLower(v.Op)})
	case *ast.CallExpr:
		g.compileCall(v, false)
	case *ast.FieldExpr:
		// Parameterless constructor on a class type name: TClass.Create.
		if id, ok := v.Expr.(*ast.Ident); ok {
			if ct := g.types[strings.ToLower(id.Name)]; ct != nil && ct.kind == ktObject && ct.isClass && g.lookup(id.Name) == nil {
				g.compileClassCreate(ct, v.Field, nil, false)
				break
			}
		}
		// A property read resolves through its `read` specifier: a getter
		// method becomes a call, a backing field becomes a field load. A bare
		// parameterless method used in an expression is also a call.
		if bt := g.typeOf(v.Expr); bt != nil && bt.kind == ktObject {
			if pr, ok := bt.prop(v.Field); ok && pr.read != "" {
				if bt.hasMethod(pr.read) {
					g.compileMethodCall(v.Expr, pr.read, nil, false)
				} else {
					g.compileFieldAddr(v.Expr, pr.read)
					g.fn.Emit(ir.Instr{Op: ir.OPLoadRef})
				}
				break
			}
			if bt.hasMethod(v.Field) {
				g.compileMethodCall(v.Expr, v.Field, nil, false)
				break
			}
		}
		g.compileAddr(v)
		g.fn.Emit(ir.Instr{Op: ir.OPLoadRef})
	case *ast.IndexExpr:
		if bt := g.typeOf(v.Expr); bt != nil && bt.kind == ktString {
			// s[i] : 1-based character access.
			g.compileExpr(v.Expr)
			g.compileExpr(v.Index)
			g.fn.Emit(ir.Instr{Op: ir.OPStrChar})
		} else {
			g.compileAddr(v)
			g.fn.Emit(ir.Instr{Op: ir.OPLoadRef})
		}
	case *ast.CaretExpr:
		// p^ : load the pointer, then dereference it.
		g.compileExpr(v.Expr)
		g.fn.Emit(ir.Instr{Op: ir.OPLoadRef})
	case *ast.AnonFunc:
		g.compileAnonFunc(v)
	case *ast.AtExpr:
		// @Routine yields a callable value (a closure with no captures).
		if id, ok := v.Expr.(*ast.Ident); ok && g.lookup(id.Name) == nil {
			if fe, isFn := g.funcs[strings.ToLower(id.Name)]; isFn {
				g.fn.Emit(ir.Instr{Op: ir.OPMakeClosure, S: fe.irName, A: 0})
				break
			}
		}
		g.compileAddr(v.Expr)
	case *ast.SetExpr:
		g.compileSet(v)
	case *ast.InExpr:
		g.compileExpr(v.Left)
		g.compileExpr(v.Right)
		g.fn.Emit(ir.Instr{Op: ir.OPIn})
	default:
		g.errf("unsupported expression %T (not yet implemented in this phase)", e)
	}
}

func (g *gen) compileIdent(v *ast.Ident) {
	switch v.Lower {
	case "true":
		g.fn.Emit(ir.Instr{Op: ir.OPPushBool, A: 1})
		return
	case "false":
		g.fn.Emit(ir.Instr{Op: ir.OPPushBool, A: 0})
		return
	case "nil":
		g.fn.Emit(ir.Instr{Op: ir.OPPushNil})
		return
	}
	g.loadVar(v.Name)
}

func (g *gen) loadVar(name string) {
	vi := g.lookup(name)
	if vi == nil {
		if g.withFieldAddr(name) {
			g.fn.Emit(ir.Instr{Op: ir.OPLoadRef})
			return
		}
		if g.selfFieldAddr(name) {
			g.fn.Emit(ir.Instr{Op: ir.OPLoadRef})
			return
		}
		g.errf("unknown identifier %q", name)
		return
	}
	switch vi.kind {
	case vConst:
		g.pushConst(vi.constVal)
	case vGlobal:
		g.fn.Emit(ir.Instr{Op: ir.OPLoadGlobal, S: vi.gname})
	case vLocal, vParam:
		g.fn.Emit(ir.Instr{Op: ir.OPLoadLocal, A: int64(vi.slot)})
	case vVarParam:
		g.fn.Emit(ir.Instr{Op: ir.OPLoadLocal, A: int64(vi.slot)})
		g.fn.Emit(ir.Instr{Op: ir.OPLoadRef})
	case vResult:
		g.fn.Emit(ir.Instr{Op: ir.OPLoadResult})
	}
}

// compileAddr pushes a reference to an lvalue (used for var arguments,
// assignment targets and field/element access).
func (g *gen) compileAddr(e ast.Expr) {
	switch v := e.(type) {
	case *ast.Ident:
		g.compileAddrIdent(v)
	case *ast.FieldExpr:
		bt := g.typeOf(v.Expr)
		field := bt.backingField(v.Field)
		if bt != nil && bt.kind == ktObject && bt.isClass {
			g.compileExpr(v.Expr) // a class reference already points at the record
		} else {
			g.compileAddr(v.Expr)
		}
		g.fn.Emit(ir.Instr{Op: ir.OPFieldAddr, S: strings.ToLower(field)})
	case *ast.IndexExpr:
		g.compileAddr(v.Expr)
		bt := g.typeOf(v.Expr)
		g.compileExpr(v.Index)
		if bt != nil && bt.kind == ktArray && bt.lo != 0 {
			g.fn.Emit(ir.Instr{Op: ir.OPPushInt, A: bt.lo})
			g.fn.Emit(ir.Instr{Op: ir.OPBinary, S: "-"})
		}
		g.fn.Emit(ir.Instr{Op: ir.OPElemAddr})
	case *ast.CaretExpr:
		// p^ : the pointer value already is a reference to the pointee.
		g.compileExpr(v.Expr)
	default:
		g.errf("cannot take address of %T", e)
	}
}

func (g *gen) compileAddrIdent(id *ast.Ident) {
	vi := g.lookup(id.Name)
	if vi == nil {
		if g.withFieldAddr(id.Name) {
			return
		}
		if g.selfFieldAddr(id.Name) {
			return
		}
		g.errf("unknown identifier %q", id.Name)
		return
	}
	switch vi.kind {
	case vGlobal:
		g.fn.Emit(ir.Instr{Op: ir.OPAddrGlobal, S: vi.gname})
	case vLocal, vParam:
		g.fn.Emit(ir.Instr{Op: ir.OPAddrLocal, A: int64(vi.slot)})
	case vVarParam:
		// The slot already holds a reference to the caller's storage.
		g.fn.Emit(ir.Instr{Op: ir.OPLoadLocal, A: int64(vi.slot)})
	case vResult:
		// Reference to the result cell (so `Result.field := ...` works).
		g.fn.Emit(ir.Instr{Op: ir.OPAddrResult})
	default:
		g.errf("cannot take the address of %q", id.Name)
	}
}

// compileTry lowers try..except / try..finally. For finally, the finally body
// is emitted on both the normal and exception paths (the latter re-raises).
func (g *gen) compileTry(v *ast.TryStmt) {
	if v.Finally != nil {
		enter := g.fn.Emit(ir.Instr{Op: ir.OPEnterTry, B: 1})
		for _, s := range v.Body {
			g.compileStmt(s)
		}
		g.fn.Emit(ir.Instr{Op: ir.OPPopTry})
		for _, s := range v.Finally { // normal path
			g.compileStmt(s)
		}
		jmpEnd := g.fn.Emit(ir.Instr{Op: ir.OPJump})
		g.fn.Code[enter].A = int64(len(g.fn.Code)) // exception path entry
		for _, s := range v.Finally {
			g.compileStmt(s)
		}
		g.fn.Emit(ir.Instr{Op: ir.OPReraise})
		g.fn.Patch(jmpEnd, len(g.fn.Code))
		return
	}
	enter := g.fn.Emit(ir.Instr{Op: ir.OPEnterTry, B: 0})
	for _, s := range v.Body {
		g.compileStmt(s)
	}
	g.fn.Emit(ir.Instr{Op: ir.OPPopTry})
	jmpEnd := g.fn.Emit(ir.Instr{Op: ir.OPJump})
	g.fn.Code[enter].A = int64(len(g.fn.Code)) // except handler entry
	for _, s := range v.Except {
		g.compileStmt(s)
	}
	g.fn.Patch(jmpEnd, len(g.fn.Code))
}

// compileWith compiles `with rec do body`, making rec's fields resolvable as
// bare identifiers in the body.
func (g *gen) compileWith(v *ast.WithStmt) {
	ti := g.typeOf(v.Rec)
	if ti == nil || (ti.kind != ktRecord && ti.kind != ktObject) {
		g.errf("with requires a record or object variable")
		return
	}
	g.withs = append(g.withs, withCtx{expr: v.Rec, ti: ti})
	g.compileStmt(v.Body)
	g.withs = g.withs[:len(g.withs)-1]
}

// withFieldAddr emits the address of a field reachable through an active `with`
// (innermost first), returning true if it handled the name.
func (g *gen) withFieldAddr(name string) bool {
	for i := len(g.withs) - 1; i >= 0; i-- {
		w := g.withs[i]
		if w.ti != nil && w.ti.field(name) != nil {
			g.compileAddr(w.expr)
			g.fn.Emit(ir.Instr{Op: ir.OPFieldAddr, S: strings.ToLower(name)})
			return true
		}
	}
	return false
}

// isFileExpr reports whether an expression is a Text/File variable.
func (g *gen) isFileExpr(e ast.Expr) bool {
	t := g.typeOf(e)
	return t != nil && t.kind == ktFile
}

// isTypedFileExpr reports whether an expression is a typed file (`file of T`).
func (g *gen) isTypedFileExpr(e ast.Expr) bool {
	t := g.typeOf(e)
	return t != nil && t.kind == ktFile && t.elem != nil
}

// typeOf returns the resolved type of an lvalue expression, or nil.
func (g *gen) typeOf(e ast.Expr) *typeInfo {
	switch v := e.(type) {
	case *ast.Ident:
		if vi := g.lookup(v.Name); vi != nil {
			return vi.ti
		}
	case *ast.FieldExpr:
		if bt := g.typeOf(v.Expr); bt != nil {
			if pr, ok := bt.prop(v.Field); ok && pr.read != "" && bt.hasMethod(pr.read) {
				return g.methodResultType(bt, pr.read)
			}
			if f := bt.field(bt.backingField(v.Field)); f != nil {
				return f.ti
			}
		}
	case *ast.IndexExpr:
		if bt := g.typeOf(v.Expr); bt != nil && bt.kind == ktArray {
			return bt.elem
		}
	case *ast.CaretExpr:
		return g.pointerElem(g.typeOf(v.Expr))
	}
	return nil
}

func (g *gen) compileBinary(v *ast.BinaryExpr) {
	op := strings.ToLower(v.Op)
	// Operator overloading: a user-declared `operator <op> (a, b: T)` takes
	// precedence over the built-in operator when both operand types match.
	if fn := g.operatorFor(v.Op, v.Left, v.Right); fn != "" {
		g.compileExpr(v.Left)
		g.compileExpr(v.Right)
		g.fn.Emit(ir.Instr{Op: ir.OPCall, S: fn, A: 2})
		return
	}
	if isCompareOp(op) {
		g.compileExpr(v.Left)
		g.compileExpr(v.Right)
		g.fn.Emit(ir.Instr{Op: ir.OPCompare, S: op})
		return
	}
	// Boolean and/or use short-circuit evaluation (TP7 default {$B-}).
	if (op == "and" || op == "or") && g.exprType(v.Left) == tBool {
		g.compileExpr(v.Left)
		if op == "and" {
			j := g.fn.Emit(ir.Instr{Op: ir.OPJumpIfFalse})
			g.compileExpr(v.Right)
			jend := g.fn.Emit(ir.Instr{Op: ir.OPJump})
			g.fn.Patch(j, len(g.fn.Code))
			g.fn.Emit(ir.Instr{Op: ir.OPPushBool, A: 0})
			g.fn.Patch(jend, len(g.fn.Code))
		} else {
			j := g.fn.Emit(ir.Instr{Op: ir.OPJumpIfTrue})
			g.compileExpr(v.Right)
			jend := g.fn.Emit(ir.Instr{Op: ir.OPJump})
			g.fn.Patch(j, len(g.fn.Code))
			g.fn.Emit(ir.Instr{Op: ir.OPPushBool, A: 1})
			g.fn.Patch(jend, len(g.fn.Code))
		}
		return
	}
	g.compileExpr(v.Left)
	g.compileExpr(v.Right)
	g.fn.Emit(ir.Instr{Op: ir.OPBinary, S: op})
}

func (g *gen) pushConst(val ir.Value) {
	switch val.Kind {
	case ir.VKReal:
		g.fn.Emit(ir.Instr{Op: ir.OPPushReal, R: val.Real})
	case ir.VKStr:
		g.fn.Emit(ir.Instr{Op: ir.OPPushStr, S: val.Str})
	case ir.VKBool:
		a := int64(0)
		if val.Bool {
			a = 1
		}
		g.fn.Emit(ir.Instr{Op: ir.OPPushBool, A: a})
	case ir.VKChar:
		g.fn.Emit(ir.Instr{Op: ir.OPPushChar, A: int64(val.Ch)})
	default:
		g.fn.Emit(ir.Instr{Op: ir.OPPushInt, A: val.Int})
	}
}

// --- compile-time helpers ---

func (g *gen) constValue(e ast.Expr) (ir.Value, vtype, bool) {
	switch v := e.(type) {
	case *ast.IntLit:
		return ir.Value{Kind: ir.VKInt, Int: v.Value}, tInt, true
	case *ast.RealLit:
		return ir.Value{Kind: ir.VKReal, Real: v.Value}, tReal, true
	case *ast.StringLit:
		return ir.Value{Kind: ir.VKStr, Str: v.Value}, tStr, true
	case *ast.CharLit:
		return ir.Value{Kind: ir.VKChar, Ch: byte(v.Value)}, tChar, true
	case *ast.Ident:
		switch v.Lower {
		case "true":
			return ir.Value{Kind: ir.VKBool, Bool: true}, tBool, true
		case "false":
			return ir.Value{Kind: ir.VKBool, Bool: false}, tBool, true
		}
		if vi := g.lookup(v.Name); vi != nil && vi.kind == vConst {
			return vi.constVal, vi.typ, true
		}
	case *ast.UnaryExpr:
		if inner, t, ok := g.constValue(v.Expr); ok && v.Op == "-" {
			if inner.Kind == ir.VKReal {
				return ir.Value{Kind: ir.VKReal, Real: -inner.Real}, tReal, true
			}
			return ir.Value{Kind: ir.VKInt, Int: -inner.Int}, t, true
		}
	}
	return ir.Value{}, tUnknown, false
}

// exprType is a best-effort static type for operator selection.
func (g *gen) exprType(e ast.Expr) vtype {
	switch v := e.(type) {
	case *ast.IntLit:
		return tInt
	case *ast.RealLit:
		return tReal
	case *ast.StringLit:
		return tStr
	case *ast.CharLit:
		return tChar
	case *ast.Ident:
		switch v.Lower {
		case "true", "false":
			return tBool
		}
		if vi := g.lookup(v.Name); vi != nil {
			return vi.typ
		}
	case *ast.BinaryExpr:
		op := strings.ToLower(v.Op)
		if isCompareOp(op) || op == "and" || op == "or" || op == "not" {
			return tBool
		}
		if g.exprType(v.Left) == tReal || g.exprType(v.Right) == tReal {
			return tReal
		}
		return tInt
	case *ast.UnaryExpr:
		if strings.ToLower(v.Op) == "not" {
			return tBool
		}
		return g.exprType(v.Expr)
	case *ast.CallExpr:
		if id, ok := v.Func.(*ast.Ident); ok {
			if fe := g.funcs[strings.ToLower(id.Name)]; fe != nil {
				return fe.resultType
			}
		}
	case *ast.FieldExpr, *ast.IndexExpr, *ast.CaretExpr:
		return g.typeOf(e).vt()
	}
	return tUnknown
}

func isCompareOp(op string) bool {
	switch op {
	case "=", "<>", "<", "<=", ">", ">=":
		return true
	}
	return false
}

// vtypeOf maps an AST type expression to a coarse vtype.
func vtypeOf(t ast.TypeExpr) vtype {
	switch v := t.(type) {
	case *ast.OrdType:
		switch v.Kind {
		case "Boolean":
			return tBool
		case "Char":
			return tChar
		default:
			return tInt
		}
	case *ast.FloatType:
		return tReal
	case *ast.StringType:
		return tStr
	case *ast.TypeRef:
		switch strings.ToLower(v.Name) {
		case "string":
			return tStr
		case "char":
			return tChar
		case "boolean":
			return tBool
		case "real", "single", "double", "extended", "comp":
			return tReal
		case "integer", "longint", "word", "byte", "shortint":
			return tInt
		}
	}
	return tUnknown
}

func vtypeOfRef(r *ast.TypeRef) vtype {
	if r == nil {
		return tUnknown
	}
	return vtypeOf(r)
}

func typeTag(t vtype) string {
	switch t {
	case tReal:
		return "real"
	case tStr:
		return "string"
	case tChar:
		return "char"
	case tBool:
		return "boolean"
	}
	return "integer"
}
