package mz

import (
	"bytes"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	img := New()
	img.AddSegment([]byte("hello world!"))
	img.AddRelocation(0, 2)
	img.Entry = 0
	img.EntrySegment = 0
	data, err := img.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if data[0] != 'M' || data[1] != 'Z' {
		t.Errorf("expected MZ signature, got %x %x", data[0], data[1])
	}
	got, err := Parse(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if got.Header.Signature != 0x5A4D {
		t.Error("signature lost")
	}
	if len(got.Relocations) != 1 {
		t.Errorf("expected 1 relocation, got %d", len(got.Relocations))
	}
}

func TestEmptyImage(t *testing.T) {
	img := New()
	data, err := img.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 28 {
		t.Error("MZ header too short")
	}
}

func TestAddSegmentPadding(t *testing.T) {
	img := New()
	img.AddSegment([]byte("short")) // 5 bytes
	if len(img.Segments[0])%16 != 0 {
		t.Errorf("segment not paragraph-aligned: %d", len(img.Segments[0]))
	}
}

func TestParseNotMZ(t *testing.T) {
	_, err := Parse(bytes.NewReader([]byte("not an exe")))
	if err == nil {
		t.Error("expected error for non-MZ file")
	}
}

func TestMultipleRelocations(t *testing.T) {
	img := New()
	img.AddSegment([]byte("a b c d e f g h i j k l m n o p"))
	img.AddRelocation(0, 0)
	img.AddRelocation(0, 2)
	img.AddRelocation(0, 4)
	data, err := img.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Relocations) != 3 {
		t.Errorf("expected 3 relocations, got %d", len(got.Relocations))
	}
}
