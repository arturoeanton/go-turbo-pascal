// Package tpu implements BPGo's unit file format. The .tpu format is
// Borland's compiled-unit format; rather than reverse-engineer the
// proprietary layout, BPGo uses its own .bpu container that is
// conceptually compatible. The bpu format is a small Go-friendly
// binary blob that stores the public interface of a compiled unit
// (names, signatures, code references) and the embedded machine code
// for the dos16 backend or IR for the vm backend.
package tpu

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
)

// Magic identifies a BPU file.
var Magic = [4]byte{'B', 'P', 'U', 1}

// Header is the fixed-size header of a BPU file.
type Header struct {
	Magic      [4]byte
	UnitName   [32]byte
	Backend    uint8 // 0 = vm, 1 = dos16
	NumImports uint16
	NumExports uint16
	NumSymbols uint16
	NumRelocs  uint16
	DataOffset uint32
	DataSize   uint32
	CodeOffset uint32
	CodeSize   uint32
}

// Export is a single exported symbol.
type Export struct {
	Name [32]byte
	Kind uint8 // 0=procedure, 1=function, 2=variable, 3=type, 4=const
	Off  uint16
}

// Import is an external symbol reference.
type Import struct {
	Name [32]byte
	Unit [32]byte
	Kind uint8
}

// Symbol is a generic name in the unit (for the browser).
type Symbol struct {
	Name [32]byte
	Kind uint8
}

// Relocation is a code relocation entry.
type Relocation struct {
	Offset uint32
	SymIdx uint16
}

// File is the in-memory representation of a BPU file.
type File struct {
	Header
	Exports []Export
	Imports []Import
	Symbols []Symbol
	Relocs  []Relocation
	Code    []byte
	Data    []byte
	IR      *ir.Program
}

// Write serializes a File to w.
func (f *File) Write(w io.Writer) error {
	f.Header.Magic = Magic
	if err := binary.Write(w, binary.LittleEndian, &f.Header); err != nil {
		return err
	}
	for _, e := range f.Exports {
		if err := binary.Write(w, binary.LittleEndian, e); err != nil {
			return err
		}
	}
	for _, i := range f.Imports {
		if err := binary.Write(w, binary.LittleEndian, i); err != nil {
			return err
		}
	}
	for _, s := range f.Symbols {
		if err := binary.Write(w, binary.LittleEndian, s); err != nil {
			return err
		}
	}
	for _, r := range f.Relocs {
		if err := binary.Write(w, binary.LittleEndian, r); err != nil {
			return err
		}
	}
	if _, err := w.Write(f.Data); err != nil {
		return err
	}
	if _, err := w.Write(f.Code); err != nil {
		return err
	}
	return nil
}

// Read parses a BPU file from r.
func Read(r io.Reader) (*File, error) {
	var h Header
	if err := binary.Read(r, binary.LittleEndian, &h); err != nil {
		return nil, err
	}
	if h.Magic != Magic {
		return nil, fmt.Errorf("not a BPU file: magic=%x", h.Magic)
	}
	f := &File{Header: h}
	for i := uint16(0); i < h.NumExports; i++ {
		var e Export
		if err := binary.Read(r, binary.LittleEndian, &e); err != nil {
			return nil, err
		}
		f.Exports = append(f.Exports, e)
	}
	for i := uint16(0); i < h.NumImports; i++ {
		var im Import
		if err := binary.Read(r, binary.LittleEndian, &im); err != nil {
			return nil, err
		}
		f.Imports = append(f.Imports, im)
	}
	for i := uint16(0); i < h.NumSymbols; i++ {
		var s Symbol
		if err := binary.Read(r, binary.LittleEndian, &s); err != nil {
			return nil, err
		}
		f.Symbols = append(f.Symbols, s)
	}
	for i := uint16(0); i < h.NumRelocs; i++ {
		var rl Relocation
		if err := binary.Read(r, binary.LittleEndian, &rl); err != nil {
			return nil, err
		}
		f.Relocs = append(f.Relocs, rl)
	}
	rest, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if int(h.DataSize) <= len(rest) {
		f.Data = rest[:h.DataSize]
		f.Code = rest[h.DataSize:][:h.CodeSize]
	}
	return f, nil
}

// Bytes returns the serialized file.
func (f *File) Bytes() ([]byte, error) {
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// MakeExport creates an Export entry from a name and kind.
func MakeExport(name string, kind uint8, off uint16) Export {
	var e Export
	copy(e.Name[:], name)
	e.Kind = kind
	e.Off = off
	return e
}

// ErrInvalidUnit indicates a malformed BPU.
var ErrInvalidUnit = errors.New("invalid bpu unit")
