package crt

import (
	"strings"
	"testing"
)

func TestClrScrAndWindow(t *testing.T) {
	s := NewScreen()
	s.ClrScr()
	if strings.TrimSpace(s.Snapshot()) != "" {
		t.Error("after ClrScr snapshot should be blank")
	}
}

func TestWriteAndCursor(t *testing.T) {
	s := NewScreen()
	s.GotoXY(1, 1)
	s.Write("hi")
	if s.WhereX() != 3 {
		t.Errorf("cursor X = %d", s.WhereX())
	}
	snap := s.Snapshot()
	if !strings.HasPrefix(snap, "hi") {
		t.Errorf("expected 'hi' at top, got %q", snap[:20])
	}
}

func TestTextColorAndBackground(t *testing.T) {
	s := NewScreen()
	s.TextColor(4)
	s.TextBackground(1)
	if s.Attr()&0x0F != 4 {
		t.Errorf("color = %d", s.Attr()&0x0F)
	}
	if (s.Attr()>>4)&0x07 != 1 {
		t.Errorf("background = %d", (s.Attr()>>4)&0x07)
	}
}

func TestHighLowNormVideo(t *testing.T) {
	s := NewScreen()
	s.NormVideo()
	if s.Attr() != 0x07 {
		t.Errorf("NormVideo = %x", s.Attr())
	}
	s.HighVideo()
	if s.Attr()&0x08 == 0 {
		t.Error("HighVideo should set intensity")
	}
	s.LowVideo()
	if s.Attr()&0x08 != 0 {
		t.Error("LowVideo should clear intensity")
	}
}

func TestWindowAndClrScr(t *testing.T) {
	s := NewScreen()
	s.Window(10, 5, 20, 10)
	s.GotoXY(11, 6)
	s.Write("X")
	if s.CellAt(11, 6).Ch != 'X' {
		t.Error("expected X at (11,6)")
	}
	s.ClrScr()
	if s.WhereX() != 10 || s.WhereY() != 5 {
		t.Errorf("ClrScr should reset cursor to window origin, got (%d,%d)", s.WhereX(), s.WhereY())
	}
}

func TestWindowClipping(t *testing.T) {
	s := NewScreen()
	s.Window(5, 5, 15, 15)
	s.GotoXY(4, 4) // outside window; should be clamped
	if s.WhereX() != 5 || s.WhereY() != 5 {
		t.Errorf("GotoXY outside window should clamp, got (%d,%d)", s.WhereX(), s.WhereY())
	}
}

func TestKeyQueue(t *testing.T) {
	s := NewScreen()
	if s.KeyPressed() {
		t.Error("empty queue should not be pressed")
	}
	s.PushKey(KeyEvent{Ch: 'a'})
	if !s.KeyPressed() {
		t.Error("after push should be pressed")
	}
	k := s.ReadKey()
	if k.Ch != 'a' {
		t.Errorf("ReadKey = %v", k)
	}
}

func TestScrollUp(t *testing.T) {
	s := NewScreen()
	s.GotoXY(1, 1)
	s.Write("a")
	s.GotoXY(1, 2)
	s.Write("b")
	// Move cursor to last line and write to force scroll.
	s.GotoXY(1, 25)
	s.Write("c")
	snap := s.Snapshot()
	if !strings.Contains(snap, "a") || !strings.Contains(snap, "b") || !strings.Contains(snap, "c") {
		t.Errorf("expected all three lines, got %q", snap)
	}
}

func TestInsLine(t *testing.T) {
	s := NewScreen()
	s.GotoXY(1, 1)
	s.Write("hello")
	s.GotoXY(1, 2)
	s.Write("world")
	s.GotoXY(1, 1)
	s.InsLine()
	if s.CellAt(1, 1).Ch != ' ' {
		t.Error("InsLine should blank cursor row")
	}
	if s.CellAt(1, 2).Ch != 'h' {
		t.Error("InsLine should push 'hello' down")
	}
	if s.CellAt(1, 3).Ch != 'w' {
		t.Error("InsLine should push 'world' further down")
	}
}

func TestDelLine(t *testing.T) {
	s := NewScreen()
	s.GotoXY(1, 1)
	s.Write("a")
	s.GotoXY(1, 2)
	s.Write("b")
	s.GotoXY(1, 1)
	s.DelLine()
	if s.CellAt(1, 1).Ch != 'b' {
		t.Error("DelLine should pull next line up")
	}
}

func TestClrEol(t *testing.T) {
	s := NewScreen()
	s.GotoXY(1, 1)
	s.Write("hello world")
	s.GotoXY(6, 1)
	s.ClrEol()
	if s.CellAt(1, 1).Ch != 'h' {
		t.Error("ClrEol should not erase before cursor")
	}
	if s.CellAt(6, 1).Ch != ' ' {
		t.Error("ClrEol should erase from cursor")
	}
}
