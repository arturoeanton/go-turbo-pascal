// Package colorsel implements the Turbo Vision ColorSel unit. The
// unit provides the colour selector dialog used by the IDE to edit
// the palette of the TV application.
package colorsel

import "github.com/arturoeanton/go-turbo-pascal/internal/tv/views"

// TColorSelector is the colour grid.
type TColorSelector struct {
	views.TView
	Palette [16]byte
	Sel     int
}

// Init creates a new colour selector.
func (s *TColorSelector) Init(bounds views.TRect) *TColorSelector {
	s.TView.Init(bounds)
	for i := 0; i < 16; i++ {
		s.Palette[i] = byte(i)
	}
	return s
}

// Selected returns the currently selected colour index.
func (s *TColorSelector) Selected() int {
	return s.Sel
}

// Select sets the selected colour.
func (s *TColorSelector) Select(i int) {
	if i < 0 || i >= 16 {
		return
	}
	s.Lock()
	defer s.Unlock()
	s.Sel = i
}

// TColorDisplay is the preview area.
type TColorDisplay struct {
	views.TView
	Color byte
}

// Init creates a new colour display.
func (d *TColorDisplay) Init(bounds views.TRect) *TColorDisplay {
	d.TView.Init(bounds)
	return d
}

// SetColor sets the displayed colour.
func (d *TColorDisplay) SetColor(c byte) {
	d.Lock()
	defer d.Unlock()
	d.Color = c
}
