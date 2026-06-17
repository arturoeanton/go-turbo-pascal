package ide

import "testing"

func newTestIDE() *IDE {
	return New(&Project{Name: "T", Source: "hello.pas", Output: "hello.exe"}, &stubCompiler{}, &stubRunner{}, &stubDebugger{})
}

type stubCompiler struct{}

func (s *stubCompiler) Compile(src, output string) (string, error) {
	return "ok", nil
}

type captureCompiler struct {
	src string
}

func (c *captureCompiler) Compile(src, output string) (string, error) {
	c.src = src
	return "ok", nil
}

type stubRunner struct{}

func (s *stubRunner) Run(exe string, args []string) (string, int, error) {
	return "Hello, world!\r\n", 0, nil
}

type stubDebugger struct{}

func (s *stubDebugger) SetBreakpoint(file string, line int) {}
func (s *stubDebugger) Step() (string, error)               { return "step", nil }
func (s *stubDebugger) Continue() (string, error)           { return "cont", nil }
func (s *stubDebugger) Watch(expr string) (string, error)   { return "0", nil }

func TestBufferInsert(t *testing.T) {
	b := NewBuffer()
	b.InsertString("hello")
	if b.Lines[0] != "hello" {
		t.Errorf("Lines[0]: %q", b.Lines[0])
	}
}

func TestBufferBackspace(t *testing.T) {
	b := NewBuffer()
	b.InsertString("hello")
	b.CursorX = 5
	b.Backspace()
	if b.Lines[0] != "hell" {
		t.Errorf("Backspace: %q", b.Lines[0])
	}
}

func TestBufferNewline(t *testing.T) {
	b := NewBuffer()
	b.InsertString("abcde")
	b.CursorX = 3
	b.InsertNewline()
	if b.Lines[0] != "abc" || b.Lines[1] != "de" {
		t.Errorf("Newline: %v", b.Lines)
	}
}

func TestBufferDelete(t *testing.T) {
	b := NewBuffer()
	b.InsertString("hello")
	b.CursorX = 0
	b.Delete()
	if b.Lines[0] != "ello" {
		t.Errorf("Delete: %q", b.Lines[0])
	}
}

func TestBufferMoveCursor(t *testing.T) {
	b := NewBuffer()
	b.InsertString("abc")
	b.CursorX = 1
	b.MoveCursor(1, 0)
	if b.CursorX != 2 {
		t.Errorf("CursorX: %d", b.CursorX)
	}
}

func TestBufferGotoLine(t *testing.T) {
	b := NewBuffer()
	b.SetText("a\nb\nc")
	if len(b.Lines) != 3 {
		t.Fatalf("Lines: %d", len(b.Lines))
	}
	b.GotoLine(2)
	if b.CursorY != 1 {
		t.Errorf("CursorY: %d", b.CursorY)
	}
}

func TestBufferWordLeft(t *testing.T) {
	b := NewBuffer()
	b.InsertString("foo bar baz")
	b.CursorX = len("foo bar baz")
	b.WordLeft()
	if b.CursorX < 4 {
		t.Errorf("WordLeft: %d", b.CursorX)
	}
}

func TestBufferWordRight(t *testing.T) {
	b := NewBuffer()
	b.InsertString("foo bar")
	b.CursorX = 0
	b.WordRight()
	if b.CursorX == 0 {
		t.Error("WordRight")
	}
}

func TestBufferSetText(t *testing.T) {
	b := NewBuffer()
	b.SetText("a\nb\nc")
	if len(b.Lines) != 3 {
		t.Errorf("Lines: %d", len(b.Lines))
	}
}

func TestBufferFind(t *testing.T) {
	b := NewBuffer()
	b.SetText("foo bar foo")
	if !b.Find("bar") {
		t.Error("Find")
	}
}

func TestBufferReplaceAll(t *testing.T) {
	b := NewBuffer()
	b.SetText("foo bar foo")
	n := b.ReplaceAll("foo", "baz")
	if n != 2 {
		t.Errorf("ReplaceAll: %d", n)
	}
	if b.Text() != "baz bar baz" {
		t.Errorf("Text: %q", b.Text())
	}
}

func TestBufferClampCursor(t *testing.T) {
	b := NewBuffer()
	b.InsertString("abc")
	b.MoveCursor(100, 100)
	if b.CursorX > 3 || b.CursorY > 0 {
		t.Errorf("Cursor: %d/%d", b.CursorX, b.CursorY)
	}
}

func TestBufferClampCursorNegative(t *testing.T) {
	b := NewBuffer()
	b.MoveCursor(-10, -10)
	if b.CursorX != 0 || b.CursorY != 0 {
		t.Errorf("Cursor: %d/%d", b.CursorX, b.CursorY)
	}
}

func TestIDEInit(t *testing.T) {
	ide := newTestIDE()
	if len(ide.Buffers) != 1 {
		t.Errorf("Buffers: %d", len(ide.Buffers))
	}
}

func TestIDERunCommandNew(t *testing.T) {
	ide := newTestIDE()
	if _, err := ide.RunCommand("New"); err != nil {
		t.Error(err)
	}
	if len(ide.Buffers) != 2 {
		t.Errorf("Buffers: %d", len(ide.Buffers))
	}
}

func TestIDERunCommandSave(t *testing.T) {
	ide := newTestIDE()
	ide.Buffers[0].SetText("program T; begin end.")
	if out, err := ide.RunCommand("Save"); err != nil || out == "" {
		t.Errorf("Save: %q %v", out, err)
	}
}

func TestIDERunCommandCompile(t *testing.T) {
	ide := newTestIDE()
	if out, err := ide.RunCommand("Compile"); err != nil || out != "ok" {
		t.Errorf("Compile: %q %v", out, err)
	}
}

func TestIDERunCommandCompileUsesActiveBuffer(t *testing.T) {
	cc := &captureCompiler{}
	ide := New(&Project{Name: "T", Source: "hello.pas", Output: "hello.exe"}, cc, &stubRunner{}, &stubDebugger{})
	ide.Buffers[0].SetText("program T; begin end.")
	if _, err := ide.RunCommand("Compile"); err != nil {
		t.Fatal(err)
	}
	if cc.src != "program T; begin end." {
		t.Errorf("compiler source: %q", cc.src)
	}
}

func TestIDERunCommandRun(t *testing.T) {
	ide := newTestIDE()
	if out, err := ide.RunCommand("Run"); err != nil {
		t.Errorf("Run: %q %v", out, err)
	}
	if ide.LastOut == "" {
		t.Error("LastOut empty")
	}
}

func TestIDERunCommandGotoLine(t *testing.T) {
	ide := newTestIDE()
	ide.Buffers[0].SetText("a\nb\nc")
	if _, err := ide.RunCommand("GotoLine", "2"); err != nil {
		t.Error(err)
	}
	if ide.Buffers[0].CursorY != 1 {
		t.Errorf("CursorY: %d", ide.Buffers[0].CursorY)
	}
}

func TestIDERunCommandFind(t *testing.T) {
	ide := newTestIDE()
	ide.Buffers[0].SetText("hello world")
	if out, _ := ide.RunCommand("Find", "world"); out != "found" {
		t.Errorf("Find: %q", out)
	}
}

func TestIDERunCommandReplace(t *testing.T) {
	ide := newTestIDE()
	ide.Buffers[0].SetText("foo foo")
	if out, _ := ide.RunCommand("Replace", "foo", "bar"); out != "2" {
		t.Errorf("Replace: %q", out)
	}
}

func TestIDERunCommandSetBreakpoint(t *testing.T) {
	ide := newTestIDE()
	if _, err := ide.RunCommand("SetBreakpoint", "10"); err != nil {
		t.Error(err)
	}
}

func TestIDERunCommandDebugCommands(t *testing.T) {
	ide := newTestIDE()
	if out, err := ide.RunCommand("DebugStep"); err != nil || out != "step" {
		t.Errorf("DebugStep: %q %v", out, err)
	}
	if out, err := ide.RunCommand("DebugContinue"); err != nil || out != "cont" {
		t.Errorf("DebugContinue: %q %v", out, err)
	}
	if out, err := ide.RunCommand("Watch", "X"); err != nil || out != "0" {
		t.Errorf("Watch: %q %v", out, err)
	}
}

func TestIDERunCommandMouse(t *testing.T) {
	ide := newTestIDE()
	if _, err := ide.RunCommand("Mouse", "10", "5", "1", "1"); err != nil {
		t.Fatal(err)
	}
	if ide.Mouse.X != 10 || ide.Mouse.Y != 5 || ide.Mouse.Button != 1 || !ide.Mouse.Pressed {
		t.Errorf("mouse: %+v", ide.Mouse)
	}
}

func TestIDERunCommandOpenProject(t *testing.T) {
	ide := newTestIDE()
	if out, err := ide.RunCommand("OpenProject", "Demo", "demo.pas", "demo.exe"); err != nil || out != "Demo" {
		t.Fatalf("OpenProject: %q %v", out, err)
	}
	if ide.Project.Source != "demo.pas" || ide.Project.Output != "demo.exe" {
		t.Errorf("project: %+v", ide.Project)
	}
}

func TestIDERunCommandExit(t *testing.T) {
	ide := newTestIDE()
	if _, err := ide.RunCommand("Exit"); err != nil {
		t.Error(err)
	}
	if !ide.Exited {
		t.Error("Exited")
	}
}

func TestIDERunCommandUnknown(t *testing.T) {
	ide := newTestIDE()
	if _, err := ide.RunCommand("Xyz"); err == nil {
		t.Error("expected error for unknown command")
	}
}

func TestIDEFileMenu(t *testing.T) {
	ide := newTestIDE()
	cmds := ide.CommandsForMenu("File")
	if len(cmds) == 0 {
		t.Error("File menu empty")
	}
}

func TestIDEAllCommands(t *testing.T) {
	ide := newTestIDE()
	all := ide.MenuCommands()
	if len(all) == 0 {
		t.Error("menu empty")
	}
}

func TestIDEFindMenuCommand(t *testing.T) {
	ide := newTestIDE()
	if ide.FindMenuCommand("Run") == nil {
		t.Error("FindMenuCommand")
	}
}

func TestIDEOpenFile(t *testing.T) {
	ide := newTestIDE()
	ide.OpenFile("test.pas", "program T; begin end.")
	if len(ide.Buffers) != 2 {
		t.Errorf("Buffers: %d", len(ide.Buffers))
	}
}

func TestIDESaveAs(t *testing.T) {
	ide := newTestIDE()
	if _, err := ide.RunCommand("SaveAs", "new.pas"); err != nil {
		t.Error(err)
	}
	if ide.Buffers[0].Filename != "new.pas" {
		t.Errorf("Filename: %s", ide.Buffers[0].Filename)
	}
}
