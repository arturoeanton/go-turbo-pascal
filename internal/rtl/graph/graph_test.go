package graph

import (
	"math"
	"testing"
)

func TestLine(t *testing.T) {
	d := New()
	d.Line(0, 0, 10, 0)
	if d.GetPixel(5, 0) != d.Color {
		t.Error("expected colored pixel at (5,0)")
	}
}

func TestCircle(t *testing.T) {
	d := New()
	d.Circle(50, 50, 10)
	if d.GetPixel(50, 40) != d.Color {
		t.Error("expected colored pixel at top of circle")
	}
}

func TestRect(t *testing.T) {
	d := New()
	d.Rectangle(0, 0, 100, 100)
	if d.GetPixel(0, 0) != d.Color {
		t.Error("corner not drawn")
	}
}

func TestBar(t *testing.T) {
	d := New()
	d.SetFillStyle(SolidFill, 12)
	d.Bar(0, 0, 10, 10)
	if d.GetPixel(5, 5) != 12 {
		t.Errorf("Bar should fill with 12, got %d", d.GetPixel(5, 5))
	}
}

func TestViewPort(t *testing.T) {
	d := New()
	d.SetViewPort(10, 10, 100, 100, true)
	if d.ViewPort.Left != 10 {
		t.Error("viewport not set")
	}
	d.ClearViewPort()
}

func TestPutPixel(t *testing.T) {
	d := New()
	d.PutPixel(0, 0, 5)
	if d.GetPixel(0, 0) != 5 {
		t.Error("pixel not set")
	}
}

func TestPages(t *testing.T) {
	d := New()
	d.SetActivePage(1)
	d.PutPixel(0, 0, 3)
	if d.GetPixel(0, 0) != 3 {
		t.Error("page 1 should have pixel")
	}
	d.SetActivePage(0)
	if d.GetPixel(0, 0) == 3 {
		t.Error("page 0 should not have pixel from page 1")
	}
}

func TestInitAndClose(t *testing.T) {
	d := New()
	if d.InitGraph(Detect, VGAHi, "") != Ok {
		t.Error("InitGraph failed")
	}
	d.CloseGraph()
	if d.GraphResult() != Ok {
		t.Error("Result should be Ok")
	}
}

func TestEllipse(t *testing.T) {
	d := New()
	d.Ellipse(100, 100, 0, 360, 20, 10)
	// Verify some pixels on the boundary were drawn.
	found := false
	for x := 100 - 20; x <= 100+20; x++ {
		if d.GetPixel(x, 100) == d.Color {
			found = true
			break
		}
	}
	if !found {
		t.Error("ellipse should draw on horizontal axis")
	}
}

func TestFillEllipse(t *testing.T) {
	d := New()
	d.SetFillStyle(SolidFill, 9)
	d.FillEllipse(100, 100, 20, 20)
	if d.GetPixel(100, 100) != 9 {
		t.Error("ellipse not filled")
	}
}

func TestImageRoundTrip(t *testing.T) {
	d := New()
	d.PutPixel(0, 0, 7)
	d.PutPixel(5, 5, 8)
	img := d.GetImage(0, 0, 9, 9)
	if img == nil {
		t.Fatal("GetImage returned nil")
	}
	if d.ImageSize(0, 0, 9, 9) != img.Size {
		t.Error("ImageSize mismatch")
	}
	d2 := New()
	d2.PutImage(0, 0, img, CopyPut)
	if d2.GetPixel(0, 0) != 7 {
		t.Error("PutImage did not restore pixel 7")
	}
}

func TestHash(t *testing.T) {
	d := New()
	h1 := d.Hash()
	d.PutPixel(0, 0, 5)
	h2 := d.Hash()
	if h1 == h2 {
		t.Error("hash should change after drawing")
	}
}

func TestOutText(t *testing.T) {
	d := New()
	d.OutTextXY(0, 0, "A")
	found := false
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if d.GetPixel(x, y) == d.Color {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("character not drawn anywhere")
	}
}

func TestFloodFill(t *testing.T) {
	d := New()
	d.Rectangle(10, 10, 20, 20)
	d.SetFillStyle(SolidFill, 5)
	d.FloodFill(15, 15, d.Color)
	if d.GetPixel(15, 15) != 5 {
		t.Error("FloodFill should fill interior")
	}
}

func TestErrorMsg(t *testing.T) {
	if GraphErrorMsg(Ok) != "OK" {
		t.Error("GraphErrorMsg")
	}
}

func TestPieSlice(t *testing.T) {
	d := New()
	d.PieSlice(50, 50, 0, 90, 30)
}

func TestSector(t *testing.T) {
	d := New()
	d.Sector(50, 50, 0, 90, 30, 30)
}

func TestBar3D(t *testing.T) {
	d := New()
	d.Bar3D(0, 0, 10, 10, 5, true)
	if d.GetPixel(5, 5) == 0 {
		t.Error("Bar3D should fill")
	}
}

func TestFillPoly(t *testing.T) {
	d := New()
	d.SetFillStyle(SolidFill, 4)
	poly := []Point{{10, 10}, {10, 20}, {20, 20}, {20, 10}}
	d.FillPoly(4, poly)
	if d.GetPixel(15, 15) != 4 {
		t.Error("FillPoly should fill interior")
	}
}

func TestRegisterDriver(t *testing.T) {
	d := New()
	if r := d.RegisterBGIDriver(nil); r != 0 {
		t.Errorf("RegisterBGIDriver: %d", r)
	}
	if r := d.RegisterBGIFont(nil); r != 0 {
		t.Errorf("RegisterBGIFont: %d", r)
	}
	if r := d.InstallUserDriver("", nil); r != 0 {
		t.Errorf("InstallUserDriver: %d", r)
	}
	if r := d.InstallUserFont(""); r != 0 {
		t.Errorf("InstallUserFont: %d", r)
	}
}

func TestArcAndSectorDrawing(t *testing.T) {
	d := New()
	d.Arc(50, 50, 0, 180, 20)
	if math.Abs(float64(d.GetMaxX())-float64(d.Width-1)) > 0 {
		t.Error("GetMaxX")
	}
}

func TestSetWriteMode(t *testing.T) {
	d := New()
	d.SetWriteMode(XorPut)
	d.PutPixel(0, 0, 5)
	d.PutPixel(0, 0, 5)
	if d.GetPixel(0, 0) != 0 {
		t.Error("XOR should cancel")
	}
}

func TestSetRGBPalette(t *testing.T) {
	d := New()
	d.SetRGBPalette(1, 100, 50, 25)
	var p PaletteType
	d.GetPalette(&p)
	if p.Colors[1].R != 100 {
		t.Error("RGB palette not set")
	}
}
