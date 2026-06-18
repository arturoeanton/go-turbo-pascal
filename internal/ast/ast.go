// Package ast defines the Abstract Syntax Tree for BPGo. The tree is
// designed to be the common representation used by the semantic
// analyzer, IR generator, codegen and IDE tooling. Every node has a
// position (file/line/column) and an optional "original" textual form
// used for source maps.
package ast

import (
	"sort"
	"strings"
)

type Pos struct {
	File string
	Line int
	Col  int
}

type Node interface {
	Pos() Pos
	String() string
}

type Base struct {
	P Pos
}

func (b Base) Pos() Pos               { return b.P }
func (b Base) WithFile(f string) Base { b.P.File = f; return b }

type Program struct {
	Base
	Name  string
	Uses  *UsesClause
	Block *Block
	Body  *BlockBody
}

func (p Program) String() string { return "program " + p.Name }

type Unit struct {
	Base
	Name           string
	Interface      *InterfaceSection
	Implementation *ImplementationSection
	Init           *BlockBody
}

func (u Unit) String() string { return "unit " + u.Name }

type InterfaceSection struct {
	Base
	Uses  *UsesClause
	Decls []Decl
}

func (i InterfaceSection) String() string { return "interface" }

type ImplementationSection struct {
	Base
	Uses  *UsesClause
	Decls []Decl
}

func (i ImplementationSection) String() string { return "implementation" }

type UsesClause struct {
	Base
	Items []UnitRef
}

func (u UsesClause) String() string {
	parts := make([]string, len(u.Items))
	for i, it := range u.Items {
		parts[i] = it.Name
	}
	return "uses " + strings.Join(parts, ",")
}

type UnitRef struct {
	Base
	Name string
	In   string
}

func (u UnitRef) String() string { return u.Name }

type Block struct {
	Base
	Labels []int
	Consts []Decl
	Types  []Decl
	Vars   []Decl
	Procs  []Decl
	Body   *BlockBody
	// SymScope is set by the semantic analyzer; nil for unanalyzed trees.
	SymScope interface{}
}

func (Block) String() string { return "block" }

type BlockBody struct {
	Base
	Stmts []Stmt
}

func (BlockBody) String() string { return "body" }

type Decl interface {
	Node
	declNode()
}

type Stmt interface {
	Node
	stmtNode()
}

type Expr interface {
	Node
	exprNode()
	exprKind() ExprKind
}

type ExprKind int

const (
	EKInt ExprKind = iota
	EKReal
	EKString
	EKChar
	EKIdent
	EKBinary
	EKUnary
	EKCaret
	EKAt
	EKCall
	EKField
	EKIndex
	EKTypeCast
	EKSet
	EKRange
	EKIn
	EKWriteArg
	EKAnonFunc
)

type IntLit struct {
	Base
	Value int64
	Hex   bool
}

func (IntLit) exprNode()          {}
func (IntLit) exprKind() ExprKind { return EKInt }
func (i IntLit) String() string   { return IntString(i.Value, i.Hex) }

func IntString(v int64, hex bool) string {
	if hex {
		return "$" + strings.ToUpper(formatHex(v))
	}
	return formatInt(v)
}

func formatInt(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [32]byte
	pos := len(buf)
	for v > 0 {
		pos--
		buf[pos] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

func formatHex(v int64) string {
	if v == 0 {
		return "0"
	}
	const hex = "0123456789ABCDEF"
	var buf [16]byte
	pos := len(buf)
	for v > 0 {
		pos--
		buf[pos] = hex[v&0xF]
		v >>= 4
	}
	return string(buf[pos:])
}

type RealLit struct {
	Base
	Value float64
}

func (RealLit) exprNode()          {}
func (RealLit) exprKind() ExprKind { return EKReal }
func (r RealLit) String() string {
	if r.Value == 0 {
		return "0"
	}
	return formatReal(r.Value)
}

func formatReal(v float64) string {
	// Match TP7 default representation: at least one decimal point.
	s := formatFloat(v)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}

func formatFloat(v float64) string {
	out := []byte{}
	neg := v < 0
	if neg {
		v = -v
	}
	whole := int64(v)
	frac := v - float64(whole)
	if whole == 0 && neg {
		out = append(out, '-', '0')
	} else {
		if neg {
			out = append(out, '-')
		}
		out = append(out, []byte(formatInt(whole))...)
	}
	// Up to 12 fractional digits.
	out = append(out, '.')
	for i := 0; i < 12; i++ {
		frac *= 10
		d := int(frac)
		out = append(out, byte('0'+d))
		frac -= float64(d)
	}
	// Trim trailing zeros.
	for len(out) > 0 && out[len(out)-1] == '0' {
		out = out[:len(out)-1]
	}
	if len(out) > 0 && out[len(out)-1] == '.' {
		out = out[:len(out)-1]
	}
	return string(out)
}

type StringLit struct {
	Base
	Value string
}

func (StringLit) exprNode()          {}
func (StringLit) exprKind() ExprKind { return EKString }
func (s StringLit) String() string {
	var b strings.Builder
	b.WriteByte('\'')
	for _, r := range s.Value {
		if r == '\'' {
			b.WriteString("''")
		} else if r < 32 || r > 126 {
			b.WriteString("#")
			b.WriteString(formatInt(int64(r)))
		} else {
			b.WriteRune(r)
		}
	}
	b.WriteByte('\'')
	return b.String()
}

type CharLit struct {
	Base
	Value byte
}

func (CharLit) exprNode()          {}
func (CharLit) exprKind() ExprKind { return EKChar }
func (c CharLit) String() string {
	return "#" + formatInt(int64(c.Value))
}

type Ident struct {
	Base
	Name  string
	Lower string
}

func (Ident) exprNode()          {}
func (Ident) exprKind() ExprKind { return EKIdent }
func (i Ident) String() string   { return i.Name }

type BinaryExpr struct {
	Base
	Op      string
	Left    Expr
	Right   Expr
	IsRange bool
}

func (BinaryExpr) exprNode()          {}
func (BinaryExpr) exprKind() ExprKind { return EKBinary }
func (b BinaryExpr) String() string {
	return "(" + b.Left.String() + " " + b.Op + " " + b.Right.String() + ")"
}

type UnaryExpr struct {
	Base
	Op   string
	Expr Expr
}

func (UnaryExpr) exprNode()          {}
func (UnaryExpr) exprKind() ExprKind { return EKUnary }
func (u UnaryExpr) String() string {
	return "(" + u.Op + " " + u.Expr.String() + ")"
}

type CaretExpr struct {
	Base
	Expr Expr
}

func (CaretExpr) exprNode()          {}
func (CaretExpr) exprKind() ExprKind { return EKCaret }
func (c CaretExpr) String() string   { return c.Expr.String() + "^" }

type AtExpr struct {
	Base
	Expr Expr
}

// AnonFunc is an anonymous method expression:
// `procedure(params) begin ... end` or `function(params): T begin ... end`.
// It may capture variables from the enclosing scope (by reference).
type AnonFunc struct {
	Base
	IsFunc bool
	Params []Param
	Result *TypeRef
	Body   *BlockBody
}

func (AnonFunc) exprNode()          {}
func (AnonFunc) exprKind() ExprKind { return EKAnonFunc }
func (a AnonFunc) String() string {
	if a.IsFunc {
		return "function(...)"
	}
	return "procedure(...)"
}

func (AtExpr) exprNode()          {}
func (AtExpr) exprKind() ExprKind { return EKAt }
func (a AtExpr) String() string   { return "@" + a.Expr.String() }

type CallExpr struct {
	Base
	Func Expr
	Args []Expr
}

func (CallExpr) exprNode()          {}
func (CallExpr) exprKind() ExprKind { return EKCall }
func (c CallExpr) String() string {
	args := make([]string, len(c.Args))
	for i, a := range c.Args {
		args[i] = a.String()
	}
	return c.Func.String() + "(" + strings.Join(args, ", ") + ")"
}

type FieldExpr struct {
	Base
	Expr  Expr
	Field string
}

func (FieldExpr) exprNode()          {}
func (FieldExpr) exprKind() ExprKind { return EKField }
func (f FieldExpr) String() string   { return f.Expr.String() + "." + f.Field }

type IndexExpr struct {
	Base
	Expr  Expr
	Index Expr
}

func (IndexExpr) exprNode()          {}
func (IndexExpr) exprKind() ExprKind { return EKIndex }
func (i IndexExpr) String() string   { return i.Expr.String() + "[" + i.Index.String() + "]" }

type TypeCastExpr struct {
	Base
	Type Expr
	Expr Expr
}

func (TypeCastExpr) exprNode()          {}
func (TypeCastExpr) exprKind() ExprKind { return EKTypeCast }
func (t TypeCastExpr) String() string {
	return t.Type.String() + "(" + t.Expr.String() + ")"
}

type SetExpr struct {
	Base
	Elements []SetElement
}

type SetElement struct {
	Base
	Lo Expr
	Hi Expr // may be nil for a single element
}

func (SetExpr) exprNode()          {}
func (SetExpr) exprKind() ExprKind { return EKSet }
func (s SetExpr) String() string {
	if len(s.Elements) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(s.Elements))
	for _, e := range s.Elements {
		if e.Hi == nil {
			parts = append(parts, e.Lo.String())
		} else {
			parts = append(parts, e.Lo.String()+".."+e.Hi.String())
		}
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

type RangeExpr struct {
	Base
	Lo Expr
	Hi Expr
}

func (RangeExpr) exprNode()          {}
func (RangeExpr) exprKind() ExprKind { return EKRange }
func (r RangeExpr) String() string {
	return r.Lo.String() + ".." + r.Hi.String()
}

// WriteArg is a Write/WriteLn argument with optional field-width and decimal
// formatting: `value`, `value:width`, or `value:width:decimals`.
type WriteArg struct {
	Base
	Value    Expr
	Width    Expr // may be nil
	Decimals Expr // may be nil
}

func (WriteArg) exprNode()          {}
func (WriteArg) exprKind() ExprKind { return EKWriteArg }
func (w WriteArg) String() string {
	s := w.Value.String()
	if w.Width != nil {
		s += ":" + w.Width.String()
	}
	if w.Decimals != nil {
		s += ":" + w.Decimals.String()
	}
	return s
}

type InExpr struct {
	Base
	Left  Expr
	Right Expr
}

func (InExpr) exprNode()          {}
func (InExpr) exprKind() ExprKind { return EKIn }
func (i InExpr) String() string {
	return i.Left.String() + " in " + i.Right.String()
}

// Declarations

type ConstDecl struct {
	Base
	Name  string
	Type  TypeExpr
	Value Expr
}

func (ConstDecl) declNode() {}
func (c ConstDecl) String() string {
	return "const " + c.Name + " = " + c.Value.String()
}

type TypeDecl struct {
	Base
	Name       string
	TypeParams []string // generic type parameters: TList<T> = ...
	Type       TypeExpr
}

func (TypeDecl) declNode() {}
func (t TypeDecl) String() string {
	return "type " + t.Name + " = " + t.Type.String()
}

type VarDecl struct {
	Base
	Names     []string
	Type      TypeExpr // nil = infer from Init ({$MODE BPGO}: var x := expr)
	Abs       Expr
	Init      Expr
	Immutable bool // `let` binding: reassignment is a compile error
}

func (VarDecl) declNode() {}
func (v VarDecl) String() string {
	if v.Abs != nil {
		return "var " + strings.Join(v.Names, ",") + ": " + v.Type.String() + " absolute " + v.Abs.String()
	}
	return "var " + strings.Join(v.Names, ",") + ": " + v.Type.String()
}

type ProcDecl struct {
	Base
	Name          string
	OperatorSym   string   // non-empty for an operator overload: `operator + (...)`
	TypeParams    []string // generic type parameters: function Max<T>(...)
	Params        []Param
	Result        *TypeRef
	Body          *BlockBody
	Nested        *Block // local nested procedures and locals
	SymScope      interface{}
	Forward       bool
	External      string
	Inline        []string
	Assembler     bool
	Interrupt     bool
	Far           bool
	Near          bool
	IsFunc        bool
	IsMethod      bool
	IsConstructor bool
	IsDestructor  bool
	OfObject      string
}

type Param struct {
	Base
	Names []string
	Type  TypeExpr
	Var   bool
	Const bool
	Open  bool // V+ open string
}

func (Param) declNode() {}
func (Param) stmtNode() {}
func (Param) exprNode() {}
func (p Param) String() string {
	if p.Type != nil {
		return strings.Join(p.Names, ",") + ": " + p.Type.String()
	}
	return strings.Join(p.Names, ",")
}
func (Param) exprKind() ExprKind { return EKIdent }

func (p ProcDecl) declNode() {}
func (p ProcDecl) String() string {
	if p.Forward {
		return "procedure " + p.Name + "; forward"
	}
	if p.External != "" {
		return "procedure " + p.Name + "; external '" + p.External + "'"
	}
	return "procedure " + p.Name
}

type LabelDecl struct {
	Base
	Names []int
}

func (LabelDecl) declNode() {}
func (l LabelDecl) String() string {
	parts := make([]string, len(l.Names))
	for i, n := range l.Names {
		parts[i] = formatInt(int64(n))
	}
	return "label " + strings.Join(parts, ",")
}

// Statements

type AssignStmt struct {
	Base
	Dest Expr
	Expr Expr
}

func (AssignStmt) stmtNode() {}
func (a AssignStmt) String() string {
	return a.Dest.String() + " := " + a.Expr.String()
}

type CallStmt struct {
	Base
	Call CallExpr
}

func (CallStmt) stmtNode()        {}
func (c CallStmt) String() string { return c.Call.String() }

type CompoundStmt struct {
	Base
	Stmts []Stmt
}

func (CompoundStmt) stmtNode() {}
func (c CompoundStmt) String() string {
	parts := make([]string, len(c.Stmts))
	for i, s := range c.Stmts {
		parts[i] = s.String()
	}
	return "begin " + strings.Join(parts, "; ") + " end"
}

type IfStmt struct {
	Base
	Cond Expr
	Then Stmt
	Else Stmt
}

func (IfStmt) stmtNode() {}
func (i IfStmt) String() string {
	if i.Else == nil {
		return "if " + i.Cond.String() + " then " + i.Then.String()
	}
	return "if " + i.Cond.String() + " then " + i.Then.String() + " else " + i.Else.String()
}

type CaseStmt struct {
	Base
	Expr  Expr
	Cases []CaseBranch
	Else  Stmt
}

type CaseBranch struct {
	Base
	Values []Expr
	Body   Stmt
}

func (CaseStmt) stmtNode() {}
func (c CaseStmt) String() string {
	parts := make([]string, len(c.Cases))
	for i, br := range c.Cases {
		vals := make([]string, len(br.Values))
		for j, v := range br.Values {
			vals[j] = v.String()
		}
		parts[i] = strings.Join(vals, ", ") + ": " + br.Body.String()
	}
	out := "case " + c.Expr.String() + " of " + strings.Join(parts, "; ")
	if c.Else != nil {
		out += "; else " + c.Else.String()
	}
	return out + " end"
}

type WhileStmt struct {
	Base
	Cond Expr
	Body Stmt
}

func (WhileStmt) stmtNode() {}
func (w WhileStmt) String() string {
	return "while " + w.Cond.String() + " do " + w.Body.String()
}

type RepeatStmt struct {
	Base
	Cond Expr
	Body Stmt
}

func (RepeatStmt) stmtNode() {}
func (r RepeatStmt) String() string {
	return "repeat " + r.Body.String() + " until " + r.Cond.String()
}

type ForStmt struct {
	Base
	Var  string
	Lo   Expr
	Hi   Expr
	Down bool
	Body Stmt
}

// TryStmt is `try Body except Except end` or `try Body finally Finally end`.
// Exactly one of Except/Finally is non-nil.
type TryStmt struct {
	Base
	Body    []Stmt
	Except  []Stmt
	Finally []Stmt
}

func (TryStmt) stmtNode()      {}
func (TryStmt) String() string { return "try" }

// RaiseStmt is `raise Expr` (Expr nil = re-raise).
type RaiseStmt struct {
	Base
	Expr Expr
}

func (RaiseStmt) stmtNode()      {}
func (RaiseStmt) String() string { return "raise" }

// ForInStmt is `for Var in Coll do Body` (arrays, strings).
type ForInStmt struct {
	Base
	Var  string
	Coll Expr
	Body Stmt
}

func (ForInStmt) stmtNode() {}
func (f ForInStmt) String() string {
	return "for " + f.Var + " in " + f.Coll.String() + " do " + f.Body.String()
}

func (ForStmt) stmtNode() {}
func (f ForStmt) String() string {
	dir := "to"
	if f.Down {
		dir = "downto"
	}
	return "for " + f.Var + " := " + f.Lo.String() + " " + dir + " " + f.Hi.String() + " do " + f.Body.String()
}

type WithStmt struct {
	Base
	Rec  Expr
	Body Stmt
}

func (WithStmt) stmtNode() {}
func (w WithStmt) String() string {
	return "with " + w.Rec.String() + " do " + w.Body.String()
}

type GotoStmt struct {
	Base
	Label int
}

func (GotoStmt) stmtNode()        {}
func (g GotoStmt) String() string { return "goto " + formatInt(int64(g.Label)) }

type LabelStmt struct {
	Base
	Label int
}

func (LabelStmt) stmtNode()        {}
func (l LabelStmt) String() string { return formatInt(int64(l.Label)) + ":" }

type BreakStmt struct{ Base }

func (BreakStmt) stmtNode()        {}
func (b BreakStmt) String() string { return "break" }

type ContinueStmt struct{ Base }

func (ContinueStmt) stmtNode()        {}
func (c ContinueStmt) String() string { return "continue" }

type ExitStmt struct{ Base }

func (ExitStmt) stmtNode()      {}
func (ExitStmt) String() string { return "exit" }

type HaltStmt struct {
	Base
	Code Expr
}

func (HaltStmt) stmtNode() {}
func (h HaltStmt) String() string {
	if h.Code == nil {
		return "halt"
	}
	return "halt(" + h.Code.String() + ")"
}

type AsmStmt struct {
	Base
	Body string
}

func (AsmStmt) stmtNode()        {}
func (a AsmStmt) String() string { return "asm ... end" }

type InlineStmt struct {
	Base
	Body string
}

func (InlineStmt) stmtNode()      {}
func (InlineStmt) String() string { return "inline(...)" }

type InheritedStmt struct {
	Base
	Call *CallExpr
}

func (InheritedStmt) stmtNode() {}
func (i InheritedStmt) String() string {
	if i.Call == nil {
		return "inherited"
	}
	return "inherited " + i.Call.String()
}

// Type expressions

type TypeExpr interface {
	Node
	typeExpr()
	String() string
}

type TypeRef struct {
	Base
	Name  string
	Lower string
	Args  []TypeExpr
}

func (TypeRef) typeExpr()            {}
func (t TypeRef) exprNode()          {}
func (t TypeRef) exprKind() ExprKind { return EKIdent }
func (t TypeRef) String() string {
	if len(t.Args) == 0 {
		return t.Name
	}
	args := make([]string, len(t.Args))
	for i, a := range t.Args {
		args[i] = a.String()
	}
	return t.Name + "<" + strings.Join(args, ",") + ">"
}

type ArrayType struct {
	Base
	Packed  bool
	Index   []RangeExpr
	Element TypeExpr
}

func (ArrayType) typeExpr() {}
func (a ArrayType) String() string {
	idxs := make([]string, len(a.Index))
	for i, r := range a.Index {
		idxs[i] = r.String()
	}
	prefix := ""
	if a.Packed {
		prefix = "packed "
	}
	return prefix + "array[" + strings.Join(idxs, ",") + "] of " + a.Element.String()
}

type RecordType struct {
	Base
	Packed  bool
	Fields  []RecordField
	Variant *VariantPart
}

type RecordField struct {
	Base
	Names []string
	Type  TypeExpr
}

type VariantPart struct {
	Base
	Tag     Expr     // selector field name (empty for an anonymous selector)
	TagType TypeExpr // selector field type, if any
	Cases   []VariantCase
}

type VariantCase struct {
	Base
	Values []Expr
	Fields []RecordField
}

func (RecordType) typeExpr() {}
func (r RecordType) String() string {
	parts := make([]string, len(r.Fields))
	for i, f := range r.Fields {
		parts[i] = strings.Join(f.Names, ",") + ": " + f.Type.String()
	}
	return "record " + strings.Join(parts, "; ") + " end"
}

type SetType struct {
	Base
	Packed  bool
	Element TypeExpr
}

func (SetType) typeExpr() {}
func (s SetType) String() string {
	return "set of " + s.Element.String()
}

type FileType struct {
	Base
	Text    bool
	Element TypeExpr
}

func (FileType) typeExpr() {}
func (f FileType) String() string {
	if f.Text {
		return "Text"
	}
	if f.Element == nil {
		return "File"
	}
	return "File of " + f.Element.String()
}

type PointerType struct {
	Base
	Target TypeExpr
}

func (PointerType) typeExpr() {}
func (p PointerType) String() string {
	return "^" + p.Target.String()
}

type ProcType struct {
	Base
	IsFunc bool
	Params []Param
	Result *TypeRef
}

func (ProcType) typeExpr() {}
func (p ProcType) String() string {
	kind := "procedure"
	if p.IsFunc {
		kind = "function"
	}
	return kind + " (TP)"
}

type StringType struct {
	Base
	Len Expr
}

func (StringType) typeExpr() {}
func (s StringType) String() string {
	if s.Len == nil {
		return "String"
	}
	return "String[" + s.Len.String() + "]"
}

type RangeType struct {
	Base
	Lo Expr
	Hi Expr
}

func (RangeType) typeExpr() {}
func (r RangeType) String() string {
	return r.Lo.String() + ".." + r.Hi.String()
}

type EnumType struct {
	Base
	Names []string
}

func (EnumType) typeExpr() {}
func (e EnumType) String() string {
	return "(" + strings.Join(e.Names, ", ") + ")"
}

type ObjectType struct {
	Base
	Name       string
	Parent     string
	Fields     []RecordField
	Methods    []ProcDecl
	Properties  []PropertyDef
	Implements  []string // interfaces a class implements: class(TParent, IFoo, IBar)
	IsClass     bool     // `class` (reference type) vs `object` (value type)
	IsInterface bool     // `interface` type (reference, methods only)
	Packed      bool
}

// PropertyDef is `property Name: Type read ReadField write WriteField;`.
type PropertyDef struct {
	Base
	Name  string
	Read  string
	Write string
}

func (ObjectType) typeExpr() {}
func (o ObjectType) String() string {
	return "object"
}

type ConstrainedArrayType struct {
	Base
	Index   TypeExpr
	Open    bool
	Element TypeExpr
}

func (ConstrainedArrayType) typeExpr() {}
func (c ConstrainedArrayType) String() string {
	return "open array"
}

type FloatType struct {
	Base
	Kind string // Real, Single, Double, Extended, Comp
}

func (FloatType) typeExpr()        {}
func (f FloatType) String() string { return f.Kind }

type OrdType struct {
	Base
	Kind string // Integer, LongInt, ShortInt, Byte, Word, Boolean, Char
}

func (OrdType) typeExpr()        {}
func (o OrdType) String() string { return o.Kind }

// Walk utility used by codegen, debug, IDE.

func Walk(n Node, fn func(Node) bool) {
	if n == nil {
		return
	}
	if !fn(n) {
		return
	}
	switch v := n.(type) {
	case *Program:
		Walk(v.Uses, fn)
		Walk(v.Block, fn)
		Walk(v.Body, fn)
	case *Unit:
		Walk(v.Interface, fn)
		Walk(v.Implementation, fn)
		Walk(v.Init, fn)
	case *InterfaceSection:
		Walk(v.Uses, fn)
		for _, d := range v.Decls {
			Walk(d, fn)
		}
	case *ImplementationSection:
		Walk(v.Uses, fn)
		for _, d := range v.Decls {
			Walk(d, fn)
		}
	case *Block:
		for _, d := range v.Consts {
			Walk(d, fn)
		}
		for _, d := range v.Types {
			Walk(d, fn)
		}
		for _, d := range v.Vars {
			Walk(d, fn)
		}
		for _, d := range v.Procs {
			Walk(d, fn)
		}
		Walk(v.Body, fn)
	case *BlockBody:
		for _, s := range v.Stmts {
			Walk(s, fn)
		}
	case *CompoundStmt:
		for _, s := range v.Stmts {
			Walk(s, fn)
		}
	case *IfStmt:
		Walk(v.Cond, fn)
		Walk(v.Then, fn)
		Walk(v.Else, fn)
	case *CaseStmt:
		Walk(v.Expr, fn)
		for _, c := range v.Cases {
			for _, vv := range c.Values {
				Walk(vv, fn)
			}
			Walk(c.Body, fn)
		}
		Walk(v.Else, fn)
	case *WhileStmt:
		Walk(v.Cond, fn)
		Walk(v.Body, fn)
	case *RepeatStmt:
		Walk(v.Body, fn)
		Walk(v.Cond, fn)
	case *ForStmt:
		Walk(v.Lo, fn)
		Walk(v.Hi, fn)
		Walk(v.Body, fn)
	case *WithStmt:
		Walk(v.Rec, fn)
		Walk(v.Body, fn)
	case *AssignStmt:
		Walk(v.Dest, fn)
		Walk(v.Expr, fn)
	case *CallStmt:
		Walk(&v.Call, fn)
	case *InheritedStmt:
		if v.Call != nil {
			Walk(v.Call, fn)
		}
	case *HaltStmt:
		Walk(v.Code, fn)
	case *BinaryExpr:
		Walk(v.Left, fn)
		Walk(v.Right, fn)
	case *UnaryExpr:
		Walk(v.Expr, fn)
	case *CaretExpr:
		Walk(v.Expr, fn)
	case *AtExpr:
		Walk(v.Expr, fn)
	case *CallExpr:
		Walk(v.Func, fn)
		for _, a := range v.Args {
			Walk(a, fn)
		}
	case *FieldExpr:
		Walk(v.Expr, fn)
	case *IndexExpr:
		Walk(v.Expr, fn)
		Walk(v.Index, fn)
	case *TypeCastExpr:
		Walk(v.Type, fn)
		Walk(v.Expr, fn)
	case *SetExpr:
		for _, e := range v.Elements {
			Walk(e.Lo, fn)
			Walk(e.Hi, fn)
		}
	case *InExpr:
		Walk(v.Left, fn)
		Walk(v.Right, fn)
	case *ProcDecl:
		for _, p := range v.Params {
			Walk(p.Type, fn)
		}
		Walk(v.Body, fn)
	case *VarDecl:
		Walk(v.Type, fn)
		Walk(v.Abs, fn)
		Walk(v.Init, fn)
	case *ConstDecl:
		Walk(v.Type, fn)
		Walk(v.Value, fn)
	case *TypeDecl:
		Walk(v.Type, fn)
	}
}

// Children returns the immediate child nodes for IDE outline views.
func Children(n Node) []Node {
	switch v := n.(type) {
	case *Program:
		out := []Node{}
		if v.Block != nil {
			out = append(out, v.Block)
		}
		if v.Body != nil {
			out = append(out, v.Body)
		}
		return out
	case *Block:
		out := make([]Node, 0, len(v.Procs))
		for _, d := range v.Procs {
			out = append(out, d)
		}
		for _, d := range v.Types {
			out = append(out, d)
		}
		return out
	}
	return nil
}

// SortedNames is a helper for symbol browser UIs.
func SortedNames(ns []string) []string {
	out := append([]string(nil), ns...)
	sort.Strings(out)
	return out
}

// String returns a deterministic textual form of a node for snapshot
// tests and source maps.
func Dump(n Node) string {
	if n == nil {
		return "<nil>"
	}
	return n.String()
}
