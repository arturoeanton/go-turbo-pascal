package objects

import "testing"

func TestTObjectInit(t *testing.T) {
	o := (&TObject{}).Init()
	if !o.Instance {
		t.Error("should be initialized")
	}
	if len(o.VMT) == 0 {
		t.Error("VMT not set")
	}
	o.Free()
	if o.Instance {
		t.Error("should be freed")
	}
}

func TestTStream(t *testing.T) {
	s := (&TStream{}).Init()
	if s.Pos != 0 {
		t.Error("pos not zero")
	}
}

func TestTBufStream(t *testing.T) {
	b := (&TBufStream{}).Init()
	if b.Buffer == nil {
		t.Error("Buffer not initialized")
	}
}

func TestTDosStream(t *testing.T) {
	d := (&TDosStream{}).Init("test.txt", 1)
	if d.Filename != "test.txt" {
		t.Error("filename not set")
	}
}

func TestTCollectionBasic(t *testing.T) {
	c := (&TCollection{}).Init(10, 5)
	if c.Limit != 10 {
		t.Error("limit")
	}
	c.Insert("a")
	c.Insert("b")
	c.Insert("c")
	if c.Count() != 3 {
		t.Errorf("Count = %d", c.Count())
	}
	if c.At(1) != "b" {
		t.Error("At(1)")
	}
	c.PutItem(1, "B")
	if c.At(1) != "B" {
		t.Error("PutItem")
	}
	c.Delete(0)
	if c.Count() != 2 {
		t.Error("Delete")
	}
}

func TestTCollectionNegative(t *testing.T) {
	c := (&TCollection{}).Init(10, 5)
	if c.At(-1) != nil {
		t.Error("At(-1)")
	}
	if c.At(100) != nil {
		t.Error("At(100)")
	}
	c.PutItem(-1, "x") // should no-op
}

func TestTSortedCollection(t *testing.T) {
	s := (&TSortedCollection{}).Init(10, 5)
	s.Compare = func(a, b interface{}) int {
		ai, bi := a.(int), b.(int)
		if ai < bi {
			return -1
		}
		if ai > bi {
			return 1
		}
		return 0
	}
	s.Insert(3)
	s.Insert(1)
	s.Insert(4)
	s.Insert(1)
	s.Insert(5)
	s.Insert(9)
	s.Insert(2)
	s.Insert(6)
	if s.At(0) != 1 {
		t.Errorf("sorted[0] = %v", s.At(0))
	}
	if s.At(s.Count()-1) != 9 {
		t.Errorf("sorted[end] = %v", s.At(s.Count()-1))
	}
}

func TestTStringCollection(t *testing.T) {
	s := (&TStringCollection{}).Init(10, 5)
	s.Insert("banana")
	s.Insert("apple")
	s.Insert("cherry")
	if s.At(0) != "apple" {
		t.Errorf("sorted[0] = %v", s.At(0))
	}
}

func TestTResourceCollection(t *testing.T) {
	r := (&TResourceCollection{}).Init(10, 5)
	r.Insert(42)
	if r.Count() != 1 {
		t.Error("Count")
	}
}

func TestTStrListMaker(t *testing.T) {
	m := (&TStrListMaker{}).Init(10, 5)
	m.Put("hello")
	m.Put("world")
	if m.Get(0) != "hello" {
		t.Error("Get(0)")
	}
	if m.Get(1) != "world" {
		t.Error("Get(1)")
	}
}

func TestFreeAll(t *testing.T) {
	c := (&TCollection{}).Init(10, 5)
	c.Insert(1)
	c.Insert(2)
	c.FreeAll()
	if c.Count() != 0 {
		t.Error("FreeAll should clear")
	}
}
