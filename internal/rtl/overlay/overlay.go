// Package overlay implements the Overlay unit runtime. Overlays let
// large TP7 programs load code on demand from an .OVR file. BPGo
// implements the overlay manager for both the VM backend (where
// overlays are simply IR segments) and the dos16 backend (where the
// .OVR file is a sidecar of the .EXE with relocation entries and a
// load counter).
package overlay

import (
	"fmt"
	"sync"
)

// Manager keeps the runtime state of the overlay system.
type Manager struct {
	mu          sync.Mutex
	loaded      map[string]int
	traps       int
	bufSize     int
	readBuf     []byte
	loadedCount int
	Result      int
	FileMode    byte
}

// New creates a default Manager.
func New() *Manager {
	return &Manager{
		loaded:  map[string]int{},
		readBuf: make([]byte, 0, 4096),
	}
}

// OvrInit initialises the manager. The OVR file is opened by the
// loader; here we just record the path and return success.
func (m *Manager) OvrInit(ovrFile string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ovrFile == "" {
		m.Result = -2
		return m.Result
	}
	m.Result = 0
	return 0
}

// OvrInitEMS is the EMS-backed variant.
func (m *Manager) OvrInitEMS(ovrFile string, handle uint16) int {
	return m.OvrInit(ovrFile)
}

// OvrSetBuf configures the read buffer size in bytes.
func (m *Manager) OvrSetBuf(size int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bufSize = size
	if cap(m.readBuf) < size {
		m.readBuf = make([]byte, 0, size)
	}
}

// OvrGetBuf returns the current buffer size.
func (m *Manager) OvrGetBuf() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.bufSize
}

// OvrClearBuf empties the read buffer.
func (m *Manager) OvrClearBuf() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readBuf = m.readBuf[:0]
}

// OvrSetRetry configures the retry buffer size.
func (m *Manager) OvrSetRetry(size int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cap(m.readBuf) < size {
		m.readBuf = make([]byte, 0, size)
	}
}

// OvrGetRetry returns the retry size.
func (m *Manager) OvrGetRetry() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return cap(m.readBuf)
}

// OvrLoadCount returns the number of overlay loads.
func (m *Manager) OvrLoadCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.loadedCount
}

// OvrTrapCount returns the number of trap calls.
func (m *Manager) OvrTrapCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.traps
}

// OvrResult returns the last result code.
func (m *Manager) OvrResult() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Result
}

// OvrFileMode returns the file mode (read/write).
func (m *Manager) OvrFileMode() byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.FileMode
}

// Load simulates a load of an overlay named `name`. The dos16 backend
// performs an actual disk read; the VM backend simply increments the
// counter.
func (m *Manager) Load(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loaded[name] = m.loaded[name] + 1
	m.loadedCount++
	if _, ok := m.loaded[name]; !ok {
		return fmt.Errorf("overlay not found: %s", name)
	}
	return nil
}

// Trap simulates a trap call.
func (m *Manager) Trap() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.traps++
}

// Stats returns a snapshot of the manager state.
type Stats struct {
	Loaded     int
	Traps      int
	BufferSize int
	LoadCount  int
}

func (m *Manager) Stats() Stats {
	m.mu.Lock()
	defer m.mu.Unlock()
	return Stats{
		Loaded:     len(m.loaded),
		Traps:      m.traps,
		BufferSize: m.bufSize,
		LoadCount:  m.loadedCount,
	}
}
