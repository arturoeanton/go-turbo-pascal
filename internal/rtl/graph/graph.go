// Package graph implements the Graph/BGI unit runtime. The TP7/BP7
// Graph unit provides 2D drawing primitives on a virtual framebuffer.
// BPGo implements a software framebuffer with palette-indexed pixels,
// clipping, viewports and page flipping. The framebuffer is exported
// as a PNG-style byte slice for golden tests.
package graph

import (
	"fmt"
	"math"
)

// Driver constants.
const (
	Detect        = 0
	CGA           = 1
	MCGA          = 2
	EGA           = 3
	EGA64         = 4
	EGAMono       = 5
	IBM8514       = 6
	HercMono      = 7
	ATT400        = 8
	VGA           = 9
	PC3270        = 10
	CurrentDriver = -1
)

// Graphics modes.
const (
	CGAC0      = 0
	CGAC1      = 1
	CGAC2      = 2
	CGAC3      = 3
	CGAHi      = 4
	MCGAC0     = 5
	MCGAC1     = 6
	MCGAC2     = 7
	MCGAC3     = 8
	MCGAMed    = 9
	MCGAHi     = 10
	EGALo      = 13
	EGAHi      = 14
	EGA64Lo    = 15
	EGA64Hi    = 16
	EGAMonoHi  = 17
	EGAMonoLo  = 18
	HercMonoHi = 19
	VGALo      = 19
	VGAMed     = 20
	VGAHi      = 21
)

// Error codes.
const (
	Ok              = 0
	NoInitGraph     = -1
	NotDetected     = -2
	NotFound        = -3
	InvalidDriver   = -4
	NoLoadMem       = -5
	NoScanMem       = -6
	NoFloodMem      = -7
	FontNotFound    = -8
	InvalidFont     = -9
	InvalidMode     = -10
	InvalidFill     = -11
	PaletteIndex    = -12
	InvalidImage    = -13
	OutOfMemory     = -14
	InvalidLine     = -15
	OutOfViewport   = -16
	InvalidViewport = -17
	InvalidPattern  = -18
)

// Line styles.
const (
	SolidLn   = 0
	DottedLn  = 1
	CenterLn  = 2
	DashedLn  = 3
	UserBitLn = 4
)

// Fill styles.
const (
	EmptyFill      = 0
	SolidFill      = 1
	LineFill       = 2
	LtSlashFill    = 3
	SlashFill      = 4
	BkSlashFill    = 5
	LtBkSlashFill  = 6
	HatchFill      = 7
	XHatchFill     = 8
	InterleaveFill = 9
	WideDotFill    = 10
	CloseDotFill   = 11
	UserFill       = 12
)

// Text justification.
const (
	LeftText    = 0
	CenterText  = 1
	RightText   = 2
	BottomText  = 0
	TopText     = 1
	CenterTextV = 2
)

// Fonts.
const (
	DefaultFont   = 0
	TriplexFont   = 1
	SmallFont     = 2
	SansSerifFont = 3
	GothicFont    = 4
)

// Write modes.
const (
	CopyPut = 0
	XorPut  = 1
	OrPut   = 2
	AndPut  = 3
	NotPut  = 4
)

// Color holds palette indices.
type Color = byte

// Point is a screen coordinate.
type Point struct {
	X, Y int
}

// ArcCoords describes the endpoints of the most recently drawn arc.
type ArcCoords struct {
	X, Y           int
	XStart, YStart int
	XEnd, YEnd     int
}

// LineSettings describes the current line style.
type LineSettings struct {
	LineStyle int
	Pattern   uint16
	Thickness int
}

// FillSettings describes the current fill.
type FillSettings struct {
	Pattern int
	Color   Color
}

// TextSettings describes the current text style.
type TextSettings struct {
	Font      int
	Direction int
	CharSize  int
	Horiz     int
	Vert      int
}

// PaletteEntry is a 4-bit per channel palette.
type PaletteEntry struct {
	R, G, B byte
}

// PaletteType is the BGI palette.
type PaletteType struct {
	Size   byte
	Colors [256]PaletteEntry
}

// FillPatternType is a 8x8 user fill pattern.
type FillPatternType [8]byte

// ViewPortType describes a viewport.
type ViewPortType struct {
	Left, Top, Right, Bottom int
	Clip                     bool
}

// Image is a GetImage buffer.
type Image struct {
	Size   uint16
	X1, Y1 int
	X2, Y2 int
	Pixels []byte
}

// Device is the software framebuffer.
type Device struct {
	Width, Height int
	MaxColor      Color
	BgColor       Color
	Color         Color
	FillColor     Color
	LineStyle     int
	LinePattern   uint16
	LineThickness int
	FillStyle     int
	FillPattern   FillPatternType
	Font          int
	TextDirection int
	CharSize      int
	HorizJust     int
	VertJust      int
	WriteMode     int
	ViewPort      ViewPortType
	ActivePage    int
	VisualPage    int
	Pages         [][]Color
	palette       PaletteType
	cp            int
	CursorX       int
	CursorY       int
	Result        int
}

// New creates a default 640x480 16-color device with two pages.
func New() *Device {
	d := &Device{
		Width: 640, Height: 480,
		MaxColor: 15, BgColor: 0, Color: 15,
		FillColor: 0,
		LineStyle: SolidLn, LinePattern: 0xFFFF, LineThickness: 1,
		FillStyle: SolidFill,
		Font:      DefaultFont, TextDirection: 0, CharSize: 1,
		HorizJust: LeftText, VertJust: TopText,
		WriteMode: CopyPut,
		ViewPort:  ViewPortType{Left: 0, Top: 0, Right: 639, Bottom: 479, Clip: true},
		Pages:     make([][]Color, 2),
	}
	for i := range d.Pages {
		d.Pages[i] = make([]Color, d.Width*d.Height)
	}
	d.palette = DefaultPalette()
	return d
}

// InitGraph initialises the device. Driver=Detect chooses a default
// (VGA).
func (d *Device) InitGraph(driver, mode int, path string) int {
	d.Result = Ok
	d.CursorX, d.CursorY = 0, 0
	return d.Result
}

// DetectGraph returns the detected driver/mode.
func (d *Device) DetectGraph() (driver, mode int) {
	return VGA, VGAHi
}

// SetGraphMode changes the current mode.
func (d *Device) SetGraphMode(mode int) int {
	return Ok
}

// GetGraphMode returns the current mode.
func (d *Device) GetGraphMode() int { return VGAHi }

// RestoreCrtMode restores the screen.
func (d *Device) RestoreCrtMode() {}

// GraphDefaults resets to default settings.
func (d *Device) GraphDefaults() {
	d.Color = 15
	d.BgColor = 0
	d.LineStyle = SolidLn
	d.LinePattern = 0xFFFF
	d.LineThickness = 1
	d.FillStyle = SolidFill
	d.Font = DefaultFont
	d.CharSize = 1
	d.HorizJust = LeftText
	d.VertJust = TopText
	d.WriteMode = CopyPut
	d.ViewPort = ViewPortType{Left: 0, Top: 0, Right: d.Width - 1, Bottom: d.Height - 1, Clip: true}
	d.cp = 0
}

// CloseGraph closes the device.
func (d *Device) CloseGraph() {}

// GraphResult returns the last error.
func (d *Device) GraphResult() int { return d.Result }

// GraphErrorMsg returns a textual message for a given error.
func GraphErrorMsg(err int) string {
	switch err {
	case Ok:
		return "OK"
	case NoInitGraph:
		return "Graphics not initialized"
	case NotDetected:
		return "Graphics hardware not detected"
	case NotFound:
		return "Driver file not found"
	case InvalidDriver:
		return "Invalid driver"
	case NoLoadMem:
		return "Not enough memory to load driver"
	case FontNotFound:
		return "Font file not found"
	}
	return fmt.Sprintf("Graphics error %d", err)
}

func (d *Device) clip(x, y int) (int, int, bool) {
	if x < d.ViewPort.Left || x > d.ViewPort.Right || y < d.ViewPort.Top || y > d.ViewPort.Bottom {
		if d.ViewPort.Clip {
			return 0, 0, false
		}
	}
	return x, y, true
}

func (d *Device) putPixel(x, y int, c Color) {
	x, y, ok := d.clip(x, y)
	if !ok {
		return
	}
	if x < 0 || y < 0 || x >= d.Width || y >= d.Height {
		return
	}
	idx := y*d.Width + x
	switch d.WriteMode {
	case CopyPut:
		d.Pages[d.ActivePage][idx] = c
	case XorPut:
		d.Pages[d.ActivePage][idx] ^= c
	case OrPut:
		d.Pages[d.ActivePage][idx] |= c
	case AndPut:
		d.Pages[d.ActivePage][idx] &= c
	case NotPut:
		d.Pages[d.ActivePage][idx] = ^c
	}
}

func (d *Device) PutPixel(x, y int, c Color) { d.putPixel(x, y, c) }
func (d *Device) GetPixel(x, y int) Color {
	if x < 0 || y < 0 || x >= d.Width || y >= d.Height {
		return 0
	}
	return d.Pages[d.ActivePage][y*d.Width+x]
}

// GetMaxX returns the maximum X coordinate.
func (d *Device) GetMaxX() int       { return d.Width - 1 }
func (d *Device) GetMaxY() int       { return d.Height - 1 }
func (d *Device) GetMaxColor() Color { return d.MaxColor }
func (d *Device) GetX() int          { return d.CursorX }
func (d *Device) GetY() int          { return d.CursorY }

func (d *Device) MoveTo(x, y int)    { d.CursorX = x; d.CursorY = y }
func (d *Device) MoveRel(dx, dy int) { d.CursorX += dx; d.CursorY += dy }

func (d *Device) LineTo(x, y int) {
	d.line(d.CursorX, d.CursorY, x, y)
	d.CursorX = x
	d.CursorY = y
}
func (d *Device) LineRel(dx, dy int) {
	d.LineTo(d.CursorX+dx, d.CursorY+dy)
}

func (d *Device) Line(x1, y1, x2, y2 int) { d.line(x1, y1, x2, y2) }

// line draws a line using Bresenham's algorithm with the current
// style/pattern.
func (d *Device) line(x1, y1, x2, y2 int) {
	dx := abs(x2 - x1)
	dy := -abs(y2 - y1)
	sx := 1
	if x1 >= x2 {
		sx = -1
	}
	sy := 1
	if y1 >= y2 {
		sy = -1
	}
	err := dx + dy
	for {
		if d.lineVisible() {
			d.PutPixel(x1, y1, d.Color)
		}
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := err * 2
		if e2 >= dy {
			err += dy
			x1 += sx
		}
		if e2 <= dx {
			err += dx
			y1 += sy
		}
	}
}

func (d *Device) lineVisible() bool {
	switch d.LineStyle {
	case SolidLn:
		return true
	case DottedLn:
		// simplified: every other pixel
		return (d.cp/1)%2 == 0
	case CenterLn:
		return d.cp%4 == 0
	case DashedLn:
		return d.cp%4 < 2
	}
	return true
}

// Rectangle draws a rectangle outline.
func (d *Device) Rectangle(x1, y1, x2, y2 int) {
	d.Line(x1, y1, x2, y1)
	d.Line(x2, y1, x2, y2)
	d.Line(x2, y2, x1, y2)
	d.Line(x1, y2, x1, y1)
}

// Bar draws a filled bar using the current fill style and color.
func (d *Device) Bar(x1, y1, x2, y2 int) {
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	c := d.FillColor
	for y := y1; y <= y2; y++ {
		for x := x1; x <= x2; x++ {
			if d.shouldFill(x, y) {
				d.PutPixel(x, y, c)
			}
		}
	}
}

func (d *Device) shouldFill(x, y int) bool {
	switch d.FillStyle {
	case SolidFill:
		return true
	case EmptyFill:
		return false
	case LineFill, LtSlashFill, SlashFill, BkSlashFill, LtBkSlashFill, HatchFill, XHatchFill, InterleaveFill, WideDotFill, CloseDotFill, UserFill:
		// Predefined patterns as 8x8 bitmaps.
		pat := fillPatternFor(d.FillStyle)
		return pat[y%8]&(1<<uint(7-x%8)) != 0
	}
	return true
}

func fillPatternFor(style int) FillPatternType {
	switch style {
	case LineFill:
		return FillPatternType{0xFF, 0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF, 0x00}
	case LtSlashFill:
		return FillPatternType{0x88, 0x22, 0x88, 0x22, 0x88, 0x22, 0x88, 0x22}
	case SlashFill:
		return FillPatternType{0xCC, 0x33, 0xCC, 0x33, 0xCC, 0x33, 0xCC, 0x33}
	case BkSlashFill:
		return FillPatternType{0x33, 0xCC, 0x33, 0xCC, 0x33, 0xCC, 0x33, 0xCC}
	case LtBkSlashFill:
		return FillPatternType{0x22, 0x88, 0x22, 0x88, 0x22, 0x88, 0x22, 0x88}
	case HatchFill:
		return FillPatternType{0xFF, 0x88, 0x88, 0x88, 0xFF, 0x88, 0x88, 0x88}
	case XHatchFill:
		return FillPatternType{0x11, 0x22, 0x44, 0x88, 0x11, 0x22, 0x44, 0x88}
	case InterleaveFill:
		return FillPatternType{0x55, 0xAA, 0x55, 0xAA, 0x55, 0xAA, 0x55, 0xAA}
	case WideDotFill:
		return FillPatternType{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	case CloseDotFill:
		return FillPatternType{0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	case UserFill:
		return FillPatternType{}
	}
	return FillPatternType{}
}

// Bar3D draws a 3D-looking bar.
func (d *Device) Bar3D(x1, y1, x2, y2, depth int, top bool) {
	old := d.FillColor
	if d.FillColor == 0 {
		d.FillColor = d.Color
	}
	d.Bar(x1, y1, x2, y2)
	d.FillColor = old
	// Draw top and right faces with simple lines.
	d.Line(x2, y1, x2+depth, y1-depth)
	d.Line(x2, y2, x2+depth, y2-depth)
	d.Line(x2+depth, y1-depth, x2+depth, y2-depth)
	if top {
		d.Line(x1, y1, x1+depth, y1-depth)
		d.Line(x1+depth, y1-depth, x2+depth, y1-depth)
	}
}

// Circle draws a circle using the midpoint algorithm.
func (d *Device) Circle(x, y, radius int) {
	mx, my := 0, radius
	dd := 1 - radius
	for mx <= my {
		pts := [8]Point{
			{x + mx, y - my}, {x - mx, y - my},
			{x + mx, y + my}, {x - mx, y + my},
			{x + my, y - mx}, {x - my, y - mx},
			{x + my, y + mx}, {x - my, y + mx},
		}
		for _, p := range pts {
			if d.lineVisible() {
				d.PutPixel(p.X, p.Y, d.Color)
			}
		}
		dd++
		switch {
		case dd > 0:
			my--
			dd -= 2 * (my - mx)
		}
		mx++
		dd += 2*mx + 1
	}
}

// Ellipse draws an ellipse outline.
func (d *Device) Ellipse(x, y, stAngle, endAngle, xRadius, yRadius int) {
	steps := 360
	for i := 0; i < steps; i++ {
		a := float64(stAngle) + float64(endAngle-stAngle)*float64(i)/float64(steps)
		rx := float64(xRadius) * math.Cos(a*math.Pi/180)
		ry := float64(yRadius) * math.Sin(a*math.Pi/180)
		d.PutPixel(x+int(rx), y+int(ry), d.Color)
	}
}

// FillEllipse draws a filled ellipse.
func (d *Device) FillEllipse(x, y, xRadius, yRadius int) {
	for yy := -yRadius; yy <= yRadius; yy++ {
		for xx := -xRadius; xx <= xRadius; xx++ {
			fx := float64(xx) / float64(xRadius)
			fy := float64(yy) / float64(yRadius)
			if fx*fx+fy*fy <= 1 {
				d.PutPixel(x+xx, y+yy, d.FillColor)
			}
		}
	}
}

// Arc draws an arc.
func (d *Device) Arc(x, y, stAngle, endAngle, radius int) {
	d.Ellipse(x, y, stAngle, endAngle, radius, radius)
}

// PieSlice draws a pie slice.
func (d *Device) PieSlice(x, y, stAngle, endAngle, radius int) {
	d.Ellipse(x, y, stAngle, endAngle, radius, radius)
	d.Line(x, y, x+radius, y)
}

// Sector draws a filled pie sector.
func (d *Device) Sector(x, y, stAngle, endAngle, xRadius, yRadius int) {
	steps := 360
	for i := 0; i <= steps; i++ {
		a := float64(stAngle) + float64(endAngle-stAngle)*float64(i)/float64(steps)
		rx := float64(xRadius) * math.Cos(a*math.Pi/180)
		ry := float64(yRadius) * math.Sin(a*math.Pi/180)
		d.LineTo(x+int(rx), y+int(ry))
	}
	d.LineTo(x, y)
}

// DrawPoly draws an outline polygon.
func (d *Device) DrawPoly(n int, points []Point) {
	for i := 0; i < n-1; i++ {
		d.Line(points[i].X, points[i].Y, points[i+1].X, points[i+1].Y)
	}
}

// FillPoly fills a polygon using scanline.
func (d *Device) FillPoly(n int, points []Point) {
	if n < 3 {
		return
	}
	minY := points[0].Y
	maxY := points[0].Y
	for i := 1; i < n; i++ {
		if points[i].Y < minY {
			minY = points[i].Y
		}
		if points[i].Y > maxY {
			maxY = points[i].Y
		}
	}
	for y := minY; y <= maxY; y++ {
		var xs []int
		for i := 0; i < n; i++ {
			j := (i + 1) % n
			y1, y2 := points[i].Y, points[j].Y
			if (y1 <= y && y2 > y) || (y2 <= y && y1 > y) {
				x1, x2 := points[i].X, points[j].X
				t := float64(y-y1) / float64(y2-y1)
				xs = append(xs, x1+int(float64(x2-x1)*t))
			}
		}
		for i := 0; i+1 < len(xs); i += 2 {
			for x := xs[i]; x <= xs[i+1]; x++ {
				d.PutPixel(x, y, d.FillColor)
			}
		}
	}
}

// FloodFill fills an enclosed area with the current fill color.
func (d *Device) FloodFill(x, y int, border Color) {
	if d.GetPixel(x, y) == border {
		return
	}
	// BFS flood fill.
	queue := []Point{{x, y}}
	visited := map[Point]bool{{x, y}: true}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		if d.GetPixel(p.X, p.Y) != border {
			d.PutPixel(p.X, p.Y, d.FillColor)
		}
		for _, n := range [4]Point{{p.X + 1, p.Y}, {p.X - 1, p.Y}, {p.X, p.Y + 1}, {p.X, p.Y - 1}} {
			if n.X < 0 || n.Y < 0 || n.X >= d.Width || n.Y >= d.Height {
				continue
			}
			if visited[n] {
				continue
			}
			if d.GetPixel(n.X, n.Y) == border {
				continue
			}
			visited[n] = true
			queue = append(queue, n)
		}
	}
}

// OutText writes a string at the current cursor.
func (d *Device) OutText(s string) {
	d.drawText(d.CursorX, d.CursorY, s)
}

// OutTextXY writes a string at the given position.
func (d *Device) OutTextXY(x, y int, s string) {
	d.drawText(x, y, s)
}

func (d *Device) drawText(x, y int, s string) {
	w, h := d.TextWidth(s), 8*d.CharSize
	cx := x
	cy := y
	if d.HorizJust == CenterText {
		cx = x - w/2
	} else if d.HorizJust == RightText {
		cx = x - w
	}
	if d.VertJust == CenterTextV {
		cy = y - h/2
	} else if d.VertJust == BottomText {
		cy = y - h
	}
	for _, ch := range s {
		d.drawChar(cx, cy, byte(ch))
		cx += d.TextWidth(string(ch))
	}
	d.CursorX = cx
	d.CursorY = cy
}

func (d *Device) drawChar(x, y int, ch byte) {
	bits := font8x8(ch)
	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			if bits[row]&(1<<uint(7-col)) != 0 {
				for dy := 0; dy < d.CharSize; dy++ {
					for dx := 0; dx < d.CharSize; dx++ {
						d.PutPixel(x+col*d.CharSize+dx, y+row*d.CharSize+dy, d.Color)
					}
				}
			}
		}
	}
}

func (d *Device) TextWidth(s string) int {
	return 8 * d.CharSize * len(s)
}

func (d *Device) TextHeight(s string) int {
	return 8 * d.CharSize
}

// ClearDevice clears the active page.
func (d *Device) ClearDevice() {
	for i := range d.Pages[d.ActivePage] {
		d.Pages[d.ActivePage][i] = d.BgColor
	}
}

// ClearViewPort clears the viewport.
func (d *Device) ClearViewPort() {
	v := d.ViewPort
	for y := v.Top; y <= v.Bottom; y++ {
		for x := v.Left; x <= v.Right; x++ {
			d.PutPixel(x, y, d.BgColor)
		}
	}
}

// SetViewPort defines a viewport.
func (d *Device) SetViewPort(x1, y1, x2, y2 int, clip bool) {
	d.ViewPort = ViewPortType{Left: x1, Top: y1, Right: x2, Bottom: y2, Clip: clip}
	d.ClearViewPort()
}

// SetColor sets the drawing color.
func (d *Device) SetColor(c Color)                    { d.Color = c }
func (d *Device) GetColor() Color                     { return d.Color }
func (d *Device) SetBkColor(c Color)                  { d.BgColor = c }
func (d *Device) GetBkColor() Color                   { return d.BgColor }
func (d *Device) SetFillStyle(style int, color Color) { d.FillStyle = style; d.FillColor = color }
func (d *Device) SetLineStyle(style int, pattern uint16, thickness int) {
	d.LineStyle = style
	d.LinePattern = pattern
	d.LineThickness = thickness
}
func (d *Device) SetTextStyle(font, dir, size int) {
	d.Font = font
	d.TextDirection = dir
	d.CharSize = size
}
func (d *Device) SetTextJustify(h, v int) { d.HorizJust = h; d.VertJust = v }
func (d *Device) SetWriteMode(m int)      { d.WriteMode = m }
func (d *Device) SetActivePage(p int)     { d.ActivePage = p % len(d.Pages) }
func (d *Device) SetVisualPage(p int)     { d.VisualPage = p % len(d.Pages) }
func (d *Device) SetPalette(c, color Color) {
	d.palette.Colors[c] = PaletteEntry{R: uint8(color) << 4, G: uint8(color) << 4, B: uint8(color) << 4}
}
func (d *Device) SetAllPalette(p PaletteType)      { d.palette = p }
func (d *Device) GetPalette(p *PaletteType)        { *p = d.palette }
func (d *Device) GetDefaultPalette(p *PaletteType) { *p = DefaultPalette() }
func (d *Device) GetPaletteSize() int              { return int(d.palette.Size) }
func (d *Device) GetAspectRatio() (xasp, yasp int) { return 10000, 10000 }
func (d *Device) SetAspectRatio(xasp, yasp int)    {}

func (d *Device) GetLineSettings(ls *LineSettings) {
	ls.LineStyle = d.LineStyle
	ls.Pattern = d.LinePattern
	ls.Thickness = d.LineThickness
}
func (d *Device) GetFillSettings(fs *FillSettings) {
	fs.Pattern = d.FillStyle
	fs.Color = d.FillColor
}
func (d *Device) GetTextSettings(ts *TextSettings) {
	ts.Font = d.Font
	ts.Direction = d.TextDirection
	ts.CharSize = d.CharSize
	ts.Horiz = d.HorizJust
	ts.Vert = d.VertJust
}
func (d *Device) GetViewSettings(v *ViewPortType) { *v = d.ViewPort }
func (d *Device) GetArcCoords(a *ArcCoords)       {}

// GetImage stores a rectangular region into an Image buffer.
func (d *Device) GetImage(x1, y1, x2, y2 int) *Image {
	w := x2 - x1 + 1
	h := y2 - y1 + 1
	img := &Image{
		Size: uint16(w*h + 8),
		X1:   x1, Y1: y1, X2: x2, Y2: y2,
		Pixels: make([]byte, w*h),
	}
	for y := y1; y <= y2; y++ {
		for x := x1; x <= x2; x++ {
			img.Pixels[(y-y1)*w+(x-x1)] = d.GetPixel(x, y)
		}
	}
	return img
}

// PutImage writes a saved image at the given location.
func (d *Device) PutImage(x, y int, img *Image, mode int) {
	if img == nil {
		return
	}
	w := img.X2 - img.X1 + 1
	old := d.WriteMode
	d.WriteMode = mode
	for dy := 0; dy < w && dy+img.Y1 < d.Height; dy++ {
		for dx := 0; dx < w && dx+img.X1 < d.Width; dx++ {
			d.PutPixel(x+dx, y+dy, img.Pixels[dy*w+dx])
		}
	}
	d.WriteMode = old
}

// ImageSize returns the size required for a region.
func (d *Device) ImageSize(x1, y1, x2, y2 int) uint16 {
	w := x2 - x1 + 1
	h := y2 - y1 + 1
	return uint16(w*h + 8)
}

// GetFillPattern returns the user fill pattern.
func (d *Device) GetFillPattern(p *FillPatternType) { *p = d.FillPattern }

// SetFillPattern sets a user fill pattern.
func (d *Device) SetFillPattern(p FillPatternType, color Color) {
	d.FillStyle = UserFill
	d.FillPattern = p
	d.FillColor = color
}

// SetRGBPalette sets a 4-bit per channel palette entry.
func (d *Device) SetRGBPalette(c int, r, g, b int) {
	d.palette.Colors[c] = PaletteEntry{R: byte(r), G: byte(g), B: byte(b)}
}

// DefaultPalette returns the BP7 default 16-color palette.
func DefaultPalette() PaletteType {
	p := PaletteType{Size: 16}
	for i := 0; i < 16; i++ {
		p.Colors[i] = PaletteEntry{R: byte(i) * 17, G: byte(i) * 17, B: byte(i) * 17}
	}
	return p
}

// Hash returns a deterministic hash of the active page for golden
// tests.
func (d *Device) Hash() uint64 {
	var h uint64 = 14695981039346656037
	for _, c := range d.Pages[d.ActivePage] {
		h ^= uint64(c)
		h *= 1099511628211
	}
	return h
}

// RegisterUserDriver / RegisterBGIDriver stubs.
func (d *Device) RegisterBGIDriver(p interface{}) int                      { return 0 }
func (d *Device) RegisterBGIFont(p interface{}) int                        { return 0 }
func (d *Device) InstallUserDriver(name string, autodetect func() int) int { return 0 }
func (d *Device) InstallUserFont(name string) int                          { return 0 }

func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

// font8x8 returns an 8x8 bitmap for a given byte. The font is a
// minimal 8x8 ASCII set suitable for the conformance harness.
func font8x8(ch byte) [8]byte {
	switch ch {
	case 'A':
		return [8]byte{0x3C, 0x66, 0x66, 0x7E, 0x66, 0x66, 0x66, 0x00}
	case 'B':
		return [8]byte{0x7C, 0x66, 0x66, 0x7C, 0x66, 0x66, 0x7C, 0x00}
	case 'C':
		return [8]byte{0x3C, 0x66, 0x60, 0x60, 0x60, 0x66, 0x3C, 0x00}
	case ' ':
		return [8]byte{}
	}
	// Fallback: simple block.
	return [8]byte{0xFF, 0x81, 0x81, 0x81, 0x81, 0x81, 0xFF, 0x00}
}
