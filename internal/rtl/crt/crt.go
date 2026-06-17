// Package crt implements the Crt unit runtime. The Crt unit provides
// text-mode screen, keyboard, sound and windowed output for BPGo
// programs. The implementation uses a virtual 80x25 screen buffer that
// is rendered to the host terminal on request. Sound is reported but
// not emitted (no audio device).
package crt

import (
	"sync"
	"time"
)

// Cell represents a single screen cell: ASCII code + attribute byte.
type Cell struct {
	Ch   byte
	Attr byte
}

// Screen is a virtual 80x25 text-mode display with attribute byte per
// cell. The default TP7 attributes use 16 foreground and 8 background
// colors, blink bit and intensity bit.
type Screen struct {
	mu      sync.Mutex
	cols    int
	rows    int
	buf     []Cell
	cursorX int
	cursorY int
	attr    byte
	winMinX int
	winMinY int
	winMaxX int
	winMaxY int
	keys    chan KeyEvent
}

// KeyEvent is a single keyboard event. Scan is 0 for ASCII keys, the
// XT scan code for extended keys, prefixed by a 0 byte.
type KeyEvent struct {
	Ch   byte
	Scan byte
}

func NewScreen() *Screen {
	s := &Screen{cols: 80, rows: 25, attr: 0x07, keys: make(chan KeyEvent, 64)}
	s.buf = make([]Cell, 80*25)
	s.winMinX = 1
	s.winMinY = 1
	s.winMaxX = 80
	s.winMaxY = 25
	return s
}

// ClrScr clears the active window.
func (s *Screen) ClrScr() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for y := s.winMinY; y <= s.winMaxY; y++ {
		for x := s.winMinX; x <= s.winMaxX; x++ {
			s.setCell(x, y, ' ')
		}
	}
	s.cursorX = s.winMinX
	s.cursorY = s.winMinY
}

// ClrEol clears from cursor to end of line within the window.
func (s *Screen) ClrEol() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for x := s.cursorX; x <= s.winMaxX; x++ {
		s.setCell(x, s.cursorY, ' ')
	}
}

// InsLine inserts a blank line at the cursor row, scrolling lower lines
// down.
func (s *Screen) InsLine() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cursorY < s.winMinY || s.cursorY >= s.winMaxY {
		return
	}
	for y := s.winMaxY; y > s.cursorY; y-- {
		for x := s.winMinX; x <= s.winMaxX; x++ {
			s.buf[s.idx(x, y)] = s.buf[s.idx(x, y-1)]
		}
	}
	for x := s.winMinX; x <= s.winMaxX; x++ {
		s.setCell(x, s.cursorY, ' ')
	}
}

// DelLine deletes the current line, scrolling lines below up.
func (s *Screen) DelLine() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cursorY < s.winMinY || s.cursorY >= s.winMaxY {
		return
	}
	for y := s.cursorY; y < s.winMaxY; y++ {
		for x := s.winMinX; x <= s.winMaxX; x++ {
			s.buf[s.idx(x, y)] = s.buf[s.idx(x, y+1)]
		}
	}
	for x := s.winMinX; x <= s.winMaxX; x++ {
		s.setCell(x, s.winMaxY, ' ')
	}
}

// GotoXY moves the cursor. (1,1) is the top-left corner of the active
// window.
func (s *Screen) GotoXY(x, y int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cursorX = clamp(x, s.winMinX, s.winMaxX)
	s.cursorY = clamp(y, s.winMinY, s.winMaxY)
}

func (s *Screen) WhereX() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cursorX
}

func (s *Screen) WhereY() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cursorY
}

// Window sets a text window. Coordinates are absolute screen
// coordinates.
func (s *Screen) Window(x1, y1, x2, y2 int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if x1 < 1 {
		x1 = 1
	}
	if y1 < 1 {
		y1 = 1
	}
	if x2 > s.cols {
		x2 = s.cols
	}
	if y2 > s.rows {
		y2 = s.rows
	}
	if x1 >= x2 || y1 >= y2 {
		return
	}
	s.winMinX = x1
	s.winMinY = y1
	s.winMaxX = x2
	s.winMaxY = y2
	s.cursorX = x1
	s.cursorY = y1
}

func (s *Screen) TextColor(c byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attr = (s.attr & 0x70) | (c & 0x0F)
}

func (s *Screen) TextBackground(c byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attr = (s.attr & 0x0F) | ((c & 0x07) << 4)
}

func (s *Screen) HighVideo() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attr |= 0x08
}

func (s *Screen) LowVideo() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attr &^= 0x08
}

func (s *Screen) NormVideo() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attr = 0x07
}

func (s *Screen) Attr() byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.attr
}

// Write writes a string at the current cursor position, scrolling if
// the cursor goes past the bottom of the window.
func (s *Screen) Write(s2 string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := 0; i < len(s2); i++ {
		ch := s2[i]
		if ch == '\n' {
			s.cursorX = s.winMinX
			s.cursorY++
			if s.cursorY > s.winMaxY {
				s.scrollUp()
				s.cursorY = s.winMaxY
			}
			continue
		}
		if ch == '\r' {
			s.cursorX = s.winMinX
			continue
		}
		s.setCell(s.cursorX, s.cursorY, ch)
		s.cursorX++
		if s.cursorX > s.winMaxX {
			s.cursorX = s.winMinX
			s.cursorY++
			if s.cursorY > s.winMaxY {
				s.scrollUp()
				s.cursorY = s.winMaxY
			}
		}
	}
}

func (s *Screen) setCell(x, y int, ch byte) {
	if x < 1 || y < 1 || x > s.cols || y > s.rows {
		return
	}
	s.buf[s.idx(x, y)] = Cell{Ch: ch, Attr: s.attr}
}

func (s *Screen) idx(x, y int) int {
	return (y-1)*s.cols + (x - 1)
}

func (s *Screen) scrollUp() {
	for y := s.winMinY; y < s.winMaxY; y++ {
		for x := s.winMinX; x <= s.winMaxX; x++ {
			s.buf[s.idx(x, y)] = s.buf[s.idx(x, y+1)]
		}
	}
	for x := s.winMinX; x <= s.winMaxX; x++ {
		s.setCell(x, s.winMaxY, ' ')
	}
}

// Snapshot returns a textual representation of the screen for golden
// tests.
func (s *Screen) Snapshot() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var b []byte
	for y := 1; y <= s.rows; y++ {
		for x := 1; x <= s.cols; x++ {
			c := s.buf[s.idx(x, y)]
			if c.Ch == 0 {
				b = append(b, ' ')
			} else {
				b = append(b, c.Ch)
			}
		}
		b = append(b, '\n')
	}
	return string(b)
}

func (s *Screen) CellAt(x, y int) Cell {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf[s.idx(x, y)]
}

func (s *Screen) PushKey(k KeyEvent) {
	select {
	case s.keys <- k:
	default:
	}
}

func (s *Screen) KeyPressed() bool {
	return len(s.keys) > 0
}

func (s *Screen) ReadKey() KeyEvent {
	return <-s.keys
}

// Delay pauses for the given number of milliseconds, calibrated so
// that the value 200 (the famous TP7 delay-loop bug) does not
// divide-by-zero. The actual sleep is bounded by the maximum sleep
// parameter to prevent runaway delays in tests.
func Delay(ms int) {
	if ms <= 0 {
		return
	}
	if ms > 10000 {
		ms = 10000
	}
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
