package drivers

import "testing"

func TestDriverInit(t *testing.T) {
	d := NewDriver()
	if d.width != 80 || d.height != 25 {
		t.Errorf("size: %dx%d", d.width, d.height)
	}
}

func TestSetAndGetCell(t *testing.T) {
	d := NewDriver()
	d.SetCell(0, 0, uint16('A')|0x0700)
	if d.GetCell(0, 0) != uint16('A')|0x0700 {
		t.Error("cell not set/get")
	}
}

func TestDrawBufferFillChar(t *testing.T) {
	var b TDrawBuffer
	b.FillChar('A', 0x07, 5)
	for i := 0; i < 5; i++ {
		if b[i] != uint16('A')|0x0700 {
			t.Errorf("cell %d: %x", i, b[i])
		}
	}
}

func TestDrawBufferFillCStr(t *testing.T) {
	var b TDrawBuffer
	b.FillCStr("Hello", 0x07)
	if b[0] != uint16('H')|0x0700 {
		t.Error("FillCStr")
	}
}

func TestDrawBufferPutChar(t *testing.T) {
	var b TDrawBuffer
	b.PutChar('X', 0x07, 3)
	if b[3] != uint16('X')|0x0700 {
		t.Error("PutChar")
	}
}

func TestPutEvent(t *testing.T) {
	d := NewDriver()
	d.PutEvent(TEvent{What: EvKeyDown})
	var ev TEvent
	if !d.GetKeyEvent(&ev) {
		t.Error("expected key event")
	}
}

func TestUpdateCursor(t *testing.T) {
	d := NewDriver()
	d.UpdateCursor(10, 5)
	if d.cursorX != 10 || d.cursorY != 5 {
		t.Error("UpdateCursor")
	}
}

func TestShowCursor(t *testing.T) {
	d := NewDriver()
	d.ShowCursor(true)
	if !d.showC {
		t.Error("ShowCursor")
	}
}

func TestSnapshot(t *testing.T) {
	d := NewDriver()
	d.SetCell(0, 0, uint16('H')|0x0700)
	s := d.Snapshot()
	if s[0] != 'H' {
		t.Errorf("Snapshot: %q", s[:5])
	}
}
