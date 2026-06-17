// Package histlist implements the Turbo Vision HistList unit. The
// unit provides THistory (the popup button) and THistList (a bounded
// FIFO of strings). BPGo implements them as Go data structures.
package histlist

import (
	"sync"
)

// THistory is the popup history button. It is a thin wrapper over a
// TView pointer; the actual rendering is done by the IDE.
type THistory struct {
	mu        sync.Mutex
	Items     []string
	MaxItems  int
	HistoryID uint16
}

// Init creates a history with the given max length.
func (h *THistory) Init(id uint16, max int) *THistory {
	h.HistoryID = id
	h.MaxItems = max
	h.Items = []string{}
	return h
}

// Add appends an item, removing duplicates and enforcing MaxItems.
func (h *THistory) Add(s string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, x := range h.Items {
		if x == s {
			h.Items = append(h.Items[:i], h.Items[i+1:]...)
			break
		}
	}
	h.Items = append([]string{s}, h.Items...)
	if h.MaxItems > 0 && len(h.Items) > h.MaxItems {
		h.Items = h.Items[:h.MaxItems]
	}
}

// Count returns the number of items.
func (h *THistory) Count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.Items)
}

// At returns the i-th item.
func (h *THistory) At(i int) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if i < 0 || i >= len(h.Items) {
		return ""
	}
	return h.Items[i]
}

// THistList is a TStringCollection wrapper used for the IDE's input
// history.
type THistList struct {
	THistory
}

// Init creates a hist list.
func (l *THistList) Init(id uint16, max int) *THistList {
	l.THistory.Init(id, max)
	return l
}

// Recall finds the first item containing `sub`. Returns -1 if not
// found.
func (l *THistList) Recall(sub string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, s := range l.Items {
		if indexOf(s, sub) >= 0 {
			return i
		}
	}
	return -1
}

func indexOf(s, sub string) int {
	if sub == "" {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
