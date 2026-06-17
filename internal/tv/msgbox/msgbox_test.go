package msgbox

import (
	"testing"

	"github.com/arturoeanton/go-turbo-pascal/internal/tv/dialogs"
)

func TestMessageBoxYes(t *testing.T) {
	if r := MessageBox("?", nil, BFYes|BFNo); r != dialogs.CmdYes {
		t.Errorf("got %d", r)
	}
}

func TestMessageBoxOk(t *testing.T) {
	if r := MessageBox("?", nil, BFOk|BFCancel); r != dialogs.CmdOK {
		t.Errorf("got %d", r)
	}
}

func TestInputBox(t *testing.T) {
	v, ok := InputBox("T", "L", "default", 10)
	if !ok {
		t.Error("ok")
	}
	if v != "default" {
		t.Errorf("got %q", v)
	}
}
