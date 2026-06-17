package stddlg

import "testing"

func TestFileDialogDefault(t *testing.T) {
	got := FileDialog("/tmp", "Open", "*.pas", false)
	if got == "" {
		t.Error("FileDialog returned empty")
	}
}

func TestFileDialogSave(t *testing.T) {
	got := FileDialog("/tmp", "Save As", "*.pas", true)
	if got == "" {
		t.Error("FileDialog returned empty for save")
	}
}

func TestInputBox(t *testing.T) {
	v, ok := InputBox("T", "L", "default", 10)
	if !ok || v != "default" {
		t.Errorf("got %q, ok=%v", v, ok)
	}
}

func TestPatternToFirst(t *testing.T) {
	if got := patternToFirst("*.pas", false); got != "FIRST.PAS" {
		t.Errorf("first: %q", got)
	}
	if got := patternToFirst("*.pas", true); got != "NEW.PAS" {
		t.Errorf("new: %q", got)
	}
}
