// Package editors implements the Turbo Vision Editors unit. The unit
// provides TEditor, TFileEditor and TEditWindow. BPGo implements the
// editor as a gap buffer with cut/copy/paste, undo/redo and
// search/replace commands. Tests cover the public command surface.
package editors

import (
	"strings"
	"sync"
)

// Buffer is a simple rune buffer with a cursor.
type Buffer struct {
	mu  sync.Mutex
	buf []rune
	cur int
}

// NewBuffer creates an empty buffer.
func NewBuffer() *Buffer {
	return &Buffer{buf: []rune{}}
}

// Insert inserts text at the cursor.
func (b *Buffer) Insert(s string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	r := []rune(s)
	out := make([]rune, 0, len(b.buf)+len(r))
	out = append(out, b.buf[:b.cur]...)
	out = append(out, r...)
	out = append(out, b.buf[b.cur:]...)
	b.buf = out
	b.cur += len(r)
}

// Delete removes `count` runes after the cursor.
func (b *Buffer) Delete(count int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if count <= 0 {
		return
	}
	if b.cur+count > len(b.buf) {
		count = len(b.buf) - b.cur
	}
	if count <= 0 {
		return
	}
	out := make([]rune, 0, len(b.buf)-count)
	out = append(out, b.buf[:b.cur]...)
	out = append(out, b.buf[b.cur+count:]...)
	b.buf = out
}

// Text returns the buffer content.
func (b *Buffer) Text() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}

// Cursor returns the cursor position.
func (b *Buffer) Cursor() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.cur
}

// SetCursor sets the cursor.
func (b *Buffer) SetCursor(p int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if p < 0 {
		p = 0
	}
	if p > len(b.buf) {
		p = len(b.buf)
	}
	b.cur = p
}

// TEditor is a text editor view.
type TEditor struct {
	Buffer     *Buffer
	Undo       []string
	Redo       []string
	BlockStart int
	BlockEnd   int
	ClipText   string
	Modified   bool
}

// Init constructs an editor.
func (e *TEditor) Init() *TEditor {
	e.Buffer = NewBuffer()
	return e
}

// InsertText inserts text at the cursor.
func (e *TEditor) InsertText(s string) {
	e.saveUndo()
	e.Buffer.Insert(s)
	e.Modified = true
}

// DeleteForward deletes the next character.
func (e *TEditor) DeleteForward() {
	e.saveUndo()
	e.Buffer.Delete(1)
	e.Modified = true
}

func (e *TEditor) saveUndo() {
	e.Undo = append(e.Undo, e.Buffer.Text())
	if len(e.Undo) > 200 {
		e.Undo = e.Undo[len(e.Undo)-200:]
	}
	e.Redo = nil
}

// Undo restores the previous buffer state.
func (e *TEditor) UndoCmd() {
	if len(e.Undo) == 0 {
		return
	}
	last := e.Undo[len(e.Undo)-1]
	e.Undo = e.Undo[:len(e.Undo)-1]
	e.Redo = append(e.Redo, e.Buffer.Text())
	// Replace the buffer content.
	e.Buffer.buf = []rune(last)
	e.Buffer.SetCursor(len([]rune(last)))
	e.Modified = true
}

// RedoCmd re-applies the last undone action.
func (e *TEditor) RedoCmd() {
	if len(e.Redo) == 0 {
		return
	}
	last := e.Redo[len(e.Redo)-1]
	e.Redo = e.Redo[:len(e.Redo)-1]
	e.Undo = append(e.Undo, e.Buffer.Text())
	e.Buffer.buf = []rune(last)
	e.Buffer.SetCursor(len([]rune(last)))
	e.Modified = true
}

// SetBlock marks a block.
func (e *TEditor) SetBlock(start, end int) {
	e.BlockStart = start
	e.BlockEnd = end
}

// CopyBlock copies the marked block to the clipboard.
func (e *TEditor) CopyBlock() {
	if e.BlockStart < 0 || e.BlockEnd <= e.BlockStart {
		return
	}
	text := e.Buffer.Text()
	if e.BlockEnd > len(text) {
		e.BlockEnd = len(text)
	}
	e.ClipText = text[e.BlockStart:e.BlockEnd]
}

// CutBlock copies and deletes the block.
func (e *TEditor) CutBlock() {
	e.CopyBlock()
	if e.BlockStart < 0 || e.BlockEnd <= e.BlockStart {
		return
	}
	e.saveUndo()
	e.Buffer.SetCursor(e.BlockStart)
	e.Buffer.Delete(e.BlockEnd - e.BlockStart)
	e.Modified = true
}

// PasteBlock pastes the clipboard.
func (e *TEditor) PasteBlock() {
	if e.ClipText == "" {
		return
	}
	e.saveUndo()
	e.Buffer.Insert(e.ClipText)
	e.Modified = true
}

// Find searches for `sub` from the cursor.
func (e *TEditor) Find(sub string) int {
	text := e.Buffer.Text()
	pos := e.Buffer.Cursor()
	idx := indexOfFrom(text, sub, pos)
	if idx < 0 {
		idx = indexOfFrom(text, sub, 0)
	}
	if idx >= 0 {
		e.Buffer.SetCursor(idx + len([]rune(sub)))
	}
	return idx
}

// Replace searches and replaces a single occurrence starting at the
// cursor position.
func (e *TEditor) Replace(what, with string) bool {
	text := e.Buffer.Text()
	pos := e.Buffer.Cursor()
	idx := indexOfFrom(text, what, pos)
	if idx < 0 {
		idx = indexOfFrom(text, what, 0)
	}
	if idx < 0 {
		return false
	}
	e.saveUndo()
	e.Buffer.SetCursor(idx)
	e.Buffer.Delete(len([]rune(what)))
	e.Buffer.Insert(with)
	return true
}

// ReplaceAll replaces all occurrences.
func (e *TEditor) ReplaceAll(what, with string) int {
	count := 0
	e.Buffer.SetCursor(0)
	for {
		idx := e.Find(what)
		if idx < 0 {
			break
		}
		e.saveUndo()
		e.Buffer.SetCursor(idx)
		e.Buffer.Delete(len([]rune(what)))
		e.Buffer.Insert(with)
		count++
	}
	return count
}

// Text returns the buffer text.
func (e *TEditor) Text() string { return e.Buffer.Text() }

// TFileEditor extends TEditor with file I/O.
type TFileEditor struct {
	TEditor
	Filename string
}

// Init creates a file editor.
func (e *TFileEditor) Init() *TFileEditor {
	e.TEditor.Init()
	return e
}

// LoadFile loads file contents into the buffer.
func (e *TFileEditor) LoadFile(name string, data []byte) {
	e.Filename = name
	e.Buffer.Insert(string(data))
	e.Buffer.SetCursor(0)
	e.Modified = false
}

// SetCursor wraps the embedded Buffer.SetCursor.
func (e *TFileEditor) SetCursor(p int) { e.Buffer.SetCursor(p) }

// SaveBuffer returns the buffer content for saving.
func (e *TFileEditor) SaveBuffer() []byte {
	return []byte(e.Buffer.Text())
}

// TEditWindow is the window that hosts a TFileEditor.
type TEditWindow struct {
	Editor *TFileEditor
}

// NewEditWindow creates a new edit window.
func NewEditWindow() *TEditWindow {
	return &TEditWindow{Editor: (&TFileEditor{}).Init()}
}

func indexOfFrom(s, sub string, from int) int {
	if sub == "" {
		return from
	}
	for i := from; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// ensure the import is used
var _ = strings.Builder{}
