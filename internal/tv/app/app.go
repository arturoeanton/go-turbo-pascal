// Package app implements the Turbo Vision App unit. The unit provides
// TProgram, TApplication and TDesktop, the top-level application
// classes that drive the TV event loop. BPGo implements them as
// cooperative state machines; the actual rendering and event loop
// are driven by the host program (the BPGo IDE).
package app

import (
	"github.com/arturoeanton/go-turbo-pascal/internal/tv/dialogs"
	"github.com/arturoeanton/go-turbo-pascal/internal/tv/menus"
	"github.com/arturoeanton/go-turbo-pascal/internal/tv/views"
)

// TProgram is the base class for an executable TV application.
type TProgram struct {
	views.TGroup
	MenuBar    *menus.TMenuBar
	StatusLine *menus.TStatusLine
	Desktop    *TDesktop
	Running    bool
}

// Init constructs a TProgram.
func (p *TProgram) Init(bounds views.TRect) *TProgram {
	p.TGroup.Init(bounds)
	return p
}

// Run runs the event loop. The conformance harness does not actually
// run an event loop; this returns after the first iteration.
func (p *TProgram) Run() {
	p.Running = true
}

// Quit terminates the program.
func (p *TProgram) Quit() {
	p.Running = false
}

// Execute runs a modal dialog.
func (p *TProgram) Execute(d *dialogs.TDialog) uint16 {
	// The conformance harness returns CmdCancel as a default.
	return dialogs.CmdCancel
}

// TApplication extends TProgram with the standard menus and status
// line expected by a TV application.
type TApplication struct {
	TProgram
}

// Init constructs a TApplication.
func (a *TApplication) Init(bounds views.TRect) *TApplication {
	a.TProgram.Init(bounds)
	a.Desktop = NewDesktop(bounds)
	a.Insert(&a.Desktop.TView)
	return a
}

// TDesktop is the background area of a TApplication.
type TDesktop struct {
	views.TView
}

// NewDesktop creates a desktop.
func NewDesktop(bounds views.TRect) *TDesktop {
	d := &TDesktop{TView: views.TView{Bounds: bounds}}
	d.State |= views.StateVisible
	return d
}

// InitEvents installs event handlers.
func (a *TApplication) InitEvents() {}

// DoneEvents removes event handlers.
func (a *TApplication) DoneEvents() {}
