package colorsel

import (
	"testing"

	"github.com/arturoeanton/go-turbo-pascal/internal/tv/views"
)

func TestColorSelectorInit(t *testing.T) {
	s := (&TColorSelector{}).Init(views.TRect{views.TPoint{0, 0}, views.TPoint{16, 1}})
	if s.Palette[0] != 0 {
		t.Error("Palette")
	}
}

func TestColorSelectorSelect(t *testing.T) {
	s := (&TColorSelector{}).Init(views.TRect{views.TPoint{0, 0}, views.TPoint{16, 1}})
	s.Select(5)
	if s.Selected() != 5 {
		t.Error("Selected")
	}
}

func TestColorSelectorBounds(t *testing.T) {
	s := (&TColorSelector{}).Init(views.TRect{views.TPoint{0, 0}, views.TPoint{16, 1}})
	s.Select(-1)
	if s.Selected() != 0 {
		t.Error("negative select should be ignored")
	}
	s.Select(100)
	if s.Selected() != 0 {
		t.Error("large select should be ignored")
	}
}

func TestColorDisplay(t *testing.T) {
	d := (&TColorDisplay{}).Init(views.TRect{views.TPoint{0, 0}, views.TPoint{10, 1}})
	d.SetColor(7)
	if d.Color != 7 {
		t.Error("Color")
	}
}
