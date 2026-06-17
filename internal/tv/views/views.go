// Package views implements the Turbo Vision Views unit. The unit
// provides the visual hierarchy: TPoint, TRect, TView, TGroup,
// TFrame, TScrollBar, TScroller, TListViewer, TStaticText,
// TParamText, TLabel, THistory, TInputLine, TButton, TCluster,
// TRadioButtons, TCheckBoxes.
package views

import (
	"sync"
)

// TPoint is a screen coordinate.
type TPoint struct {
	X, Y int
}

// TRect is a screen rectangle.
type TRect struct {
	A, B TPoint
}

// Empty reports whether the rectangle is empty.
func (r TRect) Empty() bool {
	return r.A.X >= r.B.X || r.A.Y >= r.B.Y
}

// Equal reports whether two rectangles are equal.
func (r TRect) Equal(o TRect) bool {
	return r.A == o.A && r.B == o.B
}

// Contains reports whether the point is inside the rectangle.
func (r TRect) Contains(p TPoint) bool {
	return p.X >= r.A.X && p.X < r.B.X && p.Y >= r.A.Y && p.Y < r.B.Y
}

// Intersect returns the intersection of two rectangles.
func (r TRect) Intersect(o TRect) TRect {
	if r.Empty() || o.Empty() {
		return TRect{}
	}
	a := TPoint{X: max(r.A.X, o.A.X), Y: max(r.A.Y, o.A.Y)}
	b := TPoint{X: min(r.B.X, o.B.X), Y: min(r.B.Y, o.B.Y)}
	if a.X >= b.X || a.Y >= b.Y {
		return TRect{}
	}
	return TRect{a, b}
}

// Union returns the bounding box of two rectangles.
func (r TRect) Union(o TRect) TRect {
	if r.Empty() {
		return o
	}
	if o.Empty() {
		return r
	}
	return TRect{
		TPoint{X: min(r.A.X, o.A.X), Y: min(r.A.Y, o.A.Y)},
		TPoint{X: max(r.B.X, o.B.X), Y: max(r.B.Y, o.B.Y)},
	}
}

// Move offsets the rectangle.
func (r TRect) Move(dx, dy int) TRect {
	return TRect{TPoint{r.A.X + dx, r.A.Y + dy}, TPoint{r.B.X + dx, r.B.Y + dy}}
}

// Grow expands the rectangle.
func (r TRect) Grow(dx, dy int) TRect {
	return TRect{TPoint{r.A.X - dx, r.A.Y - dy}, TPoint{r.B.X + dx, r.B.Y + dy}}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// State flags for views.
const (
	StateVisible = 1 << iota
	StateCursor
	StateShadow
	StateActive
	StateFocused
	StateSelected
	StateDisabled
	StateModal
	StateExposed
)

// DrawBuffer is a placeholder for a TV draw buffer line.
type DrawBuffer struct {
	Cells []uint16
}

// DrawBufferAt writes a cell at offset.
func (d *DrawBuffer) DrawBufferAt(offset int, c byte, attr byte, count int) {
	for i := 0; i < count; i++ {
		if offset+i >= len(d.Cells) {
			break
		}
		d.Cells[offset+i] = uint16(c) | (uint16(attr) << 8)
	}
}

// TView is the base visual class.
type TView struct {
	mu      sync.Mutex
	Bounds  TRect
	State   uint16
	Options uint16
	Owner   *TGroup
	Focus   *TView
	Next    *TView
	Prev    *TView
	HelpCtx uint16
	Command uint16
	OnEvent func(*TView, interface{}) bool
}

// Init constructs a TView.
func (v *TView) Init(bounds TRect) *TView {
	v.Bounds = bounds
	v.State = StateVisible
	return v
}

// Show makes the view visible.
func (v *TView) Show() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.State |= StateVisible
}

// Hide hides the view.
func (v *TView) Hide() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.State &^= StateVisible
}

// Lock locks the view's mutex.
func (v *TView) Lock() { v.mu.Lock() }

// Unlock unlocks the view's mutex.
func (v *TView) Unlock() { v.mu.Unlock() }

// Draw draws the view (default no-op).
func (v *TView) Draw() {}

// HandleEvent dispatches an event.
func (v *TView) HandleEvent(ev interface{}) bool {
	if v.OnEvent != nil {
		return v.OnEvent(v, ev)
	}
	return false
}

// Locate finds a subview at the given point.
func (v *TView) Locate(p TPoint) *TView {
	if !v.Bounds.Contains(p) {
		return nil
	}
	return v
}

// TGroup is a container of views.
type TGroup struct {
	TView
	Children []*TView
	Current  *TView
}

// Init constructs a TGroup.
func (g *TGroup) Init(bounds TRect) *TGroup {
	g.TView.Init(bounds)
	g.State |= StateShadow
	return g
}

// Insert adds a child view.
func (g *TGroup) Insert(child *TView) {
	g.mu.Lock()
	defer g.mu.Unlock()
	child.Owner = g
	g.Children = append(g.Children, child)
	if g.Current == nil {
		g.Current = child
	}
}

// Remove removes a child view.
func (g *TGroup) Remove(child *TView) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i, c := range g.Children {
		if c == child {
			g.Children = append(g.Children[:i], g.Children[i+1:]...)
			child.Owner = nil
			if g.Current == child {
				if i < len(g.Children) {
					g.Current = g.Children[i]
				} else if i > 0 {
					g.Current = g.Children[i-1]
				} else {
					g.Current = nil
				}
			}
			return
		}
	}
}

// Draw draws all children.
func (g *TGroup) Draw() {
	for _, c := range g.Children {
		c.Draw()
	}
}

// TFrame draws a frame around its bounds.
type TFrame struct {
	TView
}

// Init constructs a TFrame.
func (f *TFrame) Init(bounds TRect) *TFrame {
	f.TView.Init(bounds)
	return f
}

// Draw draws the frame characters.
func (f *TFrame) Draw() {
	// Default: just mark as drawn by setting exposed.
	f.State |= StateExposed
}

// TScrollBar is a vertical or horizontal scroll bar.
type TScrollBar struct {
	TView
	Min, Max, Pos, PgStep int
	Arrows                bool
}

// Init constructs a TScrollBar.
func (s *TScrollBar) Init(bounds TRect) *TScrollBar {
	s.TView.Init(bounds)
	return s
}

// SetRange sets the scroll range.
func (s *TScrollBar) SetRange(min, max int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Min = min
	s.Max = max
}

// SetPosition moves the thumb.
func (s *TScrollBar) SetPosition(p int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Pos = p
}

// TScroller is a viewport over a larger virtual area.
type TScroller struct {
	TView
	HScroll, VScroll *TScrollBar
	Delta            TPoint
}

// Init constructs a TScroller.
func (s *TScroller) Init(bounds TRect, h, v *TScrollBar) *TScroller {
	s.TView.Init(bounds)
	s.HScroll = h
	s.VScroll = v
	return s
}

// TListViewer is a scrollable list.
type TListViewer struct {
	TScroller
	Items   []string
	Focused int
}

// Init constructs a TListViewer.
func (l *TListViewer) Init(bounds TRect, columns int, h, v *TScrollBar) *TListViewer {
	l.TScroller.Init(bounds, h, v)
	return l
}

// SetRange sets the visible range.
func (l *TListViewer) SetRange(n int) {
	if l.VScroll != nil {
		l.VScroll.SetRange(0, n)
	}
}

// TStaticText is a non-interactive text label.
type TStaticText struct {
	TView
	Text string
}

// Init constructs a TStaticText.
func (s *TStaticText) Init(bounds TRect, text string) *TStaticText {
	s.TView.Init(bounds)
	s.Text = text
	return s
}

// TParamText is text with parameter substitution.
type TParamText struct {
	TStaticText
	Params []string
}

// Init constructs a TParamText.
func (p *TParamText) Init(bounds TRect, text string, count int) *TParamText {
	p.TStaticText.Init(bounds, text)
	p.Params = make([]string, count)
	return p
}

// GetText returns the text with parameter substitution.
func (p *TParamText) GetText() string {
	out := p.Text
	for i, s := range p.Params {
		out = strings_Replace(out, "%"+string(rune('0'+i)), s, 1)
	}
	return out
}

func strings_Replace(s, old, new string, n int) string {
	if old == "" {
		return s
	}
	for i := 0; i < n; i++ {
		idx := indexOf(s, old)
		if idx < 0 {
			break
		}
		s = s[:idx] + new + s[idx+len(old):]
	}
	return s
}

func indexOf(s, sub string) int {
	if len(sub) == 0 {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// TLabel is a hotkey-aware static text.
type TLabel struct {
	TStaticText
	Link *TView
}

// Init constructs a TLabel.
func (l *TLabel) Init(bounds TRect, text string, link *TView) *TLabel {
	l.TStaticText.Init(bounds, text)
	l.Link = link
	return l
}

// THistory is a button for opening a history list.
type THistory struct {
	TView
	Link      *TView
	HistoryID uint16
}

// Init constructs a THistory.
func (h *THistory) Init(bounds TRect, link *TView, id uint16) *THistory {
	h.TView.Init(bounds)
	h.Link = link
	h.HistoryID = id
	return h
}

// TInputLine is a single-line text editor.
type TInputLine struct {
	TView
	Data     []byte
	MaxLen   int
	FirstPos int
	Cursor   int
}

// Init constructs a TInputLine.
func (i *TInputLine) Init(bounds TRect, maxLen int) *TInputLine {
	i.TView.Init(bounds)
	i.MaxLen = maxLen
	i.Data = make([]byte, 0, maxLen)
	return i
}

// SetData sets the contents.
func (i *TInputLine) SetData(s string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.Data = []byte(s)
	i.Cursor = len(i.Data)
}

// GetData returns the contents.
func (i *TInputLine) GetData() string {
	i.mu.Lock()
	defer i.mu.Unlock()
	return string(i.Data)
}

// TButton is a clickable button.
type TButton struct {
	TView
	Title   string
	Command uint16
	Down    bool
}

// Init constructs a TButton.
func (b *TButton) Init(bounds TRect, title string, cmd uint16, flags uint16) *TButton {
	b.TView.Init(bounds)
	b.Title = title
	b.Command = cmd
	return b
}

// TCluster is the base for radio/checkbox groups.
type TCluster struct {
	TView
	Items []string
	Value uint16
	Sel   int
}

// Init constructs a TCluster.
func (c *TCluster) Init(bounds TRect, items []string) *TCluster {
	c.TView.Init(bounds)
	c.Items = items
	return c
}

// TRadioButtons is a set of radio buttons.
type TRadioButtons struct {
	TCluster
}

// Init constructs a TRadioButtons.
func (r *TRadioButtons) Init(bounds TRect, items []string) *TRadioButtons {
	r.TCluster.Init(bounds, items)
	return r
}

// Press selects a button.
func (r *TRadioButtons) Press(item int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Value = uint16(item)
	r.Sel = item
}

// TCheckBoxes is a set of checkboxes.
type TCheckBoxes struct {
	TCluster
}

// Init constructs a TCheckBoxes.
func (c *TCheckBoxes) Init(bounds TRect, items []string) *TCheckBoxes {
	c.TCluster.Init(bounds, items)
	return c
}

// Toggle toggles a checkbox.
func (c *TCheckBoxes) Toggle(item int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Value ^= (1 << uint(item))
}
