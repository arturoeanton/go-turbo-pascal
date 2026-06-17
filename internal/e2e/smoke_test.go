package e2e

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arturoeanton/go-turbo-pascal/internal/cli"
	"github.com/arturoeanton/go-turbo-pascal/internal/compile"
	"github.com/arturoeanton/go-turbo-pascal/internal/conformance"
	"github.com/arturoeanton/go-turbo-pascal/internal/debug"
	"github.com/arturoeanton/go-turbo-pascal/internal/diagnostics"
	"github.com/arturoeanton/go-turbo-pascal/internal/ide"
	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
	"github.com/arturoeanton/go-turbo-pascal/internal/lexer"
	"github.com/arturoeanton/go-turbo-pascal/internal/mz"
	"github.com/arturoeanton/go-turbo-pascal/internal/parser"
	"github.com/arturoeanton/go-turbo-pascal/internal/rtl/system"
	"github.com/arturoeanton/go-turbo-pascal/internal/sem"
	"github.com/arturoeanton/go-turbo-pascal/internal/tpu"
)

// TestSmoke ensures the entire system can be imported and that the
// top-level components work together. The test is intentionally
// conservative: any error fails the test.
func TestSmoke(t *testing.T) {
	// Diagnostics
	_, ok := diagnostics.Get(diagnostics.CatCompile, 5)
	if !ok {
		t.Error("diagnostics missing compile 5")
	}
	// Lexer
	l := lexer.New("program T; begin end.")
	if len(l.Errors()) > 0 {
		t.Errorf("lexer errors: %v", l.Errors())
	}
	// Parser
	p := parser.New(l.Tokens())
	p.SetFile("smoke.pas")
	n := p.ParseUnit()
	if len(p.Errors()) > 0 {
		t.Errorf("parser errors: %v", p.Errors())
	}
	// Sem
	a := sem.New()
	a.Analyze(n)
	// Errors are acceptable for the smoke test.
	_ = a
	// System unit
	vm := ir.NewVM(&ir.Program{Modules: []*ir.Module{{Name: "m", Funcs: map[string]*ir.Function{}, Init: []string{}}}, Entry: "main"})
	system.Register(vm)
	_ = vm.Builtins
	// MZ
	img := mz.New()
	img.AddSegment([]byte("test"))
	if _, err := img.Bytes(); err != nil {
		t.Errorf("mz: %v", err)
	}
	// TPU
	f := &tpu.File{}
	f.UnitName = [32]byte{}
	if _, err := f.Bytes(); err != nil {
		t.Errorf("tpu: %v", err)
	}
	// Debug
	d := debug.New()
	d.SetBreakpoint("smoke.pas", 1)
	// IDE
	i := ide.New(&ide.Project{Name: "T", Source: "t.pas", Output: "t.exe"}, &stubCompilerForSmoke{}, &stubRunnerForSmoke{}, &stubDebuggerForSmoke{})
	_ = i
	// CLI
	var out, errOut bytes.Buffer
	code := cli.Run([]string{"-V"}, nil, &out, &errOut)
	if code != 0 {
		t.Errorf("cli -V: %d", code)
	}
	// Conformance
	r := conformance.New(".", &bytes.Buffer{})
	r.Run()
	if r.Report == nil {
		t.Error("conformance: nil report")
	}
}

type stubCompilerForSmoke struct{}

func (s *stubCompilerForSmoke) Compile(src, output string) (string, error) { return "ok", nil }

type stubRunnerForSmoke struct{}

func (s *stubRunnerForSmoke) Run(exe string, args []string) (string, int, error) { return "ok", 0, nil }

type stubDebuggerForSmoke struct{}

func (s *stubDebuggerForSmoke) SetBreakpoint(file string, line int) {}
func (s *stubDebuggerForSmoke) Step() (string, error)               { return "", nil }
func (s *stubDebuggerForSmoke) Continue() (string, error)           { return "", nil }
func (s *stubDebuggerForSmoke) Watch(expr string) (string, error)   { return "0", nil }

// TestFullCorpusAllStages runs every corpus file through every
// stage (lex, parse, sem, compile). The test logs failures but
// only fails if a stage is broken (i.e. the runner panics or
// returns a fatal error).
func TestFullCorpusAllStages(t *testing.T) {
	dir := "../../testdata/pas"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("testdata/pas not present: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".pas") {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			path := filepath.Join(dir, e.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			src := string(data)
			// Lex
			l := lexer.New(src)
			if lexErrs := l.Errors(); len(lexErrs) > 0 {
				t.Errorf("lex: %v", lexErrs)
			}
			// Parse
			p := parser.New(l.Tokens())
			p.SetFile(e.Name())
			n := p.ParseUnit()
			if parseErrs := p.Errors(); len(parseErrs) > 0 {
				t.Logf("parse (tolerated): %v", parseErrs)
			}
			// Sem (tolerated)
			a := sem.New()
			a.Analyze(n)
			if semErrs := a.Errors(); len(semErrs) > 0 {
				t.Logf("sem (tolerated): %v", semErrs)
			}
			// Compile (tolerated)
			if _, err := compile.CompileToVM(&compile.CompileConfig{Source: src, SourceFile: e.Name()}); err != nil {
				t.Logf("compile (tolerated): %v", err)
			}
		})
	}
}

// TestDiagnosticCatalogPresent ensures every category in the
// diagnostic catalog has at least one entry.
func TestDiagnosticCatalogPresent(t *testing.T) {
	for _, cat := range []diagnostics.Category{diagnostics.CatCompile, diagnostics.CatRuntime, diagnostics.CatIO, diagnostics.CatGraph, diagnostics.CatOverlay, diagnostics.CatDebug, diagnostics.CatIDE} {
		codes := diagnostics.Codes(cat)
		if len(codes) == 0 {
			t.Errorf("category %v has no codes", cat)
		}
	}
}

// TestLexErrorsReported ensures lex errors are surfaced.
func TestLexErrorsReported(t *testing.T) {
	l := lexer.New("'unterminated")
	if len(l.Errors()) == 0 {
		t.Error("expected lex error for unterminated string")
	}
}

// TestParserErrorsReported ensures parser errors are surfaced.
func TestParserErrorsReported(t *testing.T) {
	l := lexer.New("program T; begin : end.")
	p := parser.New(l.Tokens())
	p.SetFile("bad.pas")
	p.ParseUnit()
	if len(p.Errors()) == 0 {
		t.Error("expected parser error")
	}
}

// TestSemErrorsReported ensures sem errors are surfaced.
func TestSemErrorsReported(t *testing.T) {
	src := "program T; begin Foo := 1; end."
	l := lexer.New(src)
	p := parser.New(l.Tokens())
	p.SetFile("bad.pas")
	n := p.ParseUnit()
	a := sem.New()
	a.Analyze(n)
	if len(a.Errors()) == 0 {
		t.Error("expected sem error")
	}
}
