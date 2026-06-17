package diagnostics

import (
	"testing"
)

func TestRegisteredCodes(t *testing.T) {
	for code := range Codes(CatCompile) {
		if code < 0 {
			t.Errorf("compile code %d must be >= 0", code)
		}
	}
	for code := range Codes(CatRuntime) {
		if code < 0 {
			t.Errorf("runtime code %d should not be negative", code)
		}
	}
}

func TestCompileAndRuntimeErrors(t *testing.T) {
	_, ok := Get(CatCompile, 5)
	if !ok {
		t.Fatal("compile error 5 not registered")
	}
	_, ok = Get(CatRuntime, 200)
	if !ok {
		t.Fatal("runtime error 200 not registered")
	}
}

func TestFormat(t *testing.T) {
	got := Format(CatCompile, 5, "hello.pas", 10, 3)
	if got != "Compile 5: Syntax error (hello.pas, line 10, col 3)" {
		t.Errorf("unexpected format: %q", got)
	}
}

func TestSearch(t *testing.T) {
	got := Search("range")
	if len(got) == 0 {
		t.Error("expected at least one match for 'range'")
	}
}

func TestGraphErrors(t *testing.T) {
	_, ok := Get(CatGraph, -2)
	if !ok {
		t.Fatal("graph error -2 not registered")
	}
}
