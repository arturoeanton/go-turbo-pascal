package menus

import (
	"testing"

	"github.com/arturoeanton/go-turbo-pascal/internal/tv/views"
)

func TestNewMenuItem(t *testing.T) {
	it := NewMenuItem("~F~ile", 100)
	if it.Text != "~F~ile" {
		t.Error("Text")
	}
	if it.Command != 100 {
		t.Error("Command")
	}
}

func TestTMenu(t *testing.T) {
	m := NewMenu(
		NewMenuItem("Open", 1),
		NewMenuItem("Save", 2),
	)
	if len(m.Items) != 2 {
		t.Error("Items")
	}
}

func TestMenuBar(t *testing.T) {
	m := NewMenu(NewMenuItem("File", 1))
	bar := NewMenuBar(m)
	if bar.Menu != m {
		t.Error("Menu")
	}
}

func TestMenuBox(t *testing.T) {
	m := NewMenu(NewMenuItem("Cut", 1))
	b := NewMenuBox(m, views.TRect{A: views.TPoint{X: 0, Y: 0}, B: views.TPoint{X: 10, Y: 10}})
	if b.Frame == nil {
		t.Error("Frame")
	}
}

func TestStatusLine(t *testing.T) {
	def := &TStatusDef{Min: 0, Max: 100, Items: []*TStatusItem{
		TStatusItemFromText("Help", 1),
		TStatusItemFromText("Save", 2),
	}}
	line := NewStatusLine(views.TRect{A: views.TPoint{X: 0, Y: 0}, B: views.TPoint{X: 80, Y: 1}}, def)
	line.Update(1)
	if len(line.Items) != 1 {
		t.Errorf("Items: %d", len(line.Items))
	}
	if line.Items[0].Text != "Help" {
		t.Error("Text")
	}
}

func TestStatusItemDisabled(t *testing.T) {
	it := NewMenuItem("X", 1)
	it.Disabled = true
	if !it.Disabled {
		t.Error("Disabled")
	}
}
