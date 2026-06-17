// Package graph3 implements the Graph3 unit runtime. Graph3 is the
// legacy graphics unit from Turbo Pascal 3. BPGo implements it as a
// shim over the modern Graph unit, mapping the older procedure names
// (Plot, LineTo3, MoveTo3) to their Graph equivalents. The legacy
// coordinate system uses viewport coordinates centered at (0,0) with
// Y going up; the shim converts to the BGI coordinate system where Y
// goes down.
package graph3

import "github.com/arturoeanton/go-turbo-pascal/internal/rtl/graph"

var (
	GraphDriver int
	GraphMode   int
	GraphColor  byte
)

// InitGraph is a thin wrapper over the modern Graph unit that sets
// the device into 320x200 CGA mode.
func InitGraph() {
	GraphDriver = graph.CGA
	GraphMode = graph.CGAC1
}

// CloseGraph is a no-op shim; the modern Graph unit manages device
// lifetime.
func CloseGraph() {}

// Plot sets a single pixel in viewport coordinates.
func Plot(x, y int) {
	// Viewport conversion: (x+160, 100-y).
	g.PutPixel(x+160, 100-y, GraphColor)
}

var g = graph.New()

// MoveTo3 updates the cursor in viewport coordinates.
func MoveTo3(x, y int) { g.MoveTo(x+160, 100-y) }

// LineTo3 draws a line to a viewport coordinate from the current
// cursor.
func LineTo3(x, y int) { g.LineTo(x+160, 100-y) }
