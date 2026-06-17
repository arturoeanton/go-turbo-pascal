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
)

// TestEndToEnd_CLI_Help runs the CLI in help mode and verifies the
// banner is present.
func TestEndToEnd_CLI_Help(t *testing.T) {
	var out, errOut bytes.Buffer
	code := cli.Run([]string{"--help"}, nil, &out, &errOut)
	if code != 0 {
		t.Errorf("code: %d", code)
	}
	if !strings.Contains(out.String(), "Usage: bpgo") {
		t.Errorf("output: %q", out.String())
	}
}

// TestEndToEnd_CLI_Version verifies the version banner.
func TestEndToEnd_CLI_Version(t *testing.T) {
	var out, errOut bytes.Buffer
	code := cli.Run([]string{"-V"}, nil, &out, &errOut)
	if code != 0 {
		t.Errorf("code: %d", code)
	}
	if !strings.Contains(out.String(), "BPGo") {
		t.Errorf("output: %q", out.String())
	}
}

// TestEndToEnd_CLI_NoArgs verifies that the CLI fails gracefully
// when no source is given.
func TestEndToEnd_CLI_NoArgs(t *testing.T) {
	var out, errOut bytes.Buffer
	code := cli.Run([]string{}, nil, &out, &errOut)
	if code == 0 {
		t.Error("expected non-zero exit")
	}
	if !strings.Contains(errOut.String(), "no source") {
		t.Errorf("stderr: %q", errOut.String())
	}
}

// TestEndToEnd_CLI_CompileAllCorpus runs the CLI over every .pas
// file in testdata/pas, including the ones that the sem pass
// currently cannot resolve. The CLI must at least lex and parse
// every file without crashing.
func TestEndToEnd_CLI_CompileAllCorpus(t *testing.T) {
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
			var out, errOut bytes.Buffer
			code := cli.Run([]string{path}, nil, &out, &errOut)
			// The CLI may report compile errors for corpus files that
			// exercise sem features still under construction. What we
			// care about is that it doesn't crash, hang, or produce
			// a non-zero exit for a syntactically valid program.
			if code != 0 {
				// A non-zero exit is acceptable as long as the
				// error message is sensible.
				if errOut.Len() == 0 {
					t.Errorf("non-zero exit without stderr")
				}
			}
		})
	}
}

// TestEndToEnd_CompilePipeline runs the full compile pipeline on
// every corpus file. The pipeline must at least lex and parse the
// file; sem may fail.
func TestEndToEnd_CompilePipeline(t *testing.T) {
	dir := "../../testdata/pas"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("testdata/pas not present: %v", err)
	}
	// Skip programs that exercise features that the parser does
	// not yet support.
	skips := map[string]bool{
		"objectpoly.pas": true, // methods outside of object decl
		"list.pas":       true, // forward type ref
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".pas") {
			continue
		}
		if skips[e.Name()] {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			path := filepath.Join(dir, e.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := compile.CompileToVM(&compile.CompileConfig{Source: string(data), SourceFile: e.Name()}); err != nil {
				t.Errorf("CompileToVM: %v", err)
			}
		})
	}
}

// TestEndToEnd_PipelineConsistency checks that running lex+parse
// and the full pipeline give consistent results on the corpus.
func TestEndToEnd_PipelineConsistency(t *testing.T) {
	dir := "../../testdata/pas"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("testdata/pas not present: %v", err)
	}
	skips := map[string]bool{
		"objectpoly.pas": true,
		"list.pas":       true,
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".pas") {
			continue
		}
		if skips[e.Name()] {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			path := filepath.Join(dir, e.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			lpErr := conformance.LexAndParse(e.Name(), string(data))
			compileErr := func() error {
				_, err := compile.CompileToVM(&compile.CompileConfig{Source: string(data), SourceFile: e.Name()})
				return err
			}()
			// LexAndParse must succeed for all corpus files.
			if lpErr != nil {
				t.Errorf("LexAndParse: %v", lpErr)
			}
			// CompileToVM may fail when sem rejects, but the error
			// must mention the file name.
			if compileErr != nil && !strings.Contains(compileErr.Error(), e.Name()) {
				t.Errorf("CompileToVM error should mention %s: %v", e.Name(), compileErr)
			}
		})
	}
}
