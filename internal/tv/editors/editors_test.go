package editors

import "testing"

func TestBufferInsertDelete(t *testing.T) {
	b := NewBuffer()
	b.Insert("hello")
	if b.Text() != "hello" {
		t.Errorf("Text: %q", b.Text())
	}
	b.SetCursor(0)
	b.Insert("X")
	if b.Text() != "Xhello" {
		t.Errorf("Text: %q", b.Text())
	}
}

func TestBufferDelete(t *testing.T) {
	b := NewBuffer()
	b.Insert("hello world")
	b.SetCursor(5)
	b.Delete(6)
	if b.Text() != "hello" {
		t.Errorf("Text: %q", b.Text())
	}
}

func TestEditorInsertAndUndo(t *testing.T) {
	e := (&TEditor{}).Init()
	e.InsertText("a")
	e.InsertText("b")
	if e.Text() != "ab" {
		t.Errorf("Text: %q", e.Text())
	}
	e.UndoCmd()
	if e.Text() != "a" {
		t.Errorf("after undo: %q", e.Text())
	}
	e.RedoCmd()
	if e.Text() != "ab" {
		t.Errorf("after redo: %q", e.Text())
	}
}

func TestEditorBlock(t *testing.T) {
	e := (&TEditor{}).Init()
	e.InsertText("hello world")
	e.SetBlock(0, 5)
	e.CopyBlock()
	if e.ClipText != "hello" {
		t.Errorf("clip: %q", e.ClipText)
	}
	e.CutBlock()
	if e.Text() != " world" {
		t.Errorf("after cut: %q", e.Text())
	}
	e.PasteBlock()
	if e.Text() != "hello world" {
		t.Errorf("after paste: %q", e.Text())
	}
}

func TestEditorFindReplace(t *testing.T) {
	e := (&TEditor{}).Init()
	e.InsertText("foo bar foo")
	e.Buffer.SetCursor(0)
	if idx := e.Find("foo"); idx != 0 {
		t.Errorf("find: %d", idx)
	}
	e.Buffer.SetCursor(0)
	if !e.Replace("foo", "baz") {
		t.Error("replace should succeed")
	}
	if e.Text() != "baz bar foo" {
		t.Errorf("after replace: %q", e.Text())
	}
	e.Buffer.SetCursor(0)
	if n := e.ReplaceAll("foo", "x"); n != 1 {
		t.Errorf("replaceAll: %d", n)
	}
}

func TestFileEditorLoad(t *testing.T) {
	fe := (&TFileEditor{}).Init()
	fe.LoadFile("test.pas", []byte("program T; begin end."))
	if fe.Text() != "program T; begin end." {
		t.Errorf("Text: %q", fe.Text())
	}
	if fe.Filename != "test.pas" {
		t.Errorf("Filename: %s", fe.Filename)
	}
}

func TestEditWindow(t *testing.T) {
	w := NewEditWindow()
	if w.Editor == nil {
		t.Error("Editor not created")
	}
}

func TestEditorFindMiss(t *testing.T) {
	e := (&TEditor{}).Init()
	e.InsertText("hello")
	if idx := e.Find("xyz"); idx >= 0 {
		t.Errorf("find miss: %d", idx)
	}
}
