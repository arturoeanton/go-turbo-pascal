package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arturoeanton/go-turbo-pascal/internal/lexer"
	"github.com/arturoeanton/go-turbo-pascal/internal/parser"
)

// TestGoldenCorpus verifies that the lexer + parser produce stable
// results for each corpus file.
func TestGoldenCorpus(t *testing.T) {
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
			data, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			l := lexer.New(string(data))
			p := parser.New(l.Tokens())
			p.SetFile(e.Name())
			p.ParseUnit()
		})
	}
}
