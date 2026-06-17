// Package msgbox implements the Turbo Vision MsgBox unit. The unit
// provides MessageBox and InputBox. BPGo provides them as Go
// functions that emit the standard command codes.
package msgbox

import (
	"github.com/arturoeanton/go-turbo-pascal/internal/tv/dialogs"
	"github.com/arturoeanton/go-turbo-pascal/internal/tv/views"
)

// Standard message box flags.
const (
	FWarning      = 0x0000
	FError        = 0x0001
	FInformation  = 0x0002
	FConfirmation = 0x0003
)

// Standard button flags.
const (
	BFYes    = 0x0001
	BFNo     = 0x0002
	BFOk     = 0x0004
	BFCancel = 0x0008
)

// MessageBox returns the command selected by the user. The
// conformance harness always returns the first button.
func MessageBox(msg string, params []string, buttons uint16) uint16 {
	return firstButton(buttons)
}

// MessageBoxRect is the position-aware variant.
func MessageBoxRect(rect views.TRect, msg string, params []string, buttons uint16) uint16 {
	return firstButton(buttons)
}

func firstButton(buttons uint16) uint16 {
	if buttons&BFYes != 0 {
		return dialogs.CmdYes
	}
	if buttons&BFOk != 0 {
		return dialogs.CmdOK
	}
	if buttons&BFNo != 0 {
		return dialogs.CmdNo
	}
	if buttons&BFCancel != 0 {
		return dialogs.CmdCancel
	}
	return dialogs.CmdOK
}

// InputBox prompts the user for a string. The harness returns the
// default value.
func InputBox(title, label, def string, limit int) (string, bool) {
	return def, true
}
