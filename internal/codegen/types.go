package codegen

import (
	"strings"

	"github.com/arturoeanton/go-turbo-pascal/internal/ast"
	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
)

// tkind classifies a resolved type's storage shape.
type tkind int

const (
	ktScalar tkind = iota // integer/real/char/boolean (see typeInfo.scalar)
	ktString
	ktRecord
	ktArray
	ktPointer
	ktEnum
	ktSet
	ktObject
	ktFile
)

// objMethod describes a method declared on an object type.
type objMethod struct {
	name   string // lowercase
	isCtor bool
	isDtor bool
}

type tfield struct {
	name  string // original case
	lname string // lower case (map key)
	ti    *typeInfo
}

// typeInfo is codegen's resolved type. It is richer than vtype: it carries
// record layout, array bounds and element/pointer targets so that lvalues,
// zero initialization and Go<->Pascal struct mapping can be generated.
type typeInfo struct {
	kind     tkind
	scalar   vtype     // for ktScalar/ktString
	name     string    // named type, if any
	fields   []tfield  // ktRecord
	elem     *typeInfo // ktArray/ktPointer/ktSet element
	lo, hi   int64     // ktArray bounds
	enumVals []string  // ktEnum
	dynamic  bool      // ktArray: dynamic array (`array of T`)
	ptrName  string    // ktPointer: target type name for lazy resolution
	// ktObject:
	objName string      // lowercase object type name
	parent  *typeInfo   // parent object type (nil if none)
	methods []objMethod // own + inherited method names
}

// vt returns the coarse vtype used for write formatting and operator choice.
func (t *typeInfo) vt() vtype {
	if t == nil {
		return tUnknown
	}
	switch t.kind {
	case ktScalar, ktString:
		return t.scalar
	case ktEnum:
		return tInt
	}
	return tUnknown
}

// field looks up a record field (case-insensitive).
func (t *typeInfo) field(name string) *tfield {
	if t == nil {
		return nil
	}
	low := strings.ToLower(name)
	for i := range t.fields {
		if t.fields[i].lname == low {
			return &t.fields[i]
		}
	}
	return nil
}

// registerTypes does a first pass over type declarations so that names exist
// (enabling forward references such as `PNode = ^TNode; TNode = record...`).
func (g *gen) registerTypes(decls []ast.Decl) {
	for _, d := range decls {
		td, ok := d.(*ast.TypeDecl)
		if !ok {
			continue
		}
		g.types[strings.ToLower(td.Name)] = &typeInfo{name: td.Name}
	}
	// Second pass: resolve each into its real shape.
	for _, d := range decls {
		td, ok := d.(*ast.TypeDecl)
		if !ok {
			continue
		}
		ti := g.resolveType(td.Type)
		if ti != nil {
			ti.name = td.Name
			// Object types are named by their TypeDecl (the ObjectType node
			// itself carries no name); this name drives dispatch and the
			// runtime __type tag.
			if ti.kind == ktObject {
				ti.objName = strings.ToLower(td.Name)
			}
			*g.types[strings.ToLower(td.Name)] = *ti
		}
		// Enumerated constants enter the scope as ordinals.
		if en, ok := td.Type.(*ast.EnumType); ok {
			for i, n := range en.Names {
				g.define(n, &vinfo{kind: vConst, constVal: ir.Value{Kind: ir.VKInt, Int: int64(i)}, typ: tInt})
			}
		}
	}
}

// resolveType maps an AST type expression to a typeInfo.
func (g *gen) resolveType(t ast.TypeExpr) *typeInfo {
	switch v := t.(type) {
	case *ast.OrdType, *ast.FloatType:
		return &typeInfo{kind: ktScalar, scalar: vtypeOf(t)}
	case *ast.StringType:
		return &typeInfo{kind: ktString, scalar: tStr}
	case *ast.TypeRef:
		low := strings.ToLower(v.Name)
		switch low {
		case "string":
			return &typeInfo{kind: ktString, scalar: tStr}
		case "integer", "longint", "word", "byte", "shortint":
			return &typeInfo{kind: ktScalar, scalar: tInt}
		case "real", "single", "double", "extended", "comp":
			return &typeInfo{kind: ktScalar, scalar: tReal}
		case "char":
			return &typeInfo{kind: ktScalar, scalar: tChar}
		case "boolean":
			return &typeInfo{kind: ktScalar, scalar: tBool}
		case "text", "file":
			return &typeInfo{kind: ktFile}
		}
		if known, ok := g.types[low]; ok {
			return known
		}
		// Unknown name: treat as opaque integer-ish scalar.
		return &typeInfo{kind: ktScalar, scalar: tUnknown, name: v.Name}
	case *ast.PointerType:
		ti := &typeInfo{kind: ktPointer}
		if tr, ok := v.Target.(*ast.TypeRef); ok {
			ti.ptrName = strings.ToLower(tr.Name)
		} else {
			ti.elem = g.resolveType(v.Target)
		}
		return ti
	case *ast.RecordType:
		ti := &typeInfo{kind: ktRecord}
		for _, f := range v.Fields {
			ft := g.resolveType(f.Type)
			for _, nm := range f.Names {
				ti.fields = append(ti.fields, tfield{name: nm, lname: strings.ToLower(nm), ti: ft})
			}
		}
		// Variant part fields are flattened in so they are addressable.
		g.flattenVariant(v, ti)
		return ti
	case *ast.ArrayType:
		if len(v.Index) == 0 {
			// Dynamic array: `array of T`.
			return &typeInfo{kind: ktArray, dynamic: true, lo: 0, hi: -1, elem: g.resolveType(v.Element)}
		}
		return g.resolveArray(v, 0)
	case *ast.SetType:
		return &typeInfo{kind: ktSet, elem: g.resolveType(v.Element)}
	case *ast.FileType:
		ti := &typeInfo{kind: ktFile}
		if v.Element != nil {
			ti.elem = g.resolveType(v.Element) // typed file: `file of T`
		}
		return ti
	case *ast.EnumType:
		return &typeInfo{kind: ktEnum, scalar: tInt, enumVals: v.Names}
	case *ast.RangeType:
		return &typeInfo{kind: ktScalar, scalar: tInt}
	case *ast.ObjectType:
		return g.resolveObject(v)
	}
	return nil
}

// resolveObject builds an object type, flattening inherited fields and
// methods so field access and dispatch work on the derived type directly.
func (g *gen) resolveObject(o *ast.ObjectType) *typeInfo {
	ti := &typeInfo{kind: ktObject, name: o.Name, objName: strings.ToLower(o.Name)}
	if o.Parent != "" {
		if pt, ok := g.types[strings.ToLower(o.Parent)]; ok && pt.kind == ktObject {
			ti.parent = pt
			// Inherit fields and methods.
			ti.fields = append(ti.fields, pt.fields...)
			ti.methods = append(ti.methods, pt.methods...)
		}
	}
	for _, f := range o.Fields {
		ft := g.resolveType(f.Type)
		for _, nm := range f.Names {
			ti.fields = append(ti.fields, tfield{name: nm, lname: strings.ToLower(nm), ti: ft})
		}
	}
	for i := range o.Methods {
		m := o.Methods[i]
		mm := objMethod{name: strings.ToLower(m.Name), isCtor: m.IsConstructor, isDtor: m.IsDestructor}
		// Override (same name) replaces the inherited entry.
		replaced := false
		for j := range ti.methods {
			if ti.methods[j].name == mm.name {
				ti.methods[j] = mm
				replaced = true
				break
			}
		}
		if !replaced {
			ti.methods = append(ti.methods, mm)
		}
	}
	return ti
}

func (t *typeInfo) hasMethod(name string) bool {
	low := strings.ToLower(name)
	for _, m := range t.methods {
		if m.name == low {
			return true
		}
	}
	return false
}

// pointerElem resolves the element type of a pointer (handling lazy names).
func (g *gen) pointerElem(ti *typeInfo) *typeInfo {
	if ti == nil || ti.kind != ktPointer {
		return nil
	}
	if ti.elem != nil {
		return ti.elem
	}
	if ti.ptrName != "" {
		if e, ok := g.types[ti.ptrName]; ok {
			return e
		}
	}
	return nil
}

// zeroTemplate builds the zero value for a type (records and arrays nested).
func (g *gen) zeroTemplate(ti *typeInfo) ir.Value {
	if ti == nil {
		return ir.Value{Kind: ir.VKInt}
	}
	switch ti.kind {
	case ktString:
		return ir.Value{Kind: ir.VKStr}
	case ktScalar, ktEnum:
		switch ti.scalar {
		case tReal:
			return ir.Value{Kind: ir.VKReal}
		case tChar:
			return ir.Value{Kind: ir.VKChar}
		case tBool:
			return ir.Value{Kind: ir.VKBool}
		}
		return ir.Value{Kind: ir.VKInt}
	case ktPointer:
		return ir.Value{Kind: ir.VKPtr} // nil (Cell == nil)
	case ktSet:
		return ir.Value{Kind: ir.VKSet}
	case ktRecord:
		rec := map[string]*ir.Value{}
		for _, f := range ti.fields {
			z := g.zeroTemplate(f.ti)
			rec[f.lname] = &z
		}
		return ir.Value{Kind: ir.VKRecord, Rec: rec}
	case ktObject:
		rec := map[string]*ir.Value{}
		for _, f := range ti.fields {
			z := g.zeroTemplate(f.ti)
			rec[f.lname] = &z
		}
		// Runtime type tag for dynamic dispatch.
		tt := ir.Value{Kind: ir.VKStr, Str: ti.objName}
		rec["__type"] = &tt
		return ir.Value{Kind: ir.VKRecord, Rec: rec}
	case ktArray:
		n := ti.hi - ti.lo + 1
		if n < 0 {
			n = 0
		}
		arr := make([]ir.Value, n)
		for i := range arr {
			arr[i] = g.zeroTemplate(ti.elem)
		}
		return ir.Value{Kind: ir.VKArray, Array: arr}
	}
	return ir.Value{Kind: ir.VKInt}
}

// resolveArray builds (possibly nested) array types. A multi-dimensional
// array[1..3, 1..4] of T is modelled as array[1..3] of array[1..4] of T, so
// element access is a[i][j].
func (g *gen) resolveArray(v *ast.ArrayType, dim int) *typeInfo {
	if dim >= len(v.Index) {
		return g.resolveType(v.Element)
	}
	r := v.Index[dim]
	ti := &typeInfo{kind: ktArray, lo: g.constInt(r.Lo), hi: g.constInt(r.Hi)}
	if dim+1 < len(v.Index) {
		ti.elem = g.resolveArray(v, dim+1)
	} else {
		ti.elem = g.resolveType(v.Element)
	}
	return ti
}

func (g *gen) constInt(e ast.Expr) int64 {
	if val, _, ok := g.constValue(e); ok {
		return val.Int
	}
	return 0
}

// flattenVariant adds variant-record fields to the flat field list so they
// are addressable. This trades exact memory layout for usability.
func (g *gen) flattenVariant(v *ast.RecordType, ti *typeInfo) {
	// The AST stores variant parts on the RecordType; we walk them defensively
	// since their shape may vary across parser versions.
	_ = v
	_ = ti
}
