package codegen

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/arturoeanton/go-turbo-pascal/internal/ast"
	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
	"github.com/arturoeanton/go-turbo-pascal/internal/lexer"
	"github.com/arturoeanton/go-turbo-pascal/internal/parser"
	"github.com/arturoeanton/go-turbo-pascal/internal/rtl/crt"
)

// rtlUnits are provided by the runtime as VM builtins, so `uses` of them needs
// no source file.
var rtlUnits = map[string]bool{
	"system": true, "crt": true, "dos": true, "strings": true,
	"windos": true, "printer": true, "graph": true, "graph3": true,
	"overlay": true, "turbo3": true,
}

// useUnit handles a single entry of a uses clause.
func (g *gen) useUnit(name string) {
	low := strings.ToLower(name)
	if rtlUnits[low] {
		// RTL units are provided as VM builtins; expose their callable names so
		// calls compile (the runtime registers the actual builtins).
		for _, n := range rtlExports[low] {
			g.externals[strings.ToLower(n)] = true
		}
		return
	}
	g.loadUnit(low)
}

// rtlExports lists the callable names each RTL unit adds to scope via `uses`.
var rtlExports = map[string][]string{
	"crt": crt.CrtExports,
}

// loadUnit compiles a user unit found next to the main source: it merges the
// unit's interface and implementation declarations into the program and
// registers its initialization section to run before the program body.
func (g *gen) loadUnit(name string) {
	if g.loaded[name] {
		return
	}
	g.loaded[name] = true

	path := g.findUnitFile(name)
	if path == "" {
		g.errf("unit %q not found", name)
		return
	}
	src, err := os.ReadFile(path)
	if err != nil {
		g.errf("cannot read unit %q: %v", name, err)
		return
	}
	l := lexer.New(string(src))
	if errs := l.Errors(); len(errs) > 0 {
		g.errf("unit %s: lex: %v", name, errs)
		return
	}
	p := parser.New(l.Tokens())
	p.SetFile(path)
	node := p.ParseUnit()
	if errs := p.Errors(); len(errs) > 0 {
		g.errf("unit %s: parse: %v", name, errs)
		return
	}
	u, ok := node.(*ast.Unit)
	if !ok {
		g.errf("%q is not a unit", name)
		return
	}

	// Load units this one depends on first.
	if u.Interface != nil && u.Interface.Uses != nil {
		for _, it := range u.Interface.Uses.Items {
			g.useUnit(it.Name)
		}
	}
	if u.Implementation != nil && u.Implementation.Uses != nil {
		for _, it := range u.Implementation.Uses.Items {
			g.useUnit(it.Name)
		}
	}

	var decls []ast.Decl
	if u.Interface != nil {
		decls = append(decls, u.Interface.Decls...)
	}
	if u.Implementation != nil {
		decls = append(decls, u.Implementation.Decls...)
	}
	g.processUnitDecls(decls)

	if u.Init != nil {
		initName := "__init_" + name
		fn := ir.NewFunction(initName)
		prev := g.fn
		g.fn = fn
		for _, s := range u.Init.Stmts {
			g.compileStmt(s)
		}
		fn.Emit(ir.Instr{Op: ir.OPReturn})
		g.fn = prev
		g.mod.Funcs[initName] = fn
		g.initFuncs = append(g.initFuncs, initName)
	}
}

// processUnitDecls registers a unit's declarations (types, consts, globals,
// signatures) and compiles its procedure/function/method bodies, in order.
func (g *gen) processUnitDecls(decls []ast.Decl) {
	g.registerTypes(decls)
	g.declareConsts(decls, true)
	g.declareGlobals(decls)
	for _, d := range decls {
		if pd, ok := d.(*ast.ProcDecl); ok && !strings.Contains(pd.Name, ".") {
			g.registerSignature(pd)
		}
	}
	for _, d := range decls {
		if pd, ok := d.(*ast.ProcDecl); ok && pd.Body != nil {
			if strings.Contains(pd.Name, ".") {
				g.compileMethod(pd)
			} else {
				g.compileProc(pd)
			}
		}
	}
}

// findUnitFile resolves a unit name to a source file beside the main program,
// case-insensitively.
func (g *gen) findUnitFile(name string) string {
	for _, ext := range []string{".pas", ".pp"} {
		p := filepath.Join(g.dir, name+ext)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	entries, err := os.ReadDir(g.dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		en := strings.ToLower(e.Name())
		if en == name+".pas" || en == name+".pp" {
			return filepath.Join(g.dir, e.Name())
		}
	}
	return ""
}
