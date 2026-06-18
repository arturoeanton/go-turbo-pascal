package lsp

import "testing"

const sampleSrc = `program Demo;
const Pi = 3;
type TColor = (Red, Green, Blue);
var count: Integer;
function Square(n: Integer): Integer;
begin
  Square := n * n;
end;
begin
  count := Square(4);
end.`

func TestSymbolsExtractsDeclarations(t *testing.T) {
	syms := Symbols(sampleSrc)
	want := map[string]int{
		"Demo":   symModule,
		"Pi":     symConstant,
		"TColor": symEnum,
		"count":  symVariable,
		"Square": symFunction,
	}
	got := map[string]int{}
	for _, s := range syms {
		got[s.Name] = s.Kind
	}
	for name, kind := range want {
		if got[name] != kind {
			t.Errorf("symbol %q: kind = %d, want %d (got %+v)", name, got[name], kind, got)
		}
	}
}

func TestSymbolSignatureForFunction(t *testing.T) {
	sym, ok := findSymbol(sampleSrc, "Square")
	if !ok {
		t.Fatal("Square not found")
	}
	if sym.Detail != "function Square(n: Integer): Integer" {
		t.Fatalf("signature = %q", sym.Detail)
	}
}

func TestWordAtResolvesIdentifier(t *testing.T) {
	// Line 10 (0-based 9): "  count := Square(4);" — cursor on "Square".
	w := wordAt(sampleSrc, 9, 12)
	if w != "Square" {
		t.Fatalf("wordAt = %q, want Square", w)
	}
}

func TestHoverReturnsSignature(t *testing.T) {
	s := &Server{docs: map[string]string{"u": sampleSrc}}
	h := s.hover("u", 9, 12) // on Square
	if h == nil {
		t.Fatal("expected hover content for Square")
	}
	m := h.(map[string]interface{})
	contents := m["contents"].(map[string]interface{})
	if got := contents["value"].(string); got == "" {
		t.Fatal("hover value empty")
	}
}

func TestDefinitionPointsAtDeclaration(t *testing.T) {
	s := &Server{docs: map[string]string{"u": sampleSrc}}
	d := s.definition("u", 9, 12) // use of Square -> its declaration
	if d == nil {
		t.Fatal("expected a definition location")
	}
	m := d.(map[string]interface{})
	rng := m["range"].(map[string]interface{})
	start := rng["start"].(map[string]int)
	if start["line"] != 4 { // Square declared on line 5 (0-based 4)
		t.Fatalf("definition line = %d, want 4", start["line"])
	}
}

func TestCompletionIncludesSymbolsAndKeywords(t *testing.T) {
	s := &Server{docs: map[string]string{"u": sampleSrc}}
	res := s.completion("u")
	items := res["items"].([]map[string]interface{})
	var hasSquare, hasBegin bool
	for _, it := range items {
		switch it["label"] {
		case "Square":
			hasSquare = true
		case "begin":
			hasBegin = true
		}
	}
	if !hasSquare {
		t.Error("completion missing the Square symbol")
	}
	if !hasBegin {
		t.Error("completion missing the 'begin' keyword")
	}
}
