package ir

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"hash/fnv"
	"sort"
)

// Snapshot/resume gives the VM deterministic durable execution: the full
// machine state (globals, heap, operand stack, call stack with program
// counters, RNG seed, exception state) is serialized to a portable byte slice
// and can be restored later — even in another process — to continue exactly
// where it left off. Combined with deterministic mode (VM.Deterministic) this
// is the basis of phase F: pausable, replayable, auditable business logic.
//
// Scope (v1): non-concurrent programs (no live fibers/channels) and no open
// files. These are the cases that matter for embedded business rules; the
// encoder returns a clear error otherwise rather than producing a partial
// snapshot.

const snapshotVersion = 1

// snapshot is the serialized, pointer-free representation of a VM's state.
// Every internal pointer is replaced by an integer id so the object graph
// (including aliasing via var-parameters and pointers) round-trips exactly.
type snapshot struct {
	Version     int
	Fingerprint uint64 // identifies the program the snapshot belongs to

	Steps         int
	RandomState   uint32
	ExitCode      int
	RuntimeError  int
	Deterministic bool
	DetRandSeed   int64

	ExcActive bool
	Exc       cellSnap

	Globals  []globalSnap
	Heap     []cellSnap
	Stack    []cellSnap
	Frames   []frameSnap // call stack, bottom (main) first
	Handlers []handlerSnap
	// Orphans are heap cells (allocated by New) reachable only through pointers,
	// not through the ownership tree. Each is standalone storage on resume.
	Orphans []cellSnap
}

type globalSnap struct {
	Name string
	Cell cellSnap
}

type frameSnap struct {
	FuncName  string
	PC        int
	Locals    []cellSnap
	Result    cellSnap
	CallerIdx int // index into Frames, or -1
	StaticIdx int // index into Frames, or -1
}

type handlerSnap struct {
	FrameIdx   int
	PC         int
	StackDepth int
	CallDepth  int
	IsFinally  bool
}

// cellSnap is the serialized form of one Value. SelfID (>0) marks a cell that
// some pointer aliases; on restore its address is registered so the pointer can
// be rewired. References to other cells use ids assigned in the same pass.
type cellSnap struct {
	SelfID int
	Kind   int

	Int  int64
	Real float64
	Bool bool
	Ch   byte
	Str  string

	HasSet bool
	Set    [32]byte

	// VKPtr: 0 = nil, 1 = cell alias (TargetID), 2 = heap index (PtrHeap).
	PtrKind  int
	TargetID int
	PtrHeap  int64

	Elems   []cellSnap // VKArray / VKFunc captured cells
	RecKeys []string   // VKRecord field names (sorted for determinism)
	RecVals []cellSnap
}

// progFingerprint derives a stable id from the program's code so a snapshot is
// only ever resumed against the program that produced it.
func progFingerprint(p *Program) uint64 {
	if p == nil {
		return 0
	}
	h := fnv.New64a()
	fmt.Fprintf(h, "main=%s entry=%s\n", p.Main, p.Entry)
	for _, m := range p.Modules {
		names := make([]string, 0, len(m.Funcs))
		for n := range m.Funcs {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			fn := m.Funcs[n]
			fmt.Fprintf(h, "%s/%s:%d:%d:%d\n", m.Name, n, len(fn.Code), len(fn.Params), len(fn.Locals))
			// Hash the full instruction stream so any source change (including
			// literals, which keep the structure identical) invalidates a stale
			// snapshot — PCs and slot indices would otherwise silently mismatch.
			for _, in := range fn.Code {
				fmt.Fprintf(h, "%d|%d|%d|%s|%v;", in.Op, in.A, in.B, in.S, in.R)
			}
			h.Write([]byte{'\n'})
		}
	}
	return h.Sum64()
}

// --- encoding ---

type snapEncoder struct {
	targetID  map[*Value]int  // alias-target cell address -> id
	next      int             //
	ownedAddr map[*Value]bool // cells that live in some ownership tree
	visited   map[*Value]bool // cells already scheduled as an orphan tree
	orphans   []*Value        // discovered orphan-cell roots, in id order
}

func (e *snapEncoder) assignID(p *Value) int {
	if id, ok := e.targetID[p]; ok {
		return id
	}
	e.next++
	e.targetID[p] = e.next
	return e.next
}

// markOwned records p and its aggregate sub-cells as living in an ownership
// tree (so pointers to them resolve to their slot, not a standalone cell).
func (e *snapEncoder) markOwned(p *Value) {
	if e.ownedAddr[p] {
		return
	}
	e.ownedAddr[p] = true
	switch p.Kind {
	case VKArray, VKFunc:
		for i := range p.Array {
			e.markOwned(&p.Array[i])
		}
	case VKRecord:
		for i := range p.Rec {
			e.markOwned(p.Rec[i].Cell)
		}
	}
}

// discover walks the ownership tree at p, following aggregates and pointers.
// Every pointer target gets an id; targets that are not owned (allocated by
// New, reachable only via pointers) become orphan roots, which are themselves
// marked owned and walked, until the whole reachable graph is covered.
func (e *snapEncoder) discover(p *Value) {
	switch p.Kind {
	case VKPtr:
		if p.Cell != nil {
			e.assignID(p.Cell)
			if !e.ownedAddr[p.Cell] && !e.visited[p.Cell] {
				e.visited[p.Cell] = true
				e.orphans = append(e.orphans, p.Cell)
				e.markOwned(p.Cell)
				e.discover(p.Cell)
			}
		}
	case VKArray, VKFunc:
		for i := range p.Array {
			e.discover(&p.Array[i])
		}
	case VKRecord:
		for i := range p.Rec {
			e.discover(p.Rec[i].Cell)
		}
	}
}

// Snapshot serializes the VM's current execution state. It must be called at an
// instruction boundary (e.g. after the VM halts or suspends); the result can be
// passed to RestoreVM to continue execution later.
func (vm *VM) Snapshot() ([]byte, error) {
	if vm.cur != nil || (vm.Program != nil && vm.Program.Concurrent && len(vm.allFibers) > 0) {
		return nil, fmt.Errorf("ir: cannot snapshot a concurrent program with live fibers")
	}
	enc := &snapEncoder{
		targetID:  map[*Value]int{},
		ownedAddr: map[*Value]bool{},
		visited:   map[*Value]bool{},
	}

	// Pass A: mark every ownership-tree cell, then walk to assign ids to every
	// pointer target and to discover orphan heap cells (reachable only via
	// pointers, e.g. allocated by New).
	roots := vm.snapshotRoots()
	for _, r := range roots {
		enc.markOwned(r)
	}
	for _, r := range roots {
		enc.discover(r)
	}

	// Pass B: encode the ownership tree, tagging targets and rewriting pointers.
	frameIdx := map[*Frame]int{}
	for i, f := range vm.CallStack {
		frameIdx[f] = i
	}

	s := snapshot{
		Version:       snapshotVersion,
		Fingerprint:   progFingerprint(vm.Program),
		Steps:         vm.Steps,
		RandomState:   vm.RandomState,
		ExitCode:      vm.ExitCode,
		RuntimeError:  vm.RuntimeError,
		Deterministic: vm.Deterministic,
		DetRandSeed:   vm.DetRandSeed,
		ExcActive:     vm.excActive,
	}

	var err error
	s.Exc = enc.encode(&vm.excValue, &err)

	globalNames := make([]string, 0, len(vm.Globals))
	for n := range vm.Globals {
		globalNames = append(globalNames, n)
	}
	sort.Strings(globalNames)
	for _, n := range globalNames {
		s.Globals = append(s.Globals, globalSnap{Name: n, Cell: enc.encode(vm.Globals[n], &err)})
	}

	s.Heap = make([]cellSnap, len(vm.Heap))
	for i := range vm.Heap {
		s.Heap[i] = enc.encode(&vm.Heap[i], &err)
	}
	s.Stack = make([]cellSnap, len(vm.Stack))
	for i := range vm.Stack {
		s.Stack[i] = enc.encode(&vm.Stack[i], &err)
	}

	for _, f := range vm.CallStack {
		fs := frameSnap{FuncName: f.Func.Name, PC: f.PC, CallerIdx: -1, StaticIdx: -1}
		if idx, ok := frameIdx[f.Caller]; ok {
			fs.CallerIdx = idx
		}
		if idx, ok := frameIdx[f.Static]; ok {
			fs.StaticIdx = idx
		}
		fs.Locals = make([]cellSnap, len(f.Locals))
		for i := range f.Locals {
			fs.Locals[i] = enc.encode(&f.Locals[i], &err)
		}
		fs.Result = enc.encode(&f.Result, &err)
		s.Frames = append(s.Frames, fs)
	}

	for _, h := range vm.handlers {
		hs := handlerSnap{PC: h.pc, StackDepth: h.stackDepth, CallDepth: h.callDepth, IsFinally: h.isFinally, FrameIdx: -1}
		if idx, ok := frameIdx[h.frame]; ok {
			hs.FrameIdx = idx
		}
		s.Handlers = append(s.Handlers, hs)
	}

	// Orphan heap cells (only reachable via pointers): each is standalone
	// storage, encoded with its SelfID so resume can rebuild and register it.
	s.Orphans = make([]cellSnap, len(enc.orphans))
	for i, o := range enc.orphans {
		s.Orphans[i] = enc.encode(o, &err)
	}

	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if e := gob.NewEncoder(&buf).Encode(&s); e != nil {
		return nil, fmt.Errorf("ir: snapshot encode: %w", e)
	}
	return buf.Bytes(), nil
}

// snapshotRoots returns the addresses of every root cell to walk, in a
// deterministic order (globals sorted by name) so snapshots are reproducible.
func (vm *VM) snapshotRoots() []*Value {
	roots := []*Value{&vm.excValue}
	gnames := make([]string, 0, len(vm.Globals))
	for n := range vm.Globals {
		gnames = append(gnames, n)
	}
	sort.Strings(gnames)
	for _, n := range gnames {
		roots = append(roots, vm.Globals[n])
	}
	for i := range vm.Heap {
		roots = append(roots, &vm.Heap[i])
	}
	for i := range vm.Stack {
		roots = append(roots, &vm.Stack[i])
	}
	for _, f := range vm.CallStack {
		for i := range f.Locals {
			roots = append(roots, &f.Locals[i])
		}
		roots = append(roots, &f.Result)
	}
	return roots
}

// encode produces the cellSnap for the cell at address p, recursing into
// aggregates. Errors (open files, live channels) are reported via *errp.
func (e *snapEncoder) encode(p *Value, errp *error) cellSnap {
	c := cellSnap{Kind: int(p.Kind), SelfID: e.targetID[p]}
	switch p.Kind {
	case VKInt, VKCurrency:
		c.Int = p.Int
	case VKReal:
		c.Real = p.Real
	case VKBool:
		c.Bool = p.Bool
	case VKChar:
		c.Ch = p.Ch
	case VKStr:
		c.Str = p.Str
	case VKNil:
		// nothing
	case VKSet:
		if p.Set != nil {
			c.HasSet = true
			c.Set = *p.Set
		}
	case VKPtr:
		switch {
		case p.Cell != nil:
			c.PtrKind = 1
			c.TargetID = e.targetID[p.Cell]
			if c.TargetID == 0 && *errp == nil {
				*errp = fmt.Errorf("ir: snapshot found a pointer to an untracked cell")
			}
		default:
			c.PtrKind = 2
			c.PtrHeap = p.Ptr
		}
	case VKArray, VKFunc:
		c.Str = p.Str // VKFunc: function name
		c.Elems = make([]cellSnap, len(p.Array))
		for i := range p.Array {
			c.Elems[i] = e.encode(&p.Array[i], errp)
		}
	case VKRecord:
		// Serialize fields sorted by name for deterministic output (stable
		// snapshot bytes regardless of insertion order).
		order := make([]int, len(p.Rec))
		for i := range order {
			order[i] = i
		}
		sort.Slice(order, func(a, b int) bool { return p.Rec[order[a]].Name < p.Rec[order[b]].Name })
		c.RecKeys = make([]string, len(order))
		c.RecVals = make([]cellSnap, len(order))
		for i, idx := range order {
			c.RecKeys[i] = p.Rec[idx].Name
			c.RecVals[i] = e.encode(p.Rec[idx].Cell, errp)
		}
	case VKFile:
		if *errp == nil {
			*errp = fmt.Errorf("ir: cannot snapshot while a file is open")
		}
	case VKChan:
		if *errp == nil {
			*errp = fmt.Errorf("ir: cannot snapshot a live channel")
		}
	}
	return c
}

// --- decoding ---

// RestoreVM rebuilds a VM from a snapshot taken by Snapshot. The program must be
// the same one that produced the snapshot. The returned VM has its execution
// state restored but no Builtins/Output wired — the caller attaches those (as
// the original run did) before calling RunResume.
func RestoreVM(prog *Program, data []byte) (*VM, error) {
	var s snapshot
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&s); err != nil {
		return nil, fmt.Errorf("ir: snapshot decode: %w", err)
	}
	if s.Version != snapshotVersion {
		return nil, fmt.Errorf("ir: snapshot version %d not supported (want %d)", s.Version, snapshotVersion)
	}
	if fp := progFingerprint(prog); fp != s.Fingerprint {
		return nil, fmt.Errorf("ir: snapshot does not match this program (fingerprint mismatch)")
	}

	vm := NewVM(prog)
	vm.Steps = s.Steps
	vm.RandomState = s.RandomState
	vm.ExitCode = s.ExitCode
	vm.RuntimeError = s.RuntimeError
	vm.Deterministic = s.Deterministic
	vm.DetRandSeed = s.DetRandSeed
	vm.excActive = s.ExcActive

	dec := &snapDecoder{idMap: map[int]*Value{}}

	// Heap first: stable backing for &Heap[i] addresses.
	vm.Heap = make([]Value, len(s.Heap))
	for i := range s.Heap {
		dec.rebuild(s.Heap[i], &vm.Heap[i])
	}

	vm.Globals = make(map[string]*Value, len(s.Globals))
	for _, g := range s.Globals {
		c := new(Value)
		dec.rebuild(g.Cell, c)
		vm.Globals[g.Name] = c
	}

	vm.Stack = make([]Value, len(s.Stack))
	for i := range s.Stack {
		dec.rebuild(s.Stack[i], &vm.Stack[i])
	}

	dec.rebuild(s.Exc, &vm.excValue)

	// Frames: allocate first (so &Locals[i] / &Result are stable), then fill
	// and link Caller/Static by index.
	frames := make([]*Frame, len(s.Frames))
	for i := range s.Frames {
		fn, ok := findFunc(prog, s.Frames[i].FuncName)
		if !ok {
			return nil, fmt.Errorf("ir: snapshot references unknown function %q", s.Frames[i].FuncName)
		}
		// Validate the resume PC against the function's code so a corrupt snapshot
		// is a clean error here rather than an out-of-range access at run time.
		if pc := s.Frames[i].PC; pc < 0 || pc > len(fn.Code) {
			return nil, fmt.Errorf("ir: snapshot frame %d has out-of-range PC %d (func %q has %d instructions)", i, pc, s.Frames[i].FuncName, len(fn.Code))
		}
		frames[i] = &Frame{Func: fn, PC: s.Frames[i].PC, Locals: make([]Value, len(s.Frames[i].Locals))}
	}
	for i, fsnap := range s.Frames {
		f := frames[i]
		for j := range fsnap.Locals {
			dec.rebuild(fsnap.Locals[j], &f.Locals[j])
		}
		dec.rebuild(fsnap.Result, &f.Result)
		if fsnap.CallerIdx >= 0 && fsnap.CallerIdx < len(frames) {
			f.Caller = frames[fsnap.CallerIdx]
		}
		if fsnap.StaticIdx >= 0 && fsnap.StaticIdx < len(frames) {
			f.Static = frames[fsnap.StaticIdx]
		}
	}
	vm.CallStack = frames

	for _, h := range s.Handlers {
		th := tryHandler{pc: h.PC, stackDepth: h.StackDepth, callDepth: h.CallDepth, isFinally: h.IsFinally}
		if h.FrameIdx >= 0 && h.FrameIdx < len(frames) {
			th.frame = frames[h.FrameIdx]
		}
		vm.handlers = append(vm.handlers, th)
	}

	// Orphan heap cells: standalone storage, registered by SelfID so pointers
	// to them resolve in the fixup pass.
	for i := range s.Orphans {
		c := new(Value)
		dec.rebuild(s.Orphans[i], c)
	}

	// Second pass: now that every cell address is registered, wire pointers.
	for _, fx := range dec.fixups {
		fx()
	}
	return vm, nil
}

type snapDecoder struct {
	idMap  map[int]*Value
	fixups []func()
}

// rebuild reconstructs the cell described by c into the canonical storage at
// dst, recursing into aggregates. Pointer targets are resolved in a later pass
// via fixups, once every cell address is known.
func (d *snapDecoder) rebuild(c cellSnap, dst *Value) {
	dst.Kind = ValKind(c.Kind)
	if c.SelfID != 0 {
		d.idMap[c.SelfID] = dst
	}
	switch ValKind(c.Kind) {
	case VKInt, VKCurrency:
		dst.Int = c.Int
	case VKReal:
		dst.Real = c.Real
	case VKBool:
		dst.Bool = c.Bool
	case VKChar:
		dst.Ch = c.Ch
	case VKStr:
		dst.Str = c.Str
	case VKNil:
		// nothing
	case VKSet:
		if c.HasSet {
			set := c.Set
			dst.Set = &set
		}
	case VKPtr:
		switch c.PtrKind {
		case 1:
			tid := c.TargetID
			d.fixups = append(d.fixups, func() { dst.Cell = d.idMap[tid] })
		case 2:
			dst.Ptr = c.PtrHeap
		}
	case VKArray, VKFunc:
		dst.Str = c.Str
		arr := make([]Value, len(c.Elems))
		dst.Array = arr
		for i := range c.Elems {
			d.rebuild(c.Elems[i], &arr[i])
		}
	case VKRecord:
		rec := make([]RecField, len(c.RecKeys))
		dst.Rec = rec
		for i, k := range c.RecKeys {
			fc := new(Value)
			rec[i] = RecField{Name: k, Cell: fc}
			d.rebuild(c.RecVals[i], fc)
		}
	}
}
