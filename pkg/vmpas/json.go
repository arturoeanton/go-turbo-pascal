package vmpas

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
)

// registerJSON installs the JSON accessor builtins. JSON parsing is pure
// computation (no I/O), so it is always available — it needs no capability.
//
//	JsonValid(text): Boolean             -> is text well-formed JSON
//	JsonStr(text, path): string          -> value at path as string
//	JsonInt(text, path): Integer         -> value at path as integer
//	JsonBool(text, path): Boolean        -> value at path as boolean
//	JsonLen(text, path): Integer         -> length of the array/object at path
//
// Building (each returns the updated JSON document, creating intermediate
// objects/arrays as needed; start from '' or '{}'):
//
//	JsonSetStr(text, path, value): string
//	JsonSetInt(text, path, value): string
//	JsonSetBool(text, path, value): string
//	JsonEscape(value): string            -> the value JSON-encoded (with quotes)
//
// path is dot-separated; a numeric segment indexes into an array. An empty path
// addresses the root. Example: JsonStr(body, 'user.name'), JsonInt(body,
// 'items.0.id'), JsonSetStr('{}', 'user.name', 'alice').
func registerJSON(vm *ir.VM) {
	vm.Builtins["JsonValid"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		return ir.Value{Kind: ir.VKBool, Bool: json.Valid([]byte(jsonArg(a, 0)))}
	}
	vm.Builtins["JsonStr"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		v, _ := jsonLookup(jsonArg(a, 0), jsonArg(a, 1))
		return ir.Value{Kind: ir.VKStr, Str: jsonToStr(v)}
	}
	vm.Builtins["JsonInt"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		v, _ := jsonLookup(jsonArg(a, 0), jsonArg(a, 1))
		return ir.Value{Kind: ir.VKInt, Int: jsonToInt(v)}
	}
	vm.Builtins["JsonBool"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		v, _ := jsonLookup(jsonArg(a, 0), jsonArg(a, 1))
		b, _ := v.(bool)
		return ir.Value{Kind: ir.VKBool, Bool: b}
	}
	vm.Builtins["JsonLen"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		v, _ := jsonLookup(jsonArg(a, 0), jsonArg(a, 1))
		switch x := v.(type) {
		case []any:
			return ir.Value{Kind: ir.VKInt, Int: int64(len(x))}
		case map[string]any:
			return ir.Value{Kind: ir.VKInt, Int: int64(len(x))}
		}
		return ir.Value{Kind: ir.VKInt, Int: 0}
	}
	vm.Builtins["JsonSetStr"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		return ir.Value{Kind: ir.VKStr, Str: jsonSet(jsonArg(a, 0), jsonArg(a, 1), jsonArg(a, 2))}
	}
	vm.Builtins["JsonSetInt"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		return ir.Value{Kind: ir.VKStr, Str: jsonSet(jsonArg(a, 0), jsonArg(a, 1), toInt64(jsonArgVal(a, 2)))}
	}
	vm.Builtins["JsonSetBool"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		v := jsonArgVal(a, 2)
		return ir.Value{Kind: ir.VKStr, Str: jsonSet(jsonArg(a, 0), jsonArg(a, 1), v.Kind == ir.VKBool && v.Bool)}
	}
	vm.Builtins["JsonEscape"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		b, err := json.Marshal(jsonArg(a, 0))
		if err != nil {
			return ir.Value{Kind: ir.VKStr, Str: `""`}
		}
		return ir.Value{Kind: ir.VKStr, Str: string(b)}
	}
}

func jsonArgVal(a []ir.Value, i int) ir.Value {
	if i < len(a) {
		return a[i]
	}
	return ir.Value{Kind: ir.VKNil}
}

// jsonSet parses text (treating ''/invalid as an empty object), sets value at
// the dot-separated path creating intermediate objects/arrays as needed, and
// returns the re-encoded JSON document.
func jsonSet(text, path string, value any) string {
	var root any
	if strings.TrimSpace(text) == "" || json.Unmarshal([]byte(text), &root) != nil {
		root = map[string]any{}
	}
	if strings.TrimSpace(path) == "" {
		root = value
	} else {
		root = jsonSetPath(root, strings.Split(path, "."), value)
	}
	b, err := json.Marshal(root)
	if err != nil {
		return text
	}
	return string(b)
}

func jsonSetPath(node any, segs []string, value any) any {
	if len(segs) == 0 {
		return value
	}
	if idx, err := strconv.Atoi(segs[0]); err == nil && idx >= 0 {
		arr, _ := node.([]any)
		for len(arr) <= idx {
			arr = append(arr, nil)
		}
		arr[idx] = jsonSetPath(arr[idx], segs[1:], value)
		return arr
	}
	m, ok := node.(map[string]any)
	if !ok {
		m = map[string]any{}
	}
	m[segs[0]] = jsonSetPath(m[segs[0]], segs[1:], value)
	return m
}

func jsonArg(a []ir.Value, i int) string {
	if i < len(a) {
		return irToStr(a[i])
	}
	return ""
}

// jsonLookup parses text and walks the dot-separated path, indexing objects by
// key and arrays by numeric segment. It returns the located value and whether
// it was found.
func jsonLookup(text, path string) (any, bool) {
	var root any
	if err := json.Unmarshal([]byte(text), &root); err != nil {
		return nil, false
	}
	cur := root
	if strings.TrimSpace(path) == "" {
		return cur, true
	}
	for _, seg := range strings.Split(path, ".") {
		switch node := cur.(type) {
		case map[string]any:
			v, ok := node[seg]
			if !ok {
				return nil, false
			}
			cur = v
		case []any:
			idx, err := strconv.Atoi(seg)
			if err != nil || idx < 0 || idx >= len(node) {
				return nil, false
			}
			cur = node[idx]
		default:
			return nil, false
		}
	}
	return cur, true
}

func jsonToStr(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64)
	case bool:
		if x {
			return "true"
		}
		return "false"
	default:
		// Objects/arrays: re-encode compactly.
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(b)
	}
}

func jsonToInt(v any) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case bool:
		if x {
			return 1
		}
		return 0
	case string:
		n, _ := strconv.ParseInt(x, 10, 64)
		return n
	}
	return 0
}
