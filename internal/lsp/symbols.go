package lsp

import (
	"strings"

	"github.com/arturoeanton/go-turbo-pascal/internal/ast"
	"github.com/arturoeanton/go-turbo-pascal/internal/lexer"
	"github.com/arturoeanton/go-turbo-pascal/internal/parser"
)

// LSP SymbolKind values used by the document-symbol, hover and completion
// features.
const (
	symModule   = 2
	symClass    = 5
	symEnum     = 10
	symFunction = 12
	symVariable = 13
	symConstant = 14
	symKeyword  = 14 // completion item kind: Keyword
)

// Symbol is a named declaration with its 1-based source position.
type Symbol struct {
	Name      string
	Detail    string // signature or type, shown on hover/completion
	Kind      int    // LSP SymbolKind
	Line, Col int    // declaration position (1-based)
}

// Symbols parses src and returns its top-level declarations (program/unit name,
// consts, types, vars, routines). Parsing is best-effort: whatever parsed is
// returned even if the document has later errors.
func Symbols(src string) []Symbol {
	l := lexer.New(src)
	p := parser.New(l.Tokens())
	p.SetFile("document.pas")
	node := p.ParseUnit()

	var syms []Symbol
	add := func(name, detail string, kind, line, col int) {
		if name == "" {
			return
		}
		syms = append(syms, Symbol{Name: name, Detail: detail, Kind: kind, Line: line, Col: col})
	}

	collectDecls := func(decls []ast.Decl) {
		for _, d := range decls {
			collectDecl(d, add)
		}
	}

	switch n := node.(type) {
	case *ast.Program:
		add(n.Name, "program "+n.Name, symModule, n.P.Line, n.P.Col)
		if n.Block != nil {
			collectDecls(n.Block.Consts)
			collectDecls(n.Block.Types)
			collectDecls(n.Block.Vars)
			collectDecls(n.Block.Procs)
		}
	case *ast.Unit:
		add(n.Name, "unit "+n.Name, symModule, n.P.Line, n.P.Col)
		if n.Interface != nil {
			collectDecls(n.Interface.Decls)
		}
		if n.Implementation != nil {
			collectDecls(n.Implementation.Decls)
		}
	}
	return syms
}

func collectDecl(d ast.Decl, add func(name, detail string, kind, line, col int)) {
	switch v := d.(type) {
	case *ast.ConstDecl:
		add(v.Name, "const "+v.Name, symConstant, v.P.Line, v.P.Col)
	case *ast.TypeDecl:
		kind := symClass
		if isEnumType(v.Type) {
			kind = symEnum
		}
		add(v.Name, "type "+v.Name+" = "+typeText(v.Type), kind, v.P.Line, v.P.Col)
	case *ast.VarDecl:
		for _, name := range v.Names {
			add(name, "var "+name+": "+typeText(v.Type), symVariable, v.P.Line, v.P.Col)
		}
	case *ast.ProcDecl:
		add(v.Name, routineSignature(v), symFunction, v.P.Line, v.P.Col)
	}
}

// routineSignature renders a "procedure Name(p: T; ...): Result" string.
func routineSignature(pd *ast.ProcDecl) string {
	kw := "procedure"
	if pd.Result != nil {
		kw = "function"
	}
	var b strings.Builder
	b.WriteString(kw)
	b.WriteString(" ")
	b.WriteString(pd.Name)
	if len(pd.Params) > 0 {
		b.WriteString("(")
		for i, par := range pd.Params {
			if i > 0 {
				b.WriteString("; ")
			}
			if par.Var {
				b.WriteString("var ")
			} else if par.Const {
				b.WriteString("const ")
			}
			b.WriteString(strings.Join(par.Names, ", "))
			if par.Type != nil {
				b.WriteString(": ")
				b.WriteString(typeText(par.Type))
			}
		}
		b.WriteString(")")
	}
	if pd.Result != nil {
		b.WriteString(": ")
		b.WriteString(typeText(pd.Result))
	}
	return b.String()
}

func typeText(t ast.TypeExpr) string {
	if t == nil {
		return ""
	}
	return t.String()
}

func isEnumType(t ast.TypeExpr) bool {
	_, ok := t.(*ast.EnumType)
	return ok
}

// wordAt returns the identifier token covering the 0-based LSP position
// (line, character), or "" if none. Token positions are 1-based.
func wordAt(src string, line, character int) string {
	tok := tokenAt(src, line, character)
	if tok == nil || tok.Kind != lexer.TokIdent {
		return ""
	}
	return tok.Text
}

// tokenAt returns the token covering the 0-based LSP position, or nil.
func tokenAt(src string, line, character int) *lexer.Token {
	l := lexer.New(src)
	toks := l.Tokens()
	wantLine := line + 1
	wantCol := character + 1
	for i := range toks {
		t := &toks[i]
		if t.Line != wantLine {
			continue
		}
		if wantCol >= t.Col && wantCol < t.Col+len(t.Text) {
			return t
		}
	}
	return nil
}
