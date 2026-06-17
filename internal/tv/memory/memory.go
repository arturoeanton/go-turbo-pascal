// Package memory implements the Turbo Vision Memory unit. The unit
// provides a memory manager wrapper around the System unit heap. It
// also installs a default out-of-memory handler. BPGo's Memory unit
// delegates to the System unit heap and provides a small API for
// long-lived object pools used by TV applications.
package memory

import (
	"sync"
)

// Manager wraps the TV memory manager.
type Manager struct {
	mu       sync.Mutex
	inUse    int64
	limit    int64
	LowMem   bool
	OnLowMem func()
}

// New creates a default memory manager.
func New() *Manager {
	return &Manager{limit: 16 * 1024 * 1024}
}

// InitMemory initializes the manager.
func (m *Manager) InitMemory() { m.inUse = 0 }

// DoneMemory releases all blocks.
func (m *Manager) DoneMemory() { m.inUse = 0 }

// MemoryError is the default OOM handler. It can be replaced by
// assigning a different function to the global OnLowMem.
func (m *Manager) MemoryError() {
	m.LowMem = true
	if m.OnLowMem != nil {
		m.OnLowMem()
	}
}

// Reserve marks a block of `n` bytes as in use.
func (m *Manager) Reserve(n int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.inUse+n > m.limit {
		m.MemoryError()
		return false
	}
	m.inUse += n
	return true
}

// Release returns `n` bytes to the free pool.
func (m *Manager) Release(n int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inUse -= n
	if m.inUse < 0 {
		m.inUse = 0
	}
}

// Limit sets the maximum number of bytes that can be reserved.
func (m *Manager) Limit(n int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.limit = n
}

// InUse returns the current allocation.
func (m *Manager) InUse() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.inUse
}
