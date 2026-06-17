package omf

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"
)

func makeOMF() []byte {
	buf := &bytes.Buffer{}
	// THEADR
	binary.Write(buf, binary.LittleEndian, uint8(RecTheadr))
	binary.Write(buf, binary.LittleEndian, uint16(4))
	buf.WriteByte(3)
	buf.WriteString("TST")
	// LNAMES: CODE, DATA
	names := []string{"CODE", "DATA"}
	body := &bytes.Buffer{}
	for _, n := range names {
		body.WriteByte(byte(len(n)))
		body.WriteString(n)
	}
	binary.Write(buf, binary.LittleEndian, uint8(RecLnames))
	binary.Write(buf, binary.LittleEndian, uint16(body.Len()))
	buf.Write(body.Bytes())
	// SEGDEF (1 segment)
	body.Reset()
	body.WriteByte(0x70) // attr
	body.WriteByte(0x28) // align=0, combine, big
	body.WriteByte(0)    // frame
	body.WriteByte(0)    // offset
	body.WriteByte(0)
	body.WriteByte(0x10) // length
	body.WriteByte(0x00)
	body.WriteByte(4)
	body.WriteString("CODE")
	binary.Write(buf, binary.LittleEndian, uint8(RecSegdef))
	binary.Write(buf, binary.LittleEndian, uint16(body.Len()))
	buf.Write(body.Bytes())
	// PUBDEF
	body.Reset()
	binary.Write(body, binary.LittleEndian, uint16(0))
	body.WriteByte(5)
	body.WriteString("HELLO")
	binary.Write(body, binary.LittleEndian, uint16(0x100))
	binary.Write(buf, binary.LittleEndian, uint8(RecPubdef))
	binary.Write(buf, binary.LittleEndian, uint16(body.Len()))
	buf.Write(body.Bytes())
	// MODEND
	binary.Write(buf, binary.LittleEndian, uint8(RecModend))
	binary.Write(buf, binary.LittleEndian, uint16(1))
	buf.WriteByte(0)
	return buf.Bytes()
}

func TestOMFParse(t *testing.T) {
	obj, err := ReadStream(bytes.NewReader(makeOMF()))
	if err != nil {
		t.Fatal(err)
	}
	if obj.ModuleName != "TST" {
		t.Errorf("module: %q", obj.ModuleName)
	}
	if len(obj.Names) != 2 {
		t.Errorf("names: %v", obj.Names)
	}
	if len(obj.Segments) != 1 {
		t.Errorf("segments: %d", len(obj.Segments))
	}
	if len(obj.Publics) != 1 || obj.Publics[0].Name != "HELLO" {
		t.Errorf("publics: %v", obj.Publics)
	}
}

func TestOMFString(t *testing.T) {
	obj, _ := ReadStream(bytes.NewReader(makeOMF()))
	if obj.String() == "" {
		t.Error("String returned empty")
	}
}

func TestIsObject(t *testing.T) {
	tmp := t.TempDir() + "/tst.obj"
	data := makeOMF()
	if err := writeFile(tmp, data); err != nil {
		t.Fatal(err)
	}
	if !IsObject(tmp) {
		t.Error("IsObject should return true for valid OMF")
	}
}

func TestIsObjectInvalid(t *testing.T) {
	tmp := t.TempDir() + "/bad.obj"
	if err := writeFile(tmp, []byte("not omf")); err != nil {
		t.Fatal(err)
	}
	if IsObject(tmp) {
		t.Error("IsObject should return false for invalid OMF")
	}
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}
