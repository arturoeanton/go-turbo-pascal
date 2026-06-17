package tpu

import (
	"bytes"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	f := &File{}
	f.UnitName = strToName("TestUnit")
	f.Backend = 0
	f.NumExports = 1
	f.NumImports = 0
	f.NumSymbols = 0
	f.NumRelocs = 0
	f.DataSize = 0
	f.CodeSize = 0
	f.Exports = []Export{MakeExport("Hello", 0, 0)}
	data, err := f.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if data[0] != 'B' || data[1] != 'P' {
		t.Errorf("expected BPU signature, got %x %x", data[0], data[1])
	}
	got, err := Read(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if got.Header.UnitName != strToName("TestUnit") {
		t.Error("unit name lost")
	}
	if len(got.Exports) != 1 {
		t.Errorf("expected 1 export, got %d", len(got.Exports))
	}
}

func TestInvalidMagic(t *testing.T) {
	_, err := Read(bytes.NewReader([]byte("NOPE")))
	if err == nil {
		t.Error("expected error for invalid magic")
	}
}

func TestEmptyUnit(t *testing.T) {
	f := &File{}
	f.UnitName = strToName("Empty")
	data, err := f.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	got, err := Read(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if got.Header.UnitName != strToName("Empty") {
		t.Error("unit name lost")
	}
}

func strToName(s string) [32]byte {
	var n [32]byte
	copy(n[:], s)
	return n
}
