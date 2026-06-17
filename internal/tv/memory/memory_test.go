package memory

import "testing"

func TestInitDone(t *testing.T) {
	m := New()
	m.InitMemory()
	if m.InUse() != 0 {
		t.Error("InitMemory should reset")
	}
	m.Reserve(100)
	m.DoneMemory()
	if m.InUse() != 0 {
		t.Error("DoneMemory should reset")
	}
}

func TestReserveRelease(t *testing.T) {
	m := New()
	if !m.Reserve(100) {
		t.Error("Reserve should succeed")
	}
	if m.InUse() != 100 {
		t.Errorf("InUse: %d", m.InUse())
	}
	m.Release(50)
	if m.InUse() != 50 {
		t.Errorf("InUse after release: %d", m.InUse())
	}
}

func TestMemoryError(t *testing.T) {
	m := New()
	called := false
	m.OnLowMem = func() { called = true }
	m.Limit(10)
	if m.Reserve(100) {
		t.Error("Reserve over limit should fail")
	}
	if !called {
		t.Error("OnLowMem should be called")
	}
	if !m.LowMem {
		t.Error("LowMem flag should be set")
	}
}

func TestReleaseNegative(t *testing.T) {
	m := New()
	m.Release(100)
	if m.InUse() != 0 {
		t.Error("Release below zero should clamp to 0")
	}
}
