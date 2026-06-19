// Package vmpas embeds the BPGo Pascal engine in Go programs.
//
// vmpas is a fast, strongly-typed dynamic-code engine: it compiles and
// type-checks Turbo Pascal 7 source ahead of execution (catching errors
// before the first run) and executes it on the embedded bytecode VM
// (internal/ir). Go variables, functions and structs can be bound so guest
// Pascal code can read/write them and call into the host.
//
// Security: every Engine runs under a capability sandbox (see Capabilities).
// The default (New / Restricted) denies filesystem, network, process-exec and
// environment access, and the sandbox can also bound execution by step count,
// heap allocations and wall-clock time. Use Full to allow everything (e.g. for
// a trusted TP7 IDE). Gated host builtins (GetEnv/Exec/HttpGet) are registered
// only when their capability is granted, so under the default sandbox they are
// simply unknown identifiers.
//
// vmpas has zero external dependencies (enforced by a test), so importing it
// never pulls in the editor/IDE tooling.
package vmpas

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/arturoeanton/go-turbo-pascal/internal/codegen"
	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
	"github.com/arturoeanton/go-turbo-pascal/internal/rtl/crt"
	"github.com/arturoeanton/go-turbo-pascal/internal/rtl/system"
)

// Capabilities is the sandbox configuration for an Engine. The zero value is
// fully restricted (default-deny).
type Capabilities struct {
	FileSystem   bool          // allow file I/O (Assign/Reset/Rewrite/...)
	Network      bool          // allow the HTTP host builtins (Http* )
	Exec         bool          // allow the Exec host builtin (run processes)
	Env          bool          // allow the GetEnv host builtin
	Database     bool          // allow the SQL host builtins (Db*); needs UseDB
	MaxSteps     int           // VM step limit (0 = engine default)
	MaxHeap      int           // max heap allocations (0 = unlimited)
	MaxOutput    int           // max captured output bytes (0 = unlimited)
	MaxCallDepth int           // max call-stack depth (0 = unlimited)
	MaxDuration  time.Duration // wall-clock execution limit (0 = none)
	// Deterministic makes execution fully reproducible: Randomize seeds the RNG
	// from Seed (not host entropy), so the same source + inputs always yield the
	// same output and state. Required for reliable snapshot/resume (phase F).
	Deterministic bool
	Seed          int64
	// Audit records every call to a capability-gated host builtin (file,
	// network, exec, env, database) in execution order; retrieve it with
	// Engine.AuditLog after a run.
	Audit bool
	// LiveBindings keeps bound Go variables (Var) in sync with the script around
	// every call into a bound Go function/procedure: the script's current values
	// are written back to Go before the call and the host's mutations are made
	// visible to the script after it. Off by default (values sync only at the
	// start and end of a run). Adds per-call overhead proportional to the number
	// of bound variables.
	LiveBindings bool
}

// Sandboxed returns a safe, bounded capability set suitable for running
// untrusted per-tenant scripts in a multi-tenant service: default-deny on
// all host access, plus conservative ceilings on steps, heap, output, call
// depth and wall-clock time. Adjust the limits to taste.
func Sandboxed() Capabilities {
	return Capabilities{
		MaxSteps:     5_000_000,
		MaxHeap:      1_000_000,
		MaxOutput:    1 << 20, // 1 MiB
		MaxCallDepth: 5000,
		MaxDuration:  2 * time.Second,
	}
}

// Restricted returns a safe, default-deny capability set.
func Restricted() Capabilities { return Capabilities{} }

// Full returns a capability set that allows everything (use only for trusted
// code, e.g. a full TP7-compatible IDE).
func Full() Capabilities {
	return Capabilities{FileSystem: true, Network: true, Exec: true, Env: true, Database: true}
}

// fileBuiltins are RTL builtins gated by Capabilities.FileSystem.
var fileBuiltins = []string{
	"Assign", "Reset", "Rewrite", "Append", "Close", "Erase", "Rename",
	"BlockRead", "BlockWrite", "Flush", "Seek", "FilePos", "FileSize",
	"Truncate", "SetTextBuf",
}

// Engine owns a set of Go bindings and executes Pascal against them.
type Engine struct {
	mu     sync.Mutex
	caps   Capabilities
	vars   map[string]reflect.Value // name -> pointer (or value) Value
	funcs  map[string]reflect.Value // value-returning callables
	procs  map[string]reflect.Value // procedures
	output string
	// builtins caches the prepared builtin table (RTL + host funcs). It is
	// VM-independent (builtins use the vm passed at call time), so it is reused
	// across runs to avoid re-registering ~100 builtins each time. Invalidated
	// when bindings or capabilities change.
	builtins map[string]ir.Builtin
	// Host integration state (single-threaded per run; guarded by mu).
	db          SQLDB             // database handle injected via UseDB (Database cap)
	cursor      *dbCursor         // active query cursor (Db* dataset-style API)
	dbErr       string            // last DB error message (DbError)
	httpStatus  int               // status code of the last HTTP call (HttpLastStatus)
	httpHeaders map[string]string // headers applied to subsequent HTTP requests
	suspendTag  string            // tag from the last Suspend call (durable runs)
	audit       []AuditEntry      // capability-gated calls recorded this run (Audit cap)
	hostErr     string            // message of the last host error raised as an exception
}

// UseDB binds a database handle for the Db* host builtins. The handle is the
// host's responsibility (it brings the driver), keeping vmpas dependency-free.
// Wrap a *sql.DB with WrapSQLDB. Has no effect unless Capabilities.Database.
func (e *Engine) UseDB(db SQLDB) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.db = db
	e.builtins = nil // re-register Db* builtins
}

// New creates an isolated, fully-restricted Pascal engine.
func New() *Engine { return NewWith(Restricted()) }

// NewWith creates an engine with the given sandbox capabilities.
func NewWith(caps Capabilities) *Engine {
	return &Engine{
		caps:  caps,
		vars:  map[string]reflect.Value{},
		funcs: map[string]reflect.Value{},
		procs: map[string]reflect.Value{},
	}
}

// RunSandboxed compiles and runs untrusted Pascal source on a fresh,
// share-nothing engine under the given capabilities, returning the captured
// output and any error. It is the recommended one-shot entry point for
// multi-tenant services: every call is fully isolated (no bindings, no state
// shared with other tenants or prior calls). Pass Sandboxed() for safe
// defaults, or a custom Capabilities to tune limits / grant host access.
func RunSandboxed(code string, caps Capabilities) (string, error) {
	e := NewWith(caps)
	err := e.Run(code)
	return e.Output(), err
}

var defaultEngine = New()

// Package-level convenience wrappers operate on a shared default engine.
func Var(name string, ptr any) error      { return defaultEngine.Var(name, ptr) }
func Function(name string, fn any) error  { return defaultEngine.Function(name, fn) }
func Func(name string, fn any) error      { return Function(name, fn) }
func Process(name string, fn any) error   { return defaultEngine.Process(name, fn) }
func Procedure(name string, fn any) error { return Process(name, fn) }
func Run(code string) error               { return defaultEngine.Run(code) }
func Output() string                      { return defaultEngine.Output() }
func Reset()                              { defaultEngine = New() }
func SetCapabilities(caps Capabilities)   { defaultEngine.SetCapabilities(caps) }

// SetCapabilities updates the sandbox configuration.
func (e *Engine) SetCapabilities(caps Capabilities) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.caps = caps
	e.builtins = nil // capability change affects which builtins are registered
}

// Var binds a Go variable. Pass a pointer for read/write; a non-pointer is
// read-only (seeded into Pascal but not written back).
func (e *Engine) Var(name string, ptr any) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if name == "" {
		return fmt.Errorf("vmpas: empty variable name")
	}
	v := reflect.ValueOf(ptr)
	if !v.IsValid() {
		return fmt.Errorf("vmpas: invalid variable %q", name)
	}
	e.vars[strings.ToLower(name)] = v
	return nil
}

// Function binds a Go function callable from Pascal as a function.
func (e *Engine) Function(name string, fn any) error { return e.bind(e.funcs, name, fn) }

// Process binds a Go procedure callable from Pascal.
func (e *Engine) Process(name string, fn any) error { return e.bind(e.procs, name, fn) }

func (e *Engine) bind(dst map[string]reflect.Value, name string, fn any) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if name == "" {
		return fmt.Errorf("vmpas: empty callable name")
	}
	v := reflect.ValueOf(fn)
	if !v.IsValid() || v.Kind() != reflect.Func {
		return fmt.Errorf("vmpas: %q is not a function", name)
	}
	dst[strings.ToLower(name)] = v
	e.builtins = nil // new host callable changes the builtin table
	return nil
}

// Output returns text written by Pascal Write/WriteLn during the last Run.
func (e *Engine) Output() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.output
}

// Script is a Pascal program compiled once against an engine's bindings. Run
// it many times without recompiling — bound Go variables are re-seeded on each
// run. Bind variables and functions before calling Compile.
type Script struct {
	eng  *Engine
	prog *ir.Program
}

// Compile parses, type-checks and compiles code once, returning a reusable
// Script. This is the fast path for the compile-once / run-many model.
func (e *Engine) Compile(code string) (*Script, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	prog, err := e.compileLocked(code)
	if err != nil {
		return nil, err
	}
	return &Script{eng: e, prog: prog}, nil
}

// Run executes the compiled script under the engine's sandbox, re-seeding bound
// Go variables and reading them back afterward.
func (s *Script) Run() error {
	s.eng.mu.Lock()
	defer s.eng.mu.Unlock()
	return s.eng.execLocked(s.prog)
}

// Output returns the text produced by the most recent run of this script.
func (s *Script) Output() string { return s.eng.Output() }

// Run compiles and executes Pascal source (or a statement fragment) against the
// bound variables and functions, under the engine's sandbox. For repeated
// execution prefer Compile + Script.Run.
func (e *Engine) Run(code string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	prog, err := e.compileLocked(code)
	if err != nil {
		return err
	}
	return e.execLocked(prog)
}

// compileLocked compiles code against the current bindings (caller holds mu).
func (e *Engine) compileLocked(code string) (*ir.Program, error) {
	src, err := e.wrap(code)
	if err != nil {
		return nil, err
	}
	// Enumerate the builtins permitted under the current sandbox so the
	// compiler knows which RTL/host names are callable (anything else is a
	// compile-time "unknown identifier" error — strong pre-execution checking).
	externals := map[string]bool{}
	for name := range e.prepareBuiltins() {
		externals[strings.ToLower(name)] = true
	}
	presets := map[string]bool{}
	for n := range e.vars {
		presets[n] = true
	}
	return codegen.CompileWithOptions(src, "inline.pas", codegen.Options{
		Externals:           externals,
		PresetGlobals:       presets,
		AutoDeclareLoopVars: true,
	})
}

// execLocked runs an already-compiled program (caller holds mu). It reuses the
// cached builtin table and only sets up fresh per-run state (globals, stack).
func (e *Engine) execLocked(prog *ir.Program) error {
	vm := ir.NewVM(prog)
	e.applyLimits(vm)
	if e.caps.MaxDuration > 0 {
		vm.Deadline = time.Now().Add(e.caps.MaxDuration)
	}
	vm.Builtins = e.prepareBuiltins()
	// Reset per-run host state so nothing leaks across runs (important when an
	// engine is reused for successive tenant requests).
	e.cursor, e.dbErr, e.httpStatus, e.httpHeaders = nil, "", 0, nil
	e.audit = nil
	e.hostErr = ""
	e.seedVars(vm)

	vm.Run()

	e.output = vm.Output.String()
	if vm.RuntimeError != 0 {
		if e.hostErr != "" {
			// A bound Go function returned an error that propagated uncaught.
			return fmt.Errorf("vmpas: %s", e.hostErr)
		}
		return newRuntimeError(vm.RuntimeError)
	}
	e.readbackVars(vm)
	return nil
}

// prepareBuiltins builds (once) and returns the VM-independent builtin table
// for the current bindings and sandbox. Cached until bindings/caps change.
func (e *Engine) prepareBuiltins() map[string]ir.Builtin {
	if e.builtins != nil {
		return e.builtins
	}
	tmp := ir.NewVM(nil)
	e.registerRuntime(tmp)
	aliasLowercase(tmp)
	e.builtins = tmp.Builtins
	return e.builtins
}

// aliasLowercase registers a lowercase alias for every builtin so that
// externals (which codegen emits in lowercase) resolve regardless of the RTL's
// PascalCase registration names.
func aliasLowercase(vm *ir.VM) {
	keys := make([]string, 0, len(vm.Builtins))
	for k := range vm.Builtins {
		keys = append(keys, k)
	}
	for _, k := range keys {
		low := strings.ToLower(k)
		if _, ok := vm.Builtins[low]; !ok {
			vm.Builtins[low] = vm.Builtins[k]
		}
	}
}

// wrap turns a statement fragment into a full program, injecting type and var
// declarations for bound variables. Full programs/units are compiled as-is
// (bindings are still seeded/read back by global name).
func (e *Engine) wrap(code string) (string, error) {
	s := strings.TrimSpace(code)
	// Recognize a full program/unit even behind leading compiler directives or
	// comments (e.g. "{$MODE BPGO} program P; ..."), which must be compiled
	// as-is rather than wrapped as a statement fragment.
	low := strings.ToLower(skipLeadingComments(s))
	if strings.HasPrefix(low, "program ") || strings.HasPrefix(low, "unit ") {
		return s, nil
	}
	typeDecls, varDecls, err := e.declarations()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("program VMPasHost;\n")
	if typeDecls != "" {
		b.WriteString("type\n")
		b.WriteString(typeDecls)
	}
	if varDecls != "" {
		b.WriteString("var\n")
		b.WriteString(varDecls)
	}
	body := s
	if !strings.HasPrefix(low, "begin") {
		if !strings.HasSuffix(body, ";") {
			body += ";"
		}
		body = "begin\n" + body + "\nend."
	} else if !strings.HasSuffix(body, ".") {
		body += "."
	}
	b.WriteString(body)
	return b.String(), nil
}

// skipLeadingComments returns s with any leading whitespace and Pascal comments
// / compiler directives ({...} or (* *)) removed, so the program/unit keyword
// that follows them can be detected.
func skipLeadingComments(s string) string {
	for {
		s = strings.TrimLeft(s, " \t\r\n")
		switch {
		case strings.HasPrefix(s, "{"):
			if i := strings.IndexByte(s, '}'); i >= 0 {
				s = s[i+1:]
				continue
			}
			return s
		case strings.HasPrefix(s, "(*"):
			if i := strings.Index(s, "*)"); i >= 0 {
				s = s[i+2:]
				continue
			}
			return s
		}
		return s
	}
}

// declarations builds Pascal type and var declarations for bound variables in
// a deterministic order.
func (e *Engine) declarations() (string, string, error) {
	names := make([]string, 0, len(e.vars))
	for n := range e.vars {
		names = append(names, n)
	}
	sort.Strings(names)

	var types, vars strings.Builder
	for _, n := range names {
		t := derefType(e.vars[n].Type())
		if t.Kind() == reflect.Struct {
			recName := "T_" + n
			fmt.Fprintf(&types, "  %s = record\n", recName)
			for i := 0; i < t.NumField(); i++ {
				f := t.Field(i)
				name, skip := pascalFieldName(f)
				if skip {
					continue
				}
				pt, ok := pascalScalarType(f.Type)
				if !ok {
					continue
				}
				fmt.Fprintf(&types, "    %s: %s;\n", name, pt)
			}
			types.WriteString("  end;\n")
			fmt.Fprintf(&vars, "  %s: %s;\n", n, recName)
			continue
		}
		if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
			pt, ok := pascalScalarType(t.Elem())
			if !ok {
				return "", "", fmt.Errorf("vmpas: variable %q: array element type %s not supported here "+
					"(declare the array type explicitly in a full program)", n, t.Elem())
			}
			hi := derefValue(e.vars[n]).Len() - 1
			if hi < 0 {
				hi = 0
			}
			fmt.Fprintf(&vars, "  %s: array[0..%d] of %s;\n", n, hi, pt)
			continue
		}
		pt, ok := pascalScalarType(t)
		if !ok {
			return "", "", fmt.Errorf("vmpas: cannot map variable %q of type %s to Pascal", n, t)
		}
		fmt.Fprintf(&vars, "  %s: %s;\n", n, pt)
	}
	return types.String(), vars.String(), nil
}

// registerRuntime installs the RTL builtins permitted by the sandbox, the host
// functions/procedures and the output-capturing Write/WriteLn.
func (e *Engine) registerRuntime(vm *ir.VM) {
	system.Register(vm)
	crt.Register(vm)
	if !e.caps.FileSystem {
		for _, n := range fileBuiltins {
			delete(vm.Builtins, n)
		}
	}
	e.registerHostCaps(vm)
	registerJSON(vm) // JSON is pure computation; no capability required
	// Output capture (matches codegen's default Write formatting).
	vm.Builtins["write"] = func(vm *ir.VM, args []ir.Value) ir.Value {
		for _, a := range args {
			vm.Output.WriteString(formatWrite(a))
		}
		return ir.Value{Kind: ir.VKNil}
	}
	vm.Builtins["writeln"] = func(vm *ir.VM, args []ir.Value) ir.Value {
		for _, a := range args {
			vm.Output.WriteString(formatWrite(a))
		}
		vm.Output.WriteString("\n")
		return ir.Value{Kind: ir.VKNil}
	}
	vm.Builtins["__writefmt"] = func(vm *ir.VM, args []ir.Value) ir.Value {
		if len(args) < 3 {
			if len(args) > 0 {
				return ir.Value{Kind: ir.VKStr, Str: formatWrite(args[0])}
			}
			return ir.Value{Kind: ir.VKStr}
		}
		val, width, dec := args[0], int(args[1].Int), int(args[2].Int)
		var s string
		if val.Kind == ir.VKReal && dec >= 0 {
			s = strconv.FormatFloat(val.Real, 'f', dec, 64)
		} else {
			s = formatWrite(val)
		}
		if width > len(s) {
			s = strings.Repeat(" ", width-len(s)) + s
		}
		return ir.Value{Kind: ir.VKStr, Str: s}
	}
	// Suspend(tag) pauses durable execution (RunDurable/ResumeDurable). It is
	// pure control flow (no capability needed); a non-durable Run that calls it
	// simply halts cleanly with no output past this point.
	vm.Builtins["suspend"] = func(vm *ir.VM, args []ir.Value) ir.Value {
		if len(args) > 0 {
			e.suspendTag = irStr(args[0])
		}
		vm.Suspended = true
		return ir.Value{Kind: ir.VKNil}
	}
	for n, fn := range e.funcs {
		vm.Builtins[n] = e.makeBuiltin(fn)
	}
	for n, fn := range e.procs {
		vm.Builtins[n] = e.makeBuiltin(fn)
	}
	// Wrap gated builtins for the audit log (before aliasLowercase mirrors them,
	// so the lowercase aliases codegen calls point at the wrapped functions).
	e.installAudit(vm)
}

func (e *Engine) seedVars(vm *ir.VM) {
	for n, v := range e.vars {
		vm.SetGlobal(n, goToIR(v))
	}
}

func (e *Engine) readbackVars(vm *ir.VM) {
	for n, v := range e.vars {
		if v.Kind() != reflect.Ptr || v.IsNil() {
			continue // read-only binding
		}
		val, ok := vm.GetGlobal(n)
		if !ok {
			continue
		}
		irToGo(val, v.Elem())
	}
}

// errorInterface is the reflect.Type of the built-in error interface.
var errorInterface = reflect.TypeOf((*error)(nil)).Elem()

// makeBuiltin adapts a Go function into a VM builtin.
//
//   - If the function's last result is an error, a non-nil error is surfaced as a
//     Pascal exception (catchable with try/except); the value result, if any, is
//     ignored on the error path.
//   - When Capabilities.LiveBindings is set, bound Go variables are synced with
//     the VM around the call (Pascal state in before the call, host mutations out
//     after it), so callbacks see and can modify live binding state.
func (e *Engine) makeBuiltin(fn reflect.Value) ir.Builtin {
	t := fn.Type()
	// Number of fixed parameters (variadic tail is not bridged).
	n := t.NumIn()
	if t.IsVariadic() {
		n--
	}
	// Detect a trailing error result.
	errIdx := -1
	if no := t.NumOut(); no > 0 && t.Out(no-1) == errorInterface {
		errIdx = no - 1
	}
	return func(vm *ir.VM, args []ir.Value) ir.Value {
		if e.caps.LiveBindings {
			e.readbackVars(vm) // Pascal -> Go: callback sees current values
		}
		in := make([]reflect.Value, n)
		for i := 0; i < n; i++ {
			if i < len(args) {
				in[i] = irToGoNew(args[i], t.In(i))
			} else {
				in[i] = reflect.Zero(t.In(i))
			}
		}
		out := fn.Call(in)
		if e.caps.LiveBindings {
			e.seedVars(vm) // Go -> Pascal: host mutations are visible (in-place)
		}
		if errIdx >= 0 && !out[errIdx].IsNil() {
			msg := out[errIdx].Interface().(error).Error()
			e.hostErr = msg
			vm.RaiseValue(ir.Value{Kind: ir.VKStr, Str: msg})
			return ir.Value{Kind: ir.VKNil}
		}
		if len(out) == 0 || errIdx == 0 {
			return ir.Value{Kind: ir.VKNil}
		}
		return goToIR(out[0])
	}
}

// --- Go <-> IR value conversion ---

func goToIR(v reflect.Value) ir.Value {
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return ir.Value{Kind: ir.VKNil} // nil Go pointer -> Pascal nil
	}
	v = derefValue(v)
	switch v.Kind() {
	case reflect.Bool:
		return ir.Value{Kind: ir.VKBool, Bool: v.Bool()}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return ir.Value{Kind: ir.VKInt, Int: v.Int()}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return ir.Value{Kind: ir.VKInt, Int: int64(v.Uint())}
	case reflect.Float32, reflect.Float64:
		return ir.Value{Kind: ir.VKReal, Real: v.Float()}
	case reflect.String:
		return ir.Value{Kind: ir.VKStr, Str: v.String()}
	case reflect.Struct:
		t := v.Type()
		rec := make([]ir.RecField, 0, t.NumField())
		for i := 0; i < t.NumField(); i++ {
			name, skip := pascalFieldName(t.Field(i))
			if skip {
				continue
			}
			fv := goToIR(v.Field(i))
			rec = append(rec, ir.RecField{Name: strings.ToLower(name), Cell: &fv})
		}
		return ir.Value{Kind: ir.VKRecord, Rec: rec}
	case reflect.Slice, reflect.Array:
		n := v.Len()
		arr := make([]ir.Value, n)
		for i := 0; i < n; i++ {
			arr[i] = goToIR(v.Index(i))
		}
		return ir.Value{Kind: ir.VKArray, Array: arr}
	}
	return ir.Value{Kind: ir.VKNil}
}

// irToGoNew builds a new Go value of type t from an IR value.
func irToGoNew(val ir.Value, t reflect.Type) reflect.Value {
	p := reflect.New(t)
	irToGo(val, p.Elem())
	return p.Elem()
}

// irToGo writes an IR value into a settable Go value.
func irToGo(val ir.Value, dst reflect.Value) {
	if !dst.CanSet() {
		return
	}
	switch dst.Kind() {
	case reflect.Ptr:
		// nil IR value clears the Go pointer; otherwise allocate (if needed) and
		// write through, so nested *Struct fields and pointer args/returns round-trip.
		if val.Kind == ir.VKNil || (val.Kind == ir.VKPtr && val.Cell == nil) {
			dst.Set(reflect.Zero(dst.Type()))
			return
		}
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		irToGo(val, dst.Elem())
	case reflect.Bool:
		dst.SetBool(irBool(val))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		dst.SetInt(irInt(val))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		dst.SetUint(uint64(irInt(val)))
	case reflect.Float32, reflect.Float64:
		dst.SetFloat(irFloat(val))
	case reflect.String:
		dst.SetString(irStr(val))
	case reflect.Struct:
		if val.Kind != ir.VKRecord {
			return
		}
		t := dst.Type()
		for i := 0; i < t.NumField(); i++ {
			name, skip := pascalFieldName(t.Field(i))
			if skip {
				continue
			}
			if cell := val.Field(strings.ToLower(name)); cell != nil {
				irToGo(*cell, dst.Field(i))
			}
		}
	case reflect.Slice:
		if val.Kind != ir.VKArray {
			return
		}
		n := len(val.Array)
		s := reflect.MakeSlice(dst.Type(), n, n)
		for i := 0; i < n; i++ {
			irToGo(val.Array[i], s.Index(i))
		}
		dst.Set(s)
	case reflect.Array:
		if val.Kind != ir.VKArray {
			return
		}
		for i := 0; i < dst.Len() && i < len(val.Array); i++ {
			irToGo(val.Array[i], dst.Index(i))
		}
	}
}

func irBool(v ir.Value) bool {
	switch v.Kind {
	case ir.VKBool:
		return v.Bool
	case ir.VKInt:
		return v.Int != 0
	}
	return false
}

func irInt(v ir.Value) int64 {
	switch v.Kind {
	case ir.VKInt:
		return v.Int
	case ir.VKReal:
		return int64(v.Real)
	case ir.VKChar:
		return int64(v.Ch)
	case ir.VKBool:
		if v.Bool {
			return 1
		}
	}
	return 0
}

func irFloat(v ir.Value) float64 {
	switch v.Kind {
	case ir.VKReal:
		return v.Real
	case ir.VKInt:
		return float64(v.Int)
	}
	return 0
}

func irStr(v ir.Value) string {
	switch v.Kind {
	case ir.VKStr:
		return v.Str
	case ir.VKChar:
		return string([]byte{v.Ch})
	}
	return v.String()
}

// --- reflection helpers ---

func derefValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return reflect.Zero(v.Type().Elem())
		}
		v = v.Elem()
	}
	return v
}

func derefType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// pascalFieldName returns the Pascal record-field name for a Go struct field,
// honoring a `vmpas:"name"` (preferred) or `json:"name"` tag. It reports skip
// when the field is unexported or tagged "-".
func pascalFieldName(f reflect.StructField) (name string, skip bool) {
	if f.PkgPath != "" { // unexported field
		return "", true
	}
	tag := f.Tag.Get("vmpas")
	if tag == "" {
		tag = f.Tag.Get("json")
	}
	if tag != "" {
		n := tag
		if i := strings.IndexByte(n, ','); i >= 0 { // drop ",omitempty" and friends
			n = n[:i]
		}
		if n == "-" {
			return "", true
		}
		if n != "" {
			return n, false
		}
	}
	return f.Name, false
}

// pascalScalarType maps a Go scalar type to a Pascal type name.
func pascalScalarType(t reflect.Type) (string, bool) {
	switch t.Kind() {
	case reflect.Bool:
		return "Boolean", true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "LongInt", true
	case reflect.Float32, reflect.Float64:
		return "Real", true
	case reflect.String:
		return "String", true
	}
	return "", false
}

// formatWrite renders a value the way TP7's Write does by default.
func formatWrite(v ir.Value) string {
	switch v.Kind {
	case ir.VKInt:
		return fmt.Sprintf("%d", v.Int)
	case ir.VKBool:
		if v.Bool {
			return "TRUE"
		}
		return "FALSE"
	case ir.VKChar:
		return string([]byte{v.Ch})
	case ir.VKStr:
		return v.Str
	case ir.VKReal:
		return fmt.Sprintf(" %.10E", v.Real)
	case ir.VKNil:
		return ""
	}
	return v.String()
}
