// Package mz implements the DOS MZ executable format writer. The MZ
// (Mark Zbikowski) format is the canonical DOS EXE format used by
// TP7/BP7 programs. The writer produces a binary image with proper
// PSP, header, segment layout, relocations, stack and entry point.
// The format is described in the RBIL / doswiki and matches the
// Borland TLINK output.
package mz

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Header is the 28-byte MZ header at the start of every DOS EXE.
type Header struct {
	Signature       uint16 // 'M' 'Z'
	BytesOnLastPage uint16
	Pages           uint16
	Relocations     uint16
	HeaderSize      uint16 // in 16-byte paragraphs
	MinAlloc        uint16
	MaxAlloc        uint16
	InitialSS       uint16
	InitialSP       uint16
	Checksum        uint16
	InitialIP       uint16
	InitialCS       uint16
	RelocationTable uint16
	OverlayNumber   uint16
}

// Image is a segment-aligned executable image.
type Image struct {
	Header       Header
	Segments     [][]byte
	Relocations  []Relocation
	StackSize    uint16
	Entry        uint16
	EntrySegment uint16
}

// Relocation is a single MZ relocation entry. Each entry patches a
// 16-bit word at the given segment:offset by adding the base segment
// of the loaded image.
type Relocation struct {
	Segment uint16
	Offset  uint16
}

// New creates a new empty image.
func New() *Image {
	img := &Image{
		Header: Header{
			Signature:  0x5A4D, // 'M' 'Z'
			HeaderSize: 32,     // 0x200 bytes / 16
			MinAlloc:   0x1000,
			MaxAlloc:   0xFFFF,
			InitialSS:  0,
			InitialSP:  0xFFFE,
			InitialIP:  0,
			InitialCS:  0,
		},
	}
	return img
}

// AddSegment appends a 16-byte-aligned segment.
func (i *Image) AddSegment(seg []byte) {
	if len(seg)%16 != 0 {
		// Pad with zeros to paragraph boundary.
		pad := 16 - (len(seg) % 16)
		seg = append(seg, make([]byte, pad)...)
	}
	i.Segments = append(i.Segments, seg)
}

// AddRelocation adds a relocation. Segment and offset are within the
// image; the loader adds the base segment value.
func (i *Image) AddRelocation(seg, off uint16) {
	i.Relocations = append(i.Relocations, Relocation{Segment: seg, Offset: off})
}

// Write serializes the image to w.
func (i *Image) Write(w io.Writer) error {
	// Update header counts.
	totalSize := 0
	for _, s := range i.Segments {
		totalSize += len(s)
	}
	headerBytes := (int(i.Header.HeaderSize) * 16)
	// Header is followed by relocations.
	relocs := i.Relocations
	i.Header.Relocations = uint16(len(relocs))
	i.Header.RelocationTable = uint16(headerBytes / 16)
	// Compute pages: each page is 512 bytes.
	bodySize := headerBytes + len(relocs)*4 + totalSize
	pages := bodySize / 512
	last := bodySize % 512
	if last > 0 {
		last = 512 - last
		pages++
	}
	i.Header.Pages = uint16(pages)
	i.Header.BytesOnLastPage = uint16(last)

	// Write header.
	if err := binary.Write(w, binary.LittleEndian, &i.Header); err != nil {
		return err
	}
	// Pad header to HeaderSize * 16.
	if _, err := w.Write(make([]byte, headerBytes-28)); err != nil {
		return err
	}
	// Relocations.
	for _, r := range relocs {
		if err := binary.Write(w, binary.LittleEndian, r); err != nil {
			return err
		}
	}
	// Segments.
	for _, s := range i.Segments {
		if _, err := w.Write(s); err != nil {
			return err
		}
	}
	return nil
}

// Bytes returns the serialized image as a byte slice.
func (i *Image) Bytes() ([]byte, error) {
	buf := make([]byte, 0, 4096)
	if err := i.Write(&byteWriter{buf: &buf}); err != nil {
		return nil, err
	}
	return buf, nil
}

type byteWriter struct {
	buf *[]byte
}

func (w *byteWriter) Write(p []byte) (int, error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}

// Parse reads an MZ image and returns an Image.
func Parse(r io.Reader) (*Image, error) {
	var hdr Header
	if err := binary.Read(r, binary.LittleEndian, &hdr); err != nil {
		return nil, err
	}
	if hdr.Signature != 0x5A4D {
		return nil, fmt.Errorf("not an MZ file: signature=%x", hdr.Signature)
	}
	// Skip remaining header bytes.
	headerRest := make([]byte, int(hdr.HeaderSize)*16-28)
	if _, err := io.ReadFull(r, headerRest); err != nil {
		return nil, err
	}
	// Read relocations.
	relocs := make([]Relocation, hdr.Relocations)
	for j := range relocs {
		if err := binary.Read(r, binary.LittleEndian, &relocs[j]); err != nil {
			return nil, err
		}
	}
	img := &Image{Header: hdr, Relocations: relocs}
	// Read remaining bytes as a single payload (segmented at runtime by
	// loader; we keep it as-is).
	remaining, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	img.Segments = [][]byte{remaining}
	return img, nil
}
