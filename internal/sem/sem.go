// Package sem implements the BPGo semantic analyzer and type system.
// It walks the AST produced by the parser, resolves identifiers, builds
// the symbol table and computes type information used by code
// generation. The semantic pass does not emit code; it only validates
// the program and decorates nodes with type/layout information.
package sem

import (
	"fmt"
	"sort"
	"strings"

	"github.com/arturoeanton/go-turbo-pascal/internal/ast"
	"github.com/arturoeanton/go-turbo-pascal/internal/diagnostics"
)

// Basic type kinds. These match the TP7 / BP7 categories.
type BasicKind int

const (
	BKUnknown BasicKind = iota
	BKShortInt
	BKByte
	BKInteger
	BKWord
	BKLongInt
	BKBoolean
	BKChar
	BKReal
	BKSingle
	BKDouble
	BKExtended
	BKComp
	BKString
	BKPChar
	BKPointer
	BKText
)

func (b BasicKind) String() string {
	switch b {
	case BKShortInt:
		return "ShortInt"
	case BKByte:
		return "Byte"
	case BKInteger:
		return "Integer"
	case BKWord:
		return "Word"
	case BKLongInt:
		return "LongInt"
	case BKBoolean:
		return "Boolean"
	case BKChar:
		return "Char"
	case BKReal:
		return "Real"
	case BKSingle:
		return "Single"
	case BKDouble:
		return "Double"
	case BKExtended:
		return "Extended"
	case BKComp:
		return "Comp"
	case BKString:
		return "String"
	case BKPChar:
		return "PChar"
	case BKPointer:
		return "Pointer"
	case BKText:
		return "Text"
	}
	return "Unknown"
}

// Size returns the byte size for a basic type, matching TP7/BP7.
func (b BasicKind) Size() int64 {
	switch b {
	case BKShortInt, BKByte, BKChar, BKBoolean:
		return 1
	case BKInteger, BKWord:
		return 2
	case BKLongInt, BKSingle:
		return 4
	case BKReal:
		return 6
	case BKDouble:
		return 8
	case BKExtended:
		return 10
	case BKComp:
		return 8
	case BKPChar, BKPointer:
		return 4 // far pointer segment:offset
	case BKString:
		return 256 // default String[255] + length byte
	case BKText:
		return 256 // TTextRec-ish opaque record
	}
	return 0
}

// Type represents any Pascal type. The semantic analyzer maintains a
// graph of Type values; equal() compares identity.
type Type struct {
	Kind  BasicKind
	Name  string
	Lower string
	// For complex types:
	Element   *Type
	Key       *Type // map key or set element
	Index     []*Type
	IsPacked  bool
	IsOpen    bool // open array
	IsText    bool
	IsFile    bool
	IsPointer bool
	IsRecord  bool
	IsObject  bool
	IsArray   bool
	IsSet     bool
	IsRange   bool
	IsEnum    bool
	IsString  bool
	IsProc    bool
	IsFunc    bool
	Fields    []*Field
	Methods   []*Method
	Parent    *Type
	VMT       []string
	Lo, Hi    int64
	EnumVals  []string
	Len       int64 // string length (0 means default 255)
	Size      int64
	Align     int64
}

func (t *Type) String() string {
	if t == nil {
		return "<nil>"
	}
	if t.Name != "" {
		return t.Name
	}
	return t.Kind.String()
}

func (t *Type) Equals(o *Type) bool {
	if t == o {
		return true
	}
	if t == nil || o == nil {
		return false
	}
	if t.Kind != o.Kind {
		return false
	}
	// Both anonymous: equal if same kind.
	if t.Name == "" && o.Name == "" {
		return true
	}
	// If both have names (named types), compare by lower-case name.
	if t.Name != "" && o.Name != "" {
		return strings.EqualFold(t.Name, o.Name)
	}
	// One has name, one is anonymous: equality holds if the kind matches and
	// the anonymous one represents the corresponding built-in.
	if t.Name != "" {
		return basicTypeForName(t.Name) == o.Kind
	}
	return basicTypeForName(o.Name) == t.Kind
}

// basicTypeForName returns the BasicKind corresponding to the canonical
// TP7/BP7 name, or BKUnknown for non-basic names.
func basicTypeForName(name string) BasicKind {
	switch strings.ToLower(name) {
	case "shortint":
		return BKShortInt
	case "byte":
		return BKByte
	case "integer":
		return BKInteger
	case "word":
		return BKWord
	case "longint":
		return BKLongInt
	case "boolean":
		return BKBoolean
	case "char":
		return BKChar
	case "real":
		return BKReal
	case "single":
		return BKSingle
	case "double":
		return BKDouble
	case "extended":
		return BKExtended
	case "comp":
		return BKComp
	case "string":
		return BKString
	case "pchar":
		return BKPChar
	case "pointer":
		return BKPointer
	case "text":
		return BKText
	}
	return BKUnknown
}

type Field struct {
	Name string
	Off  int64
	Type *Type
	// For variant records:
	Tag    *Type
	Cases  []VariantCase
	Offset int64
}

type VariantCase struct {
	Values []int64
	Fields []*Field
}

type Method struct {
	Name string
	Off  int64
	Proc *ast.ProcDecl
	// VMT index (starting at 0; -1 if not virtual).
	VMTIndex int
	IsVirt   bool
	Parent   *Method
}

type SymKind int

const (
	SymVar SymKind = iota
	SymConst
	SymType
	SymProc
	SymFunc
	SymParam
	SymUnit
	SymLabel
	SymField
)

type Symbol struct {
	Name   string
	Lower  string
	Kind   SymKind
	Type   *Type
	Value  ast.Expr
	Decl   ast.Node
	Offset int64
}

type Scope struct {
	Parent  *Scope
	Symbols map[string]*Symbol
	Nodes   []ast.Node
	// For nested scopes:
	Owner ast.Node
}

func NewScope(parent *Scope) *Scope {
	return &Scope{Parent: parent, Symbols: map[string]*Symbol{}}
}

func (s *Scope) Lookup(name string) *Symbol {
	if s == nil {
		return nil
	}
	low := strings.ToLower(name)
	if sym, ok := s.Symbols[low]; ok {
		return sym
	}
	if s.Parent != nil {
		return s.Parent.Lookup(name)
	}
	return nil
}

func (s *Scope) Define(sym *Symbol) {
	if s.Symbols == nil {
		s.Symbols = map[string]*Symbol{}
	}
	low := strings.ToLower(sym.Name)
	if _, ok := s.Symbols[low]; ok {
		return
	}
	s.Symbols[low] = sym
}

type Program struct {
	Scope     *Scope
	Types     map[string]*Type
	Globals   []ast.Decl
	Body      ast.Stmt
	Errors    []string
	Source    ast.Node
	Units     map[string]*UnitInfo
	StackSize int
	HeapMin   int
	HeapMax   int
	SourceMap map[ast.Pos]string
}

type UnitInfo struct {
	Name    string
	Scope   *Scope
	Init    ast.Stmt
	Final   ast.Stmt
	Symbols map[string]*Symbol
}

// Analyzer runs over a parsed AST.
type Analyzer struct {
	errs        []string
	prog        *Program
	cur         *Scope
	curTyp      *Type
	switchTable bool
}

func New() *Analyzer {
	a := &Analyzer{prog: &Program{Types: map[string]*Type{}, Units: map[string]*UnitInfo{}, SourceMap: map[ast.Pos]string{}}}
	a.prog.Scope = NewScope(nil)
	a.cur = a.prog.Scope
	return a
}

func (a *Analyzer) Errors() []string { return a.errs }

func (a *Analyzer) Program() *Program { return a.prog }

func (a *Analyzer) errf(p ast.Pos, code int, format string, args ...any) {
	a.errs = append(a.errs, diagnostics.Format(diagnostics.CatCompile, code, p.File, p.Line, p.Col)+": "+fmt.Sprintf(format, args...))
}

func (a *Analyzer) Analyze(n ast.Node) {
	switch v := n.(type) {
	case *ast.Program:
		a.analyzeProgram(v)
	case *ast.Unit:
		a.analyzeUnit(v)
	default:
		a.analyzeBlockBody(ast.BlockBody{Base: ast.Base{P: n.Pos()}, Stmts: nil})
	}
}

func (a *Analyzer) analyzeProgram(p *ast.Program) {
	// Define built-in basic types.
	a.defineBasicTypes()
	// Define built-in identifiers.
	a.defineBuiltins()
	// Uses: register unit references.
	if p.Uses != nil {
		for _, u := range p.Uses.Items {
			a.cur.Define(&Symbol{Name: u.Name, Lower: strings.ToLower(u.Name), Kind: SymUnit, Decl: &u})
		}
	}
	// Block.
	var blockScope *Scope
	if p.Block != nil {
		blockScope = NewScope(a.cur)
		prev := a.cur
		a.cur = blockScope
		p.Block.SymScope = blockScope
		for _, d := range p.Block.Consts {
			a.analyzeDecl(d)
		}
		for _, d := range p.Block.Types {
			a.analyzeDecl(d)
		}
		for _, d := range p.Block.Vars {
			a.analyzeDecl(d)
		}
		for _, d := range p.Block.Procs {
			a.analyzeDecl(d)
		}
		// Body statement list is part of the program body, evaluated in the
		// block scope (so procs/vars are visible).
		if p.Body != nil {
			a.analyzeBlockBody(*p.Body)
		}
		a.cur = prev
	} else if p.Body != nil {
		a.analyzeBlockBody(*p.Body)
	}
	_ = blockScope
}

func (a *Analyzer) analyzeUnit(u *ast.Unit) {
	a.defineBasicTypes()
	a.defineBuiltins()
	ui := &UnitInfo{Name: u.Name, Symbols: map[string]*Symbol{}}
	scope := NewScope(a.prog.Scope)
	ui.Scope = scope
	prev := a.cur
	a.cur = scope
	if u.Interface != nil {
		a.analyzeInterface(u.Interface)
	}
	if u.Implementation != nil {
		a.analyzeImplementation(u.Implementation)
	}
	if u.Init != nil {
		ui.Init = a.analyzeBlockBody(*u.Init)
	}
	a.prog.Units[strings.ToLower(u.Name)] = ui
	a.cur = prev
}

func (a *Analyzer) analyzeInterface(i *ast.InterfaceSection) {
	if i.Uses != nil {
		for _, u := range i.Uses.Items {
			a.cur.Define(&Symbol{Name: u.Name, Lower: strings.ToLower(u.Name), Kind: SymUnit, Decl: &u})
		}
	}
	for _, d := range i.Decls {
		a.analyzeDecl(d)
	}
}

func (a *Analyzer) analyzeImplementation(i *ast.ImplementationSection) {
	if i.Uses != nil {
		for _, u := range i.Uses.Items {
			a.cur.Define(&Symbol{Name: u.Name, Lower: strings.ToLower(u.Name), Kind: SymUnit, Decl: &u})
		}
	}
	for _, d := range i.Decls {
		a.analyzeDecl(d)
	}
}

func (a *Analyzer) analyzeBlock(b *ast.Block) {
	inner := NewScope(a.cur)
	prev := a.cur
	a.cur = inner
	b.SymScope = inner
	for _, d := range b.Consts {
		a.analyzeDecl(d)
	}
	for _, d := range b.Types {
		a.analyzeDecl(d)
	}
	for _, d := range b.Vars {
		a.analyzeDecl(d)
	}
	for _, d := range b.Procs {
		a.analyzeDecl(d)
	}
	if b.Body != nil {
		a.analyzeBlockBody(*b.Body)
	}
	a.cur = prev
}

func (a *Analyzer) analyzeDecl(d ast.Decl) {
	switch v := d.(type) {
	case *ast.ConstDecl:
		var typ *Type
		if te, ok := v.Type.(ast.TypeExpr); ok {
			typ = a.resolveType(te, v.Base.Pos())
		}
		if typ == nil {
			typ = a.inferExpr(v.Value)
		}
		sym := &Symbol{Name: v.Name, Lower: strings.ToLower(v.Name), Kind: SymConst, Type: typ, Value: v.Value, Decl: v}
		a.cur.Define(sym)
	case *ast.TypeDecl:
		typ := a.resolveType(v.Type, v.Base.Pos())
		if typ != nil {
			typ.Name = v.Name
			typ.Lower = strings.ToLower(v.Name)
			a.prog.Types[typ.Lower] = typ
		}
		sym := &Symbol{Name: v.Name, Lower: strings.ToLower(v.Name), Kind: SymType, Type: typ, Decl: v}
		a.cur.Define(sym)
		// Enumerated values become constants in the current scope.
		if en, ok := v.Type.(*ast.EnumType); ok && typ != nil {
			for i, n := range en.Names {
				a.cur.Define(&Symbol{Name: n, Lower: strings.ToLower(n), Kind: SymConst, Type: typ, Value: &ast.IntLit{Base: v.Base, Value: int64(i)}, Decl: v})
			}
		}
	case *ast.VarDecl:
		typ := a.resolveType(v.Type, v.Base.Pos())
		for i, n := range v.Names {
			sym := &Symbol{Name: n, Lower: strings.ToLower(n), Kind: SymVar, Type: typ, Decl: v, Offset: int64(i) * 0}
			a.cur.Define(sym)
		}
	case *ast.ProcDecl:
		a.analyzeProc(v)
	case *ast.LabelDecl:
		for _, l := range v.Names {
			a.cur.Define(&Symbol{Name: fmt.Sprintf("L%d", l), Lower: fmt.Sprintf("l%d", l), Kind: SymLabel, Decl: v, Value: &ast.IntLit{Base: v.Base, Value: int64(l)}})
		}
	}
}

func (a *Analyzer) analyzeProc(p *ast.ProcDecl) {
	// First, register the symbol in the outer scope.
	kind := SymProc
	if p.IsFunc {
		kind = SymFunc
	}
	sym := &Symbol{Name: p.Name, Lower: strings.ToLower(p.Name), Kind: kind, Decl: p, Type: &Type{Kind: BKPointer, Name: "Pointer"}}
	a.cur.Define(sym)
	// Inner scope.
	inner := NewScope(a.cur)
	prev := a.cur
	a.cur = inner
	p.SymScope = inner
	for _, par := range p.Params {
		ptyp := a.resolveType(par.Type, p.Base.Pos())
		for i, n := range par.Names {
			psym := &Symbol{Name: n, Lower: strings.ToLower(n), Kind: SymParam, Type: ptyp, Decl: &par, Offset: int64(i)}
			a.cur.Define(psym)
		}
	}
	if p.Result != nil {
		typ := a.resolveType(&ast.TypeRef{Base: ast.Base{P: p.Base.Pos()}, Name: p.Result.Name, Lower: p.Result.Lower}, p.Base.Pos())
		if typ != nil {
			sym.Type = typ
		}
	}
	if p.Nested != nil {
		a.analyzeBlock(p.Nested)
	}
	if p.Body != nil {
		a.analyzeBlockBody(*p.Body)
	}
	a.cur = prev
}

func (a *Analyzer) analyzeBlockBody(b ast.BlockBody) ast.Stmt {
	for _, s := range b.Stmts {
		a.analyzeStmt(s)
	}
	if len(b.Stmts) == 1 {
		return b.Stmts[0]
	}
	return &ast.CompoundStmt{Base: b.Base, Stmts: b.Stmts}
}

func (a *Analyzer) analyzeStmt(s ast.Stmt) {
	if s == nil {
		return
	}
	switch v := s.(type) {
	case *ast.AssignStmt:
		lt := a.inferExpr(v.Dest)
		rt := a.inferExpr(v.Expr)
		if !a.isAssignable(lt, rt) {
			a.errf(v.Base.Pos(), 26, "type mismatch: cannot assign %s to %s", rt, lt)
		}
	case *ast.CallStmt:
		a.inferCall(&v.Call)
	case *ast.CompoundStmt:
		for _, ss := range v.Stmts {
			a.analyzeStmt(ss)
		}
	case *ast.IfStmt:
		a.inferExpr(v.Cond)
		a.analyzeStmt(v.Then)
		a.analyzeStmt(v.Else)
	case *ast.CaseStmt:
		a.inferExpr(v.Expr)
		for _, br := range v.Cases {
			for _, val := range br.Values {
				a.inferExpr(val)
			}
			a.analyzeStmt(br.Body)
		}
		a.analyzeStmt(v.Else)
	case *ast.WhileStmt:
		a.inferExpr(v.Cond)
		a.analyzeStmt(v.Body)
	case *ast.RepeatStmt:
		a.analyzeStmt(v.Body)
		a.inferExpr(v.Cond)
	case *ast.ForStmt:
		a.inferExpr(v.Lo)
		a.inferExpr(v.Hi)
		a.analyzeStmt(v.Body)
	case *ast.WithStmt:
		a.inferExpr(v.Rec)
		a.analyzeStmt(v.Body)
	case *ast.GotoStmt:
		// No-op; we just verify the label exists at runtime.
	case *ast.LabelStmt:
		// No-op.
	case *ast.HaltStmt:
		a.inferExpr(v.Code)
	case *ast.InheritedStmt:
		if v.Call != nil {
			a.inferCall(v.Call)
		}
	}
}

func (a *Analyzer) inferExpr(e ast.Expr) *Type {
	if e == nil {
		return nil
	}
	switch v := e.(type) {
	case *ast.IntLit:
		return &Type{Kind: BKInteger}
	case *ast.RealLit:
		return &Type{Kind: BKReal}
	case *ast.StringLit:
		return &Type{Kind: BKString, IsString: true, Len: int64(len(v.Value))}
	case *ast.CharLit:
		return &Type{Kind: BKChar}
	case *ast.Ident:
		// `nil` is a special identifier that represents the null
		// pointer. It is assignable to any pointer type.
		if v.Lower == "nil" {
			return &Type{Kind: BKPointer, IsPointer: true}
		}
		sym := a.cur.Lookup(v.Name)
		if sym == nil {
			a.errf(v.Base.Pos(), 3, "unknown identifier %q", v.Name)
			return nil
		}
		return sym.Type
	case *ast.BinaryExpr:
		lt := a.inferExpr(v.Left)
		rt := a.inferExpr(v.Right)
		return a.binaryResult(v.Op, lt, rt)
	case *ast.UnaryExpr:
		t := a.inferExpr(v.Expr)
		if v.Op == "not" {
			return &Type{Kind: BKBoolean}
		}
		return t
	case *ast.CaretExpr:
		t := a.inferExpr(v.Expr)
		if t != nil {
			return &Type{Kind: t.Element.Kind, Name: t.Element.Name, Element: t.Element.Element}
		}
		return &Type{Kind: BKInteger}
	case *ast.AtExpr:
		a.inferExpr(v.Expr)
		return &Type{Kind: BKPointer}
	case *ast.CallExpr:
		return a.inferCall(v)
	case *ast.FieldExpr:
		return a.inferField(v)
	case *ast.IndexExpr:
		t := a.inferExpr(v.Expr)
		if t != nil {
			return t.Element
		}
		return nil
	case *ast.TypeCastExpr:
		if te, ok := v.Type.(ast.TypeExpr); ok {
			return a.resolveType(te, v.Base.Pos())
		}
		return nil
	case *ast.SetExpr:
		if len(v.Elements) > 0 {
			elt := a.inferExpr(v.Elements[0].Lo)
			return &Type{Kind: BKWord, IsSet: true, Element: elt}
		}
		return &Type{Kind: BKWord, IsSet: true, Element: &Type{Kind: BKChar}}
	case *ast.RangeExpr:
		return nil
	}
	return nil
}

func (a *Analyzer) inferCall(c *ast.CallExpr) *Type {
	id, ok := c.Func.(*ast.Ident)
	if !ok {
		a.inferExpr(c.Func)
		return nil
	}
	sym := a.cur.Lookup(id.Name)
	if sym == nil {
		a.errf(id.Base.Pos(), 3, "unknown identifier %q", id.Name)
		return nil
	}
	for _, arg := range c.Args {
		a.inferExpr(arg)
	}
	if sym.Kind == SymFunc && sym.Type != nil {
		return sym.Type
	}
	return nil
}

func (a *Analyzer) inferField(e *ast.FieldExpr) *Type {
	// Simplified: object/record field lookup.
	id, ok := e.Expr.(*ast.Ident)
	if !ok {
		a.inferExpr(e.Expr)
		return nil
	}
	sym := a.cur.Lookup(id.Name)
	if sym == nil || sym.Type == nil {
		return nil
	}
	for _, f := range sym.Type.Fields {
		if strings.EqualFold(f.Name, e.Field) {
			return f.Type
		}
	}
	return nil
}

func (a *Analyzer) binaryResult(op string, lt, rt *Type) *Type {
	if lt == nil || rt == nil {
		return nil
	}
	if op == "and" || op == "or" || op == "xor" {
		return &Type{Kind: BKBoolean}
	}
	if op == "=" || op == "<>" || op == "<" || op == "<=" || op == ">" || op == ">=" {
		return &Type{Kind: BKBoolean}
	}
	if op == "in" {
		return &Type{Kind: BKBoolean}
	}
	if lt.Kind == BKReal || rt.Kind == BKReal {
		return &Type{Kind: BKReal}
	}
	if lt.Kind == BKLongInt || rt.Kind == BKLongInt {
		return &Type{Kind: BKLongInt}
	}
	return &Type{Kind: BKInteger}
}

func (a *Analyzer) isAssignable(dst, src *Type) bool {
	if dst == nil || src == nil {
		return true
	}
	if dst.Equals(src) {
		return true
	}
	if (dst.IsPointer || dst.Kind == BKPointer) && src.IsPointer && src.Element == nil {
		return true
	}
	// Numeric compatibility
	if dst.Kind == BKReal && (src.Kind == BKInteger || src.Kind == BKByte || src.Kind == BKWord) {
		return true
	}
	if dst.Kind == BKLongInt && (src.Kind == BKInteger || src.Kind == BKByte || src.Kind == BKWord) {
		return true
	}
	// Integer literals are assignable to most ordinal types.
	if src.Kind == BKInteger {
		if dst.IsRange || dst.IsEnum {
			return true
		}
		switch dst.Kind {
		case BKInteger, BKWord, BKByte, BKShortInt, BKLongInt, BKBoolean, BKChar:
			return true
		}
	}
	// Subrange compatibility
	if dst.IsRange && src.IsRange {
		return dst.Lo == src.Lo && dst.Hi == src.Hi
	}
	return false
}

func (a *Analyzer) resolveType(t ast.TypeExpr, pos ast.Pos) *Type {
	if t == nil {
		return nil
	}
	switch v := t.(type) {
	case *ast.OrdType:
		switch v.Kind {
		case "Integer":
			return &Type{Kind: BKInteger, Name: "Integer", Lower: "integer"}
		case "LongInt":
			return &Type{Kind: BKLongInt, Name: "LongInt", Lower: "longint"}
		case "ShortInt":
			return &Type{Kind: BKShortInt, Name: "ShortInt", Lower: "shortint"}
		case "Byte":
			return &Type{Kind: BKByte, Name: "Byte", Lower: "byte"}
		case "Word":
			return &Type{Kind: BKWord, Name: "Word", Lower: "word"}
		case "Boolean":
			return &Type{Kind: BKBoolean, Name: "Boolean", Lower: "boolean"}
		case "Char":
			return &Type{Kind: BKChar, Name: "Char", Lower: "char"}
		}
	case *ast.FloatType:
		switch v.Kind {
		case "Real":
			return &Type{Kind: BKReal, Name: "Real", Lower: "real"}
		case "Single":
			return &Type{Kind: BKSingle, Name: "Single", Lower: "single"}
		case "Double":
			return &Type{Kind: BKDouble, Name: "Double", Lower: "double"}
		case "Extended":
			return &Type{Kind: BKExtended, Name: "Extended", Lower: "extended"}
		case "Comp":
			return &Type{Kind: BKComp, Name: "Comp", Lower: "comp"}
		}
	case *ast.StringType:
		if v.Len == nil {
			return &Type{Kind: BKString, Name: "String", Lower: "string", IsString: true, Len: 255}
		}
		if n, ok := v.Len.(*ast.IntLit); ok {
			return &Type{Kind: BKString, Name: "String", Lower: "string", IsString: true, Len: n.Value}
		}
	case *ast.ArrayType:
		elem := a.resolveType(v.Element, pos)
		idxs := []*Type{}
		for _, r := range v.Index {
			_ = r
			idxs = append(idxs, &Type{Kind: BKInteger})
		}
		t := &Type{Kind: BKInteger, IsArray: true, Element: elem, Index: idxs, IsPacked: v.Packed}
		t.Size = computeArraySize(t)
		return t
	case *ast.RecordType:
		rt := &Type{Kind: BKInteger, IsRecord: true, Name: v.String()}
		for _, f := range v.Fields {
			ft := a.resolveType(f.Type, pos)
			rt.Fields = append(rt.Fields, &Field{Name: f.Names[0], Type: ft, Off: rt.Size})
			if ft != nil {
				rt.Size += ft.Size
			}
		}
		return rt
	case *ast.SetType:
		elt := a.resolveType(v.Element, pos)
		return &Type{Kind: BKWord, IsSet: true, Element: elt}
	case *ast.FileType:
		ft := &Type{Kind: BKText, IsFile: true, IsText: v.Text}
		if v.Element != nil {
			ft.Element = a.resolveType(v.Element, pos)
		}
		return ft
	case *ast.PointerType:
		inner := a.resolveType(v.Target, pos)
		return &Type{Kind: BKPointer, IsPointer: true, Element: inner}
	case *ast.TypeRef:
		low := strings.ToLower(v.Name)
		if t2, ok := a.prog.Types[low]; ok {
			return t2
		}
		// Built-in aliases.
		switch low {
		case "string":
			return &Type{Kind: BKString, Name: "String", Lower: "string", IsString: true, Len: 255}
		case "pchar":
			return &Type{Kind: BKPChar, Name: "PChar", Lower: "pchar"}
		case "text":
			return &Type{Kind: BKText, Name: "Text", Lower: "text", IsText: true}
		}
		return &Type{Kind: BKUnknown, Name: v.Name, Lower: low}
	case *ast.EnumType:
		et := &Type{Kind: BKWord, IsEnum: true, EnumVals: v.Names}
		for i, n := range v.Names {
			et.Fields = append(et.Fields, &Field{Name: n, Off: int64(i), Type: et})
		}
		return et
	case *ast.RangeType:
		lo, hi := int64(0), int64(0)
		if l, ok := v.Lo.(*ast.IntLit); ok {
			lo = l.Value
		}
		if h, ok := v.Hi.(*ast.IntLit); ok {
			hi = h.Value
		}
		return &Type{Kind: BKInteger, IsRange: true, Lo: lo, Hi: hi}
	case *ast.ObjectType:
		ot := a.analyzeObject(v, pos)
		return ot
	}
	return nil
}

func (a *Analyzer) analyzeObject(o *ast.ObjectType, pos ast.Pos) *Type {
	t := &Type{Kind: BKPointer, IsObject: true, Name: o.Name, Lower: strings.ToLower(o.Name)}
	if o.Parent != "" {
		pname := strings.ToLower(o.Parent)
		if pt, ok := a.prog.Types[pname]; ok {
			t.Parent = pt
		}
	}
	// Methods: VMT
	vmtIdx := 0
	for _, m := range o.Methods {
		mm := &Method{Name: m.Name, Off: int64(vmtIdx) * 4, Proc: &m, VMTIndex: vmtIdx, IsVirt: true}
		t.Methods = append(t.Methods, mm)
		t.VMT = append(t.VMT, strings.ToLower(m.Name))
		vmtIdx++
	}
	// Fields layout after VMT pointer (4 bytes for segment:offset).
	off := int64(4)
	for _, f := range o.Fields {
		ft := a.resolveType(f.Type, pos)
		for _, nm := range f.Names {
			t.Fields = append(t.Fields, &Field{Name: nm, Off: off, Type: ft})
		}
		if ft != nil {
			off += ft.Size
		}
	}
	t.Size = off
	return t
}

func computeArraySize(t *Type) int64 {
	if t.Element == nil {
		return 0
	}
	count := int64(1)
	for _, idx := range t.Index {
		count *= 1 // simplified: only 0..N arrays
		_ = idx
	}
	return count * t.Element.Size
}

func (a *Analyzer) defineBasicTypes() {
	basic := []*Type{
		{Kind: BKShortInt, Name: "ShortInt", Lower: "shortint", Size: 1, Align: 1},
		{Kind: BKByte, Name: "Byte", Lower: "byte", Size: 1, Align: 1},
		{Kind: BKInteger, Name: "Integer", Lower: "integer", Size: 2, Align: 2},
		{Kind: BKWord, Name: "Word", Lower: "word", Size: 2, Align: 2},
		{Kind: BKLongInt, Name: "LongInt", Lower: "longint", Size: 4, Align: 2},
		{Kind: BKBoolean, Name: "Boolean", Lower: "boolean", Size: 1, Align: 1},
		{Kind: BKChar, Name: "Char", Lower: "char", Size: 1, Align: 1},
		{Kind: BKReal, Name: "Real", Lower: "real", Size: 6, Align: 2},
		{Kind: BKSingle, Name: "Single", Lower: "single", Size: 4, Align: 2},
		{Kind: BKDouble, Name: "Double", Lower: "double", Size: 8, Align: 2},
		{Kind: BKExtended, Name: "Extended", Lower: "extended", Size: 10, Align: 2},
		{Kind: BKComp, Name: "Comp", Lower: "comp", Size: 8, Align: 2},
		{Kind: BKString, Name: "String", Lower: "string", Size: 256, Align: 1, IsString: true, Len: 255},
		{Kind: BKPChar, Name: "PChar", Lower: "pchar", Size: 4, Align: 2},
		{Kind: BKPointer, Name: "Pointer", Lower: "pointer", Size: 4, Align: 2},
		{Kind: BKText, Name: "Text", Lower: "text", Size: 256, Align: 1, IsText: true},
	}
	for _, t := range basic {
		a.prog.Types[t.Lower] = t
	}
}

func (a *Analyzer) defineBuiltins() {
	for name, kind := range map[string]SymKind{
		"Input":  SymVar,
		"Output": SymVar,
	} {
		low := strings.ToLower(name)
		t := &Type{Kind: BKText, IsText: true}
		if kind == SymVar {
			a.cur.Define(&Symbol{Name: name, Lower: low, Kind: kind, Type: t})
		}
	}
}

// SizeOf returns the byte size of a type or expression.
func SizeOf(t *Type) int64 {
	if t == nil {
		return 0
	}
	if t.Size > 0 {
		return t.Size
	}
	return t.Kind.Size()
}

// SortedTypes returns the names of all defined types.
func (a *Analyzer) SortedTypes() []string {
	out := make([]string, 0, len(a.prog.Types))
	for n := range a.prog.Types {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}
