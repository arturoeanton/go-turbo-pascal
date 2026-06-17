// Package debug implements the BPGo source-level debugger. The
// debugger cooperates with the VM (or a future dos16 backend with a
// remote stub) to provide breakpoints, watches, call stacks and
// step/continue controls. The conformance harness uses the VM
// backend to drive the tests.
package debug

import (
	"fmt"
	"sync"
)

// Breakpoint is a single line breakpoint.
type Breakpoint struct {
	File string
	Line int
	Hits int
}

// Watch is a watch expression.
type Watch struct {
	Expr   string
	Last   string
	Breaks bool // stops execution when value changes
}

// Frame is a call-stack entry.
type Frame struct {
	Func string
	Line int
	PC   int
}

// Debugger is the in-memory debugger used by the BPGo CLI and the
// conformance harness.
type Debugger struct {
	mu          sync.Mutex
	Breakpoints []Breakpoint
	Watches     []Watch
	Stack       []Frame
	Current     Frame
	StepCount   int
	StepLimit   int
	Halted      bool
}

// New creates a new Debugger.
func New() *Debugger {
	return &Debugger{StepLimit: 100000}
}

// SetBreakpoint adds a breakpoint.
func (d *Debugger) SetBreakpoint(file string, line int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i, b := range d.Breakpoints {
		if b.File == file && b.Line == line {
			d.Breakpoints[i].Hits++
			return
		}
	}
	d.Breakpoints = append(d.Breakpoints, Breakpoint{File: file, Line: line, Hits: 1})
}

// RemoveBreakpoint removes a breakpoint.
func (d *Debugger) RemoveBreakpoint(file string, line int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i, b := range d.Breakpoints {
		if b.File == file && b.Line == line {
			d.Breakpoints = append(d.Breakpoints[:i], d.Breakpoints[i+1:]...)
			return
		}
	}
}

// AddWatch adds a watch expression.
func (d *Debugger) AddWatch(expr string, breaks bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Watches = append(d.Watches, Watch{Expr: expr, Breaks: breaks})
}

// PushFrame pushes a call-stack frame.
func (d *Debugger) PushFrame(f Frame) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Stack = append(d.Stack, f)
}

// PopFrame pops the top frame.
func (d *Debugger) PopFrame() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.Stack) == 0 {
		return
	}
	d.Stack = d.Stack[:len(d.Stack)-1]
}

// HitAt reports whether the given line has a breakpoint.
func (d *Debugger) HitAt(file string, line int) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, b := range d.Breakpoints {
		if b.File == file && b.Line == line {
			return true
		}
	}
	return false
}

// Step increments the step counter and returns true if execution
// should halt.
func (d *Debugger) Step() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.StepCount++
	if d.StepCount > d.StepLimit {
		d.Halted = true
	}
	return d.Halted
}

// Halt stops execution.
func (d *Debugger) Halt() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Halted = true
}

// Continue resumes execution.
func (d *Debugger) Continue() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Halted = false
}

// Snapshot returns a textual view of the debugger state.
func (d *Debugger) Snapshot() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	var b []byte
	b = append(b, "Breakpoints:\n"...)
	for _, bp := range d.Breakpoints {
		fmt.Fprintf(stringWriter{&b}, "  %s:%d (hits=%d)\n", bp.File, bp.Line, bp.Hits)
	}
	b = append(b, "Watches:\n"...)
	for _, w := range d.Watches {
		fmt.Fprintf(stringWriter{&b}, "  %s = %s\n", w.Expr, w.Last)
	}
	b = append(b, "Stack:\n"...)
	for _, f := range d.Stack {
		fmt.Fprintf(stringWriter{&b}, "  %s @ line %d\n", f.Func, f.Line)
	}
	return string(b)
}

type stringWriter struct {
	buf *[]byte
}

func (s stringWriter) Write(p []byte) (int, error) {
	*s.buf = append(*s.buf, p...)
	return len(p), nil
}
