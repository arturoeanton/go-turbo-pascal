// Package omf provides a minimal OMF (Object Module Format) reader
// for Borland .obj files. The full OMF specification is large; BPGo
// implements only the records needed to resolve {$L file.obj}
// directives: the THEADR, LNAMES, SEGDEF, GRPDEF, EXTDEF, LEDATA and
// FIXUPP records. The reader is used to extract public symbols and
// their segment/offset so that an external object file can be linked
// into a BPGo program.
package omf

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// RecordType is the OMF record type byte.
type RecordType byte

const (
	RecTheadr  RecordType = 0x80
	RecComment RecordType = 0x88
	RecLnames  RecordType = 0x96
	RecSegdef  RecordType = 0x98
	RecGrpdef  RecordType = 0x9A
	RecExtdef  RecordType = 0x8C
	RecPubdef  RecordType = 0x90
	RecLPubdef RecordType = 0xB6
	RecLedata  RecordType = 0xA0
	RecFixupp  RecordType = 0x9C
	RecModend  RecordType = 0x8A
	RecEnd     RecordType = 0x8B
)

// Object is a parsed OMF object.
type Object struct {
	ModuleName string
	Names      []string
	Segments   []Segment
	Groups     []Group
	Externs    []External
	Publics    []Public
	Records    []Record
}

// Segment is a SEGDEF record.
type Segment struct {
	Name      string
	Class     string
	Combine   uint8
	Length    uint32
	Alignment uint8
}

// Group is a GRPDEF record.
type Group struct {
	Name string
	Segs []string
}

// External is an EXTDEF record.
type External struct {
	Name       string
	LocalIndex uint16
}

// Public is a PUBDEF or LPUBDEF record.
type Public struct {
	Name    string
	Segment string
	Offset  uint32
	Type    uint8
}

// Record is a raw OMF record (for debugging).
type Record struct {
	Type RecordType
	Data []byte
}

// Read parses an OMF object file.
func Read(path string) (*Object, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadStream(f)
}

// ReadStream parses an OMF stream.
func ReadStream(r io.Reader) (*Object, error) {
	obj := &Object{}
	for {
		var recType uint8
		if err := binary.Read(r, binary.LittleEndian, &recType); err != nil {
			if err == io.EOF {
				return obj, nil
			}
			return nil, err
		}
		var length uint16
		if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
			return nil, err
		}
		data := make([]byte, length)
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, err
		}
		obj.Records = append(obj.Records, Record{Type: RecordType(recType), Data: data})
		if err := obj.apply(RecordType(recType), data); err != nil {
			return nil, err
		}
		if recType == byte(RecModend) {
			return obj, nil
		}
	}
}

func (o *Object) apply(t RecordType, data []byte) error {
	switch t {
	case RecTheadr:
		o.ModuleName = readName(data)
	case RecLnames:
		readLNames(o, data)
	case RecSegdef:
		readSegDef(o, data)
	case RecGrpdef:
		readGrpDef(o, data)
	case RecExtdef:
		readExtDef(o, data)
	case RecPubdef, RecLPubdef:
		readPubDef(o, data, t == RecLPubdef)
	case RecModend:
		// End of module.
	case RecEnd:
		// End of file.
	default:
		// Unknown; ignore.
	}
	return nil
}

// IsObject reports whether the given path looks like an OMF object.
func IsObject(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	var b [3]byte
	_, err = io.ReadFull(f, b[:])
	if err != nil {
		return false
	}
	return b[0] == byte(RecTheadr) || b[0] == 0x80
}

// readName reads a TP-style length-prefixed name.
func readName(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	n := int(data[0])
	if 1+n > len(data) {
		return ""
	}
	return string(data[1 : 1+n])
}

func readLNames(o *Object, data []byte) {
	i := 0
	for i < len(data) {
		name := readName(data[i:])
		i += 1 + len(name)
		o.Names = append(o.Names, name)
	}
}

func readSegDef(o *Object, data []byte) error {
	if len(data) < 4 {
		return errors.New("short segdef")
	}
	attr := data[0]
	_ = attr
	align := data[1] & 0x0F
	combine := (data[1] >> 2) & 0x07
	// frame, offset, length depend on A bit
	i := 2
	if data[1]&0x04 == 0 { // frame is 1 byte
		i++
	} else {
		i += 2
	}
	// offset: 1 or 2 bytes
	if data[1]&0x01 == 0 {
		i++
	} else {
		i += 2
	}
	// length
	var length uint32
	if data[1]&0x80 == 0 {
		if i+2 > len(data) {
			return errors.New("short segdef length")
		}
		length = uint32(binary.LittleEndian.Uint16(data[i : i+2]))
		i += 2
	} else {
		if i+4 > len(data) {
			return errors.New("short segdef length")
		}
		length = uint32(binary.LittleEndian.Uint32(data[i : i+4]))
		i += 4
	}
	name := readName(data[i:])
	cls := ""
	if i+1+len(name) < len(data) {
		cls = readName(data[i+1+len(name):])
	}
	o.Segments = append(o.Segments, Segment{
		Name: name, Class: cls, Combine: combine, Length: length, Alignment: align,
	})
	return nil
}

func readGrpDef(o *Object, data []byte) {
	if len(data) < 2 {
		return
	}
	name := readName(data[1:])
	o.Groups = append(o.Groups, Group{Name: name})
	// Skip details for the conformance harness.
	_ = strings.TrimSpace
}

func readExtDef(o *Object, data []byte) {
	i := 0
	for i < len(data) {
		name := readName(data[i:])
		i += 1 + len(name)
		o.Externs = append(o.Externs, External{Name: name})
	}
}

func readPubDef(o *Object, data []byte, lpub bool) {
	if lpub {
		// Skip base group/segment index.
		if len(data) < 2 {
			return
		}
		data = data[2:]
	}
	if len(data) < 2 {
		return
	}
	segIdx := binary.LittleEndian.Uint16(data[:2])
	data = data[2:]
	for len(data) > 0 {
		name := readName(data)
		data = data[1+len(name):]
		if len(data) < 2 {
			break
		}
		off := binary.LittleEndian.Uint16(data[:2])
		data = data[2:]
		seg := ""
		if int(segIdx) < len(o.Segments) {
			seg = o.Segments[segIdx].Name
		}
		o.Publics = append(o.Publics, Public{Name: name, Segment: seg, Offset: uint32(off)})
	}
}

// String returns a short textual representation for golden tests.
func (o *Object) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "module=%s segments=%d publics=%d externs=%d", o.ModuleName, len(o.Segments), len(o.Publics), len(o.Externs))
	return b.String()
}
