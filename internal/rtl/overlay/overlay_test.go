package overlay

import "testing"

func TestOvrInit(t *testing.T) {
	m := New()
	if r := m.OvrInit("test.ovr"); r != 0 {
		t.Errorf("OvrInit: %d", r)
	}
}

func TestOvrInitEMS(t *testing.T) {
	m := New()
	if r := m.OvrInitEMS("test.ovr", 0); r != 0 {
		t.Errorf("OvrInitEMS: %d", r)
	}
}

func TestOvrInitEmpty(t *testing.T) {
	m := New()
	if r := m.OvrInit(""); r != -2 {
		t.Errorf("OvrInit(empty) should report error, got %d", r)
	}
}

func TestOvrSetGetBuf(t *testing.T) {
	m := New()
	m.OvrSetBuf(2048)
	if m.OvrGetBuf() != 2048 {
		t.Errorf("OvrGetBuf: %d", m.OvrGetBuf())
	}
}

func TestOvrClearBuf(t *testing.T) {
	m := New()
	m.OvrSetBuf(2048)
	m.OvrClearBuf()
}

func TestOvrSetGetRetry(t *testing.T) {
	m := New()
	m.OvrSetRetry(1024)
	if m.OvrGetRetry() < 1024 {
		t.Errorf("OvrGetRetry: %d", m.OvrGetRetry())
	}
}

func TestOvrLoadCount(t *testing.T) {
	m := New()
	m.Load("foo")
	m.Load("bar")
	m.Load("foo")
	if m.OvrLoadCount() != 3 {
		t.Errorf("OvrLoadCount: %d", m.OvrLoadCount())
	}
}

func TestOvrTrapCount(t *testing.T) {
	m := New()
	m.Trap()
	m.Trap()
	m.Trap()
	if m.OvrTrapCount() != 3 {
		t.Errorf("OvrTrapCount: %d", m.OvrTrapCount())
	}
}

func TestOvrResult(t *testing.T) {
	m := New()
	m.OvrInit("a.ovr")
	if m.OvrResult() != 0 {
		t.Errorf("OvrResult: %d", m.OvrResult())
	}
}

func TestOvrFileMode(t *testing.T) {
	m := New()
	if m.OvrFileMode() != 0 {
		t.Errorf("OvrFileMode default: %d", m.OvrFileMode())
	}
	m.FileMode = 1
	if m.OvrFileMode() != 1 {
		t.Errorf("OvrFileMode set: %d", m.OvrFileMode())
	}
}

func TestOvrStats(t *testing.T) {
	m := New()
	m.Load("x")
	m.Trap()
	s := m.Stats()
	if s.Loaded != 1 || s.Traps != 1 {
		t.Errorf("Stats: %+v", s)
	}
}
