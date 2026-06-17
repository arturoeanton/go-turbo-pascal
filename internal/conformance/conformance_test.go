package conformance

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCorpus(t *testing.T) {
	r := New(t.TempDir(), &bytes.Buffer{})
	r.Run()
	if r.Report == nil {
		t.Fatal("Report is nil")
	}
	if r.Report.Total < 1 {
		t.Errorf("Total: %d", r.Report.Total)
	}
}

func TestWriteReport(t *testing.T) {
	tmp := t.TempDir()
	r := New(tmp, &bytes.Buffer{})
	r.Run()
	reportPath := filepath.Join(tmp, "compat", "report.json")
	if _, err := os.Stat(reportPath); err != nil {
		t.Fatalf("report not written: %v", err)
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var r2 Report
	if err := json.Unmarshal(data, &r2); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if r2.GeneratedAt == "" {
		t.Error("GeneratedAt missing")
	}
}

func TestLexAndParse(t *testing.T) {
	if err := LexAndParse("test.pas", "program T; begin end."); err != nil {
		t.Errorf("LexAndParse: %v", err)
	}
}

func TestLexAndParseInvalid(t *testing.T) {
	if err := LexAndParse("test.pas", "program T; var : Integer; begin end."); err == nil {
		t.Error("expected error")
	}
}

func TestLexParseSem(t *testing.T) {
	if err := LexParseSem("test.pas", "program T; var X: Integer; begin X := 1; end."); err != nil {
		t.Errorf("LexParseSem: %v", err)
	}
}

func TestLexParseSemUnknown(t *testing.T) {
	err := LexParseSem("test.pas", "program T; begin Foo := 1; end.")
	if err == nil {
		t.Error("expected error")
	}
}

func TestDirectives(t *testing.T) {
	tmp := t.TempDir()
	r := New(tmp, &bytes.Buffer{})
	r.Run()
	if len(r.Report.Directives) == 0 {
		t.Error("Directives empty")
	}
}

func TestDiagnostics(t *testing.T) {
	tmp := t.TempDir()
	r := New(tmp, &bytes.Buffer{})
	r.Run()
	if len(r.Report.Diagnostics) == 0 {
		t.Error("Diagnostics empty")
	}
}

func TestUnits(t *testing.T) {
	tmp := t.TempDir()
	r := New(tmp, &bytes.Buffer{})
	r.Run()
	if len(r.Report.Units) == 0 {
		t.Error("Units empty")
	}
}

func TestTPUFileSize(t *testing.T) {
	if TPUFileSize(nil) != 0 {
		t.Error("nil file size")
	}
}
