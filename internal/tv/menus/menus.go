// Package menus implements the Turbo Vision Menus unit. The unit
// provides menu items, popup menus, menu bar and menu box. BPGo
// implements them as simple data structures that can be rendered to a
// TDrawBuffer by the host TProgram.
package menus

import (
	"sync"

	"github.com/arturoeanton/go-turbo-pascal/internal/tv/views"
)

// TMenuItem is a single menu entry.
type TMenuItem struct {
	Text     string
	Command  uint16
	Disabled bool
	Checked  bool
	Submenu  *TMenu
	Key      uint16
	HelpCtx  uint16
}

// NewMenuItem creates a menu item from a parameter string (TP style).
func NewMenuItem(s string, cmd uint16) *TMenuItem {
	return &TMenuItem{Text: s, Command: cmd}
}

// TMenu is a list of menu items.
type TMenu struct {
	Items []*TMenuItem
}

// NewMenu creates a menu from items.
func NewMenu(items ...*TMenuItem) *TMenu {
	return &TMenu{Items: items}
}

// TMenuView is the base for menu views.
type TMenuView struct {
	views.TView
	Menu *TMenu
}

// TMenuBar is the top-of-screen menu bar.
type TMenuBar struct {
	TMenuView
}

// NewMenuBar creates a menu bar.
func NewMenuBar(menu *TMenu) *TMenuBar {
	b := &TMenuBar{TMenuView: TMenuView{Menu: menu}}
	b.State |= views.StateVisible
	return b
}

// TMenuBox is a popup menu window.
type TMenuBox struct {
	TMenuView
	Frame *views.TFrame
}

// NewMenuBox creates a popup menu box.
func NewMenuBox(menu *TMenu, bounds views.TRect) *TMenuBox {
	b := &TMenuBox{
		TMenuView: TMenuView{Menu: menu},
		Frame:     (&views.TFrame{}).Init(bounds),
	}
	b.TView.Init(bounds)
	return b
}

// TStatusLine is the bottom status line.
type TStatusLine struct {
	views.TView
	Items []*TStatusItem
	Def   *TStatusDef
	mu    sync.Mutex
}

// TStatusItem is a single status hint.
type TStatusItem struct {
	Text    string
	Key     uint16
	Command uint16
}

// TStatusDef is a list of status items for a context.
type TStatusDef struct {
	Min, Max uint16
	Items    []*TStatusItem
}

// TStatusItemFromText creates a status item from a parameter string.
func TStatusItemFromText(s string, ctx uint16) *TStatusItem {
	return &TStatusItem{Text: s, Key: ctx}
}

// NewStatusLine creates a status line.
func NewStatusLine(bounds views.TRect, def *TStatusDef) *TStatusLine {
	return &TStatusLine{TView: views.TView{Bounds: bounds}, Def: def}
}

// Update rebuilds the status line for a context.
func (s *TStatusLine) Update(ctx uint16) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Def == nil {
		return
	}
	s.Items = nil
	for _, it := range s.Def.Items {
		if it.Key == ctx {
			s.Items = append(s.Items, it)
		}
	}
}
