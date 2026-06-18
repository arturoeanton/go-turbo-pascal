package vmpas

import (
	"reflect"
	"testing"
)

// Analyze infers the capabilities a script needs from its bytecode, even on a
// restricted engine that could not actually run it.
func TestAnalyzeInfersCapabilities(t *testing.T) {
	e := New() // fully restricted
	rep, err := e.Analyze(`program P;
var body, home: string;
begin
  home := GetEnv('HOME');
  body := HttpGet('https://api.example.com/x');
  WriteLn(home, body);
end.`)
	if err != nil {
		t.Fatal(err)
	}
	want := []Capability{CapEnv, CapNetwork}
	if !reflect.DeepEqual(rep.Required, want) {
		t.Fatalf("Required = %v, want %v", rep.Required, want)
	}
	if !rep.Needs(CapNetwork) || !rep.Needs(CapEnv) {
		t.Fatal("Needs() disagrees with Required")
	}
	if rep.Needs(CapFileSystem) {
		t.Fatal("false positive: FileSystem not used")
	}
	if got := rep.Calls[CapNetwork]; len(got) != 1 || got[0] != "httpget" {
		t.Fatalf("network calls = %v, want [httpget]", got)
	}
}

// A pure-computation script needs no capabilities (least privilege: grant
// nothing).
func TestAnalyzePureScriptNeedsNothing(t *testing.T) {
	e := New()
	rep, err := e.Analyze(`program P;
var i, s: Integer;
begin
  s := 0;
  for i := 1 to 10 do s := s + i;
  WriteLn(s);
end.`)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Required) != 0 {
		t.Fatalf("expected no capabilities, got %v", rep.Required)
	}
}

// Analyze detects file and database access too.
func TestAnalyzeFileAndDb(t *testing.T) {
	e := New()
	rep, err := e.Analyze(`program P;
var f: Text;
begin
  Assign(f, 'x.txt');
  Reset(f);
  Close(f);
end.`)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Needs(CapFileSystem) {
		t.Fatalf("expected FileSystem, got %v", rep.Required)
	}
}
