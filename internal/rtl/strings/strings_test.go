package strings

import "testing"

func mk(s string) []byte {
	b := make([]byte, 64)
	copy(b, s)
	return b
}

func TestStrLen(t *testing.T) {
	if StrLen(mk("hello")) != 5 {
		t.Error("StrLen")
	}
}

func TestStrCat(t *testing.T) {
	dest := mk("foo")
	dest = StrCat(dest, mk("bar"))
	if string(dest[:6]) != "foobar" {
		t.Errorf("StrCat = %q", string(dest[:6]))
	}
}

func TestStrComp(t *testing.T) {
	if StrComp(mk("abc"), mk("abc")) != 0 {
		t.Error("StrComp equal")
	}
	if StrComp(mk("a"), mk("b")) >= 0 {
		t.Error("StrComp a<b")
	}
}

func TestStrIComp(t *testing.T) {
	if StrIComp(mk("ABC"), mk("abc")) != 0 {
		t.Error("StrIComp case-insensitive")
	}
}

func TestStrCopy(t *testing.T) {
	dest := make([]byte, 10)
	dest = StrCopy(dest, mk("hello"))
	if string(dest[:5]) != "hello" {
		t.Errorf("StrCopy = %q", string(dest[:5]))
	}
}

func TestStrEnd(t *testing.T) {
	e := StrEnd(mk("hello"))
	if e[0] != 0 {
		t.Errorf("StrEnd should point to null: %v", e)
	}
}

func TestStrLowerUpper(t *testing.T) {
	s := mk("HeLLo")
	StrLower(s)
	if string(s[:5]) != "hello" {
		t.Errorf("StrLower = %q", string(s[:5]))
	}
	s2 := mk("HeLLo")
	StrUpper(s2)
	if string(s2[:5]) != "HELLO" {
		t.Errorf("StrUpper = %q", string(s2[:5]))
	}
}

func TestStrLCopy(t *testing.T) {
	dest := make([]byte, 10)
	dest = StrLCopy(dest, mk("hello world"), 6)
	if string(dest[:5]) != "hello" {
		t.Errorf("StrLCopy = %q", string(dest[:5]))
	}
}

func TestStrPos(t *testing.T) {
	idx := StrPos(mk("lo"), mk("hello"))
	if idx == nil || string(idx[:2]) != "lo" {
		t.Errorf("StrPos")
	}
}

func TestStrScan(t *testing.T) {
	idx := StrScan(mk("hello"), 'l')
	if idx == nil || idx[0] != 'l' {
		t.Errorf("StrScan")
	}
}

func TestStrPas(t *testing.T) {
	if StrPas(mk("hello")) != "hello" {
		t.Errorf("StrPas = %q", StrPas(mk("hello")))
	}
}

func TestStrNewAndDispose(t *testing.T) {
	p := StrNew(mk("hello"))
	if p == nil {
		t.Fatal("StrNew returned nil")
	}
	if string(p[:5]) != "hello" {
		t.Errorf("StrNew = %q", string(p[:5]))
	}
	StrDispose(p)
}

func TestStrECopy(t *testing.T) {
	dest := make([]byte, 10)
	rest := StrECopy(dest, mk("hello"))
	if string(dest[:5]) != "hello" {
		t.Errorf("StrECopy = %q", string(dest[:5]))
	}
	if rest == nil {
		t.Errorf("StrECopy should return pointer to null terminator")
	}
}

func TestStrLIComp(t *testing.T) {
	if StrLIComp(mk("ABCDE"), mk("abcxy"), 3) != 0 {
		t.Error("StrLIComp should be case-insensitive for first 3")
	}
}

func TestStrLComp(t *testing.T) {
	if StrLComp(mk("abcde"), mk("abcxy"), 3) != 0 {
		t.Error("StrLComp should match for first 3")
	}
}
