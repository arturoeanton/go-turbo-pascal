package vmpas

import (
	"fmt"

	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
)

// Durable execution: a script can pause itself with Suspend(tag) and the host
// can persist the resulting State, then resume exactly where it left off —
// possibly in another process, minutes or days later. Combined with the
// Deterministic capability this gives pausable, replayable business workflows
// (wait for an approval, an external event, a scheduled time) on top of the
// same embedded engine.
//
// Contract: everything the script computes (locals, globals, the call stack)
// is captured in State and restored on resume. Bound Go variables are the host
// I/O channel — they are re-seeded on every resume, so the host injects answers
// by updating them before calling ResumeDurable, and the script reads them
// after the Suspend call returns.

// State is an opaque, serializable snapshot of a paused execution. Persist
// State.Data (e.g. to a database) and pass it back to ResumeDurable later.
type State struct {
	Tag    string // the tag passed to Suspend (why the script paused)
	Data   []byte // the VM snapshot
	Output string // output produced up to the suspension point
}

// RunDurable compiles and runs code under the engine's sandbox. If the script
// calls Suspend(tag) it returns a non-nil *State to persist and resume later;
// if it runs to completion it returns (nil, nil) and Output() holds the result.
func (e *Engine) RunDurable(code string) (*State, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	prog, err := e.compileLocked(code)
	if err != nil {
		return nil, err
	}
	return e.runDurableLocked(prog, nil)
}

// ResumeDurable restores a paused execution from st and continues it. code must
// be the exact source that produced st. It may suspend again (returns a new
// *State) or finish (returns nil). Bound Go variables are re-seeded first, so
// update them beforehand to inject inputs the script will read after resuming.
func (e *Engine) ResumeDurable(code string, st *State) (*State, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if st == nil {
		return nil, fmt.Errorf("vmpas: nil resume state")
	}
	prog, err := e.compileLocked(code)
	if err != nil {
		return nil, err
	}
	return e.runDurableLocked(prog, st)
}

// runDurableLocked drives one run-or-resume segment to completion or to the next
// suspension (caller holds mu). When st is nil it starts a fresh run; otherwise
// it restores st and continues.
func (e *Engine) runDurableLocked(prog *ir.Program, st *State) (*State, error) {
	var vm *ir.VM
	if st == nil {
		vm = ir.NewVM(prog)
	} else {
		var err error
		vm, err = ir.RestoreVM(prog, st.Data)
		if err != nil {
			return nil, err
		}
	}

	e.applyLimits(vm)
	vm.Builtins = e.prepareBuiltins()
	e.cursor, e.dbErr, e.httpStatus, e.httpHeaders = nil, "", 0, nil
	e.suspendTag = ""
	if st != nil {
		vm.Output.WriteString(st.Output) // keep output cumulative across segments
	}
	// Bound vars are the host I/O channel: (re-)seed them on start and on every
	// resume so the host can inject inputs the script reads after Suspend.
	e.seedVars(vm)

	if st == nil {
		vm.Run()
	} else {
		vm.RunResume()
	}

	e.output = vm.Output.String()
	e.readbackVars(vm)

	if vm.Suspended {
		data, err := vm.Snapshot()
		if err != nil {
			return nil, err
		}
		return &State{Tag: e.suspendTag, Data: data, Output: e.output}, nil
	}
	if vm.RuntimeError != 0 {
		return nil, fmt.Errorf("vmpas: runtime error %d", vm.RuntimeError)
	}
	return nil, nil
}

// applyLimits copies the engine's sandbox limits onto a VM (shared by the
// regular and durable run paths).
func (e *Engine) applyLimits(vm *ir.VM) {
	if e.caps.MaxSteps > 0 {
		vm.MaxSteps = e.caps.MaxSteps
	}
	if e.caps.MaxHeap > 0 {
		vm.MaxHeap = e.caps.MaxHeap
	}
	if e.caps.MaxOutput > 0 {
		vm.MaxOutput = e.caps.MaxOutput
	}
	if e.caps.MaxCallDepth > 0 {
		vm.MaxCallDepth = e.caps.MaxCallDepth
	}
	if e.caps.Deterministic {
		vm.Deterministic = true
		vm.DetRandSeed = e.caps.Seed
	}
	// Note: MaxDuration is intentionally not applied here — a durable run may be
	// paused for arbitrarily long; wall-clock limits would be measured per host
	// process, not across the whole workflow. Use MaxSteps to bound work.
}
