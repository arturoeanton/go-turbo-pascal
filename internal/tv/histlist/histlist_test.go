package histlist

import "testing"

func TestHistoryAdd(t *testing.T) {
	h := (&THistory{}).Init(1, 5)
	h.Add("foo")
	h.Add("bar")
	h.Add("foo") // moves to top
	if h.Count() != 2 {
		t.Errorf("Count: %d", h.Count())
	}
	if h.At(0) != "foo" {
		t.Errorf("At(0): %s", h.At(0))
	}
}

func TestHistoryMax(t *testing.T) {
	h := (&THistory{}).Init(1, 2)
	h.Add("a")
	h.Add("b")
	h.Add("c")
	if h.Count() != 2 {
		t.Errorf("Count: %d", h.Count())
	}
	if h.At(1) != "b" {
		t.Errorf("At(1): %s", h.At(1))
	}
}

func TestHistoryBounds(t *testing.T) {
	h := (&THistory{}).Init(1, 5)
	if h.At(-1) != "" {
		t.Error("At(-1)")
	}
	if h.At(100) != "" {
		t.Error("At(100)")
	}
}

func TestHistListRecall(t *testing.T) {
	l := (&THistList{}).Init(1, 5)
	l.Add("apple")
	l.Add("banana")
	if r := l.Recall("nan"); r != 0 {
		t.Errorf("Recall: %d", r)
	}
	if r := l.Recall("zzz"); r != -1 {
		t.Errorf("Recall miss: %d", r)
	}
}
