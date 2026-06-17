package dialogs

import (
	"testing"

	"github.com/arturoeanton/go-turbo-pascal/internal/tv/views"
)

func TestTWindowInit(t *testing.T) {
	w := (&TWindow{}).Init(views.TRect{views.TPoint{0, 0}, views.TPoint{20, 20}}, "Test", 1)
	if w.Title != "Test" {
		t.Error("Title")
	}
	if w.Number != 1 {
		t.Error("Number")
	}
	if w.Frame == nil {
		t.Error("Frame")
	}
}

func TestTDialogInit(t *testing.T) {
	d := (&TDialog{}).Init(views.TRect{views.TPoint{0, 0}, views.TPoint{40, 10}}, "Confirm")
	if d.State&views.StateModal == 0 {
		t.Error("Modal flag")
	}
}

func TestEndModal(t *testing.T) {
	d := (&TDialog{}).Init(views.TRect{views.TPoint{0, 0}, views.TPoint{40, 10}}, "X")
	d.EndModal(CmdOK)
	if d.State&views.StateModal != 0 {
		t.Error("Modal should be cleared")
	}
	if d.Command != CmdOK {
		t.Error("Command")
	}
}

func TestStandardCommands(t *testing.T) {
	if CmdOK == CmdCancel {
		t.Error("commands should differ")
	}
}
