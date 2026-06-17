package views

import "testing"

func TestTRectEmpty(t *testing.T) {
	if !(TRect{}).Empty() {
		t.Error("default rect should be empty")
	}
	r := TRect{TPoint{0, 0}, TPoint{10, 10}}
	if r.Empty() {
		t.Error("non-empty rect")
	}
}

func TestTRectContains(t *testing.T) {
	r := TRect{TPoint{0, 0}, TPoint{10, 10}}
	if !r.Contains(TPoint{5, 5}) {
		t.Error("should contain (5,5)")
	}
	if r.Contains(TPoint{15, 5}) {
		t.Error("should not contain (15,5)")
	}
}

func TestTRectIntersect(t *testing.T) {
	a := TRect{TPoint{0, 0}, TPoint{10, 10}}
	b := TRect{TPoint{5, 5}, TPoint{15, 15}}
	i := a.Intersect(b)
	if i.A.X != 5 || i.A.Y != 5 || i.B.X != 10 || i.B.Y != 10 {
		t.Errorf("intersect: %+v", i)
	}
}

func TestTRectUnion(t *testing.T) {
	a := TRect{TPoint{0, 0}, TPoint{5, 5}}
	b := TRect{TPoint{3, 3}, TPoint{10, 10}}
	u := a.Union(b)
	if u.A != (TPoint{0, 0}) || u.B != (TPoint{10, 10}) {
		t.Errorf("union: %+v", u)
	}
}

func TestTRectMoveGrow(t *testing.T) {
	r := TRect{TPoint{0, 0}, TPoint{10, 10}}
	m := r.Move(2, 3)
	if m.A != (TPoint{2, 3}) {
		t.Error("Move")
	}
	g := r.Grow(1, 1)
	if g.A != (TPoint{-1, -1}) {
		t.Error("Grow")
	}
}

func TestTViewShowHide(t *testing.T) {
	v := (&TView{}).Init(TRect{TPoint{0, 0}, TPoint{10, 10}})
	v.Show()
	if v.State&StateVisible == 0 {
		t.Error("Show")
	}
	v.Hide()
	if v.State&StateVisible != 0 {
		t.Error("Hide")
	}
}

func TestTGroupInsertRemove(t *testing.T) {
	g := (&TGroup{}).Init(TRect{TPoint{0, 0}, TPoint{80, 25}})
	v := (&TView{}).Init(TRect{TPoint{1, 1}, TPoint{10, 10}})
	g.Insert(v)
	if v.Owner != g {
		t.Error("Owner not set")
	}
	if g.Current != v {
		t.Error("Current not set")
	}
	g.Remove(v)
	if v.Owner != nil {
		t.Error("Owner not cleared")
	}
}

func TestTScrollBar(t *testing.T) {
	s := (&TScrollBar{}).Init(TRect{TPoint{0, 0}, TPoint{1, 10}})
	s.SetRange(0, 100)
	s.SetPosition(50)
	if s.Pos != 50 {
		t.Error("SetPosition")
	}
}

func TestTInputLine(t *testing.T) {
	i := (&TInputLine{}).Init(TRect{TPoint{0, 0}, TPoint{10, 1}}, 20)
	i.SetData("hello")
	if i.GetData() != "hello" {
		t.Error("GetData")
	}
}

func TestTStaticText(t *testing.T) {
	s := (&TStaticText{}).Init(TRect{TPoint{0, 0}, TPoint{10, 1}}, "Hello")
	if s.Text != "Hello" {
		t.Error("Text")
	}
}

func TestTParamText(t *testing.T) {
	p := (&TParamText{}).Init(TRect{TPoint{0, 0}, TPoint{10, 1}}, "%0 says hi", 1)
	p.Params[0] = "Alice"
	if p.GetText() != "Alice says hi" {
		t.Errorf("GetText: %q", p.GetText())
	}
}

func TestTButton(t *testing.T) {
	b := (&TButton{}).Init(TRect{TPoint{0, 0}, TPoint{10, 1}}, "OK", 1, 0)
	if b.Title != "OK" {
		t.Error("Title")
	}
	if b.Command != 1 {
		t.Error("Command")
	}
}

func TestTRadioButtons(t *testing.T) {
	r := (&TRadioButtons{}).Init(TRect{TPoint{0, 0}, TPoint{10, 5}}, []string{"A", "B", "C"})
	r.Press(1)
	if r.Value != 1 {
		t.Error("Press")
	}
}

func TestTCheckBoxes(t *testing.T) {
	c := (&TCheckBoxes{}).Init(TRect{TPoint{0, 0}, TPoint{10, 5}}, []string{"X", "Y", "Z"})
	c.Toggle(0)
	c.Toggle(2)
	if c.Value&(1<<0) == 0 || c.Value&(1<<2) == 0 {
		t.Error("Toggle")
	}
	if c.Value&(1<<1) != 0 {
		t.Error("Toggle(1) should be off")
	}
}

func TestTFrame(t *testing.T) {
	f := (&TFrame{}).Init(TRect{TPoint{0, 0}, TPoint{10, 10}})
	f.Draw()
	if f.State&StateExposed == 0 {
		t.Error("Draw should set Exposed")
	}
}

func TestTScroller(t *testing.T) {
	s := (&TScroller{}).Init(TRect{TPoint{0, 0}, TPoint{10, 10}}, nil, nil)
	if s.HScroll != nil {
		t.Error("HScroll")
	}
}

func TestTListViewer(t *testing.T) {
	l := (&TListViewer{}).Init(TRect{TPoint{0, 0}, TPoint{10, 10}}, 1, nil, nil)
	l.SetRange(5)
}

func TestTHistory(t *testing.T) {
	v := (&TView{}).Init(TRect{TPoint{0, 0}, TPoint{1, 1}})
	h := (&THistory{}).Init(TRect{TPoint{0, 0}, TPoint{1, 1}}, v, 1)
	if h.Link != v {
		t.Error("Link")
	}
}

func TestTLabel(t *testing.T) {
	v := (&TView{}).Init(TRect{TPoint{0, 0}, TPoint{1, 1}})
	l := (&TLabel{}).Init(TRect{TPoint{0, 0}, TPoint{1, 1}}, "Label", v)
	if l.Link != v {
		t.Error("Link")
	}
}
