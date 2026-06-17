// Package dialogs implements the Turbo Vision Dialogs unit. The unit
// provides TDialog and TWindow, the standard dialog and window
// classes used by the IDE and TV applications. BPGo implements the
// classes as specialised TGroup subclasses with the expected state
// flags and event handling.
package dialogs

import (
	"github.com/arturoeanton/go-turbo-pascal/internal/tv/views"
)

// TWindow is a draggable, sizeable window with a frame and title.
type TWindow struct {
	views.TGroup
	Title  string
	Number int16
	Frame  *views.TFrame
	Zoomed bool
	Flags  uint16
}

// Init constructs a TWindow.
func (w *TWindow) Init(bounds views.TRect, title string, number int16) *TWindow {
	w.TGroup.Init(bounds)
	w.Title = title
	w.Number = number
	w.Frame = (&views.TFrame{}).Init(bounds)
	frameView := &w.Frame.TView
	frameView.State |= views.StateVisible
	w.Insert(frameView)
	w.State |= views.StateShadow | views.StateActive
	return w
}

// TDialog is a modal dialog window.
type TDialog struct {
	TWindow
}

// Init constructs a TDialog.
func (d *TDialog) Init(bounds views.TRect, title string) *TDialog {
	d.TWindow.Init(bounds, title, 0)
	d.State |= views.StateModal
	return d
}

// EndModal closes the dialog with the given result code.
func (d *TDialog) EndModal(cmd uint16) {
	d.State &^= views.StateModal
	d.Command = cmd
}

// Standard dialog commands.
const (
	CmdOK     = 0x0FF01
	CmdCancel = 0x0FF02
	CmdYes    = 0x0FF03
	CmdNo     = 0x0FF04
)

// TButton is re-exported from views for convenience.
type TButton = views.TButton

// TInputLine is re-exported from views for convenience.
type TInputLine = views.TInputLine

// TLabel is re-exported from views for convenience.
type TLabel = views.TLabel

// TStaticText is re-exported from views for convenience.
type TStaticText = views.TStaticText

// TCheckBoxes is re-exported from views for convenience.
type TCheckBoxes = views.TCheckBoxes

// TRadioButtons is re-exported from views for convenience.
type TRadioButtons = views.TRadioButtons
