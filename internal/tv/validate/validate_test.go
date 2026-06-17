package validate

import "testing"

func TestRangeValidator(t *testing.T) {
	r := (&TRangeValidator{}).Init(1, 10)
	if !r.Valid("5") {
		t.Error("5 should be valid")
	}
	if r.Valid("0") {
		t.Error("0 should be invalid")
	}
	if r.Valid("11") {
		t.Error("11 should be invalid")
	}
	if r.Valid("abc") {
		t.Error("abc should be invalid")
	}
}

func TestFilterValidator(t *testing.T) {
	f := (&TFilterValidator{}).Init(func(b byte) bool {
		return b >= '0' && b <= '9'
	})
	if !f.Valid("123") {
		t.Error("123 should be valid")
	}
	if f.Valid("12a") {
		t.Error("12a should be invalid")
	}
}

func TestPXPictureValidator(t *testing.T) {
	p := (&TPXPictureValidator{}).Init("###-##-##")
	if !p.Valid("123") {
		t.Error("should be valid (simplified)")
	}
}
