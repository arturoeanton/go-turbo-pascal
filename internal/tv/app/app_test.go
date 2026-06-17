package app

import (
	"testing"

	"github.com/arturoeanton/go-turbo-pascal/internal/tv/dialogs"
	"github.com/arturoeanton/go-turbo-pascal/internal/tv/views"
)

func TestTProgramInit(t *testing.T) {
	p := (&TProgram{}).Init(views.TRect{views.TPoint{0, 0}, views.TPoint{80, 25}})
	if p.Running {
		t.Error("should not be running yet")
	}
	p.Run()
	if !p.Running {
		t.Error("should be running after Run")
	}
	p.Quit()
	if p.Running {
		t.Error("should be stopped after Quit")
	}
}

func TestExecute(t *testing.T) {
	p := (&TProgram{}).Init(views.TRect{views.TPoint{0, 0}, views.TPoint{80, 25}})
	d := (&dialogs.TDialog{}).Init(views.TRect{views.TPoint{0, 0}, views.TPoint{40, 10}}, "X")
	if r := p.Execute(d); r != dialogs.CmdCancel {
		t.Errorf("Execute: %d", r)
	}
}

func TestTApplicationInit(t *testing.T) {
	a := (&TApplication{}).Init(views.TRect{views.TPoint{0, 0}, views.TPoint{80, 25}})
	if a.Desktop == nil {
		t.Error("Desktop not created")
	}
	a.InitEvents()
	a.DoneEvents()
}

func TestNewDesktop(t *testing.T) {
	d := NewDesktop(views.TRect{views.TPoint{0, 0}, views.TPoint{80, 25}})
	if d.State&views.StateVisible == 0 {
		t.Error("Desktop not visible")
	}
}
