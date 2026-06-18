package app

import (
	"testing"

	"github.com/arturoeanton/go-turbo-pascal/internal/tv/dialogs"
	"github.com/arturoeanton/go-turbo-pascal/internal/tv/views"
)

func TestTProgramInit(t *testing.T) {
	p := (&TProgram{}).Init(views.TRect{A: views.TPoint{X: 0, Y: 0}, B: views.TPoint{X: 80, Y: 25}})
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
	p := (&TProgram{}).Init(views.TRect{A: views.TPoint{X: 0, Y: 0}, B: views.TPoint{X: 80, Y: 25}})
	d := (&dialogs.TDialog{}).Init(views.TRect{A: views.TPoint{X: 0, Y: 0}, B: views.TPoint{X: 40, Y: 10}}, "X")
	if r := p.Execute(d); r != dialogs.CmdCancel {
		t.Errorf("Execute: %d", r)
	}
}

func TestTApplicationInit(t *testing.T) {
	a := (&TApplication{}).Init(views.TRect{A: views.TPoint{X: 0, Y: 0}, B: views.TPoint{X: 80, Y: 25}})
	if a.Desktop == nil {
		t.Error("Desktop not created")
	}
	a.InitEvents()
	a.DoneEvents()
}

func TestNewDesktop(t *testing.T) {
	d := NewDesktop(views.TRect{A: views.TPoint{X: 0, Y: 0}, B: views.TPoint{X: 80, Y: 25}})
	if d.State&views.StateVisible == 0 {
		t.Error("Desktop not visible")
	}
}
