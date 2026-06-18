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
	ktFunc // procedural type / closure (procedure or function value)
	ktChan // channel (Channel<T>); concurrency
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
	objName string              // lowercase object type name
	parent  *typeInfo           // parent object type (nil if none)
	methods []objMethod         // own + inherited method names
	isClass bool                // `class` (reference type) vs `object` (value type)
	props   map[string]propInfo // property name (lower) -> backing fields
	// ktFunc:
	isFunc bool // procedural type returns a value (function vs procedure)
	// helper (record/class helper for Base):
	helperFor string // lowercase name of the extended type ("" if not a helper)
}

// propInfo maps a property to its read/write backing fields.
type propInfo struct {
	read  string
	write string
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
		// Generic type parameters are erased: each resolves to an "any" type
		// (the VM is dynamically typed, so T behaves like a dynamic value).
		for _, tp := range td.TypeParams {
			g.registerTypeParam(tp)
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
			// A helper registers its methods against the extended type so that
			// `value.Method` on that type resolves to the helper (static call).
			if ti.helperFor != "" {
				if g.helpers[ti.helperFor] == nil {
					g.helpers[ti.helperFor] = map[string]string{}
				}
				for _, m := range ti.methods {
					g.helpers[ti.helperFor][m.name] = ti.objName + "." + m.name
				}
			}
		}
		// Enumerated constants enter the scope as ordinals.
		if en, ok := td.Type.(*ast.EnumType); ok {
			for i, n := range en.Names {
				g.define(n, &vinfo{kind: vConst, constVal: ir.Value{Kind: ir.VKInt, Int: int64(i)}, typ: tInt})
			}
		}
		// Sum-type variants register as ADT constructors (name -> arity).
		if st, ok := td.Type.(*ast.SumType); ok {
			for _, v := range st.Variants {
				g.adtCtors[strings.ToLower(v.Name)] = len(v.Fields)
			}
		}
	}
}

// registerTypeParam registers a generic type-parameter name as an erased "any"
// type so references to it inside the generic declaration resolve.
func (g *gen) registerTypeParam(name string) {
	low := strings.ToLower(name)
	if _, exists := g.types[low]; !exists {
		g.types[low] = &typeInfo{kind: ktScalar, scalar: tUnknown, name: name}
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
		case "string", "ansistring", "shortstring", "widestring", "unicodestring", "pchar":
			return &typeInfo{kind: ktString, scalar: tStr}
		case "integer", "longint", "word", "byte", "shortint",
			"cardinal", "longword", "smallint", "int64", "qword", "nativeint", "nativeuint":
			return &typeInfo{kind: ktScalar, scalar: tInt}
		case "real", "single", "double", "extended", "comp":
			return &typeInfo{kind: ktScalar, scalar: tReal}
		case "char", "ansichar", "widechar":
			return &typeInfo{kind: ktScalar, scalar: tChar}
		case "boolean", "bytebool", "wordbool", "longbool":
			return &typeInfo{kind: ktScalar, scalar: tBool}
		case "currency":
			return &typeInfo{kind: ktScalar, scalar: tCurrency}
		case "channel":
			return &typeInfo{kind: ktChan}
		case "text", "file":
			return &typeInfo{kind: ktFile}
		}
		if known, ok := g.types[low]; ok {
			return known
		}
		// Genuinely undefined type name (typo, missing declaration). Report it
		// but keep going with an opaque type so later errors still surface.
		g.errfAt(v.Pos(), "unknown type %q", v.Name)
		return &typeInfo{kind: ktScalar, scalar: tUnknown, name: v.Name}
	case *ast.PointerType:
		ti := &typeInfo{kind: ktPointer}
		if tr, ok := v.Target.(*ast.TypeRef); ok {
			ti.ptrName = strings.ToLower(tr.Name)
		} else {
			ti.elem = g.resolveType(v.Target)
		}
		return ti
	case *ast.ProcType:
		return &typeInfo{kind: ktFunc, isFunc: v.IsFunc}
	case *ast.SumType:
		// A sum value is a tagged record at runtime; the type carries no fields.
		return &typeInfo{kind: ktRecord}
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
	ti := &typeInfo{kind: ktObject, name: o.Name, objName: strings.ToLower(o.Name), isClass: o.IsClass, helperFor: strings.ToLower(o.HelperFor), props: map[string]propInfo{}}
	if o.Parent != "" {
		if pt, ok := g.types[strings.ToLower(o.Parent)]; ok && pt.kind == ktObject {
			ti.parent = pt
			// Inherit fields, methods and properties.
			ti.fields = append(ti.fields, pt.fields...)
			ti.methods = append(ti.methods, pt.methods...)
			for k, v := range pt.props {
				ti.props[k] = v
			}
		}
	}
	for _, pr := range o.Properties {
		r, w := pr.Read, pr.Write
		if r == "" {
			r = w
		}
		if w == "" {
			w = r
		}
		ti.props[strings.ToLower(pr.Name)] = propInfo{read: r, write: w}
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

// classInstanceTemplate builds the zero record value for a class instance
// (used by Create before allocating it on the heap).
func (g *gen) classInstanceTemplate(ti *typeInfo) ir.Value {
	rec := map[string]*ir.Value{}
	for _, f := range ti.fields {
		z := g.zeroTemplate(f.ti)
		rec[f.lname] = &z
	}
	tt := ir.Value{Kind: ir.VKStr, Str: ti.objName}
	rec["__type"] = &tt
	return ir.Value{Kind: ir.VKRecord, Rec: rec}
}

// backingField maps a property name to its backing field; non-properties pass
// through unchanged.
func (t *typeInfo) backingField(name string) string {
	if t != nil && t.props != nil {
		if pr, ok := t.props[strings.ToLower(name)]; ok {
			if pr.read != "" {
				return pr.read
			}
			return pr.write
		}
	}
	return name
}

// prop returns the property definition for name (case-insensitive) if t is an
// object type that declares it.
func (t *typeInfo) prop(name string) (propInfo, bool) {
	if t == nil || t.props == nil {
		return propInfo{}, false
	}
	pr, ok := t.props[strings.ToLower(name)]
	return pr, ok
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
		case tCurrency:
			return ir.Value{Kind: ir.VKCurrency}
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
		if ti.isClass {
			return ir.Value{Kind: ir.VKPtr} // a class variable is a nil reference
		}
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
	case ktFunc:
		return ir.Value{Kind: ir.VKFunc} // unassigned procedural value (nil)
	case ktChan:
		return ir.Value{Kind: ir.VKChan} // unassigned channel (nil until MakeChan)
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

// flattenVariant adds the variant part of a record (the selector field and the
// fields of every case) to the flat field list so they are all addressable. We
// trade the exact union memory layout for usability: every variant field exists
// independently rather than overlapping in storage. Field names are unique
// within a record, so a name appearing in several cases is added once.
func (g *gen) flattenVariant(v *ast.RecordType, ti *typeInfo) {
	if v.Variant == nil {
		return
	}
	// Named selector field, e.g. `case kind: TKind of`.
	if id, ok := v.Variant.Tag.(*ast.Ident); ok && id.Name != "" && v.Variant.TagType != nil {
		g.addFieldUnique(ti, id.Name, g.resolveType(v.Variant.TagType))
	}
	for _, c := range v.Variant.Cases {
		for _, f := range c.Fields {
			ft := g.resolveType(f.Type)
			for _, nm := range f.Names {
				g.addFieldUnique(ti, nm, ft)
			}
		}
	}
}

// addFieldUnique appends a record field unless one with the same (lowercased)
// name already exists.
func (g *gen) addFieldUnique(ti *typeInfo, name string, ft *typeInfo) {
	lname := strings.ToLower(name)
	for _, f := range ti.fields {
		if f.lname == lname {
			return
		}
	}
	ti.fields = append(ti.fields, tfield{name: name, lname: lname, ti: ft})
}
