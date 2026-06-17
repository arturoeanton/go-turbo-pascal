package ir

// Debugger drives a VM with source-line granularity: line breakpoints,
// step-over/step-line and continue, plus inspection of globals and the call
// stack. It steps the VM cooperatively (synchronously), so the host (e.g. a
// DAP adapter) calls Continue/StepLine and gets back the stop location.
type Debugger struct {
	vm          *VM
	frame       *Frame
	breakpoints map[int]bool // source lines
	curLine     int
}

// NewDebugger prepares a debugger for a program. Call Start before stepping.
func NewDebugger(vm *VM) *Debugger {
	return &Debugger{vm: vm, breakpoints: map[int]bool{}}
}

// SetBreakpoints replaces the set of line breakpoints.
func (d *Debugger) SetBreakpoints(lines []int) {
	d.breakpoints = map[int]bool{}
	for _, l := range lines {
		d.breakpoints[l] = true
	}
}

// Start initializes execution at the program entry without running any code.
func (d *Debugger) Start() bool {
	main := d.vm.entryFunc()
	if main == nil {
		d.vm.RuntimeError = 3
		d.vm.Halted = true
		return false
	}
	d.frame = &Frame{Func: main, Locals: make([]Value, len(main.Params)+len(main.Locals))}
	d.vm.CallStack = []*Frame{d.frame}
	d.curLine = d.currentLine()
	return true
}

// Halted reports whether the program has finished.
func (d *Debugger) Halted() bool { return d.vm.Halted }

// Line returns the current source line.
func (d *Debugger) Line() int { return d.curLine }

// Output returns the program output so far.
func (d *Debugger) Output() string { return d.vm.Output.String() }

// Continue runs until a breakpoint is hit or the program halts. It returns
// true and the line when stopped at a breakpoint.
func (d *Debugger) Continue() (bool, int) {
	for !d.vm.Halted {
		if len(d.vm.CallStack) == 0 {
			break
		}
		d.vm.Step(d.vm.CallStack[len(d.vm.CallStack)-1])
		if d.vm.Halted {
			break
		}
		line := d.currentLine()
		if line != 0 && line != d.curLine {
			d.curLine = line
			if d.breakpoints[line] {
				return true, line
			}
		}
	}
	return false, 0
}

// StepLine executes until the source line changes (step into calls). It
// returns true and the new line, or false when the program halts.
func (d *Debugger) StepLine() (bool, int) {
	start := d.curLine
	for !d.vm.Halted {
		if len(d.vm.CallStack) == 0 {
			break
		}
		d.vm.Step(d.vm.CallStack[len(d.vm.CallStack)-1])
		if d.vm.Halted {
			break
		}
		line := d.currentLine()
		if line != 0 && line != start {
			d.curLine = line
			return true, line
		}
	}
	return false, 0
}

// Globals returns a snapshot of the current global values.
func (d *Debugger) Globals() map[string]Value {
	out := map[string]Value{}
	for k, v := range d.vm.Globals {
		if v != nil {
			out[k] = *v
		}
	}
	return out
}

// currentLine maps the top frame's program counter to a source line via the
// function's SourceMap (the highest mapped index <= PC).
func (d *Debugger) currentLine() int {
	if len(d.vm.CallStack) == 0 {
		return 0
	}
	f := d.vm.CallStack[len(d.vm.CallStack)-1]
	if f.Func == nil || f.Func.SourceMap == nil {
		return 0
	}
	best, bestLine := -1, 0
	for idx, ref := range f.Func.SourceMap {
		if idx <= f.PC && idx > best {
			best, bestLine = idx, ref.Line
		}
	}
	return bestLine
}

// entryFunc finds the program's entry function.
func (vm *VM) entryFunc() *Function {
	for _, m := range vm.Program.Modules {
		if f := m.Funcs[vm.Program.Entry]; f != nil {
			return f
		}
	}
	return nil
}
