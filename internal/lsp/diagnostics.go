// Package lsp implements a Language Server Protocol server for Turbo Pascal 7
// source, backed by the BPGo front-end. It provides live diagnostics (lex,
// parse and codegen errors), document symbols, hover, go-to-definition and
// completion (document symbols plus Pascal keywords).
package lsp

import (
	"regexp"
	"strconv"

	"github.com/arturoeanton/go-turbo-pascal/internal/codegen"
	"github.com/arturoeanton/go-turbo-pascal/internal/lexer"
	"github.com/arturoeanton/go-turbo-pascal/internal/parser"
)

// Diagnostic is a 1-based line/column problem report.
type Diagnostic struct {
	Line    int
	Col     int
	Message string
}

var posRe = regexp.MustCompile(`^line (\d+) col (\d+): (.*)$`)

// Analyze returns the diagnostics for a Pascal source document. It stops at the
// first failing stage (lexing blocks parsing; parsing blocks codegen) so the
// reported errors are always meaningful.
func Analyze(src string) []Diagnostic {
	var out []Diagnostic

	l := lexer.New(src)
	if errs := l.Errors(); len(errs) > 0 {
		for _, e := range errs {
			out = append(out, parseDiag(e))
		}
		return out
	}

	p := parser.New(l.Tokens())
	p.SetModern(l.ModeBPGo())
	p.SetFile("document.pas")
	p.ParseUnit()
	if errs := p.Errors(); len(errs) > 0 {
		for _, e := range errs {
			out = append(out, parseDiag(e))
		}
		return out
	}

	// Codegen catches type/semantic errors (unknown identifiers, etc.).
	if _, err := codegen.Compile(src, "document.pas"); err != nil {
		out = append(out, Diagnostic{Line: 1, Col: 1, Message: err.Error()})
	}
	return out
}

// parseDiag extracts a position from a "line N col M: message" string.
func parseDiag(s string) Diagnostic {
	if m := posRe.FindStringSubmatch(s); m != nil {
		line, _ := strconv.Atoi(m[1])
		col, _ := strconv.Atoi(m[2])
		return Diagnostic{Line: line, Col: col, Message: m[3]}
	}
	return Diagnostic{Line: 1, Col: 1, Message: s}
}
