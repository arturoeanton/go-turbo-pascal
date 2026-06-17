// Package printer implements the Printer unit runtime. In TP7 the
// Printer unit provides a Text file named Lst that writes to a printer
// device. BPGo maps the printer to a configurable output file or an
// in-memory buffer for tests.
package printer

import (
	"io"
	"os"
	"sync"
)

// Lst is the global Text file used by Write(Lst, ...) and WriteLn.
var Lst *File

// File is a simple text file with a writer.
type File struct {
	mu   sync.Mutex
	w    io.Writer
	name string
	open bool
}

// Open opens the printer file for writing. If path is empty, the
// in-memory buffer is used.
func Open(path string) error {
	Lst = &File{open: true}
	if path == "" {
		Lst.w = &memBuf{}
	} else {
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		Lst.w = f
		Lst.name = path
	}
	return nil
}

func Close() error {
	if Lst == nil {
		return nil
	}
	Lst.mu.Lock()
	defer Lst.mu.Unlock()
	Lst.open = false
	if c, ok := Lst.w.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func WriteString(s string) error {
	if Lst == nil {
		return nil
	}
	Lst.mu.Lock()
	defer Lst.mu.Unlock()
	_, err := io.WriteString(Lst.w, s)
	return err
}

func WriteLn(s string) error {
	if err := WriteString(s); err != nil {
		return err
	}
	return WriteString("\r\n")
}

// Output returns the in-memory buffer as a string, if the printer is
// configured with one.
func Output() string {
	if Lst == nil {
		return ""
	}
	if b, ok := Lst.w.(*memBuf); ok {
		return b.String()
	}
	return ""
}

type memBuf struct {
	data []byte
}

func (m *memBuf) Write(p []byte) (int, error) {
	m.data = append(m.data, p...)
	return len(p), nil
}

func (m *memBuf) String() string {
	return string(m.data)
}
