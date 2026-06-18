package vmpas

import (
	"sort"
	"strings"

	"github.com/arturoeanton/go-turbo-pascal/internal/codegen"
	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
)

// Capability inference (G1): statically determine which host capabilities a
// script actually needs by scanning the compiled bytecode for calls to the
// capability-gated host builtins. This enables least-privilege — grant only
// what a tenant's script uses, and reject scripts that would reach for more.

// Capability identifies a sandbox capability category.
type Capability string

const (
	CapFileSystem Capability = "FileSystem"
	CapNetwork    Capability = "Network"
	CapExec       Capability = "Exec"
	CapEnv        Capability = "Env"
	CapDatabase   Capability = "Database"
)

// networkBuiltins / execBuiltins / envBuiltins / dbBuiltins are the gated host
// builtins per capability (PascalCase; matched case-insensitively). fileBuiltins
// lives in vmpas.go.
var networkBuiltins = []string{
	"HttpGet", "HttpPost", "HttpPut", "HttpPatch", "HttpDelete", "HttpRequest",
	"HttpSetHeader", "HttpClearHeaders", "HttpLastStatus",
}
var execBuiltins = []string{"Exec"}
var envBuiltins = []string{"GetEnv"}
var dbBuiltins = []string{
	"DbExec", "DbOpen", "DbEof", "DbNext", "DbFieldStr", "DbFieldInt", "DbClose", "DbError",
}

// builtinCap maps every gated builtin (lowercased) to its capability.
var builtinCap = func() map[string]Capability {
	m := map[string]Capability{}
	add := func(names []string, cap Capability) {
		for _, n := range names {
			m[strings.ToLower(n)] = cap
		}
	}
	add(fileBuiltins, CapFileSystem)
	add(networkBuiltins, CapNetwork)
	add(execBuiltins, CapExec)
	add(envBuiltins, CapEnv)
	add(dbBuiltins, CapDatabase)
	return m
}()

// CapReport is the result of analyzing a script: the set of capabilities it
// needs, and the specific gated builtins that triggered each.
type CapReport struct {
	Required []Capability            // sorted, deduplicated
	Calls    map[Capability][]string // capability -> sorted gated builtins used
}

// Needs reports whether the script requires the given capability.
func (r *CapReport) Needs(c Capability) bool {
	for _, x := range r.Required {
		if x == c {
			return true
		}
	}
	return false
}

// Analyze compiles code and reports which host capabilities it requires, by
// scanning the bytecode for calls to capability-gated builtins. It compiles
// against a permissive name set (every gated builtin is resolvable) so the
// analysis works regardless of the engine's current capabilities — nothing is
// executed. Bound variables and the {$MODE BPGO} gate are honored exactly as in
// a normal run.
func (e *Engine) Analyze(code string) (*CapReport, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	src, err := e.wrap(code)
	if err != nil {
		return nil, err
	}
	// Externals = the engine's current builtins plus every gated builtin name,
	// so a call like HttpGet resolves even under a restricted engine.
	externals := map[string]bool{}
	for name := range e.prepareBuiltins() {
		externals[strings.ToLower(name)] = true
	}
	for name := range builtinCap {
		externals[name] = true
	}
	presets := map[string]bool{}
	for n := range e.vars {
		presets[n] = true
	}
	prog, err := codegen.CompileWithOptions(src, "analyze.pas", codegen.Options{
		Externals:           externals,
		PresetGlobals:       presets,
		AutoDeclareLoopVars: true,
	})
	if err != nil {
		return nil, err
	}
	return analyzeProgram(prog), nil
}

// analyzeProgram scans a compiled program for calls to gated builtins.
func analyzeProgram(prog *ir.Program) *CapReport {
	used := map[Capability]map[string]bool{}
	for _, m := range prog.Modules {
		for _, fn := range m.Funcs {
			for _, in := range fn.Code {
				if in.Op != ir.OPCallBuiltin {
					continue
				}
				if cap, ok := builtinCap[strings.ToLower(in.S)]; ok {
					if used[cap] == nil {
						used[cap] = map[string]bool{}
					}
					used[cap][strings.ToLower(in.S)] = true
				}
			}
		}
	}
	rep := &CapReport{Calls: map[Capability][]string{}}
	for cap, names := range used {
		rep.Required = append(rep.Required, cap)
		list := make([]string, 0, len(names))
		for n := range names {
			list = append(list, n)
		}
		sort.Strings(list)
		rep.Calls[cap] = list
	}
	sort.Slice(rep.Required, func(i, j int) bool { return rep.Required[i] < rep.Required[j] })
	return rep
}
