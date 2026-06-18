// Package parser implements a recursive descent parser for Turbo Pascal
// 7 / Borland Pascal 7. It produces an AST from the token stream
// returned by internal/lexer. The parser performs basic error recovery
// so that a single malformed construct does not silence every error
// after it.
//
// The parser intentionally does not enforce all type rules: that is the
// responsibility of internal/sem. The parser only enforces TP7 syntax
// (keywords, separators, structure).
package parser

import (
	"fmt"
	"strings"

	"github.com/arturoeanton/go-turbo-pascal/internal/ast"
	"github.com/arturoeanton/go-turbo-pascal/internal/lexer"
)

type Parser struct {
	tokens   []lexer.Token
	pos      int
	errs     []string
	file     string
	modeBPGo bool // modern (BPGo) extensions enabled via {$MODE BPGO}
}

// SetModern enables the modern (BPGo) language extensions. Callers pass the
// lexer's ModeBPGo() so contextual keywords (let, match, ...) activate only
// under {$MODE BPGO}.
func (p *Parser) SetModern(on bool) { p.modeBPGo = on }

// isModern reports whether the modern extensions are enabled.
func (p *Parser) isModern() bool { return p.modeBPGo }

// isContextualKw reports whether the current token is the identifier `word`
// acting as a contextual keyword: it only counts under {$MODE BPGO}, so outside
// that mode the word stays a plain identifier (full TP7 compatibility).
func (p *Parser) isContextualKw(word string) bool {
	return p.modeBPGo && p.cur().Kind == lexer.TokIdent && p.cur().Lower == word
}

func New(toks []lexer.Token) *Parser {
	return &Parser{tokens: toks, file: ""}
}

func (p *Parser) SetFile(name string) { p.file = name }

func (p *Parser) Errors() []string { return p.errs }

func (p *Parser) cur() lexer.Token { return p.tokens[p.pos] }

func (p *Parser) peekKind() lexer.TokenKind { return p.cur().Kind }

func (p *Parser) peekText() string { return p.cur().Text }

func (p *Parser) advance() lexer.Token {
	t := p.tokens[p.pos]
	if t.Kind != lexer.TokEOF {
		p.pos++
	}
	return t
}

func (p *Parser) check(k lexer.TokenKind, text ...string) bool {
	t := p.cur()
	if t.Kind != k {
		return false
	}
	if len(text) == 0 {
		return true
	}
	low := strings.ToLower(t.Text)
	for _, w := range text {
		if low == strings.ToLower(w) {
			return true
		}
	}
	return false
}

func (p *Parser) match(k lexer.TokenKind, text ...string) bool {
	if p.check(k, text...) {
		p.advance()
		return true
	}
	return false
}

func (p *Parser) matchTok(k lexer.TokenKind, text ...string) (lexer.Token, bool) {
	if p.check(k, text...) {
		return p.advance(), true
	}
	return lexer.Token{}, false
}

func (p *Parser) expect(k lexer.TokenKind, text ...string) lexer.Token {
	if p.check(k, text...) {
		return p.advance()
	}
	p.errExpected(k, p.cur(), text)
	return lexer.Token{Kind: k, Line: p.cur().Line, Col: p.cur().Col}
}

func (p *Parser) errExpected(k lexer.TokenKind, got lexer.Token, hint []string) {
	msg := fmt.Sprintf("line %d col %d: expected %v", got.Line, got.Col, k)
	if len(hint) > 0 {
		msg += " (" + strings.Join(hint, "/") + ")"
	}
	msg += ", got " + describeTok(got)
	p.errs = append(p.errs, msg)
}

func describeTok(t lexer.Token) string {
	if t.Kind == lexer.TokIdent {
		return "ident " + t.Text
	}
	if t.Kind == lexer.TokKeyword {
		return "keyword " + t.Lower
	}
	return t.Kind.String() + " " + t.Text
}

func (p *Parser) errf(format string, args ...any) {
	t := p.cur()
	p.errs = append(p.errs, fmt.Sprintf("line %d col %d: "+format, append([]any{t.Line, t.Col}, args...)...))
}

func (p *Parser) is(tok string) bool {
	t := p.cur()
	if t.Kind == lexer.TokKeyword && t.Lower == strings.ToLower(tok) {
		return true
	}
	return false
}

func (p *Parser) skipTo(set ...string) {
	for p.cur().Kind != lexer.TokEOF {
		for _, kw := range set {
			if p.is(kw) {
				return
			}
		}
		p.advance()
	}
}

func (p *Parser) curPos() ast.Pos {
	t := p.cur()
	return ast.Pos{File: p.file, Line: t.Line, Col: t.Col}
}

// ParseUnit determines whether the source is a program or a unit.
func (p *Parser) ParseUnit() ast.Node {
	if p.cur().Kind == lexer.TokKeyword && p.cur().Lower == "unit" {
		return p.parseUnitDecl()
	}
	return p.parseProgram()
}

func (p *Parser) parseProgram() ast.Node {
	start := p.curPos()
	p.advance() // program
	id := p.expect(lexer.TokIdent)
	// Optional program parameter list: program Name(Input, Output);
	if p.match(lexer.TokLParen) {
		for p.cur().Kind != lexer.TokRParen && p.cur().Kind != lexer.TokEOF {
			p.advance()
		}
		p.expect(lexer.TokRParen)
	}
	p.expect(lexer.TokSemicolon)
	// The uses clause follows the program header's semicolon.
	var uses *ast.UsesClause
	if p.is("uses") {
		uses = p.parseUses()
		p.expect(lexer.TokSemicolon)
	}
	block := p.parseBlock(true)
	body := p.parseBlockBody()
	p.expect(lexer.TokPeriod)
	return &ast.Program{Base: ast.Base{P: start}, Name: id.Text, Uses: uses, Block: block, Body: body}
}

func (p *Parser) parseUnitDecl() ast.Node {
	start := p.curPos()
	p.advance() // unit
	id := p.expect(lexer.TokIdent)
	p.expect(lexer.TokSemicolon)
	iface := p.parseInterface()
	impl := p.parseImplementation()
	var init *ast.BlockBody
	if p.is("initialization") {
		p.advance()
		// initialization is a statement sequence (no begin), ending at
		// finalization or end.
		var stmts []ast.Stmt
		for !p.is("end") && !p.is("finalization") && p.cur().Kind != lexer.TokEOF {
			if s := p.parseStmt(); s != nil {
				stmts = append(stmts, s)
			}
			if !p.match(lexer.TokSemicolon) {
				break
			}
		}
		init = &ast.BlockBody{Base: ast.Base{P: start}, Stmts: stmts}
		if p.is("finalization") {
			p.advance()
			for !p.is("end") && p.cur().Kind != lexer.TokEOF {
				p.parseStmt()
				if !p.match(lexer.TokSemicolon) {
					break
				}
			}
		}
	} else if p.is("begin") {
		// Alternative form: a begin..end initialization block.
		init = p.parseBlockBody()
	}
	if p.is("end") {
		p.advance()
	}
	p.expect(lexer.TokPeriod)
	return &ast.Unit{Base: ast.Base{P: start}, Name: id.Text, Interface: iface, Implementation: impl, Init: init}
}

func (p *Parser) parseUses() *ast.UsesClause {
	start := p.curPos()
	p.advance() // uses
	cl := &ast.UsesClause{Base: ast.Base{P: start}}
	for {
		id := p.expect(lexer.TokIdent)
		item := ast.UnitRef{Base: ast.Base{P: ast.Pos{File: p.file, Line: id.Line, Col: id.Col}}, Name: id.Text}
		if p.cur().Kind == lexer.TokKeyword && p.cur().Lower == "in" {
			p.advance()
			s := p.expect(lexer.TokString)
			item.In = s.Str
		}
		cl.Items = append(cl.Items, item)
		if !p.match(lexer.TokComma) {
			break
		}
	}
	return cl
}

func (p *Parser) parseInterface() *ast.InterfaceSection {
	start := p.curPos()
	if !p.is("interface") {
		p.errf("expected 'interface'")
		p.advance()
	}
	p.advance()
	sect := &ast.InterfaceSection{Base: ast.Base{P: start}}
	if p.is("uses") {
		sect.Uses = p.parseUses()
		p.expect(lexer.TokSemicolon)
	}
	for !(p.cur().Kind == lexer.TokKeyword && (p.cur().Lower == "implementation" || p.cur().Lower == "initialization")) {
		if p.cur().Kind == lexer.TokEOF {
			p.errf("unexpected EOF in interface")
			break
		}
		if p.cur().Kind == lexer.TokSemicolon {
			p.advance()
			continue
		}
		d := p.parseDecl(false)
		if d != nil {
			sect.Decls = append(sect.Decls, d)
		}
	}
	return sect
}

func (p *Parser) parseImplementation() *ast.ImplementationSection {
	start := p.curPos()
	if !p.is("implementation") {
		p.errf("expected 'implementation'")
		p.advance()
	}
	p.advance()
	sect := &ast.ImplementationSection{Base: ast.Base{P: start}}
	if p.is("uses") {
		sect.Uses = p.parseUses()
		p.expect(lexer.TokSemicolon)
	}
	for !(p.cur().Kind == lexer.TokKeyword && p.cur().Lower == "initialization") {
		if p.cur().Kind == lexer.TokEOF {
			p.errf("unexpected EOF in implementation")
			break
		}
		if p.cur().Kind == lexer.TokSemicolon {
			p.advance()
			continue
		}
		// end (closing the unit) stops the implementation.
		if p.cur().Kind == lexer.TokKeyword && p.cur().Lower == "end" {
			break
		}
		d := p.parseDecl(true)
		if d != nil {
			sect.Decls = append(sect.Decls, d)
		}
	}
	return sect
}

func (p *Parser) parseDecl(allowBodies bool) ast.Decl {
	start := p.curPos()
	_ = start
	t := p.cur()
	switch t.Kind {
	case lexer.TokKeyword:
		switch t.Lower {
		case "const":
			p.advance()
			return p.parseConstDecl()
		case "type":
			p.advance()
			return p.parseTypeDecl()
		case "var":
			p.advance()
			return p.parseVarDecl()
		case "procedure", "function", "constructor", "destructor":
			return p.parseProcDecl(allowBodies)
		}
	}
	p.errf("expected declaration, got %v", describeTok(t))
	p.advance()
	return nil
}

func (p *Parser) parseBlock(isProgram bool) *ast.Block {
	start := p.curPos()
	blk := &ast.Block{Base: ast.Base{P: start}}
	for {
		t := p.cur()
		// {$MODE BPGO}: `let` is a contextual keyword introducing an immutable
		// declaration alongside const/type/var. Outside modern mode it stays a
		// plain identifier and ends the declaration section as usual.
		if p.isContextualKw("let") {
			if d := p.parseLetDecl(); d != nil {
				blk.Vars = append(blk.Vars, d)
			}
			p.match(lexer.TokSemicolon)
			continue
		}
		// {$MODE BPGO}: `test 'name' begin ... end` integrated unit test.
		if p.isContextualKw("test") {
			if tb := p.parseTestBlock(); tb != nil {
				blk.Tests = append(blk.Tests, tb)
			}
			p.match(lexer.TokSemicolon)
			continue
		}
		if t.Kind != lexer.TokKeyword {
			return blk
		}
		switch t.Lower {
		case "label":
			p.advance()
			for {
				n := p.expect(lexer.TokInt)
				blk.Labels = append(blk.Labels, int(n.Int))
				if !p.match(lexer.TokComma) {
					break
				}
			}
			p.expect(lexer.TokSemicolon)
		case "const":
			p.advance()
			for !(p.cur().Kind == lexer.TokKeyword && p.isBlockStart(p.cur().Lower)) && p.cur().Kind != lexer.TokEOF {
				if p.is("procedure") || p.is("function") || p.is("constructor") || p.is("destructor") || p.is("type") || p.is("var") || p.is("label") || p.is("begin") || p.is("uses") {
					break
				}
				c := p.parseConstDecl()
				if c != nil {
					blk.Consts = append(blk.Consts, c)
				}
				p.match(lexer.TokSemicolon)
			}
		case "type":
			p.advance()
			for !(p.cur().Kind == lexer.TokKeyword && p.isBlockStart(p.cur().Lower)) && p.cur().Kind != lexer.TokEOF {
				if p.is("procedure") || p.is("function") || p.is("constructor") || p.is("destructor") || p.is("const") || p.is("var") || p.is("label") || p.is("begin") || p.is("uses") {
					break
				}
				td := p.parseTypeDecl()
				if td != nil {
					blk.Types = append(blk.Types, td)
				}
				p.match(lexer.TokSemicolon)
			}
		case "var":
			p.advance()
			for !(p.cur().Kind == lexer.TokKeyword && p.isBlockStart(p.cur().Lower)) && p.cur().Kind != lexer.TokEOF {
				if p.is("procedure") || p.is("function") || p.is("constructor") || p.is("destructor") || p.is("const") || p.is("type") || p.is("label") || p.is("begin") || p.is("uses") {
					break
				}
				v := p.parseVarDecl()
				if v != nil {
					blk.Vars = append(blk.Vars, v)
				}
				p.match(lexer.TokSemicolon)
			}
		case "procedure", "function", "constructor", "destructor":
			pr := p.parseProcDecl(true)
			if pr != nil {
				blk.Procs = append(blk.Procs, pr)
			}
			p.match(lexer.TokSemicolon)
		case "operator":
			if pr := p.parseOperatorDecl(); pr != nil {
				blk.Procs = append(blk.Procs, pr)
			}
			p.match(lexer.TokSemicolon)
		case "uses":
			// `uses` in a block is rare but allowed (TP7 tolerates it
			// for compatibility with some unit conventions). We
			// accept it and parse the following unit list.
			p.parseUses()
			p.match(lexer.TokSemicolon)
		default:
			return blk
		}
	}
}

func (p *Parser) isBlockStart(kw string) bool {
	switch kw {
	case "label", "const", "type", "var", "procedure", "function", "constructor", "destructor", "operator", "begin", "uses":
		return true
	}
	return false
}

func (p *Parser) parseConstDecl() ast.Decl {
	start := p.curPos()
	id := p.expect(lexer.TokIdent)
	var t ast.TypeExpr
	if p.match(lexer.TokColon) {
		t = p.parseType()
	}
	p.expect(lexer.TokEqual)
	v := p.parseExpr()
	return &ast.ConstDecl{Base: ast.Base{P: start}, Name: id.Text, Type: t, Value: v}
}

func (p *Parser) parseTypeDecl() ast.Decl {
	start := p.curPos()
	id := p.expect(lexer.TokIdent)
	var typeParams []string
	if p.check(lexer.TokOp, "<") { // generic type: TList<T> = ...
		typeParams = p.parseTypeParams()
	}
	p.expect(lexer.TokEqual)
	t := p.parseType()
	return &ast.TypeDecl{Base: ast.Base{P: start}, Name: id.Text, TypeParams: typeParams, Type: t}
}

// parseTypeParams parses a generic parameter list `<T1, T2: Constraint, ...>`,
// returning the parameter names. Constraints are accepted and erased.
func (p *Parser) parseTypeParams() []string {
	var params []string
	p.expect(lexer.TokOp, "<")
	for {
		id := p.expect(lexer.TokIdent)
		params = append(params, id.Text)
		if p.match(lexer.TokColon) {
			p.parseType() // constraint, erased
		}
		if !p.match(lexer.TokComma) {
			break
		}
	}
	p.expect(lexer.TokOp, ">")
	return params
}

// skipTypeArgs consumes a generic argument list `<Type, ...>` in a type
// reference. Type arguments are erased (the runtime is dynamically typed).
func (p *Parser) skipTypeArgs() {
	p.expect(lexer.TokOp, "<")
	for {
		p.parseType()
		if !p.match(lexer.TokComma) {
			break
		}
	}
	p.expect(lexer.TokOp, ">")
}

func (p *Parser) parseVarDecl() ast.Decl {
	start := p.curPos()
	names := []string{}
	for {
		id := p.expect(lexer.TokIdent)
		names = append(names, id.Text)
		if !p.match(lexer.TokComma) {
			break
		}
	}
	// {$MODE BPGO}: `var x := expr` infers the type from the initializer.
	if p.isModern() && p.check(lexer.TokAssign) {
		p.advance() // :=
		return &ast.VarDecl{Base: ast.Base{P: start}, Names: names, Init: p.parseExpr()}
	}
	p.expect(lexer.TokColon)
	t := p.parseType()
	var abs ast.Expr
	var init ast.Expr
	if p.cur().Kind == lexer.TokKeyword && p.cur().Lower == "absolute" {
		p.advance()
		abs = p.parsePrimary()
	} else if p.isModern() && p.match(lexer.TokEqual) {
		init = p.parseExpr() // var x: T = expr
	}
	return &ast.VarDecl{Base: ast.Base{P: start}, Names: names, Type: t, Abs: abs, Init: init}
}

// parseTestBlock parses `test 'name' begin ... end` ({$MODE BPGO}).
func (p *Parser) parseTestBlock() *ast.TestBlock {
	start := p.curPos()
	p.advance() // 'test'
	name := ""
	if p.check(lexer.TokString) {
		name = p.cur().Str
		p.advance()
	}
	return &ast.TestBlock{Base: ast.Base{P: start}, Name: name, Body: p.parseBlockBody()}
}

// parseLetDecl parses a modern immutable binding: `let x [: T] = expr` or
// `let x [: T] := expr`. It is a declaration-section entry ({$MODE BPGO}).
func (p *Parser) parseLetDecl() ast.Decl {
	start := p.curPos()
	p.advance() // 'let'
	name := p.expect(lexer.TokIdent).Text
	var typ ast.TypeExpr
	if p.match(lexer.TokColon) {
		typ = p.parseType()
	}
	if !p.match(lexer.TokAssign) { // accept '=' or ':='
		p.match(lexer.TokEqual)
	}
	return &ast.VarDecl{Base: ast.Base{P: start}, Names: []string{name}, Type: typ, Init: p.parseExpr(), Immutable: true}
}

func (p *Parser) parseType() ast.TypeExpr {
	start := p.curPos()
	t := p.cur()
	// Range type: const .. const
	if t.Kind == lexer.TokInt || t.Kind == lexer.TokHex || t.Kind == lexer.TokString {
		save := p.pos
		e := p.parseExpr()
		if re, ok := e.(*ast.RangeExpr); ok {
			return &ast.RangeType{Base: ast.Base{P: start}, Lo: re.Lo, Hi: re.Hi}
		}
		// Not a range; rewind and parse as type ref (rare).
		p.pos = save
	}
	if t.Kind == lexer.TokKeyword {
		switch t.Lower {
		case "packed":
			p.advance()
			inner := p.parseType()
			markPacked(inner)
			return inner
		case "array":
			return p.parseArrayType()
		case "set":
			return p.parseSetType()
		case "file":
			return p.parseFileType()
		case "string":
			p.advance()
			if p.match(lexer.TokLBracket) {
				n := p.parseExpr()
				p.expect(lexer.TokRBracket)
				return &ast.StringType{Base: ast.Base{P: start}, Len: n}
			}
			return &ast.StringType{Base: ast.Base{P: start}}
		case "record":
			return p.parseRecordType()
		case "object", "class", "interface":
			return p.parseObjectType("")
		case "procedure", "function":
			return p.parseProcType()
		}
	}
	if t.Kind == lexer.TokLParen {
		// enumerated type or ( subrange )
		p.advance()
		names := []string{}
		for {
			id := p.expect(lexer.TokIdent)
			names = append(names, id.Text)
			if !p.match(lexer.TokComma) {
				break
			}
		}
		if p.match(lexer.TokRange) {
			hi := p.parseExpr()
			p.expect(lexer.TokRParen)
			lo := &ast.Ident{Base: ast.Base{P: start}, Name: names[0], Lower: strings.ToLower(names[0])}
			return &ast.RangeType{Base: ast.Base{P: start}, Lo: lo, Hi: hi}
		}
		p.expect(lexer.TokRParen)
		return &ast.EnumType{Base: ast.Base{P: start}, Names: names}
	}
	if t.Kind == lexer.TokCaret {
		p.advance()
		inner := p.parseType()
		return &ast.PointerType{Base: ast.Base{P: start}, Target: inner}
	}
	if t.Kind == lexer.TokIdent {
		p.advance()
		name := t.Text
		low := strings.ToLower(name)
		switch low {
		case "integer", "longint", "shortint", "byte", "word", "boolean", "char":
			return &ast.OrdType{Base: ast.Base{P: start}, Kind: titleCase(low)}
		case "real", "single", "double", "extended", "comp":
			return &ast.FloatType{Base: ast.Base{P: start}, Kind: titleCase(low)}
		case "text":
			return &ast.FileType{Base: ast.Base{P: start}, Text: true}
		}
		// Generic instantiation `Base<Arg, ...>`: erase the type arguments.
		if p.check(lexer.TokOp, "<") {
			p.skipTypeArgs()
		}
		// could be a parameterized type (String[n], array[0..N] of ...)
		return &ast.TypeRef{Base: ast.Base{P: start}, Name: name, Lower: low}
	}
	// fallback: parse an expression
	e := p.parseExpr()
	return &ast.TypeRef{Base: ast.Base{P: start}, Name: e.String()}
}

func titleCase(s string) string {
	return strings.ToUpper(s[:1]) + s[1:]
}

func markPacked(t ast.TypeExpr) {
	switch v := t.(type) {
	case *ast.ArrayType:
		v.Packed = true
	case *ast.RecordType:
		v.Packed = true
	case *ast.SetType:
		v.Packed = true
	}
}

func (p *Parser) parseArrayType() ast.TypeExpr {
	start := p.curPos()
	p.advance() // array
	// Dynamic array: `array of T` (no index bounds).
	if p.is("of") {
		p.advance()
		elem := p.parseType()
		return &ast.ArrayType{Base: ast.Base{P: start}, Index: nil, Element: elem}
	}
	p.expect(lexer.TokLBracket)
	idxs := []ast.RangeExpr{}
	for {
		lo := p.parsePrimary()
		p.expect(lexer.TokRange)
		hi := p.parsePrimary()
		idxs = append(idxs, ast.RangeExpr{Base: ast.Base{P: start}, Lo: lo, Hi: hi})
		if !p.match(lexer.TokComma) {
			break
		}
	}
	p.expect(lexer.TokRBracket)
	p.expect(lexer.TokKeyword, "of")
	elem := p.parseType()
	return &ast.ArrayType{Base: ast.Base{P: start}, Index: idxs, Element: elem}
}

func (p *Parser) parseSetType() ast.TypeExpr {
	start := p.curPos()
	p.advance() // set
	p.expect(lexer.TokKeyword, "of")
	elem := p.parseType()
	return &ast.SetType{Base: ast.Base{P: start}, Element: elem}
}

func (p *Parser) parseFileType() ast.TypeExpr {
	start := p.curPos()
	p.advance() // file
	if p.is("of") {
		p.advance()
		elem := p.parseType()
		return &ast.FileType{Base: ast.Base{P: start}, Element: elem}
	}
	return &ast.FileType{Base: ast.Base{P: start}}
}

func (p *Parser) parseRecordType() ast.TypeExpr {
	start := p.curPos()
	p.advance() // record
	if p.isContextualKw("helper") { // {$MODE BPGO}: record helper for Base
		return p.parseHelperType(start, false)
	}
	r := &ast.RecordType{Base: ast.Base{P: start}}
	for {
		if p.is("end") {
			break
		}
		if p.is("case") {
			// variant part
			p.advance()
			tag := p.parseIdent()
			if !p.match(lexer.TokColon) {
				tag = &ast.Ident{Base: ast.Base{P: start}, Name: "", Lower: ""}
			}
			tp := p.parseType()
			p.expect(lexer.TokKeyword, "of")
			varCases := []ast.VariantCase{}
			for {
				if p.is("end") {
					break
				}
				values := []ast.Expr{}
				for {
					v := p.parseExpr()
					values = append(values, v)
					if !p.match(lexer.TokComma) {
						break
					}
				}
				p.expect(lexer.TokColon)
				p.expect(lexer.TokLParen)
				vfields := []ast.RecordField{}
				for {
					if p.match(lexer.TokRParen) {
						break
					}
					names := []string{}
					for {
						id := p.expect(lexer.TokIdent)
						names = append(names, id.Text)
						if !p.match(lexer.TokComma) {
							break
						}
					}
					p.expect(lexer.TokColon)
					ft := p.parseType()
					vfields = append(vfields, ast.RecordField{Base: ast.Base{P: start}, Names: names, Type: ft})
					p.match(lexer.TokSemicolon)
				}
				varCases = append(varCases, ast.VariantCase{Base: ast.Base{P: start}, Values: values, Fields: vfields})
				if !p.match(lexer.TokSemicolon) {
					break
				}
			}
			r.Variant = &ast.VariantPart{Base: ast.Base{P: start}, Tag: &ast.Ident{Base: ast.Base{P: start}, Name: tag.String()}, TagType: tp, Cases: varCases}
			break
		}
		names := []string{}
		for {
			id := p.expect(lexer.TokIdent)
			names = append(names, id.Text)
			if !p.match(lexer.TokComma) {
				break
			}
		}
		p.expect(lexer.TokColon)
		ft := p.parseType()
		r.Fields = append(r.Fields, ast.RecordField{Base: ast.Base{P: start}, Names: names, Type: ft})
		p.match(lexer.TokSemicolon)
	}
	p.expect(lexer.TokKeyword, "end")
	return r
}

func (p *Parser) parseObjectType(parentName string) ast.TypeExpr {
	start := p.curPos()
	isClass := p.is("class")
	isInterface := p.is("interface")
	p.advance() // object | class | interface
	if isClass && p.isContextualKw("helper") { // {$MODE BPGO}: class helper for Base
		return p.parseHelperType(start, true)
	}
	o := &ast.ObjectType{Base: ast.Base{P: start}, IsClass: isClass || isInterface, IsInterface: isInterface}
	if p.cur().Kind == lexer.TokLParen {
		p.advance()
		id := p.expect(lexer.TokIdent)
		o.Parent = id.Text
		// class(TParent, IFoo, IBar): the extra names are implemented interfaces.
		for p.match(lexer.TokComma) {
			o.Implements = append(o.Implements, p.expect(lexer.TokIdent).Text)
		}
		p.expect(lexer.TokRParen)
	}
	// `class` forward declaration: `TFoo = class;`
	if isClass && p.cur().Kind == lexer.TokSemicolon {
		return o
	}
	for {
		if p.is("end") || p.cur().Kind == lexer.TokEOF {
			break
		}
		// Visibility sections (private/public/...) are accepted and ignored.
		if p.is("private") || p.is("public") || p.is("protected") || p.is("published") {
			p.advance()
			continue
		}
		if p.is("property") {
			o.Properties = append(o.Properties, p.parsePropertyDef())
			for p.cur().Kind == lexer.TokSemicolon {
				p.advance()
			}
			continue
		}
		if p.is("procedure") || p.is("function") || p.is("constructor") || p.is("destructor") {
			// Inside an object only the method *signature* appears (the body is
			// a separate Type.Method declaration), so parse a signature only.
			if pr := p.parseMethodSig(); pr != nil {
				o.Methods = append(o.Methods, *pr)
			}
			for p.cur().Kind == lexer.TokSemicolon {
				p.advance()
			}
			continue
		}
		names := []string{}
		for {
			id := p.expect(lexer.TokIdent)
			names = append(names, id.Text)
			if !p.match(lexer.TokComma) {
				break
			}
		}
		p.expect(lexer.TokColon)
		ft := p.parseType()
		o.Fields = append(o.Fields, ast.RecordField{Base: ast.Base{P: start}, Names: names, Type: ft})
		p.match(lexer.TokSemicolon)
	}
	p.expect(lexer.TokKeyword, "end")
	return o
}

// parseHelperType parses the body of `record helper for Base` / `class helper
// for Base` (the keyword and `helper` were already consumed reaching here for
// the keyword; `helper` is the current token). It carries only method
// signatures; the bodies are separate Type.Method declarations.
func (p *Parser) parseHelperType(start ast.Pos, isClass bool) ast.TypeExpr {
	p.advance() // helper
	p.expect(lexer.TokKeyword, "for")
	target := p.expect(lexer.TokIdent).Text
	o := &ast.ObjectType{Base: ast.Base{P: start}, IsClass: isClass, HelperFor: target}
	for {
		if p.is("end") || p.cur().Kind == lexer.TokEOF {
			break
		}
		if p.is("private") || p.is("public") || p.is("protected") || p.is("published") {
			p.advance()
			continue
		}
		if p.is("procedure") || p.is("function") || p.is("constructor") || p.is("destructor") {
			if pr := p.parseMethodSig(); pr != nil {
				o.Methods = append(o.Methods, *pr)
			}
			for p.cur().Kind == lexer.TokSemicolon {
				p.advance()
			}
			continue
		}
		p.advance() // skip anything else defensively
	}
	p.expect(lexer.TokKeyword, "end")
	return o
}

// parsePropertyDef parses `property Name: Type [read F] [write F];`. The
// read/write specifiers are context-sensitive identifiers.
func (p *Parser) parsePropertyDef() ast.PropertyDef {
	start := p.curPos()
	p.advance() // property
	pd := ast.PropertyDef{Base: ast.Base{P: start}, Name: p.expect(lexer.TokIdent).Text}
	if p.match(lexer.TokColon) {
		p.parseType() // property type (not needed for the field mapping)
	}
	for p.cur().Kind == lexer.TokIdent {
		switch strings.ToLower(p.cur().Text) {
		case "read":
			p.advance()
			pd.Read = p.expect(lexer.TokIdent).Text
		case "write":
			p.advance()
			pd.Write = p.expect(lexer.TokIdent).Text
		default:
			return pd
		}
	}
	return pd
}

// parseMethodSig parses an object method *signature* (no body): the keyword,
// name, parameters, optional result and any trailing directives up to ';'.
func (p *Parser) parseMethodSig() *ast.ProcDecl {
	start := p.curPos()
	pr := &ast.ProcDecl{Base: ast.Base{P: start}}
	switch {
	case p.is("constructor"):
		pr.IsConstructor = true
		p.advance()
	case p.is("destructor"):
		pr.IsDestructor = true
		p.advance()
	case p.is("function"):
		pr.IsFunc = true
		p.advance()
	default: // procedure
		p.advance()
	}
	id := p.expect(lexer.TokIdent)
	pr.Name = id.Text
	if p.match(lexer.TokLParen) {
		for {
			if p.check(lexer.TokRParen) {
				break
			}
			pr.Params = append(pr.Params, p.parseParam())
			if !p.match(lexer.TokSemicolon) {
				break
			}
		}
		p.expect(lexer.TokRParen)
	}
	if pr.IsFunc && p.match(lexer.TokColon) {
		tr := p.parseType()
		if tref, ok := tr.(*ast.TypeRef); ok {
			pr.Result = tref
		} else {
			pr.Result = &ast.TypeRef{Base: ast.Base{P: start}, Name: tr.String()}
		}
	}
	p.match(lexer.TokSemicolon)
	// Method directives (virtual, abstract, override, ...).
	for p.cur().Kind == lexer.TokKeyword {
		switch p.cur().Lower {
		case "virtual", "abstract", "override", "static", "far", "near", "inline":
			p.advance()
			p.match(lexer.TokSemicolon)
		default:
			return pr
		}
	}
	return pr
}

func (p *Parser) parseProcType() ast.TypeExpr {
	start := p.curPos()
	isFunc := p.is("function")
	p.advance()
	pt := &ast.ProcType{Base: ast.Base{P: start}, IsFunc: isFunc}
	if p.match(lexer.TokLParen) {
		for {
			if p.match(lexer.TokRParen) {
				break
			}
			names := []string{}
			for {
				id := p.expect(lexer.TokIdent)
				names = append(names, id.Text)
				if !p.match(lexer.TokComma) {
					break
				}
			}
			_ = names
			p.match(lexer.TokColon)
			pt.Params = append(pt.Params, ast.Param{Base: ast.Base{P: start}})
			// Best-effort: skip tokens until ',' or ')'
			for !p.check(lexer.TokComma) && !p.check(lexer.TokRParen) && p.cur().Kind != lexer.TokEOF {
				p.advance()
			}
			if !p.match(lexer.TokComma) {
				p.expect(lexer.TokRParen)
				break
			}
		}
	}
	if isFunc && p.match(lexer.TokColon) {
		id := p.expect(lexer.TokIdent)
		pt.Result = &ast.TypeRef{Base: ast.Base{P: start}, Name: id.Text, Lower: strings.ToLower(id.Text)}
	}
	return pt
}

// parseAnonFunc parses an anonymous method expression:
// `procedure(params) begin ... end` or `function(params): T begin ... end`.
func (p *Parser) parseAnonFunc() ast.Expr {
	start := p.curPos()
	isFunc := p.is("function")
	p.advance() // procedure | function
	af := &ast.AnonFunc{Base: ast.Base{P: start}, IsFunc: isFunc}
	if p.match(lexer.TokLParen) {
		for {
			if p.check(lexer.TokRParen) {
				break
			}
			af.Params = append(af.Params, p.parseParam())
			if !p.match(lexer.TokSemicolon) {
				break
			}
		}
		p.expect(lexer.TokRParen)
	}
	if isFunc && p.match(lexer.TokColon) {
		if tr, ok := p.parseType().(*ast.TypeRef); ok {
			af.Result = tr
		}
	}
	af.Body = p.parseBlockBody()
	return af
}

func (p *Parser) parseProcDecl(allowForward bool) ast.Decl {
	start := p.curPos()
	isMethod := false
	isCtor := false
	isDtor := false
	isFunc := false
	switch {
	case p.is("constructor"):
		isCtor = true
		p.advance()
	case p.is("destructor"):
		isDtor = true
		p.advance()
	case p.is("procedure"):
		p.advance()
	case p.is("function"):
		isFunc = true
		p.advance()
	default:
		p.errf("expected procedure/function/constructor/destructor")
		p.advance()
		return nil
	}
	id := p.expect(lexer.TokIdent)
	// Qualified method name: TypeName.MethodName (e.g. TFoo.DoIt).
	for p.cur().Kind == lexer.TokPeriod {
		p.advance()
		sub := p.expect(lexer.TokIdent)
		id.Text = id.Text + "." + sub.Text
	}
	pr := &ast.ProcDecl{Base: ast.Base{P: start}, Name: id.Text, IsMethod: isMethod, IsConstructor: isCtor, IsDestructor: isDtor, IsFunc: isFunc}
	if p.check(lexer.TokOp, "<") { // generic routine: function Max<T>(...)
		pr.TypeParams = p.parseTypeParams()
	}
	if p.match(lexer.TokLParen) {
		for {
			if p.check(lexer.TokRParen) {
				break
			}
			par := p.parseParam()
			pr.Params = append(pr.Params, par)
			// Parameter groups are separated by ';' (names within a group by
			// ',', which parseParam consumes). e.g. (var a: Integer; b: Char).
			if !p.match(lexer.TokSemicolon) {
				break
			}
		}
		p.expect(lexer.TokRParen)
	}
	if pr.IsFunc && p.match(lexer.TokColon) {
		tr := p.parseType()
		if tref, ok := tr.(*ast.TypeRef); ok {
			pr.Result = tref
		} else {
			pr.Result = &ast.TypeRef{Base: ast.Base{P: start}, Name: tr.String()}
		}
	}
	// Consume optional semicolons before directive chain (TP7 allows `;` before
	// forward, external, etc.).
	for p.cur().Kind == lexer.TokSemicolon {
		p.advance()
	}
	// Parse directive chain
	for p.cur().Kind == lexer.TokKeyword {
		switch p.cur().Lower {
		case "forward":
			if allowForward {
				pr.Forward = true
			}
			p.advance()
			if !pr.Forward {
				p.errf("forward not allowed here")
			}
			return pr
		case "external":
			p.advance()
			if p.cur().Kind == lexer.TokString {
				pr.External = p.cur().Str
				p.advance()
			}
			return pr
		case "far":
			pr.Far = true
			p.advance()
		case "near":
			pr.Near = true
			p.advance()
		case "interrupt":
			pr.Interrupt = true
			p.advance()
		case "assembler":
			pr.Assembler = true
			p.advance()
		case "virtual":
			p.advance()
		case "override":
			p.advance()
		case "abstract":
			p.advance()
		case "static":
			p.advance()
		case "inline":
			p.advance()
			p.expect(lexer.TokLParen)
			for p.cur().Kind != lexer.TokRParen && p.cur().Kind != lexer.TokEOF {
				s := p.cur().Text
				p.advance()
				pr.Inline = append(pr.Inline, s)
				if !p.match(lexer.TokComma) {
					break
				}
			}
			p.expect(lexer.TokRParen)
		default:
			goto done
		}
	}
done:
	if pr.Forward || pr.External != "" {
		return pr
	}
	// In an interface section (allowForward == false) procedures are
	// signatures only — the body lives in the implementation section.
	if !allowForward {
		return pr
	}
	// body
	if p.cur().Kind == lexer.TokSemicolon {
		// optional ; before body
		p.advance()
	}
	switch p.cur().Kind {
	case lexer.TokKeyword:
		switch p.cur().Lower {
		case "const", "var", "type", "label":
			blk := p.parseBlock(false)
			pr.Nested = blk
			// parseBlock stops at `begin`; parse the routine body here.
			if p.is("begin") {
				pr.Body = p.parseBlockBody()
			} else {
				pr.Body = blk.Body
			}
			p.expect(lexer.TokSemicolon)
		case "begin":
			pr.Body = p.parseBlockBody()
			p.expect(lexer.TokSemicolon)
		case "procedure", "function", "constructor", "destructor":
			// procedure body starts with nested procs.
			blk := p.parseBlock(false)
			pr.Nested = blk
			pr.Body = blk.Body
			if p.cur().Kind == lexer.TokKeyword && p.cur().Lower == "begin" {
				body := p.parseBlockBody()
				if pr.Body == nil {
					pr.Body = body
				} else {
					pr.Body.Stmts = append(pr.Body.Stmts, body.Stmts...)
				}
				p.match(lexer.TokSemicolon)
			} else {
				p.match(lexer.TokSemicolon)
			}
		case "asm":
			pr.Body = &ast.BlockBody{Base: ast.Base{P: p.curPos()}, Stmts: []ast.Stmt{&ast.AsmStmt{Base: ast.Base{P: p.curPos()}}}}
			p.advance()
			for !(p.cur().Kind == lexer.TokKeyword && p.cur().Lower == "end") && p.cur().Kind != lexer.TokEOF {
				p.advance()
			}
			p.expect(lexer.TokKeyword, "end")
			p.expect(lexer.TokSemicolon)
		default:
			// assume external/no body
			return pr
		}
	}
	return pr
}

// operatorNames maps an operator symbol to a name fragment for the generated
// IR function. Used for both the unique function name and dispatch.
var operatorNames = map[string]string{
	"+": "add", "-": "sub", "*": "mul", "/": "div",
	"=": "eq", "<>": "ne", "<": "lt", ">": "gt", "<=": "le", ">=": "ge",
}

// parseOperatorDecl parses an FPC-style operator overload:
// `operator + (a, b: TVec): TVec; begin Result := ...; end;`.
func (p *Parser) parseOperatorDecl() ast.Decl {
	start := p.curPos()
	p.advance() // operator
	sym := p.cur().Text
	p.advance() // the operator symbol token
	pr := &ast.ProcDecl{Base: ast.Base{P: start}, IsFunc: true, OperatorSym: sym}
	if p.match(lexer.TokLParen) {
		for {
			if p.check(lexer.TokRParen) {
				break
			}
			pr.Params = append(pr.Params, p.parseParam())
			if !p.match(lexer.TokSemicolon) {
				break
			}
		}
		p.expect(lexer.TokRParen)
	}
	// Optional FPC named result: `operator + (a, b: T) r: T`.
	if p.cur().Kind == lexer.TokIdent {
		p.advance()
	}
	if p.match(lexer.TokColon) {
		if tr, ok := p.parseType().(*ast.TypeRef); ok {
			pr.Result = tr
		}
	}
	// Unique IR name from the operator and its operand type names so distinct
	// overloads do not collide.
	frag := operatorNames[sym]
	if frag == "" {
		frag = "op"
	}
	pr.Name = "$op_" + frag + "_" + operandTypeName(pr.Params, 0) + "_" + operandTypeName(pr.Params, 1)
	for p.cur().Kind == lexer.TokSemicolon {
		p.advance()
	}
	if p.is("begin") {
		pr.Body = p.parseBlockBody()
	}
	return pr
}

// operandTypeName returns a lowercase type name for the operator's i-th operand
// (grouped parameters share a type).
func operandTypeName(params []ast.Param, i int) string {
	if len(params) == 0 {
		return ""
	}
	idx := i
	if idx >= len(params) {
		idx = len(params) - 1
	}
	if tr, ok := params[idx].Type.(*ast.TypeRef); ok {
		return strings.ToLower(tr.Name)
	}
	if params[idx].Type != nil {
		return strings.ToLower(params[idx].Type.String())
	}
	return ""
}

func (p *Parser) parseParam() ast.Param {
	start := p.curPos()
	par := ast.Param{Base: ast.Base{P: start}}
	if p.cur().Kind == lexer.TokKeyword && p.cur().Lower == "var" {
		par.Var = true
		p.advance()
	}
	if p.cur().Kind == lexer.TokKeyword && p.cur().Lower == "const" {
		par.Const = true
		p.advance()
	}
	if p.cur().Kind == lexer.TokKeyword && p.cur().Lower == "out" {
		p.advance()
	}
	names := []string{}
	for {
		id := p.expect(lexer.TokIdent)
		names = append(names, id.Text)
		if !p.match(lexer.TokComma) {
			break
		}
	}
	par.Names = names
	if p.match(lexer.TokColon) {
		par.Type = p.parseType()
	}
	return par
}

func (p *Parser) parseBlockBody() *ast.BlockBody {
	start := p.curPos()
	if !p.is("begin") {
		p.errf("expected BEGIN")
		return &ast.BlockBody{Base: ast.Base{P: start}}
	}
	p.advance()
	bb := &ast.BlockBody{Base: ast.Base{P: start}}
	for {
		if p.is("end") {
			p.advance()
			return bb
		}
		if p.cur().Kind == lexer.TokEOF {
			p.errf("unexpected EOF inside BEGIN..END")
			return bb
		}
		s := p.parseStmt()
		if s != nil {
			bb.Stmts = append(bb.Stmts, s)
		}
		if !p.match(lexer.TokSemicolon) {
			if p.is("end") {
				continue
			}
			if p.cur().Kind == lexer.TokEOF {
				return bb
			}
			p.errf("expected ';' or 'end'")
		}
	}
}

func (p *Parser) parseStmt() ast.Stmt {
	start := p.curPos()
	t := p.cur()
	if t.Kind == lexer.TokSemicolon {
		return nil
	}
	switch t.Kind {
	case lexer.TokKeyword:
		switch t.Lower {
		case "begin":
			bb := p.parseBlockBody()
			return &ast.CompoundStmt{Base: ast.Base{P: ast.Pos{File: p.file, Line: t.Line, Col: t.Col}}, Stmts: bb.Stmts}
		case "if":
			return p.parseIf()
		case "while":
			return p.parseWhile()
		case "repeat":
			return p.parseRepeat()
		case "for":
			return p.parseFor()
		case "case":
			return p.parseCase()
		case "with":
			return p.parseWith()
		case "try":
			return p.parseTry()
		case "raise":
			p.advance()
			var e ast.Expr
			if !p.is(";") && !p.is("end") && !p.is("except") && !p.is("finally") && p.cur().Kind != lexer.TokSemicolon {
				e = p.parseExpr()
			}
			return &ast.RaiseStmt{Base: ast.Base{P: start}, Expr: e}
		case "goto":
			p.advance()
			n := p.expect(lexer.TokInt)
			return &ast.GotoStmt{Base: ast.Base{P: start}, Label: int(n.Int)}
		case "break":
			p.advance()
			return &ast.BreakStmt{Base: ast.Base{P: start}}
		case "continue":
			p.advance()
			return &ast.ContinueStmt{Base: ast.Base{P: start}}
		case "exit":
			p.advance()
			return &ast.ExitStmt{Base: ast.Base{P: start}}
		case "halt":
			p.advance()
			var code ast.Expr
			if p.match(lexer.TokLParen) {
				code = p.parseExpr()
				p.expect(lexer.TokRParen)
			}
			return &ast.HaltStmt{Base: ast.Base{P: start}, Code: code}
		case "asm":
			p.advance()
			for !(p.cur().Kind == lexer.TokKeyword && p.cur().Lower == "end") && p.cur().Kind != lexer.TokEOF {
				p.advance()
			}
			p.expect(lexer.TokKeyword, "end")
			return &ast.AsmStmt{Base: ast.Base{P: start}}
		case "inherited":
			p.advance()
			stmt := &ast.InheritedStmt{Base: ast.Base{P: start}}
			if p.cur().Kind == lexer.TokIdent {
				c := p.parseCall()
				stmt.Call = &c
			}
			return stmt
		}
		// label statement? integers as statement
		// TP7 allows integer as label-start of statement
	}
	// Possibly a label: integer followed by ':' then statement
	if t.Kind == lexer.TokInt && p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Kind == lexer.TokColon {
		labelTok := p.advance()
		p.advance() // ':'
		if p.is("end") || p.is(";") || p.cur().Kind == lexer.TokEOF {
			return &ast.LabelStmt{Base: ast.Base{P: ast.Pos{File: p.file, Line: labelTok.Line, Col: labelTok.Col}}, Label: int(labelTok.Int)}
		}
		sub := p.parseStmt()
		if sub == nil {
			return &ast.LabelStmt{Base: ast.Base{P: ast.Pos{File: p.file, Line: labelTok.Line, Col: labelTok.Col}}, Label: int(labelTok.Int)}
		}
		return &ast.CompoundStmt{Base: ast.Base{P: ast.Pos{File: p.file, Line: t.Line, Col: t.Col}}, Stmts: []ast.Stmt{
			&ast.LabelStmt{Base: ast.Base{P: ast.Pos{File: p.file, Line: labelTok.Line, Col: labelTok.Col}}, Label: int(labelTok.Int)},
			sub,
		}}
	}
	// Try assignment or call
	lhs := p.parseLValue()
	if p.cur().Kind == lexer.TokAssign {
		p.advance()
		rhs := p.parseExpr()
		return &ast.AssignStmt{Base: ast.Base{P: start}, Dest: lhs, Expr: rhs}
	}
	// It must be a call (procedure call as statement)
	if ce, ok := lhs.(*ast.CallExpr); ok {
		return &ast.CallStmt{Base: ast.Base{P: start}, Call: *ce}
	}
	// Field access as a statement: method call on object.
	if fe, ok := lhs.(*ast.FieldExpr); ok {
		return &ast.CallStmt{Base: ast.Base{P: start}, Call: ast.CallExpr{Base: ast.Base{P: start}, Func: fe}}
	}
	// If it parses as a single ident like "Foo" (no parens), treat as call.
	if id, ok := lhs.(*ast.Ident); ok {
		return &ast.CallStmt{Base: ast.Base{P: start}, Call: ast.CallExpr{Base: ast.Base{P: start}, Func: id}}
	}
	p.errf("expected statement, got %v", describeTok(t))
	p.advance()
	return nil
}

func (p *Parser) parseIf() ast.Stmt {
	start := p.curPos()
	p.advance() // if
	cond := p.parseExpr()
	p.expect(lexer.TokKeyword, "then")
	then := p.parseStmt()
	var els ast.Stmt
	if p.is("else") {
		p.advance()
		els = p.parseStmt()
	}
	return &ast.IfStmt{Base: ast.Base{P: start}, Cond: cond, Then: then, Else: els}
}

func (p *Parser) parseWhile() ast.Stmt {
	start := p.curPos()
	p.advance() // while
	cond := p.parseExpr()
	p.expect(lexer.TokKeyword, "do")
	body := p.parseStmt()
	return &ast.WhileStmt{Base: ast.Base{P: start}, Cond: cond, Body: body}
}

func (p *Parser) parseRepeat() ast.Stmt {
	start := p.curPos()
	p.advance() // repeat
	// A repeat..until body is a statement sequence (no begin/end needed).
	var stmts []ast.Stmt
	for !p.is("until") && p.cur().Kind != lexer.TokEOF {
		if s := p.parseStmt(); s != nil {
			stmts = append(stmts, s)
		}
		if !p.match(lexer.TokSemicolon) {
			break
		}
	}
	body := ast.Stmt(&ast.CompoundStmt{Base: ast.Base{P: start}, Stmts: stmts})
	if p.is("until") {
		p.advance()
		cond := p.parseExpr()
		return &ast.RepeatStmt{Base: ast.Base{P: start}, Cond: cond, Body: body}
	}
	p.errf("expected 'until'")
	return &ast.RepeatStmt{Base: ast.Base{P: start}, Body: body}
}

func (p *Parser) parseFor() ast.Stmt {
	start := p.curPos()
	p.advance() // for
	id := p.expect(lexer.TokIdent)
	// for x in collection do ...
	if p.is("in") {
		p.advance()
		coll := p.parseExpr()
		p.expect(lexer.TokKeyword, "do")
		body := p.parseStmt()
		return &ast.ForInStmt{Base: ast.Base{P: start}, Var: id.Text, Coll: coll, Body: body}
	}
	p.expect(lexer.TokAssign)
	lo := p.parseExpr()
	down := false
	if p.is("to") {
		p.advance()
	} else if p.is("downto") {
		p.advance()
		down = true
	} else {
		p.errf("expected to/downto")
	}
	hi := p.parseExpr()
	p.expect(lexer.TokKeyword, "do")
	body := p.parseStmt()
	return &ast.ForStmt{Base: ast.Base{P: start}, Var: id.Text, Lo: lo, Hi: hi, Down: down, Body: body}
}

func (p *Parser) parseCase() ast.Stmt {
	start := p.curPos()
	p.advance() // case
	expr := p.parseExpr()
	p.expect(lexer.TokKeyword, "of")
	cs := &ast.CaseStmt{Base: ast.Base{P: start}, Expr: expr}
	for {
		if p.is("end") {
			p.advance()
			return cs
		}
		if p.is("else") {
			p.advance()
			cs.Else = p.parseStmt()
			p.match(lexer.TokSemicolon)
			continue
		}
		vals := []ast.Expr{}
		for {
			v := p.parseExpr()
			vals = append(vals, v)
			if !p.match(lexer.TokComma) {
				break
			}
		}
		p.expect(lexer.TokColon)
		body := p.parseStmt()
		cs.Cases = append(cs.Cases, ast.CaseBranch{Base: ast.Base{P: start}, Values: vals, Body: body})
		p.match(lexer.TokSemicolon)
	}
}

// parseStmtSeqUntil parses `;`-separated statements until one of the given
// terminator keywords (or EOF) is reached.
func (p *Parser) parseStmtSeqUntil(terms ...string) []ast.Stmt {
	var stmts []ast.Stmt
	isTerm := func() bool {
		for _, t := range terms {
			if p.is(t) {
				return true
			}
		}
		return false
	}
	for !isTerm() && p.cur().Kind != lexer.TokEOF {
		if s := p.parseStmt(); s != nil {
			stmts = append(stmts, s)
		}
		if !p.match(lexer.TokSemicolon) {
			break
		}
	}
	return stmts
}

func (p *Parser) parseTry() ast.Stmt {
	start := p.curPos()
	p.advance() // try
	t := &ast.TryStmt{Base: ast.Base{P: start}, Body: p.parseStmtSeqUntil("except", "finally")}
	if p.is("finally") {
		p.advance()
		t.Finally = p.parseStmtSeqUntil("end")
	} else if p.is("except") {
		p.advance()
		t.Except = p.parseStmtSeqUntil("end")
	} else {
		p.errf("expected 'except' or 'finally'")
	}
	p.expect(lexer.TokKeyword, "end")
	return t
}

func (p *Parser) parseWith() ast.Stmt {
	start := p.curPos()
	p.advance() // with
	recs := []ast.Expr{p.parseExpr()}
	for p.match(lexer.TokComma) {
		recs = append(recs, p.parseExpr())
	}
	p.expect(lexer.TokKeyword, "do")
	body := p.parseStmt()
	// `with a, b do S` is `with a do with b do S` (a outermost).
	for i := len(recs) - 1; i >= 0; i-- {
		body = &ast.WithStmt{Base: ast.Base{P: start}, Rec: recs[i], Body: body}
	}
	return body
}

func (p *Parser) parseLValue() ast.Expr {
	// LValue: primary chain (postfix supported)
	return p.parsePostfix()
}

func (p *Parser) parseCall() ast.CallExpr {
	start := p.curPos()
	ce := ast.CallExpr{Base: ast.Base{P: start}}
	ce.Func = p.parsePrimary()
	if p.match(lexer.TokLParen) {
		for {
			if p.match(lexer.TokRParen) {
				break
			}
			ce.Args = append(ce.Args, p.parseExpr())
			if !p.match(lexer.TokComma) {
				p.expect(lexer.TokRParen)
				break
			}
		}
	}
	return ce
}

// Expression parser using Pratt-style precedence climbing.

type bindingPower struct {
	left, right int
}

var binopBP = map[string]bindingPower{
	"or":  {1, 2},
	"xor": {3, 4},
	"and": {5, 6},
	"=":   {9, 10},
	"<>":  {9, 10},
	"<":   {9, 10},
	"<=":  {9, 10},
	">":   {9, 10},
	">=":  {9, 10},
	"in":  {9, 10},
	"+":   {11, 12},
	"-":   {11, 12},
	"*":   {13, 14},
	"/":   {13, 14},
	"div": {13, 14},
	"mod": {13, 14},
	"shl": {15, 16},
	"shr": {15, 16},
}

func (p *Parser) parseExpr() ast.Expr {
	return p.parseExprBP(0)
}

func (p *Parser) parseExprBP(minBP int) ast.Expr {
	left := p.parseUnary()
	for {
		t := p.cur()
		if t.Kind == lexer.TokRange {
			p.advance()
			right := p.parseExprBP(minBP + 2)
			return &ast.RangeExpr{Base: ast.Base{P: left.Pos()}, Lo: left, Hi: right}
		}
		op := ""
		switch t.Kind {
		case lexer.TokKeyword:
			switch t.Lower {
			case "or", "xor", "and", "in", "div", "mod", "shl", "shr":
				op = t.Lower
			}
		case lexer.TokOp, lexer.TokEqual:
			op = t.Text
		}
		if op == "" {
			return left
		}
		bp, ok := binopBP[op]
		if !ok || bp.left < minBP {
			return left
		}
		p.advance()
		right := p.parseExprBP(bp.right)
		if op == "in" {
			left = &ast.InExpr{Base: ast.Base{P: left.Pos()}, Left: left, Right: right}
		} else {
			left = &ast.BinaryExpr{Base: ast.Base{P: left.Pos()}, Op: op, Left: left, Right: right}
		}
	}
}

func (p *Parser) parseUnary() ast.Expr {
	t := p.cur()
	if t.Kind == lexer.TokOp && (t.Text == "+" || t.Text == "-") {
		p.advance()
		e := p.parseUnary()
		return &ast.UnaryExpr{Base: ast.Base{P: ast.Pos{File: p.file, Line: t.Line, Col: t.Col}}, Op: t.Text, Expr: e}
	}
	if t.Kind == lexer.TokKeyword && t.Lower == "not" {
		p.advance()
		e := p.parseUnary()
		return &ast.UnaryExpr{Base: ast.Base{P: ast.Pos{File: p.file, Line: t.Line, Col: t.Col}}, Op: "not", Expr: e}
	}
	if t.Kind == lexer.TokAt {
		p.advance()
		e := p.parseUnary()
		return &ast.AtExpr{Base: ast.Base{P: ast.Pos{File: p.file, Line: t.Line, Col: t.Col}}, Expr: e}
	}
	return p.parsePostfix()
}

func (p *Parser) parsePostfix() ast.Expr {
	e := p.parsePrimary()
	for {
		t := p.cur()
		switch t.Kind {
		case lexer.TokCaret:
			p.advance()
			e = &ast.CaretExpr{Base: ast.Base{P: e.Pos()}, Expr: e}
		case lexer.TokLBracket:
			p.advance()
			idx := p.parseExpr()
			p.expect(lexer.TokRBracket)
			e = &ast.IndexExpr{Base: ast.Base{P: e.Pos()}, Expr: e, Index: idx}
		case lexer.TokPeriod:
			p.advance()
			id := p.expect(lexer.TokIdent)
			e = &ast.FieldExpr{Base: ast.Base{P: e.Pos()}, Expr: e, Field: id.Text}
		case lexer.TokLParen:
			// Detect typecast: e is a type-name ident, args must be a single expr
			// not surrounded by commas (i.e. a cast form). e.g. Integer(P).
			if id, ok := e.(*ast.Ident); ok && isTypeName(id.Lower) {
				p.advance() // '('
				inner := p.parseExpr()
				p.expect(lexer.TokRParen)
				e = &ast.TypeCastExpr{Base: ast.Base{P: e.Pos()}, Type: &ast.TypeRef{Base: ast.Base{P: id.Pos()}, Name: id.Name, Lower: id.Lower}, Expr: inner}
				continue
			}
			ce := ast.CallExpr{Base: ast.Base{P: e.Pos()}, Func: e}
			p.advance()
			// Write/WriteLn/Str arguments may carry field formatting (x:w:d).
			fmtCall := false
			if id, ok := e.(*ast.Ident); ok {
				switch id.Lower {
				case "write", "writeln":
					fmtCall = true
				}
			}
			if !p.check(lexer.TokRParen) {
				for {
					arg := p.parseExpr()
					if fmtCall && p.cur().Kind == lexer.TokColon {
						p.advance()
						wa := &ast.WriteArg{Base: ast.Base{P: arg.Pos()}, Value: arg, Width: p.parseExpr()}
						if p.cur().Kind == lexer.TokColon {
							p.advance()
							wa.Decimals = p.parseExpr()
						}
						arg = wa
					}
					ce.Args = append(ce.Args, arg)
					if !p.match(lexer.TokComma) {
						break
					}
				}
			}
			p.expect(lexer.TokRParen)
			e = &ce
		default:
			return e
		}
	}
}

func (p *Parser) parseIdent() ast.Expr {
	t := p.expect(lexer.TokIdent)
	return &ast.Ident{Base: ast.Base{P: ast.Pos{File: p.file, Line: t.Line, Col: t.Col}}, Name: t.Text, Lower: strings.ToLower(t.Text)}
}

func (p *Parser) parsePrimary() ast.Expr {
	t := p.cur()
	switch t.Kind {
	case lexer.TokInt:
		p.advance()
		return &ast.IntLit{Base: ast.Base{P: ast.Pos{File: p.file, Line: t.Line, Col: t.Col}}, Value: t.Int}
	case lexer.TokHex:
		p.advance()
		return &ast.IntLit{Base: ast.Base{P: ast.Pos{File: p.file, Line: t.Line, Col: t.Col}}, Value: int64(t.Hex), Hex: true}
	case lexer.TokReal:
		p.advance()
		return &ast.RealLit{Base: ast.Base{P: ast.Pos{File: p.file, Line: t.Line, Col: t.Col}}, Value: t.Real}
	case lexer.TokString:
		p.advance()
		// If followed by #, fold the integer into the string.
		s := t.Str
		for p.cur().Kind == lexer.TokInt && p.pos > 0 && p.tokens[p.pos-1].Kind == lexer.TokString {
			// Edge: integer token (e.g. #13) is consumed inline.
			s += string(byte(p.cur().Int))
			p.advance()
		}
		return &ast.StringLit{Base: ast.Base{P: ast.Pos{File: p.file, Line: t.Line, Col: t.Col}}, Value: s}
	case lexer.TokIdent:
		return p.parseIdent()
	case lexer.TokKeyword:
		// `nil` is the only keyword that is also a valid expression in
		// primary position: it represents the null pointer. Every other
		// keyword is a parse error in this position.
		if t.Lower == "nil" {
			p.advance()
			return &ast.Ident{Base: ast.Base{P: ast.Pos{File: p.file, Line: t.Line, Col: t.Col}}, Name: "nil", Lower: "nil"}
		}
		// Anonymous method: `procedure(...) begin ... end` / `function(...): T begin ... end`.
		if t.Lower == "procedure" || t.Lower == "function" {
			return p.parseAnonFunc()
		}
		p.errf("unexpected keyword in expression: %s", t.Lower)
		return &ast.IntLit{Base: ast.Base{P: ast.Pos{File: p.file, Line: t.Line, Col: t.Col}}, Value: 0}
	case lexer.TokLBracket:
		p.advance()
		if p.match(lexer.TokRBracket) {
			return &ast.SetExpr{Base: ast.Base{P: ast.Pos{File: p.file, Line: t.Line, Col: t.Col}}}
		}
		se := &ast.SetExpr{Base: ast.Base{P: ast.Pos{File: p.file, Line: t.Line, Col: t.Col}}}
		for {
			lo := p.parseExpr()
			hi := ast.Expr(nil)
			if p.match(lexer.TokRange) {
				hi = p.parseExpr()
			}
			se.Elements = append(se.Elements, ast.SetElement{Base: ast.Base{P: lo.Pos()}, Lo: lo, Hi: hi})
			if !p.match(lexer.TokComma) {
				break
			}
		}
		p.expect(lexer.TokRBracket)
		return se
	case lexer.TokLParen:
		p.advance()
		e := p.parseExpr()
		p.expect(lexer.TokRParen)
		return e
	case lexer.TokCaret:
		// typecast via leading ^: not used as primary; only as postfix
		p.advance()
		inner := p.parseUnary()
		return &ast.CaretExpr{Base: ast.Base{P: ast.Pos{File: p.file, Line: t.Line, Col: t.Col}}, Expr: inner}
	}
	p.errf("unexpected token in expression: %v", describeTok(t))
	p.advance()
	return &ast.IntLit{Base: ast.Base{P: ast.Pos{File: p.file, Line: t.Line, Col: t.Col}}, Value: 0}
}

func isTypeName(low string) bool {
	switch low {
	case "integer", "longint", "shortint", "byte", "word", "boolean", "char",
		"real", "single", "double", "extended", "comp",
		"string", "text", "file":
		return true
	}
	return false
}

// Helpers for AST.Pos construction.
