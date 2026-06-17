// Package outline implements the Turbo Vision Outline unit. The unit
// provides TOutline, a tree-view control used to display hierarchical
// data such as the class browser. BPGo implements the tree as a
// flat list with parent/child indices and a cursor.
package outline

import "sync"

// Node is a single tree node.
type Node struct {
	Text     string
	Parent   int
	Children []int
	Expanded bool
	Depth    int
}

// TOutline is a tree view.
type TOutline struct {
	mu       sync.Mutex
	Nodes    []Node
	Focused  int
	expanded bool
}

// New creates a new outline.
func New() *TOutline {
	return &TOutline{Nodes: []Node{}}
}

// InsertChild appends a new child to parent.
func (o *TOutline) InsertChild(parent int, text string) int {
	o.mu.Lock()
	defer o.mu.Unlock()
	id := len(o.Nodes)
	n := Node{Text: text, Parent: parent, Depth: depth(o.Nodes, parent) + 1}
	if parent >= 0 && parent < len(o.Nodes) {
		o.Nodes[parent].Children = append(o.Nodes[parent].Children, id)
	}
	o.Nodes = append(o.Nodes, n)
	return id
}

// Root inserts a root-level item.
func (o *TOutline) Root(text string) int {
	return o.InsertChild(-1, text)
}

func depth(nodes []Node, idx int) int {
	if idx < 0 || idx >= len(nodes) {
		return 0
	}
	if nodes[idx].Parent < 0 {
		return 0
	}
	return 1 + depth(nodes, nodes[idx].Parent)
}

// Expand marks a node as expanded.
func (o *TOutline) Expand(idx int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if idx < 0 || idx >= len(o.Nodes) {
		return
	}
	o.Nodes[idx].Expanded = true
}

// Collapse marks a node as collapsed.
func (o *TOutline) Collapse(idx int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if idx < 0 || idx >= len(o.Nodes) {
		return
	}
	o.Nodes[idx].Expanded = false
}

// Focus sets the focused item.
func (o *TOutline) Focus(idx int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if idx < 0 || idx >= len(o.Nodes) {
		return
	}
	o.Focused = idx
}

// Count returns the number of items.
func (o *TOutline) Count() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.Nodes)
}

// Text returns the text of an item.
func (o *TOutline) Text(idx int) string {
	o.mu.Lock()
	defer o.mu.Unlock()
	if idx < 0 || idx >= len(o.Nodes) {
		return ""
	}
	return o.Nodes[idx].Text
}

// Visible returns the items visible in the current expand state.
func (o *TOutline) Visible() []int {
	o.mu.Lock()
	defer o.mu.Unlock()
	visible := []int{}
	for i, n := range o.Nodes {
		if o.isVisibleLocked(i, n) {
			visible = append(visible, i)
		}
	}
	return visible
}

func (o *TOutline) isVisibleLocked(i int, n Node) bool {
	if n.Parent < 0 {
		return true
	}
	parent := o.Nodes[n.Parent]
	if parent.Parent < 0 {
		// Direct child of root: always visible when the parent is in
		// the visible list (which is always for root).
		return true
	}
	if !parent.Expanded {
		return false
	}
	return o.isVisibleLocked(n.Parent, parent)
}
