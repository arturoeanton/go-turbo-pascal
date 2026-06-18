package vmpas

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
)

// registerHostCaps installs the capability-gated host builtins. These are vmpas
// extensions (not part of the Turbo Pascal 7 RTL): each is registered only when
// its capability is granted, so under the default-deny sandbox calling one is a
// compile-time "unknown identifier" error rather than a silent no-op.
//
//	Env      GetEnv(name: string): string        -> os.Getenv
//	Exec     Exec(command: string): Integer       -> run via the system shell
//	Network  HttpGet(url: string): string         -> HTTP GET body (or empty)
//
// Names are registered in PascalCase; aliasLowercase mirrors them so codegen's
// lowercased externals resolve.
func (e *Engine) registerHostCaps(vm *ir.VM) {
	if e.caps.Env {
		vm.Builtins["GetEnv"] = func(_ *ir.VM, args []ir.Value) ir.Value {
			if len(args) == 0 {
				return ir.Value{Kind: ir.VKStr}
			}
			return ir.Value{Kind: ir.VKStr, Str: os.Getenv(irToStr(args[0]))}
		}
	}
	if e.caps.Exec {
		vm.Builtins["Exec"] = func(_ *ir.VM, args []ir.Value) ir.Value {
			if len(args) == 0 {
				return ir.Value{Kind: ir.VKInt, Int: -1}
			}
			var cmd *exec.Cmd
			if runtime.GOOS == "windows" {
				cmd = exec.Command("cmd", "/C", irToStr(args[0]))
			} else {
				cmd = exec.Command("/bin/sh", "-c", irToStr(args[0]))
			}
			code := 0
			if err := cmd.Run(); err != nil {
				if ee, ok := err.(*exec.ExitError); ok {
					code = ee.ExitCode()
				} else {
					code = -1
				}
			}
			return ir.Value{Kind: ir.VKInt, Int: int64(code)}
		}
	}
	if e.caps.Network {
		e.registerHTTP(vm)
	}
	if e.caps.Database && e.db != nil {
		e.registerDB(vm)
	}
}

// registerHTTP installs the HTTP client builtins (Network capability):
//
//	HttpGet(url): string                       -> response body
//	HttpPost(url, contentType, body): string   -> response body
//	HttpLastStatus: Integer                    -> status code of the last call
//
// The body is returned as-is (empty on transport error); the status code of the
// most recent call is available via HttpLastStatus (mirrors TP7's IOResult).
func (e *Engine) registerHTTP(vm *ir.VM) {
	do := func(resp *http.Response, err error) ir.Value {
		if err != nil {
			e.httpStatus = 0
			return ir.Value{Kind: ir.VKStr}
		}
		defer resp.Body.Close()
		e.httpStatus = resp.StatusCode
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return ir.Value{Kind: ir.VKStr}
		}
		return ir.Value{Kind: ir.VKStr, Str: string(body)}
	}
	vm.Builtins["HttpGet"] = func(_ *ir.VM, args []ir.Value) ir.Value {
		if len(args) == 0 {
			return ir.Value{Kind: ir.VKStr}
		}
		return do(http.Get(irToStr(args[0])))
	}
	vm.Builtins["HttpPost"] = func(_ *ir.VM, args []ir.Value) ir.Value {
		if len(args) < 3 {
			return ir.Value{Kind: ir.VKStr}
		}
		url, ctype, body := irToStr(args[0]), irToStr(args[1]), irToStr(args[2])
		return do(http.Post(url, ctype, strings.NewReader(body)))
	}
	vm.Builtins["HttpLastStatus"] = func(_ *ir.VM, _ []ir.Value) ir.Value {
		return ir.Value{Kind: ir.VKInt, Int: int64(e.httpStatus)}
	}
}

// registerDB installs the SQL builtins (Database capability + a handle from
// UseDB). The cursor API mirrors a Delphi-style dataset:
//
//	DbExec(sql [, params...]): Integer   -> affected rows (-1 on error)
//	DbOpen(sql [, params...]): Boolean   -> run a query; true if a row is ready
//	DbEof: Boolean                       -> true when past the last row
//	DbNext                               -> advance to the next row
//	DbFieldStr(i): string                -> current row, column i as string
//	DbFieldInt(i): Integer               -> current row, column i as integer
//	DbClose                              -> close the active cursor
//	DbError: string                      -> last error message ('' if none)
func (e *Engine) registerDB(vm *ir.VM) {
	vm.Builtins["DbExec"] = func(_ *ir.VM, args []ir.Value) ir.Value {
		if len(args) == 0 {
			return ir.Value{Kind: ir.VKInt, Int: -1}
		}
		n, err := e.db.Exec(irToStr(args[0]), dbParams(args[1:])...)
		if err != nil {
			e.dbErr = err.Error()
			return ir.Value{Kind: ir.VKInt, Int: -1}
		}
		e.dbErr = ""
		return ir.Value{Kind: ir.VKInt, Int: n}
	}
	vm.Builtins["DbOpen"] = func(_ *ir.VM, args []ir.Value) ir.Value {
		e.dbClose()
		if len(args) == 0 {
			return ir.Value{Kind: ir.VKBool, Bool: false}
		}
		rows, err := e.db.Query(irToStr(args[0]), dbParams(args[1:])...)
		if err != nil {
			e.dbErr = err.Error()
			return ir.Value{Kind: ir.VKBool, Bool: false}
		}
		e.dbErr = ""
		cols, _ := rows.Columns()
		e.cursor = &dbCursor{rows: rows, cols: cols}
		return ir.Value{Kind: ir.VKBool, Bool: e.dbAdvance()}
	}
	vm.Builtins["DbEof"] = func(_ *ir.VM, _ []ir.Value) ir.Value {
		return ir.Value{Kind: ir.VKBool, Bool: e.cursor == nil || !e.cursor.hasRow}
	}
	vm.Builtins["DbNext"] = func(_ *ir.VM, _ []ir.Value) ir.Value {
		e.dbAdvance()
		return ir.Value{Kind: ir.VKNil}
	}
	vm.Builtins["DbFieldStr"] = func(_ *ir.VM, args []ir.Value) ir.Value {
		return ir.Value{Kind: ir.VKStr, Str: anyToStr(e.dbField(args))}
	}
	vm.Builtins["DbFieldInt"] = func(_ *ir.VM, args []ir.Value) ir.Value {
		return ir.Value{Kind: ir.VKInt, Int: anyToInt(e.dbField(args))}
	}
	vm.Builtins["DbClose"] = func(_ *ir.VM, _ []ir.Value) ir.Value {
		e.dbClose()
		return ir.Value{Kind: ir.VKNil}
	}
	vm.Builtins["DbError"] = func(_ *ir.VM, _ []ir.Value) ir.Value {
		return ir.Value{Kind: ir.VKStr, Str: e.dbErr}
	}
}

// dbCursor is an open query result positioned on the current row.
type dbCursor struct {
	rows   SQLRows
	cols   []string
	vals   []any
	hasRow bool
}

// dbAdvance moves the cursor to the next row, scanning its values. It returns
// whether a row is now available.
func (e *Engine) dbAdvance() bool {
	c := e.cursor
	if c == nil || c.rows == nil || !c.rows.Next() {
		if c != nil {
			c.hasRow = false
		}
		return false
	}
	dest := make([]any, len(c.cols))
	ptrs := make([]any, len(c.cols))
	for i := range dest {
		ptrs[i] = &dest[i]
	}
	if err := c.rows.Scan(ptrs...); err != nil {
		e.dbErr = err.Error()
		c.hasRow = false
		return false
	}
	c.vals = dest
	c.hasRow = true
	return true
}

// dbField returns the value of the current row's column given the builtin args.
func (e *Engine) dbField(args []ir.Value) any {
	if e.cursor == nil || !e.cursor.hasRow || len(args) == 0 {
		return nil
	}
	i := int(toInt64(args[0]))
	if i < 0 || i >= len(e.cursor.vals) {
		return nil
	}
	return e.cursor.vals[i]
}

func (e *Engine) dbClose() {
	if e.cursor != nil && e.cursor.rows != nil {
		e.cursor.rows.Close()
	}
	e.cursor = nil
}

// dbParams converts Pascal argument values to Go query parameters.
func dbParams(args []ir.Value) []any {
	out := make([]any, len(args))
	for i, a := range args {
		switch a.Kind {
		case ir.VKInt:
			out[i] = a.Int
		case ir.VKReal:
			out[i] = a.Real
		case ir.VKBool:
			out[i] = a.Bool
		case ir.VKChar:
			out[i] = string(rune(a.Ch))
		default:
			out[i] = a.Str
		}
	}
	return out
}

// toInt64 extracts an integer from an IR value.
func toInt64(v ir.Value) int64 {
	switch v.Kind {
	case ir.VKInt:
		return v.Int
	case ir.VKReal:
		return int64(v.Real)
	case ir.VKChar:
		return int64(v.Ch)
	}
	return 0
}

// anyToStr renders a scanned database value as a string.
func anyToStr(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case []byte:
		return string(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64)
	case bool:
		if x {
			return "true"
		}
		return "false"
	}
	return fmt.Sprintf("%v", v)
}

// anyToInt coerces a scanned database value to an integer.
func anyToInt(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case float64:
		return int64(x)
	case bool:
		if x {
			return 1
		}
		return 0
	case []byte:
		n, _ := strconv.ParseInt(string(x), 10, 64)
		return n
	case string:
		n, _ := strconv.ParseInt(x, 10, 64)
		return n
	}
	return 0
}

// irToStr extracts a string from an IR value (string or char).
func irToStr(v ir.Value) string {
	switch v.Kind {
	case ir.VKStr:
		return v.Str
	case ir.VKChar:
		return string(rune(v.Ch))
	}
	return ""
}
