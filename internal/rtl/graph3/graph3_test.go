package graph3

import "testing"

func TestGraph3Basic(t *testing.T) {
	InitGraph()
	defer CloseGraph()
	GraphColor = 7
	Plot(0, 0)
	MoveTo3(0, 0)
	LineTo3(10, 10)
	if GraphDriver != 1 {
		t.Errorf("GraphDriver: %d", GraphDriver)
	}
}
