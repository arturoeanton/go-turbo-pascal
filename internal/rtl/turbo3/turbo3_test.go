package turbo3

import "testing"

func TestTurbo3(t *testing.T) {
	Init()
	if KbdID != 100 {
		t.Error("KbdID should be 100")
	}
}
