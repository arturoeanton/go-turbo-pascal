package ir

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"time"
)

// Value is a runtime value manipulated by the IR VM. The VM uses a
// tagged-union style with a small set of types that match the TP7 /
// BP7 categories: integers (16/32 bit), reals (TP real, IEEE single /
// double / extended), booleans, characters, strings, sets, arrays,
// records, pointers, files and the special nil pointer.
type Value struct {
	Kind  ValKind
	Int   int64
	Real  float64
	Bool  bool
	Ch    byte
	Str   string
	Set   *[32]byte // set bitmap, allocated lazily (nil = empty set)
	Array []Value
	Rec   map[string]*Value // record fields, each an addressable cell
	Ptr   int64             // address in the heap (negative = nil)
	File  *File
	Ref   interface{} // generic slot
	// Cell points at an addressable storage location when this Value is
	// used as a reference (kind VKPtr): a global cell, a frame slot, a heap
	// cell, an array element or a record field. A VKPtr value with a nil
	// Cell is the Pascal nil pointer.
	Cell *Value
}

type ValKind int

const (
	VKInt ValKind = iota
	VKReal
	VKBool
	VKChar
	VKStr
	VKSet
	VKArray
	VKRecord
	VKPtr
	VKFile
	VKNil
	// VKFunc is a callable value (procedural type / closure): Str holds the
	// IR function name and Array holds the captured reference cells (each a
	// VKPtr), bound to the function's leading var-parameters on call.
	VKFunc
)

func (v Value) String() string {
	switch v.Kind {
	case VKInt:
		return fmt.Sprintf("%d", v.Int)
	case VKReal:
		if v.Real == math.Floor(v.Real) {
			return fmt.Sprintf("%g.0", v.Real)
		}
		return fmt.Sprintf("%g", v.Real)
	case VKBool:
		if v.Bool {
			return "True"
		}
		return "False"
	case VKChar:
		return fmt.Sprintf("#%d", v.Ch)
	case VKStr:
		return v.Str
	case VKSet:
		var b strings.Builder
		b.WriteString("[")
		bits := setBits(v)
		for i, x := range bits {
			if x != 0 {
				if b.Len() > 1 {
					b.WriteString(",")
				}
				fmt.Fprintf(&b, "%d", i)
			}
		}
		b.WriteString("]")
		return b.String()
	case VKArray:
		var b strings.Builder
		b.WriteString("[")
		for i, x := range v.Array {
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString(x.String())
		}
		b.WriteString("]")
		return b.String()
	case VKRecord:
		var b strings.Builder
		b.WriteString("(")
		keys := make([]string, 0, len(v.Rec))
		for k := range v.Rec {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, k := range keys {
			if i > 0 {
				b.WriteString(",")
			}
			fmt.Fprintf(&b, "%s:%s", k, v.Rec[k].String())
		}
		b.WriteString(")")
		return b.String()
	case VKPtr:
		return fmt.Sprintf("^%d", v.Ptr)
	case VKFile:
		if v.File == nil {
			return "File(nil)"
		}
		return "File(" + v.File.Name + ")"
	case VKNil:
		return "nil"
	case VKFunc:
		if v.Str == "" {
			return "nil"
		}
		return "func(" + v.Str + ")"
	}
	return "?"
}

// File is a minimal file descriptor.
type File struct {
	Name   string
	Mode   int
	Buffer []byte
	Pos    int
	IsText bool
	Closed bool
}

type Frame struct {
	Func   *Function
	Locals []Value // frame slots: parameters first, then locals
	PC     int
	Caller *Frame
	Static *Frame
	Result Value // function result (assigned via OPSetResult)
}

type VM struct {
	Program      *Program
	Globals      map[string]*Value
	Heap         []Value
	Stack        []Value
	CallStack    []*Frame
	Output       *Output
	Input        *Input
	ExitCode     int
	Halted       bool
	RuntimeError int
	MaxSteps     int
	Steps        int
	MaxHeap      int       // max heap allocations (0 = unlimited)
	heapAllocs   int       // heap allocations so far
	Deadline     time.Time // wall-clock deadline (zero = none)
	RandomState  uint32
	Trace        bool
	Builtins     map[string]Builtin
	Args         []string
	framePool    []*Frame     // reusable call frames (reduces per-call allocations)
	handlers     []tryHandler // active exception handlers (try/except/finally)
	excValue     Value        // the value of the exception being propagated
	excActive    bool         // whether an exception is currently propagating
}

// tryHandler records a try block's recovery point.
type tryHandler struct {
	frame      *Frame
	pc         int
	stackDepth int
	callDepth  int
	isFinally  bool
}

// doUnwind transfers control to the nearest exception handler, popping any call
// frames above it and restoring the operand stack. If there is no handler, the
// program halts with an unhandled-exception runtime error.
func (vm *VM) doUnwind() {
	if len(vm.handlers) == 0 {
		vm.RuntimeError = 217 // unhandled exception
		vm.Halted = true
		return
	}
	h := vm.handlers[len(vm.handlers)-1]
	vm.handlers = vm.handlers[:len(vm.handlers)-1]
	for len(vm.CallStack) > h.callDepth {
		f := vm.CallStack[len(vm.CallStack)-1]
		vm.CallStack = vm.CallStack[:len(vm.CallStack)-1]
		vm.putFrame(f)
	}
	if h.stackDepth <= len(vm.Stack) {
		vm.Stack = vm.Stack[:h.stackDepth]
	}
	h.frame.PC = h.pc
	if !h.isFinally {
		vm.excActive = false // an except block handles the exception
	}
}

// getFrame returns a frame for fn, reusing one from the pool when possible.
func (vm *VM) getFrame(fn *Function, caller *Frame) *Frame {
	n := len(fn.Params) + len(fn.Locals)
	if k := len(vm.framePool); k > 0 {
		f := vm.framePool[k-1]
		vm.framePool = vm.framePool[:k-1]
		f.Func = fn
		f.PC = 0
		f.Caller = caller
		f.Static = nil
		f.Result = Value{}
		if cap(f.Locals) >= n {
			f.Locals = f.Locals[:n]
			for i := range f.Locals {
				f.Locals[i] = Value{}
			}
		} else {
			f.Locals = make([]Value, n)
		}
		return f
	}
	return &Frame{Func: fn, Locals: make([]Value, n), Caller: caller}
}

// putFrame returns a frame to the pool after it has returned. Safe because a
// frame's locals are no longer referenced once it returns (var-parameter
// references never outlive the call).
func (vm *VM) putFrame(f *Frame) {
	vm.framePool = append(vm.framePool, f)
}

type Builtin func(vm *VM, args []Value) Value

type Output struct {
	Buf strings.Builder
	// W, when set, receives output live (in addition to Buf) so interactive
	// programs show prompts before they read input.
	W io.Writer
}

type Input struct {
	Lines []string
	Pos   int
	// Reader, when set, is read line-by-line on demand (interactive input);
	// otherwise Lines is consumed.
	Reader *bufio.Reader
}

func (o *Output) WriteString(s string) {
	o.Buf.WriteString(s)
	if o.W != nil {
		io.WriteString(o.W, s)
	}
}
func (o *Output) String() string { return o.Buf.String() }

func (in *Input) ReadLine() string {
	if in.Reader != nil {
		line, err := in.Reader.ReadString('\n')
		if err != nil && line == "" {
			return ""
		}
		return strings.TrimRight(line, "\r\n")
	}
	if in.Pos >= len(in.Lines) {
		return ""
	}
	s := in.Lines[in.Pos]
	in.Pos++
	return s
}

func NewVM(p *Program) *VM {
	return &VM{
		Program:     p,
		Globals:     map[string]*Value{},
		Heap:        []Value{},
		Output:      &Output{},
		Input:       &Input{},
		Builtins:    map[string]Builtin{},
		MaxSteps:    10000000,
		RandomState: 1,
	}
}

// SetOutput configures a writer-like output buffer.
func (vm *VM) SetOutputText(s string) {
	vm.Output = &Output{}
	vm.Output.Buf.WriteString(s)
}

// Step executes one IR instruction. It returns false if the program has
// halted.
func (vm *VM) Step(frame *Frame) bool {
	vm.Steps++
	if vm.Steps > vm.MaxSteps {
		vm.RuntimeError = 200
		vm.Halted = true
		return false
	}
	// Wall-clock deadline: checked every 4096 steps to keep time.Now() out
	// of the per-instruction hot path.
	if !vm.Deadline.IsZero() && vm.Steps&0xFFF == 0 && time.Now().After(vm.Deadline) {
		vm.RuntimeError = 200
		vm.Halted = true
		return false
	}
	if frame.PC >= len(frame.Func.Code) {
		vm.RuntimeError = 202
		vm.Halted = true
		return false
	}
	ins := frame.Func.Code[frame.PC]
	frame.PC++
	switch ins.Op {
	case OPNoop:
		return true
	case OPEnter:
		// no-op
		return true
	case OPLeave:
		return true
	case OPHalt:
		vm.Halted = true
		return false
	case OPError:
		vm.RuntimeError = int(ins.A)
		vm.Halted = true
		return false
	case OPPushInt:
		vm.Stack = append(vm.Stack, Value{Kind: VKInt, Int: ins.A})
		return true
	case OPPushReal:
		vm.Stack = append(vm.Stack, Value{Kind: VKReal, Real: ins.R})
		return true
	case OPPushStr:
		vm.Stack = append(vm.Stack, Value{Kind: VKStr, Str: ins.S})
		return true
	case OPPushBool:
		vm.Stack = append(vm.Stack, Value{Kind: VKBool, Bool: ins.A != 0})
		return true
	case OPPushChar:
		vm.Stack = append(vm.Stack, Value{Kind: VKChar, Ch: byte(ins.A)})
		return true
	case OPPushNil:
		vm.Stack = append(vm.Stack, Value{Kind: VKNil})
		return true
	case OPLoadGlobal:
		vm.Stack = append(vm.Stack, *vm.globalCell(ins.S))
		return true
	case OPStoreGlobal:
		v := vm.pop()
		*vm.globalCell(ins.S) = deepCopy(v)
		return true
	case OPAddrGlobal:
		vm.Stack = append(vm.Stack, Value{Kind: VKPtr, Cell: vm.globalCell(ins.S)})
		return true
	case OPAddrLocal:
		idx := int(ins.A)
		for idx >= len(frame.Locals) {
			frame.Locals = append(frame.Locals, Value{})
		}
		vm.Stack = append(vm.Stack, Value{Kind: VKPtr, Cell: &frame.Locals[idx]})
		return true
	case OPLoadRef:
		r := vm.pop()
		if r.Kind != VKPtr || r.Cell == nil {
			vm.RuntimeError = 204 // nil pointer / invalid reference
			vm.Halted = true
			return false
		}
		vm.Stack = append(vm.Stack, *r.Cell)
		return true
	case OPStoreRef:
		v := vm.pop()
		r := vm.pop()
		if r.Kind != VKPtr || r.Cell == nil {
			vm.RuntimeError = 204
			vm.Halted = true
			return false
		}
		*r.Cell = deepCopy(v)
		return true
	case OPSetResult:
		frame.Result = vm.pop()
		return true
	case OPLoadResult:
		vm.Stack = append(vm.Stack, frame.Result)
		return true
	case OPPushZero:
		idx := int(ins.A)
		if idx < 0 || idx >= len(frame.Func.Templates) {
			vm.RuntimeError = 204
			vm.Halted = true
			return false
		}
		vm.Stack = append(vm.Stack, deepCopy(frame.Func.Templates[idx]))
		return true
	case OPFieldAddr:
		r := vm.pop()
		if r.Kind != VKPtr || r.Cell == nil || r.Cell.Kind != VKRecord {
			vm.RuntimeError = 204
			vm.Halted = true
			return false
		}
		fc, ok := r.Cell.Rec[ins.S]
		if !ok {
			fc = &Value{}
			r.Cell.Rec[ins.S] = fc
		}
		vm.Stack = append(vm.Stack, Value{Kind: VKPtr, Cell: fc})
		return true
	case OPElemAddr:
		idx := vm.pop()
		r := vm.pop()
		if r.Kind != VKPtr || r.Cell == nil || r.Cell.Kind != VKArray {
			vm.RuntimeError = 204
			vm.Halted = true
			return false
		}
		i := int(toInt(idx))
		if i < 0 || i >= len(r.Cell.Array) {
			vm.RuntimeError = 201 // range check error
			vm.Halted = true
			return false
		}
		vm.Stack = append(vm.Stack, Value{Kind: VKPtr, Cell: &r.Cell.Array[i]})
		return true
	case OPHeapAlloc:
		if !vm.chargeHeap() {
			return false
		}
		v := vm.pop()
		cell := new(Value)
		*cell = v
		vm.Stack = append(vm.Stack, Value{Kind: VKPtr, Cell: cell})
		return true
	case OPStrChar:
		idx := vm.pop()
		s := vm.pop()
		i := int(toInt(idx))
		if s.Kind == VKStr && i >= 1 && i <= len(s.Str) {
			vm.Stack = append(vm.Stack, Value{Kind: VKChar, Ch: s.Str[i-1]})
		} else {
			vm.Stack = append(vm.Stack, Value{Kind: VKChar, Ch: 0})
		}
		return true
	case OPStrCharStore:
		val := vm.pop()
		idx := vm.pop()
		r := vm.pop()
		if r.Kind == VKPtr && r.Cell != nil && r.Cell.Kind == VKStr {
			i := int(toInt(idx))
			if i >= 1 && i <= len(r.Cell.Str) {
				var ch byte
				switch val.Kind {
				case VKChar:
					ch = val.Ch
				case VKStr:
					if len(val.Str) > 0 {
						ch = val.Str[0]
					}
				default:
					ch = byte(toInt(val))
				}
				b := []byte(r.Cell.Str)
				b[i-1] = ch
				r.Cell.Str = string(b)
			}
		}
		return true
	case OPEnterTry:
		vm.handlers = append(vm.handlers, tryHandler{
			frame: frame, pc: int(ins.A), stackDepth: len(vm.Stack),
			callDepth: len(vm.CallStack), isFinally: ins.B != 0,
		})
		return true
	case OPPopTry:
		if len(vm.handlers) > 0 {
			vm.handlers = vm.handlers[:len(vm.handlers)-1]
		}
		return true
	case OPRaise:
		if ins.A != 0 {
			vm.excValue = vm.pop()
		}
		vm.excActive = true
		vm.doUnwind()
		return !vm.Halted
	case OPReraise:
		vm.excActive = true
		vm.doUnwind()
		return !vm.Halted
	case OPSetLength:
		tmpl := vm.pop()
		n := int(toInt(vm.pop()))
		r := vm.pop()
		if r.Kind == VKPtr && r.Cell != nil && r.Cell.Kind == VKArray {
			if n < 0 {
				n = 0
			}
			cur := r.Cell.Array
			if n <= len(cur) {
				r.Cell.Array = cur[:n]
			} else {
				na := make([]Value, n)
				copy(na, cur)
				for i := len(cur); i < n; i++ {
					na[i] = deepCopy(tmpl)
				}
				r.Cell.Array = na
			}
		}
		return true
	case OPCallMethod:
		argc := int(ins.A)
		if argc < 1 || len(vm.Stack) < argc {
			vm.RuntimeError = 204
			vm.Halted = true
			return false
		}
		base := len(vm.Stack) - argc
		self := vm.Stack[base] // Self is the first argument (a reference)
		typ := selfTypeName(self)
		fnName := lookupMethod(vm.Program, typ, ins.S)
		if fnName == "" {
			vm.RuntimeError = 210 // abstract/undefined method call
			vm.Halted = true
			return false
		}
		fn, ok := findFunc(vm.Program, fnName)
		if !ok {
			vm.RuntimeError = 3
			vm.Halted = true
			return false
		}
		newFrame := vm.getFrame(fn, frame)
		for i := 0; i < argc && i < len(newFrame.Locals); i++ {
			newFrame.Locals[i] = deepCopy(vm.Stack[base+i])
		}
		vm.Stack = vm.Stack[:base]
		vm.CallStack = append(vm.CallStack, newFrame)
		return true
	case OPLoadLocal:
		idx := int(ins.A)
		if idx < 0 || idx >= len(frame.Locals) {
			vm.RuntimeError = 204
			vm.Halted = true
			return false
		}
		vm.Stack = append(vm.Stack, frame.Locals[idx])
		return true
	case OPStoreLocal:
		idx := int(ins.A)
		for idx >= len(frame.Locals) {
			frame.Locals = append(frame.Locals, Value{})
		}
		frame.Locals[idx] = vm.pop()
		return true
	case OPJump:
		frame.PC = int(ins.A)
		return true
	case OPJumpIfFalse:
		v := vm.pop()
		if !truthy(v) {
			frame.PC = int(ins.A)
		}
		return true
	case OPJumpIfTrue:
		v := vm.pop()
		if truthy(v) {
			frame.PC = int(ins.A)
		}
		return true
	case OPReturn:
		// The function result lives in frame.Result (set via OPSetResult).
		// Procedures leave it as the zero value. Every call pushes exactly
		// one result onto the shared operand stack; statement-level calls
		// discard it with OPPop.
		rv := frame.Result
		if frame.Caller == nil {
			vm.Halted = true
			vm.ExitCode = int(rv.Int)
			return false
		}
		// Drop any exception handlers left by this frame (e.g. an `exit` from
		// inside a try) so they don't catch later, unrelated exceptions.
		for len(vm.handlers) > 0 && vm.handlers[len(vm.handlers)-1].frame == frame {
			vm.handlers = vm.handlers[:len(vm.handlers)-1]
		}
		vm.CallStack = vm.CallStack[:len(vm.CallStack)-1]
		vm.Stack = append(vm.Stack, rv)
		vm.putFrame(frame)
		return true
	case OPBinary:
		b := vm.pop()
		a := vm.pop()
		r := binaryOp(ins.S, a, b)
		vm.Stack = append(vm.Stack, r)
		return true
	case OPCompare:
		b := vm.pop()
		a := vm.pop()
		r := compareOp(ins.S, a, b)
		vm.Stack = append(vm.Stack, r)
		return true
	case OPUnary:
		a := vm.pop()
		r := unaryOp(ins.S, a)
		vm.Stack = append(vm.Stack, r)
		return true
	case OPCallBuiltin:
		args := []Value{}
		for i := int(ins.A) - 1; i >= 0; i-- {
			args = append([]Value{vm.pop()}, args...)
		}
		fn, ok := vm.Builtins[ins.S]
		if !ok {
			vm.RuntimeError = 3
			vm.Halted = true
			return false
		}
		r := fn(vm, args)
		vm.Stack = append(vm.Stack, r)
		return true
	case OPCall:
		// The call target is resolved by name once, then cached in the
		// instruction so repeated calls (notably recursion) skip the map
		// lookup. The program is immutable across runs, so the cache is
		// valid for the lifetime of the compiled program.
		fn := ins.Func
		if fn == nil {
			var ok bool
			fn, ok = findFunc(vm.Program, ins.S)
			if !ok {
				vm.RuntimeError = 3
				vm.Halted = true
				return false
			}
			frame.Func.Code[frame.PC-1].Func = fn
		}
		// Arguments sit on the operand stack in order (arg0..argN-1); bind them
		// positionally to the parameter slots without allocating an args slice.
		argc := int(ins.A)
		base := len(vm.Stack) - argc
		if base < 0 {
			base = 0
		}
		newFrame := vm.getFrame(fn, frame)
		for i := 0; base+i < len(vm.Stack) && i < len(newFrame.Locals); i++ {
			newFrame.Locals[i] = deepCopy(vm.Stack[base+i])
		}
		vm.Stack = vm.Stack[:base]
		vm.CallStack = append(vm.CallStack, newFrame)
		return true
	case OPMakeClosure:
		// Build a callable value: pop A capture references (already VKPtr),
		// keep them in order, attach the target function name.
		ncap := int(ins.A)
		base := len(vm.Stack) - ncap
		if base < 0 {
			base = 0
		}
		var caps []Value
		if ncap > 0 {
			caps = make([]Value, ncap)
			copy(caps, vm.Stack[base:])
		}
		vm.Stack = vm.Stack[:base]
		vm.Stack = append(vm.Stack, Value{Kind: VKFunc, Str: ins.S, Array: caps})
		return true
	case OPCallValue:
		// Stack: [.. closure, arg0..argN-1]. Bind captures to the leading
		// var-parameter slots, then the actual arguments.
		argc := int(ins.A)
		base := len(vm.Stack) - argc
		if base < 1 {
			vm.RuntimeError = 204
			vm.Halted = true
			return false
		}
		clo := vm.Stack[base-1]
		if clo.Kind != VKFunc {
			vm.RuntimeError = 204 // calling a nil/invalid procedural value
			vm.Halted = true
			return false
		}
		fn, ok := findFunc(vm.Program, clo.Str)
		if !ok {
			vm.RuntimeError = 3
			vm.Halted = true
			return false
		}
		newFrame := vm.getFrame(fn, frame)
		ncap := len(clo.Array)
		for i := 0; i < ncap && i < len(newFrame.Locals); i++ {
			newFrame.Locals[i] = clo.Array[i] // captured reference (VKPtr)
		}
		for i := 0; i < argc && ncap+i < len(newFrame.Locals); i++ {
			newFrame.Locals[ncap+i] = deepCopy(vm.Stack[base+i])
		}
		vm.Stack = vm.Stack[:base-1]
		vm.CallStack = append(vm.CallStack, newFrame)
		return true
	case OPDup:
		vm.Stack = append(vm.Stack, vm.Stack[len(vm.Stack)-1])
		return true
	case OPDup2:
		if len(vm.Stack) < 2 {
			vm.RuntimeError = 204
			vm.Halted = true
			return false
		}
		vm.Stack = append(vm.Stack, vm.Stack[len(vm.Stack)-2], vm.Stack[len(vm.Stack)-1])
		return true
	case OPPop:
		vm.pop()
		return true
	case OPIn:
		b := vm.pop()
		a := vm.pop()
		ok := inSet(a, b)
		vm.Stack = append(vm.Stack, Value{Kind: VKBool, Bool: ok})
		return true
	case OPMkSet:
		// Build a set from elements on the stack.
		count := int(ins.A)
		set := Value{Kind: VKSet, Set: new([32]byte)}
		for i := 0; i < count; i++ {
			v := vm.pop()
			idx := int(toInt(v))
			if idx >= 0 && idx < 256 {
				set.Set[idx/8] |= 1 << (idx % 8)
			}
		}
		vm.Stack = append(vm.Stack, set)
		return true
	case OPMkString:
		// Build a Pascal string from `count` chars.
		count := int(ins.A)
		chars := make([]byte, count)
		for i := count - 1; i >= 0; i-- {
			v := vm.pop()
			chars[i] = byte(toInt(v))
		}
		vm.Stack = append(vm.Stack, Value{Kind: VKStr, Str: string(chars)})
		return true
	case OPMkArray:
		count := int(ins.A)
		arr := make([]Value, count)
		for i := count - 1; i >= 0; i-- {
			arr[i] = vm.pop()
		}
		vm.Stack = append(vm.Stack, Value{Kind: VKArray, Array: arr})
		return true
	case OPIndex:
		idx := vm.pop()
		arr := vm.pop()
		i := int(toInt(idx))
		if arr.Kind != VKArray {
			vm.RuntimeError = 201
			vm.Halted = true
			return false
		}
		if i < 0 || i >= len(arr.Array) {
			vm.RuntimeError = 201
			vm.Halted = true
			return false
		}
		vm.Stack = append(vm.Stack, arr.Array[i])
		return true
	case OPIndexStore:
		val := vm.pop()
		idx := vm.pop()
		arr := vm.pop()
		i := int(toInt(idx))
		if arr.Kind != VKArray {
			vm.RuntimeError = 201
			vm.Halted = true
			return false
		}
		if i < 0 || i >= len(arr.Array) {
			vm.RuntimeError = 201
			vm.Halted = true
			return false
		}
		arr.Array[i] = val
		return true
	case OPNew:
		if !vm.chargeHeap() {
			return false
		}
		vm.Stack = append(vm.Stack, Value{Kind: VKPtr, Ptr: int64(len(vm.Heap))})
		vm.Heap = append(vm.Heap, Value{Kind: VKRecord, Rec: map[string]*Value{}})
		return true
	case OPDeref:
		p := vm.pop()
		if p.Kind == VKNil || p.Ptr < 0 {
			vm.RuntimeError = 204
			vm.Halted = true
			return false
		}
		vm.Stack = append(vm.Stack, vm.Heap[p.Ptr])
		return true
	case OPInc:
		vm.Stack[len(vm.Stack)-1].Int++
		return true
	case OPDec:
		vm.Stack[len(vm.Stack)-1].Int--
		return true
	case OPAndSC:
		a := vm.pop()
		if !truthy(a) {
			vm.Stack = append(vm.Stack, Value{Kind: VKBool, Bool: false})
			frame.PC = int(ins.A)
		} else {
			// Pop second operand and push it (it's the result of the AND).
			b := vm.pop()
			vm.Stack = append(vm.Stack, Value{Kind: VKBool, Bool: truthy(b)})
		}
		return true
	case OPOrSC:
		a := vm.pop()
		if truthy(a) {
			vm.Stack = append(vm.Stack, Value{Kind: VKBool, Bool: true})
			frame.PC = int(ins.A)
		} else {
			b := vm.pop()
			vm.Stack = append(vm.Stack, Value{Kind: VKBool, Bool: truthy(b)})
		}
		return true
	case OPCaseTest:
		// Compare TOS with constant, jump to label on match.
		// ins.S is the constant as string. We pop TOS and compare.
		// Layout: push case-value(s) before OPCaseTest. Jumps if any matches.
		// For simplicity we do a linear walk of subsequent CaseInstrs in the
		// interpreter by chaining through ins.A as the "next case test" PC.
		v := vm.Stack[len(vm.Stack)-1]
		if compareLiteral(ins.S, v) {
			frame.PC = int(ins.B)
		} else {
			frame.PC = int(ins.A)
		}
		return true
	}
	vm.RuntimeError = 207
	vm.Halted = true
	return false
}

// globalCell returns the addressable storage cell for a global, creating
// a zero-valued cell on first use.
func (vm *VM) globalCell(name string) *Value {
	if c, ok := vm.Globals[name]; ok {
		return c
	}
	c := &Value{Kind: VKInt}
	vm.Globals[name] = c
	return c
}

// GetGlobal reads a global by name, returning false if it does not exist.
// It is the value-semantics accessor used by the RTL builtins.
func (vm *VM) GetGlobal(name string) (Value, bool) {
	if c, ok := vm.Globals[name]; ok {
		return *c, true
	}
	return Value{}, false
}

// GlobalVal reads a global by name, returning the zero value if absent.
func (vm *VM) GlobalVal(name string) Value {
	if c, ok := vm.Globals[name]; ok {
		return *c
	}
	return Value{}
}

// SetGlobal stores a global by name, creating its cell on first use.
func (vm *VM) SetGlobal(name string, v Value) {
	if c, ok := vm.Globals[name]; ok {
		*c = v
		return
	}
	cp := v
	vm.Globals[name] = &cp
}

// deepCopy clones a value so that aggregate assignment has Pascal value
// semantics (records and arrays are copied, not aliased). References
// (VKPtr with a Cell) are preserved by identity so that var parameters and
// pointers keep aliasing their target.
func deepCopy(v Value) Value {
	switch v.Kind {
	case VKArray:
		if v.Array == nil {
			return v
		}
		cp := make([]Value, len(v.Array))
		for i := range v.Array {
			cp[i] = deepCopy(v.Array[i])
		}
		v.Array = cp
		return v
	case VKRecord:
		if v.Rec == nil {
			return v
		}
		cp := make(map[string]*Value, len(v.Rec))
		for k, fv := range v.Rec {
			nc := deepCopy(*fv)
			cp[k] = &nc
		}
		v.Rec = cp
		return v
	}
	return v
}

// zeroValueForType returns the default value for a global of the named
// type. The names match the codegen's lowercase type tags.
func zeroValueForType(typ string) Value {
	switch typ {
	case "real", "single", "double", "extended", "comp":
		return Value{Kind: VKReal}
	case "string":
		return Value{Kind: VKStr}
	case "char":
		return Value{Kind: VKChar}
	case "boolean":
		return Value{Kind: VKBool}
	case "pointer":
		return Value{Kind: VKPtr}
	}
	return Value{Kind: VKInt}
}

// chargeHeap accounts for one heap allocation and enforces MaxHeap. It returns
// false (halting the VM with a heap-overflow error) when the limit is exceeded.
func (vm *VM) chargeHeap() bool {
	vm.heapAllocs++
	if vm.MaxHeap > 0 && vm.heapAllocs > vm.MaxHeap {
		vm.RuntimeError = 203 // heap overflow
		vm.Halted = true
		return false
	}
	return true
}

func (vm *VM) pop() Value {
	if len(vm.Stack) == 0 {
		vm.RuntimeError = 204
		vm.Halted = true
		return Value{Kind: VKNil}
	}
	v := vm.Stack[len(vm.Stack)-1]
	vm.Stack = vm.Stack[:len(vm.Stack)-1]
	return v
}

// Run executes the program until it halts.
func (vm *VM) Run() {
	// Initialize globals.
	for _, m := range vm.Program.Modules {
		for _, g := range m.Globals {
			if _, ok := vm.Globals[g.Name]; !ok {
				z := zeroValueForType(g.Type)
				vm.Globals[g.Name] = &z
			}
		}
		// Run init funcs.
		for _, initName := range m.Init {
			fn := m.Funcs[initName]
			if fn == nil {
				continue
			}
			frame := &Frame{Func: fn, Locals: make([]Value, len(fn.Locals)+len(fn.Params))}
			vm.CallStack = []*Frame{frame}
			vm.runFrame(frame)
			if vm.Halted {
				return
			}
		}
	}
	// Run main.
	var main *Function
	for _, m := range vm.Program.Modules {
		if m.Funcs[vm.Program.Entry] != nil {
			main = m.Funcs[vm.Program.Entry]
			break
		}
	}
	if main == nil {
		// Try by name without module prefix.
		for _, m := range vm.Program.Modules {
			for _, f := range m.Funcs {
				if strings.EqualFold(f.Name, vm.Program.Entry) {
					main = f
				}
			}
		}
	}
	if main == nil {
		vm.RuntimeError = 3
		vm.Halted = true
		return
	}
	frame := &Frame{Func: main, Locals: make([]Value, len(main.Locals)+len(main.Params))}
	vm.CallStack = []*Frame{frame}
	vm.runFrame(frame)
}

// runFrame drives execution starting from the given frame until that frame
// (and everything it called) has returned. It always steps the current
// top-of-callstack frame, so nested and recursive calls execute correctly;
// it returns once the call stack drops below the depth at entry.
func (vm *VM) runFrame(frame *Frame) {
	base := len(vm.CallStack)
	for !vm.Halted {
		if len(vm.CallStack) < base {
			return
		}
		vm.Step(vm.CallStack[len(vm.CallStack)-1])
	}
}

// selfTypeName reads the runtime object type tag from a method receiver,
// which may be a reference to the instance record or the record itself.
func selfTypeName(self Value) string {
	rec := self
	if self.Kind == VKPtr && self.Cell != nil {
		rec = *self.Cell
	}
	if rec.Kind == VKRecord {
		if t, ok := rec.Rec["__type"]; ok && t != nil {
			return strings.ToLower(t.Str)
		}
	}
	return ""
}

// lookupMethod resolves a method through an object type's (flattened) vtable.
func lookupMethod(p *Program, typ, method string) string {
	if p == nil || p.Vtables == nil {
		return ""
	}
	if tbl, ok := p.Vtables[typ]; ok {
		if fn, ok := tbl[method]; ok {
			return fn
		}
	}
	return ""
}

// setBits returns the set bitmap of v by value, or a zero bitmap when the set
// is empty (nil). Set values are cold, so the 32-byte copy is acceptable.
func setBits(v Value) [32]byte {
	if v.Set == nil {
		return [32]byte{}
	}
	return *v.Set
}

func findFunc(p *Program, name string) (*Function, bool) {
	for _, m := range p.Modules {
		if f, ok := m.Funcs[name]; ok {
			return f, true
		}
		// Allow "module.func" lookup.
		if idx := strings.LastIndex(name, "."); idx > 0 {
			if f, ok := m.Funcs[name[idx+1:]]; ok {
				return f, true
			}
		}
	}
	return nil, false
}

func truthy(v Value) bool {
	switch v.Kind {
	case VKBool:
		return v.Bool
	case VKInt:
		return v.Int != 0
	case VKReal:
		return v.Real != 0
	case VKChar:
		return v.Ch != 0
	case VKStr:
		return v.Str != ""
	case VKPtr:
		return v.Ptr >= 0
	}
	return false
}

func toInt(v Value) int64 {
	switch v.Kind {
	case VKInt:
		return v.Int
	case VKReal:
		return int64(v.Real)
	case VKChar:
		return int64(v.Ch)
	case VKBool:
		if v.Bool {
			return 1
		}
		return 0
	}
	return 0
}

func stringOf(v Value) string {
	switch v.Kind {
	case VKStr:
		return v.Str
	case VKChar:
		return string([]byte{v.Ch})
	}
	return v.String()
}

func toReal(v Value) float64 {
	switch v.Kind {
	case VKReal:
		return v.Real
	case VKInt:
		return float64(v.Int)
	case VKChar:
		return float64(v.Ch)
	}
	return 0
}

func binaryOp(op string, a, b Value) Value {
	// In TP7 the '/' operator is always real division (use 'div' for integer
	// division), regardless of operand types.
	if op == "/" {
		y := toReal(b)
		if y == 0 {
			return Value{Kind: VKReal, Real: 0}
		}
		return Value{Kind: VKReal, Real: toReal(a) / y}
	}
	// Set operators: union (+), difference (-), intersection (*).
	if a.Kind == VKSet && b.Kind == VKSet {
		av, bv := setBits(a), setBits(b)
		r := Value{Kind: VKSet, Set: new([32]byte)}
		for i := 0; i < 32; i++ {
			switch op {
			case "+":
				r.Set[i] = av[i] | bv[i]
			case "-":
				r.Set[i] = av[i] &^ bv[i]
			case "*":
				r.Set[i] = av[i] & bv[i]
			}
		}
		return r
	}
	// Mixed real/int promotes to real.
	if a.Kind == VKReal || b.Kind == VKReal {
		x, y := toReal(a), toReal(b)
		switch op {
		case "+":
			return Value{Kind: VKReal, Real: x + y}
		case "-":
			return Value{Kind: VKReal, Real: x - y}
		case "*":
			return Value{Kind: VKReal, Real: x * y}
		case "/":
			if y == 0 {
				return Value{Kind: VKReal, Real: 0}
			}
			return Value{Kind: VKReal, Real: x / y}
		}
	}
	// String concat.
	if op == "+" && (a.Kind == VKStr || b.Kind == VKStr) {
		return Value{Kind: VKStr, Str: stringOf(a) + stringOf(b)}
	}
	x, y := toInt(a), toInt(b)
	switch op {
	case "+":
		return Value{Kind: VKInt, Int: x + y}
	case "-":
		return Value{Kind: VKInt, Int: x - y}
	case "*":
		return Value{Kind: VKInt, Int: x * y}
	case "/":
		if y == 0 {
			return Value{Kind: VKInt, Int: 0}
		}
		return Value{Kind: VKInt, Int: x / y}
	case "div":
		if y == 0 {
			return Value{Kind: VKInt, Int: 0}
		}
		return Value{Kind: VKInt, Int: x / y}
	case "mod":
		if y == 0 {
			return Value{Kind: VKInt, Int: 0}
		}
		return Value{Kind: VKInt, Int: x % y}
	case "and":
		return Value{Kind: VKInt, Int: x & y}
	case "or":
		return Value{Kind: VKInt, Int: x | y}
	case "xor":
		return Value{Kind: VKInt, Int: x ^ y}
	case "shl":
		return Value{Kind: VKInt, Int: x << uint(y)}
	case "shr":
		return Value{Kind: VKInt, Int: x >> uint(y)}
	}
	return Value{Kind: VKNil}
}

func compareOp(op string, a, b Value) Value {
	// Pointer / nil comparison: only = and <> are defined.
	if a.Kind == VKPtr || b.Kind == VKPtr || a.Kind == VKNil || b.Kind == VKNil {
		an := a.Kind == VKNil || (a.Kind == VKPtr && a.Cell == nil)
		bn := b.Kind == VKNil || (b.Kind == VKPtr && b.Cell == nil)
		var eq bool
		if an || bn {
			eq = an && bn
		} else {
			eq = a.Cell == b.Cell
		}
		switch op {
		case "=":
			return Value{Kind: VKBool, Bool: eq}
		case "<>":
			return Value{Kind: VKBool, Bool: !eq}
		}
		return Value{Kind: VKBool, Bool: false}
	}
	// Set comparison: equality and subset/superset.
	if a.Kind == VKSet && b.Kind == VKSet {
		av, bv := setBits(a), setBits(b)
		switch op {
		case "=":
			return Value{Kind: VKBool, Bool: av == bv}
		case "<>":
			return Value{Kind: VKBool, Bool: av != bv}
		case "<=": // a is a subset of b
			for i := 0; i < 32; i++ {
				if av[i]&^bv[i] != 0 {
					return Value{Kind: VKBool, Bool: false}
				}
			}
			return Value{Kind: VKBool, Bool: true}
		case ">=": // a is a superset of b
			for i := 0; i < 32; i++ {
				if bv[i]&^av[i] != 0 {
					return Value{Kind: VKBool, Bool: false}
				}
			}
			return Value{Kind: VKBool, Bool: true}
		}
		return Value{Kind: VKBool, Bool: false}
	}
	if a.Kind == VKStr || b.Kind == VKStr {
		c := strings.Compare(stringOf(a), stringOf(b))
		return cmpResult(op, c)
	}
	if a.Kind == VKReal || b.Kind == VKReal {
		x, y := toReal(a), toReal(b)
		return cmpResult(op, signum(x-y))
	}
	x, y := toInt(a), toInt(b)
	return cmpResult(op, signum64(x-y))
}

func cmpResult(op string, c int) Value {
	var r bool
	switch op {
	case "=":
		r = c == 0
	case "<>":
		r = c != 0
	case "<":
		r = c < 0
	case "<=":
		r = c <= 0
	case ">":
		r = c > 0
	case ">=":
		r = c >= 0
	}
	return Value{Kind: VKBool, Bool: r}
}

func signum(x float64) int {
	switch {
	case x < 0:
		return -1
	case x > 0:
		return 1
	}
	return 0
}

func signum64(x int64) int {
	switch {
	case x < 0:
		return -1
	case x > 0:
		return 1
	}
	return 0
}

func unaryOp(op string, a Value) Value {
	switch op {
	case "+":
		return a
	case "-":
		if a.Kind == VKReal {
			return Value{Kind: VKReal, Real: -a.Real}
		}
		return Value{Kind: VKInt, Int: -toInt(a)}
	case "not":
		return Value{Kind: VKBool, Bool: !truthy(a)}
	}
	return a
}

func inSet(a, b Value) bool {
	if b.Kind != VKSet {
		return false
	}
	idx := int(toInt(a))
	if idx < 0 || idx >= 256 || b.Set == nil {
		return false
	}
	return b.Set[idx/8]&(1<<(idx%8)) != 0
}

func compareLiteral(lit string, v Value) bool {
	if v.Kind == VKStr {
		return v.Str == lit
	}
	if len(lit) == 0 {
		return false
	}
	if lit[0] == '\'' && lit[len(lit)-1] == '\'' {
		if v.Kind == VKChar {
			return int(v.Ch) == int(lit[1])
		}
	}
	// numeric
	if i, err := parseIntLit(lit); err == nil {
		return toInt(v) == i
	}
	return false
}

func parseIntLit(s string) (int64, error) {
	var v int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("bad")
		}
		v = v*10 + int64(c-'0')
	}
	return v, nil
}
