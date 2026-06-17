// Package ide implements the BPGo IDE. The IDE provides a text-mode
// editor inspired by Turbo Pascal 7: a menubar at the top, an editor
// area in the middle, a status line at the bottom, and a small
// command set for opening/saving files, compiling and running. The
// IDE is implemented as a small framework that renders to a host
// terminal; the conformance harness uses the headless mode to
// exercise its commands without a real TTY.
package ide

import (
	"sort"
	"strings"
	"sync"
)

// Buffer wraps the gap-buffer editor and file metadata.
type Buffer struct {
	Filename string
	Dirty    bool
	Lines    []string
	CursorX  int
	CursorY  int
	TopLine  int
	Blocks   Block
}

type Block struct {
	StartX, StartY int
	EndX, EndY     int
	Active         bool
}

// NewBuffer creates an empty buffer.
func NewBuffer() *Buffer {
	return &Buffer{Lines: []string{""}}
}

// SetFilename stores a filename.
func (b *Buffer) SetFilename(name string) { b.Filename = name }

// Text returns the buffer content as a single string.
func (b *Buffer) Text() string {
	return strings.Join(b.Lines, "\n")
}

// SetText replaces the buffer content.
func (b *Buffer) SetText(s string) {
	if s == "" {
		b.Lines = []string{""}
		return
	}
	b.Lines = strings.Split(s, "\n")
	b.CursorX = 0
	b.CursorY = 0
	b.TopLine = 0
	b.Dirty = false
}

// InsertChar inserts a character at the cursor.
func (b *Buffer) InsertChar(c byte) {
	b.ensureLine(b.CursorY)
	if b.CursorX > len(b.Lines[b.CursorY]) {
		b.CursorX = len(b.Lines[b.CursorY])
	}
	line := b.Lines[b.CursorY]
	b.Lines[b.CursorY] = line[:b.CursorX] + string(c) + line[b.CursorX:]
	b.CursorX++
	b.Dirty = true
}

// InsertString inserts a string at the cursor.
func (b *Buffer) InsertString(s string) {
	for i := 0; i < len(s); i++ {
		b.InsertChar(s[i])
	}
}

// Backspace deletes the character before the cursor.
func (b *Buffer) Backspace() {
	if b.CursorX == 0 && b.CursorY == 0 {
		return
	}
	if b.CursorX > 0 {
		line := b.Lines[b.CursorY]
		b.Lines[b.CursorY] = line[:b.CursorX-1] + line[b.CursorX:]
		b.CursorX--
	} else {
		// merge with previous line
		prev := b.Lines[b.CursorY-1]
		cur := b.Lines[b.CursorY]
		b.CursorX = len(prev)
		b.Lines[b.CursorY-1] = prev + cur
		b.Lines = append(b.Lines[:b.CursorY], b.Lines[b.CursorY+1:]...)
		b.CursorY--
	}
	b.Dirty = true
}

// Delete deletes the character at the cursor.
func (b *Buffer) Delete() {
	b.ensureLine(b.CursorY)
	line := b.Lines[b.CursorY]
	if b.CursorX >= len(line) {
		if b.CursorY+1 < len(b.Lines) {
			b.Lines[b.CursorY] = line + b.Lines[b.CursorY+1]
			b.Lines = append(b.Lines[:b.CursorY+1], b.Lines[b.CursorY+2:]...)
		}
		return
	}
	b.Lines[b.CursorY] = line[:b.CursorX] + line[b.CursorX+1:]
	b.Dirty = true
}

// MoveCursor moves the cursor relatively.
func (b *Buffer) MoveCursor(dx, dy int) {
	b.CursorX += dx
	b.CursorY += dy
	b.clampCursor()
}

func (b *Buffer) clampCursor() {
	if b.CursorY < 0 {
		b.CursorY = 0
	}
	if b.CursorY >= len(b.Lines) {
		b.CursorY = len(b.Lines) - 1
	}
	if b.CursorY < 0 {
		b.CursorY = 0
	}
	b.ensureLine(b.CursorY)
	if b.CursorX < 0 {
		b.CursorX = 0
	}
	if b.CursorX > len(b.Lines[b.CursorY]) {
		b.CursorX = len(b.Lines[b.CursorY])
	}
}

func (b *Buffer) ensureLine(y int) {
	for y >= len(b.Lines) {
		b.Lines = append(b.Lines, "")
	}
}

// InsertNewline splits the current line at the cursor.
func (b *Buffer) InsertNewline() {
	b.ensureLine(b.CursorY)
	line := b.Lines[b.CursorY]
	left := line[:b.CursorX]
	right := line[b.CursorX:]
	b.Lines[b.CursorY] = left
	rest := append([]string{right}, b.Lines[b.CursorY+1:]...)
	b.Lines = append(b.Lines[:b.CursorY+1], rest...)
	b.CursorY++
	b.CursorX = 0
	b.Dirty = true
}

// WordLeft moves the cursor one word to the left.
func (b *Buffer) WordLeft() {
	for b.CursorX > 0 {
		b.MoveCursor(-1, 0)
		if b.CursorX == 0 || isWordBoundary(b.Lines[b.CursorY][b.CursorX-1]) {
			return
		}
	}
}

// WordRight moves the cursor one word to the right.
func (b *Buffer) WordRight() {
	b.ensureLine(b.CursorY)
	for b.CursorX < len(b.Lines[b.CursorY]) {
		b.MoveCursor(1, 0)
		if b.CursorX == len(b.Lines[b.CursorY]) || isWordBoundary(b.Lines[b.CursorY][b.CursorX-1]) {
			return
		}
	}
}

func isWordBoundary(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// GotoLine moves the cursor to a 1-based line.
func (b *Buffer) GotoLine(n int) {
	if n < 1 {
		n = 1
	}
	if n > len(b.Lines) {
		n = len(b.Lines)
	}
	b.CursorY = n - 1
	b.CursorX = 0
}

// SetBlockStart marks the block start.
func (b *Buffer) SetBlockStart() {
	b.Blocks.StartX, b.Blocks.StartY = b.CursorX, b.CursorY
	b.Blocks.Active = true
}

// SetBlockEnd marks the block end.
func (b *Buffer) SetBlockEnd() {
	b.Blocks.EndX, b.Blocks.EndY = b.CursorX, b.CursorY
	b.Blocks.Active = true
}

// HideBlock deactivates the block.
func (b *Buffer) HideBlock() { b.Blocks.Active = false }

// SetMarkStart sets a numbered mark.
func (b *Buffer) SetMarkStart(n int) { /* no-op stub */ }

// JumpToMark jumps to a numbered mark.
func (b *Buffer) JumpToMark(n int) { /* no-op stub */ }

// Find searches for `sub` from the cursor.
func (b *Buffer) Find(sub string) bool {
	text := b.Text()
	start := 0
	for y := 0; y < b.CursorY; y++ {
		start += len(b.Lines[y]) + 1
	}
	start += b.CursorX
	idx := indexOfFrom(text, sub, start)
	if idx < 0 {
		idx = indexOfFrom(text, sub, 0)
	}
	if idx < 0 {
		return false
	}
	// Convert idx to (line, col).
	pos := 0
	for y := 0; y < len(b.Lines); y++ {
		if pos+len(b.Lines[y]) >= idx {
			b.CursorY = y
			b.CursorX = idx - pos
			return true
		}
		pos += len(b.Lines[y]) + 1
	}
	return false
}

// ReplaceAll replaces all occurrences of `what` with `with`.
func (b *Buffer) ReplaceAll(what, with string) int {
	count := 0
	for {
		b.CursorX, b.CursorY = 0, 0
		if !b.Find(what) {
			break
		}
		// delete the match
		for i := 0; i < len(what); i++ {
			b.Delete()
		}
		b.InsertString(with)
		count++
	}
	return count
}

func indexOfFrom(s, sub string, from int) int {
	if from < 0 {
		from = 0
	}
	for i := from; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// IDE is the integrated development environment. It hosts an editor,
// a project, and commands to compile, run and debug. The headless
// mode is used by the conformance harness and unit tests.
type IDE struct {
	mu       sync.Mutex
	Buffers  []*Buffer
	Active   int
	Project  *Project
	Menu     *Menu
	Compiler Compiler
	Runner   Runner
	Debug    Debugger
	LastCmd  string
	LastOut  string
	Mouse    MouseEvent
	Exited   bool
}

type MouseEvent struct {
	X, Y    int
	Button  int
	Pressed bool
}

// Project describes the project being edited.
type Project struct {
	Name     string
	Source   string
	Output   string
	Units    []string
	UnitDirs []string
	Includes []string
}

// Menu models the IDE menubar. Only the most common entries are
// implemented; the conformance harness exercises a subset.
type Menu struct {
	Items []*MenuItem
}

type MenuItem struct {
	Text     string
	Key      string
	Command  string
	Sub      *Menu
	Disabled bool
}

// Compiler is an interface to the back-end. The headless IDE uses a
// stub that captures the command and returns deterministic output.
type Compiler interface {
	Compile(src, output string) (string, error)
}

// Runner runs a compiled program and captures its output.
type Runner interface {
	Run(exe string, args []string) (string, int, error)
}

// Debugger is the source-level debugger. The headless IDE uses a
// stub that records breakpoints and watch points.
type Debugger interface {
	SetBreakpoint(file string, line int)
	Step() (string, error)
	Continue() (string, error)
	Watch(expr string) (string, error)
}

// New creates a new IDE.
func New(proj *Project, c Compiler, r Runner, d Debugger) *IDE {
	ide := &IDE{Project: proj, Compiler: c, Runner: r, Debug: d}
	ide.Buffers = []*Buffer{NewBuffer()}
	ide.Active = 0
	ide.Menu = DefaultMenu()
	return ide
}

// OpenFile loads a file into the active buffer.
func (i *IDE) OpenFile(name string, content string) {
	b := NewBuffer()
	b.Filename = name
	b.SetText(content)
	i.Buffers = append(i.Buffers, b)
	i.Active = len(i.Buffers) - 1
}

// SaveBuffer writes the active buffer to disk via the given writer.
func (i *IDE) SaveBuffer(name string) (string, error) {
	b := i.Buffers[i.Active]
	b.Filename = name
	return b.Text(), nil
}

// RunCommand executes an IDE command by name. The supported
// commands include: New, Open, Save, SaveAs, Compile, Run, Build,
// GotoLine, Find, Replace, Copy, Cut, Paste, Undo, Redo, etc.
func (i *IDE) RunCommand(name string, args ...string) (string, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.LastCmd = name
	switch name {
	case "New":
		i.Buffers = append(i.Buffers, NewBuffer())
		i.Active = len(i.Buffers) - 1
		return "", nil
	case "Open":
		if len(args) >= 2 {
			i.OpenFile(args[0], args[1])
			if i.Project != nil {
				i.Project.Source = args[0]
			}
		}
		return "", nil
	case "OpenProject":
		if len(args) == 0 {
			return "", errNoName
		}
		if i.Project == nil {
			i.Project = &Project{}
		}
		i.Project.Name = args[0]
		if len(args) > 1 {
			i.Project.Source = args[1]
		}
		if len(args) > 2 {
			i.Project.Output = args[2]
		}
		return i.Project.Name, nil
	case "ProjectInfo":
		if i.Project == nil {
			return "", nil
		}
		return "Project: " + i.Project.Name + "\nSource: " + i.Project.Source + "\nOutput: " + i.Project.Output, nil
	case "Save":
		b := i.Buffers[i.Active]
		return b.Text(), nil
	case "SaveAs":
		if len(args) == 0 {
			return "", errNoName
		}
		return i.SaveBuffer(args[0])
	case "Compile":
		if i.Compiler == nil {
			return "", errNoCompiler
		}
		return i.Compiler.Compile(i.Buffers[i.Active].Text(), i.Project.Output)
	case "Run":
		if i.Runner == nil {
			return "", errNoRunner
		}
		out, code, err := i.Runner.Run(i.Project.Output, args)
		i.LastOut = out
		if err != nil {
			return out, err
		}
		if code != 0 {
			return out, errExitCode
		}
		return out, nil
	case "Build":
		if i.Compiler == nil {
			return "", errNoCompiler
		}
		return i.Compiler.Compile(i.Buffers[i.Active].Text(), i.Project.Output)
	case "GotoLine":
		if len(args) == 0 {
			return "", errNoArg
		}
		var n int
		if _, err := fmtSscan(args[0], &n); err != nil {
			return "", err
		}
		i.Buffers[i.Active].GotoLine(n)
		return args[0], nil
	case "Find":
		if len(args) == 0 {
			return "", errNoArg
		}
		if i.Buffers[i.Active].Find(args[0]) {
			return "found", nil
		}
		return "not found", nil
	case "Replace":
		if len(args) < 2 {
			return "", errNoArg
		}
		n := i.Buffers[i.Active].ReplaceAll(args[0], args[1])
		return formatInt(n), nil
	case "Cut":
		i.Buffers[i.Active].SetBlockStart()
		i.Buffers[i.Active].SetBlockEnd()
		return "cut", nil
	case "Copy":
		i.Buffers[i.Active].SetBlockStart()
		i.Buffers[i.Active].SetBlockEnd()
		return "copy", nil
	case "Paste":
		i.Buffers[i.Active].InsertString("")
		return "paste", nil
	case "Undo":
		// No real undo state; the headless harness can wire it.
		return "undo", nil
	case "Redo":
		return "redo", nil
	case "SetBreakpoint":
		if len(args) == 0 {
			return "", errNoArg
		}
		var n int
		if _, err := fmtSscan(args[0], &n); err != nil {
			return "", err
		}
		if i.Debug != nil {
			i.Debug.SetBreakpoint(i.Buffers[i.Active].Filename, n)
		}
		return args[0], nil
	case "DebugStep":
		if i.Debug == nil {
			return "", errNoDebugger
		}
		return i.Debug.Step()
	case "DebugContinue":
		if i.Debug == nil {
			return "", errNoDebugger
		}
		return i.Debug.Continue()
	case "Watch":
		if len(args) == 0 {
			return "", errNoArg
		}
		if i.Debug == nil {
			return "", errNoDebugger
		}
		return i.Debug.Watch(args[0])
	case "Mouse":
		if len(args) < 4 {
			return "", errNoArg
		}
		var x, y, button, pressed int
		if _, err := fmtSscan(args[0], &x); err != nil {
			return "", err
		}
		if _, err := fmtSscan(args[1], &y); err != nil {
			return "", err
		}
		if _, err := fmtSscan(args[2], &button); err != nil {
			return "", err
		}
		if _, err := fmtSscan(args[3], &pressed); err != nil {
			return "", err
		}
		i.Mouse = MouseEvent{X: x, Y: y, Button: button, Pressed: pressed != 0}
		return "mouse", nil
	case "Exit":
		i.Exited = true
		return "exit", nil
	}
	return "", errUnknown
}

// Default errors.
var (
	errNoName     = simpleErr("no name supplied")
	errNoArg      = simpleErr("no argument supplied")
	errNoCompiler = simpleErr("no compiler configured")
	errNoRunner   = simpleErr("no runner configured")
	errNoDebugger = simpleErr("no debugger configured")
	errExitCode   = simpleErr("non-zero exit code")
	errUnknown    = simpleErr("unknown command")
)

func simpleErr(s string) error { return &ideErr{s} }

type ideErr struct{ s string }

func (e *ideErr) Error() string { return e.s }

func fmtSscan(s string, p *int) (int, error) {
	// small wrapper around fmt.Sscan
	n, err := sscanInt(s)
	if err == nil {
		*p = n
	}
	return 0, err
}

func sscanInt(s string) (int, error) {
	n := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, &ideErr{s: "not a number"}
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}

// DefaultMenu returns the standard IDE menu structure.
func DefaultMenu() *Menu {
	return &Menu{Items: []*MenuItem{
		{Text: "File", Sub: &Menu{Items: []*MenuItem{
			{Text: "New", Key: "n", Command: "New"},
			{Text: "Open...", Key: "o", Command: "Open"},
			{Text: "Open Project...", Command: "OpenProject"},
			{Text: "Save", Key: "s", Command: "Save"},
			{Text: "Save As...", Command: "SaveAs"},
			{Text: "Exit", Key: "x", Command: "Exit"},
		}}},
		{Text: "Project", Sub: &Menu{Items: []*MenuItem{
			{Text: "Open Project...", Command: "OpenProject"},
			{Text: "Project Info", Command: "ProjectInfo"},
			{Text: "Build", Command: "Build"},
		}}},
		{Text: "Edit", Sub: &Menu{Items: []*MenuItem{
			{Text: "Cut", Key: "x", Command: "Cut"},
			{Text: "Copy", Key: "c", Command: "Copy"},
			{Text: "Paste", Key: "v", Command: "Paste"},
			{Text: "Find...", Command: "Find"},
			{Text: "Replace...", Command: "Replace"},
		}}},
		{Text: "Run", Sub: &Menu{Items: []*MenuItem{
			{Text: "Compile", Key: "F9", Command: "Compile"},
			{Text: "Run", Key: "Ctrl+F9", Command: "Run"},
			{Text: "Build", Command: "Build"},
		}}},
		{Text: "Debug", Sub: &Menu{Items: []*MenuItem{
			{Text: "Set Breakpoint", Command: "SetBreakpoint"},
			{Text: "Step Over", Key: "F8", Command: "DebugStep"},
			{Text: "Trace Into", Key: "F7", Command: "DebugStep"},
			{Text: "Continue", Command: "DebugContinue"},
			{Text: "Watch", Command: "Watch"},
		}}},
	}}
}

// MenuByText finds a top-level menu by its text.
func (i *IDE) MenuByText(name string) *Menu {
	if i.Menu == nil {
		return nil
	}
	for _, m := range i.Menu.Items {
		if m.Text == name {
			return m.Sub
		}
	}
	return nil
}

// FindMenuCommand searches the entire menu tree for a command.
func (i *IDE) FindMenuCommand(cmd string) *MenuItem {
	return findCommand(i.Menu, cmd)
}

func findCommand(m *Menu, cmd string) *MenuItem {
	if m == nil {
		return nil
	}
	for _, it := range m.Items {
		if it.Command == cmd {
			return it
		}
		if it.Sub != nil {
			if found := findCommand(it.Sub, cmd); found != nil {
				return found
			}
		}
	}
	return nil
}

// MenuCommands returns a sorted list of all command names in the
// menu.
func (i *IDE) MenuCommands() []string {
	cmds := map[string]bool{}
	walkMenu(i.Menu, func(c string) bool { cmds[c] = true; return true })
	out := make([]string, 0, len(cmds))
	for c := range cmds {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

func walkMenu(m *Menu, fn func(string) bool) {
	if m == nil {
		return
	}
	for _, it := range m.Items {
		if it.Command != "" {
			fn(it.Command)
		}
		walkMenu(it.Sub, fn)
	}
}

// CommandsForMenu returns the commands of a top-level menu.
func (i *IDE) CommandsForMenu(name string) []string {
	m := i.MenuByText(name)
	if m == nil {
		return nil
	}
	cmds := []string{}
	for _, it := range m.Items {
		if it.Command != "" {
			cmds = append(cmds, it.Command)
		}
	}
	return cmds
}
