package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arturoeanton/go-turbo-pascal/internal/conformance"
)

func TestInteractiveExamplesParse(t *testing.T) {
	dir := "../../examples/interactive"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("examples not present: %v", err)
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".pas") {
			continue
		}
		count++
		name := e.Name()
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(dir, name)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := conformance.LexAndParse(name, string(data)); err != nil {
				t.Errorf("LexAndParse: %v", err)
			}
		})
	}
	if count < 10 {
		t.Fatalf("expected at least 10 interactive examples, got %d", count)
	}
}
