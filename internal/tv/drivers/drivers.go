// Package drivers implements the Turbo Vision Drivers unit. The
// unit provides the event loop, keyboard/mouse/screen drivers and
// the TDrawBuffer screen line buffer used by Views.
package drivers

import (
	"sync"
)

// Event kinds.
const (
	EvNothing = iota
	EvKeyDown
	EvKeyUp
	EvMouseDown
	EvMouseUp
	EvMouseMove
	EvCommand
	EvBroadcast
	EvMessage
)

// TEvent is the unified TV event record.
type TEvent struct {
	What uint16
	Case union
	Info [4]uint16
}

type union struct {
	A uint16
	B uint16
	C uint16
	D uint16
}

// Key codes (subset of the BIOS scan codes recognised by TV).
const (
	KbEsc = 0x011B
)

// MouseState describes the current mouse position and buttons.
type MouseState struct {
	X, Y    int
	Buttons byte
	Wheel   int
}

// TDrawBuffer is an 80-char screen line used by views.
type TDrawBuffer [80]uint16

// FillChar fills a range of cells with a character + attribute.
func (b *TDrawBuffer) FillChar(c byte, attr byte, count int) {
	for i := 0; i < count && i < len(b); i++ {
		b[i] = uint16(c) | (uint16(attr) << 8)
	}
}

// FillCStr copies a string with the given attribute.
func (b *TDrawBuffer) FillCStr(s string, attr byte) {
	for i := 0; i < len(s) && i < len(b); i++ {
		b[i] = uint16(s[i]) | (uint16(attr) << 8)
	}
}

// PutChar writes a single cell.
func (b *TDrawBuffer) PutChar(c byte, attr byte, pos int) {
	if pos < 0 || pos >= len(b) {
		return
	}
	b[pos] = uint16(c) | (uint16(attr) << 8)
}

// Driver is the screen + keyboard + mouse driver.
type Driver struct {
	mu      sync.Mutex
	events  []TEvent
	screen  []uint16
	width   int
	height  int
	mouse   MouseState
	cursorX int
	cursorY int
	showC   bool
}

// NewDriver returns a default 80x25 driver.
func NewDriver() *Driver {
	return &Driver{
		width: 80, height: 25,
		screen: make([]uint16, 80*25),
	}
}

// InitEvents starts the event polling loop.
func (d *Driver) InitEvents() {}

// DoneEvents stops the event loop.
func (d *Driver) DoneEvents() {}

// GetKeyEvent returns the next key event (blocking).
func (d *Driver) GetKeyEvent(ev *TEvent) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	for len(d.events) > 0 {
		e := d.events[0]
		d.events = d.events[1:]
		if e.What == EvKeyDown || e.What == EvKeyUp {
			*ev = e
			return true
		}
	}
	return false
}

// GetMouseEvent returns the next mouse event.
func (d *Driver) GetMouseEvent(ev *TEvent) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	for len(d.events) > 0 {
		e := d.events[0]
		d.events = d.events[1:]
		if e.What == EvMouseDown || e.What == EvMouseUp || e.What == EvMouseMove {
			*ev = e
			return true
		}
	}
	return false
}

// PutEvent pushes an event.
func (d *Driver) PutEvent(e TEvent) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.events = append(d.events, e)
}

// UpdateCursor moves the text cursor to the given cell.
func (d *Driver) UpdateCursor(x, y int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cursorX = x
	d.cursorY = y
}

// ShowCursor toggles the text cursor.
func (d *Driver) ShowCursor(show bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.showC = show
}

// SetCell writes a single screen cell.
func (d *Driver) SetCell(x, y int, cell uint16) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if x < 0 || y < 0 || x >= d.width || y >= d.height {
		return
	}
	d.screen[y*d.width+x] = cell
}

// GetCell reads a single screen cell.
func (d *Driver) GetCell(x, y int) uint16 {
	d.mu.Lock()
	defer d.mu.Unlock()
	if x < 0 || y < 0 || x >= d.width || y >= d.height {
		return 0
	}
	return d.screen[y*d.width+x]
}

// Snapshot returns a textual snapshot of the screen.
func (d *Driver) Snapshot() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]byte, 0, d.width*d.height+d.height)
	for y := 0; y < d.height; y++ {
		for x := 0; x < d.width; x++ {
			c := byte(d.screen[y*d.width+x] & 0xFF)
			if c == 0 {
				c = ' '
			}
			out = append(out, c)
		}
		out = append(out, '\n')
	}
	return string(out)
}
