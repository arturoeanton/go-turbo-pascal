package codegen

import (
	"errors"
	"testing"
)

func TestDiagnosticHasPosition(t *testing.T) {
	src := "program P;\n" +
		"var i: Integer;\n" +
		"begin\n" +
		"  i := nosuchvar;\n" + // line 4
		"end.\n"
	_, err := Compile(src, "t.pas")
	if err == nil {
		t.Fatal("expected a compile error")
	}
	var ce *CompileError
	if !errors.As(err, &ce) {
		t.Fatalf("expected *CompileError, got %T", err)
	}
	if len(ce.Diags) == 0 {
		t.Fatal("expected at least one diagnostic")
	}
	d := ce.Diags[0]
	if d.Line != 4 {
		t.Fatalf("diagnostic line = %d, want 4 (%+v)", d.Line, d)
	}
}

func TestMultipleDiagnostics(t *testing.T) {
	src := "program P;\n" +
		"begin\n" +
		"  a := 1;\n" + // unknown a (line 3)
		"  b := 2;\n" + // unknown b (line 4)
		"end.\n"
	_, err := Compile(src, "t.pas")
	var ce *CompileError
	if !errors.As(err, &ce) {
		t.Fatalf("expected *CompileError, got %v", err)
	}
	if len(ce.Diags) < 2 {
		t.Fatalf("expected >= 2 diagnostics, got %d: %+v", len(ce.Diags), ce.Diags)
	}
}
