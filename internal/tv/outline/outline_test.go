package outline

import "testing"

func TestRoot(t *testing.T) {
	o := New()
	r := o.Root("A")
	if o.Text(r) != "A" {
		t.Error("Text")
	}
}

func TestChildren(t *testing.T) {
	o := New()
	r := o.Root("A")
	c1 := o.InsertChild(r, "A.1")
	c2 := o.InsertChild(r, "A.2")
	if o.Count() != 3 {
		t.Errorf("Count: %d", o.Count())
	}
	if o.Text(c1) != "A.1" {
		t.Error("c1 text")
	}
	if o.Text(c2) != "A.2" {
		t.Error("c2 text")
	}
}

func TestExpandCollapse(t *testing.T) {
	o := New()
	r := o.Root("A")
	c1 := o.InsertChild(r, "A.1")
	grand := o.InsertChild(c1, "A.1.1")
	o.Expand(c1)
	visible := o.Visible()
	if len(visible) != 3 {
		t.Errorf("Visible when expanded: %d", len(visible))
	}
	o.Collapse(c1)
	visible = o.Visible()
	if len(visible) != 2 {
		t.Errorf("Visible when collapsed: %d", len(visible))
	}
	if visible[0] != r || visible[1] != c1 {
		t.Errorf("Visible: %v", visible)
	}
	_ = grand
}

func TestFocus(t *testing.T) {
	o := New()
	r := o.Root("A")
	c := o.InsertChild(r, "A.1")
	o.Focus(c)
	if o.Focused != c {
		t.Error("Focus")
	}
	o.Focus(-1)
	if o.Focused != c {
		t.Error("Focus should be no-op for invalid")
	}
}

func TestTextBounds(t *testing.T) {
	o := New()
	if o.Text(-1) != "" {
		t.Error("Text(-1)")
	}
	if o.Text(100) != "" {
		t.Error("Text(100)")
	}
}
