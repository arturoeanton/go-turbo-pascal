// Package ir defines BPGo's intermediate representation and the
// interpreter ("virtual machine") that executes IR programs. The IR is
// the common target for both the Pascal source and the assembly-style
// MZ backend. The VM is intentionally small but faithful to TP7
// semantics: it supports the same calling conventions, short-circuit
// boolean evaluation, range/overflow checking and unit initialization
// order used by the canonical compiler.
package ir

import (
	"fmt"
	"sort"
	"strings"
)

type Op uint8

const (
	OPNoop Op = iota
	OPPushInt
	OPPushStr
	OPPushReal
	OPPushNil
	OPPushBool
	OPLoadGlobal
	OPStoreGlobal
	OPLoadLocal
	OPStoreLocal
	OPLoadField
	OPStoreField
	OPIndex
	OPIndexStore
	OPDeref
	OPAddr
	OPCall
	OPCallBuiltin
	OPEnter
	OPLeave
	OPReturn
	OPJump
	OPJumpIfFalse
	OPJumpIfTrue
	OPBinary
	OPUnary
	OPCompare
	OPIn
	OPNew
	OPDispose
	OPMkSet
	OPMkRange
	OPAndSC
	OPOrSC
	OPHalt
	OPError
	OPInc
	OPDec
	OPCaseTest
	OPDup
	OPPop
	OPDup2
	OPMkString
	OPMkArray
	OPFillChar
	OPMove
	OPBlockCopy
	// Reference / lvalue model (added for the real codegen). A reference
	// is a Value of kind VKPtr whose Cell points at an addressable storage
	// location (a global cell, a frame slot, a heap cell, an array element
	// or a record field). The codegen computes a reference, then reads it
	// with OPLoadRef or writes through it with OPStoreRef.
	OPAddrGlobal // push a reference to global cell named S
	OPAddrLocal  // push a reference to frame slot A
	OPLoadRef    // pop reference, push the value it points at
	OPStoreRef   // pop value, pop reference, store value through the reference
	OPSetResult  // pop value into the current frame's function result
	OPLoadResult // push the current frame's function result
	OPPushChar   // push a Char value with ordinal A
	// Aggregates (records, arrays) and heap pointers.
	OPFieldAddr    // pop record reference, push reference to field named S
	OPElemAddr     // pop index, pop array reference, push reference to element
	OPHeapAlloc    // pop value, allocate a heap cell holding it, push a reference
	OPPushZero     // push a deep copy of the current function's template A
	OPCallMethod   // dynamic dispatch: A args (Self first), method S, via Vtables
	OPStrChar      // pop index, pop string, push the 1-based character
	OPStrCharStore // pop char, pop index, pop string-reference; set 1-based char
	OPSetLength    // pop zero-template, new length, array reference; resize the array
	// Exceptions.
	OPEnterTry // push an exception handler (A=handler PC, B=1 if finally)
	OPPopTry   // pop the top exception handler (try body completed normally)
	OPRaise    // raise an exception (A=1: value on stack; A=0: re-raise)
	OPReraise  // continue propagating the current exception (after a finally)
	// Procedural values / closures.
	OPMakeClosure // pop A capture refs, push a VKFunc value (S=IR function name)
	OPCallValue   // pop A args then a VKFunc; call it, binding captures + args
	OPAddrResult  // push a reference to the current frame's result cell
	// Algebraic data types / pattern matching.
	OPMkTagged // pop A payloads, push a tagged record {__tag:S, __0..__(A-1)}
	OPRecover  // push the active panic value and clear it (or nil if none)
	// Concurrency (cooperative scheduler): spawn and channels.
	OPSpawn     // pop a closure value, start it as a new fiber
	OPMakeChan  // pop buffer size (A=1) or none (A=0), push a new channel
	OPChanSend  // pop value, pop channel; send (may park the fiber)
	OPChanRecv  // pop channel; push received value (may park the fiber)
	OPChanClose // pop channel; close it
)

func (o Op) String() string {
	switch o {
	case OPLoadGlobal:
		return "loadg"
	case OPStoreGlobal:
		return "storeg"
	case OPLoadLocal:
		return "loadl"
	case OPStoreLocal:
		return "storel"
	case OPBinary:
		return "bin"
	case OPCompare:
		return "cmp"
	case OPJump:
		return "jmp"
	case OPJumpIfFalse:
		return "jf"
	case OPJumpIfTrue:
		return "jt"
	case OPCall:
		return "call"
	case OPCallBuiltin:
		return "callb"
	case OPNew:
		return "new"
	case OPDeref:
		return "deref"
	case OPIndex:
		return "index"
	}
	return fmt.Sprintf("op%d", o)
}

// Function is a compiled IR routine.
type Function struct {
	Name      string
	Params    []string
	Locals    []string
	Code      []Instr
	SourceMap map[int]SourceRef
	Entry     int
	// Templates are zero-value prototypes for aggregate initialization,
	// pushed by OPPushZero (used to initialize locals/globals and to allocate
	// New() targets of the right shape).
	Templates []Value
}

// AddTemplate registers a zero-value template and returns its index.
func (f *Function) AddTemplate(v Value) int {
	f.Templates = append(f.Templates, v)
	return len(f.Templates) - 1
}

type SourceRef struct {
	File string
	Line int
	Col  int
}

type Instr struct {
	Op    Op
	A     int64
	B     int64
	S     string
	R     float64
	Func  *Function
	Label string
	Str   string
}

// Module is a compiled unit or program.
type Module struct {
	Name      string
	Funcs     map[string]*Function
	Globals   []Global
	Init      []string // function names to call at module init
	SourceMap map[string]map[int]SourceRef
}

type Global struct {
	Name string
	Size int64
	Type string
}

type Program struct {
	Modules   []*Module
	Main      string
	Entry     string
	Init      []string
	SourceMap map[string]map[int]SourceRef
	// Vtables maps a lowercased object type name to its method table
	// (lowercased method name -> IR function name). Tables are flattened to
	// include inherited methods, so OPCallMethod can resolve dynamically.
	Vtables map[string]map[string]string
	// Concurrent is set when the program uses spawn/channels. When false the VM
	// runs main on a fast single-fiber path with no scheduler overhead.
	Concurrent bool
}

func NewFunction(name string) *Function {
	return &Function{Name: name, Code: []Instr{{Op: OPNoop}}, SourceMap: map[int]SourceRef{}}
}

func (f *Function) Emit(i Instr) int {
	f.Code = append(f.Code, i)
	return len(f.Code) - 1
}

func (f *Function) Patch(jmpIdx int, target int) {
	f.Code[jmpIdx].A = int64(target)
}

func (p *Program) SortedFuncs() []string {
	out := []string{}
	for _, m := range p.Modules {
		for n := range m.Funcs {
			out = append(out, m.Name+"."+n)
		}
	}
	sort.Strings(out)
	return out
}

// DumpFunc returns a textual form of a function for debugging and
// golden tests.
func DumpFunc(f *Function) string {
	var b strings.Builder
	b.WriteString("func " + f.Name + ":\n")
	for i, ins := range f.Code {
		fmt.Fprintf(&b, "  %4d  %-10s a=%d b=%d s=%q r=%g func=%v\n", i, ins.Op, ins.A, ins.B, ins.S, ins.R, ins.Func != nil)
	}
	return b.String()
}

// StringOfProgram returns a deterministic textual form of the whole
// program (modules + functions) used by golden tests.
func StringOfProgram(p *Program) string {
	var b strings.Builder
	for _, m := range p.Modules {
		fmt.Fprintf(&b, "module %s\n", m.Name)
		gs := make([]string, 0, len(m.Globals))
		for _, g := range m.Globals {
			gs = append(gs, g.Name)
		}
		sort.Strings(gs)
		for _, n := range gs {
			fmt.Fprintf(&b, "  global %s\n", n)
		}
		fns := make([]string, 0, len(m.Funcs))
		for n := range m.Funcs {
			fns = append(fns, n)
		}
		sort.Strings(fns)
		for _, n := range fns {
			b.WriteString(DumpFunc(m.Funcs[n]))
		}
	}
	return b.String()
}
