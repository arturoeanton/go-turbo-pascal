package codegen

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
	"github.com/arturoeanton/go-turbo-pascal/internal/rtl/crt"
	"github.com/arturoeanton/go-turbo-pascal/internal/rtl/system"
)

// builtinFuncs are RTL functions that codegen lowers directly to
// OPCallBuiltin. They are pure value-returning helpers (procedures that
// mutate var arguments, like Inc/Dec, are handled specially in codegen).
var builtinFuncs = map[string]string{
	"ord":          "Ord",
	"chr":          "Chr",
	"abs":          "Abs",
	"sqr":          "Sqr",
	"sqrt":         "Sqrt",
	"sin":          "Sin",
	"cos":          "Cos",
	"arctan":       "ArcTan",
	"ln":           "Ln",
	"exp":          "Exp",
	"trunc":        "Trunc",
	"round":        "Round",
	"frac":         "Frac",
	"int":          "Int",
	"odd":          "Odd",
	"succ":         "Succ",
	"pred":         "Pred",
	"upcase":       "UpCase",
	"length":       "Length",
	"copy":         "Copy",
	"pos":          "Pos",
	"concat":       "Concat",
	"hi":           "Hi",
	"lo":           "Lo",
	"high":         "High",
	"low":          "Low",
	"inttostr":     "IntToStr",
	"strtoint":     "StrToInt",
	"strtointdef":  "StrToIntDef",
	"floattostr":   "FloatToStr",
	"strtofloat":   "StrToFloat",
	"uppercase":    "UpperCase",
	"lowercase":    "LowerCase",
	"trim":         "Trim",
	"trimleft":     "TrimLeft",
	"trimright":    "TrimRight",
	"stringofchar": "StringOfChar",
	"sqrtf":        "Sqrt",
	"eof":          "eof",
	"filesize":     "filesize",
	"filepos":      "filepos",
	"seek":         "seek",
	// Business stdlib (N3).
	"currtostr":    "CurrToStr",
	"strtocurr":    "StrToCurr",
	"min":          "Min",
	"max":          "Max",
	"clamp":        "Clamp",
	"contains":     "Contains",
	"startswith":   "StartsWith",
	"endswith":     "EndsWith",
	"isempty":      "IsEmpty",
	"dateyear":     "DateYear",
	"datemonth":    "DateMonth",
	"dateday":      "DateDay",
	"dateadddays":  "DateAddDays",
	"datediffdays": "DateDiffDays",
	"datevalid":    "DateValid",
}

func isBuiltinFunc(name string) bool {
	_, ok := builtinFuncs[strings.ToLower(name)]
	return ok
}

func canonicalBuiltin(name string) string {
	if c, ok := builtinFuncs[strings.ToLower(name)]; ok {
		return c
	}
	return name
}

// NewVM builds a ready-to-run VM for a program: it installs the System unit,
// program arguments, console/file I/O builtins and input, but does not execute
// anything. Use it for debugging (see ir.Debugger) or custom drivers.
func NewVM(prog *ir.Program, args []string, input string) *ir.VM {
	vm := ir.NewVM(prog)
	vm.Args = args
	if input != "" {
		vm.Input = &ir.Input{Lines: strings.Split(strings.TrimRight(input, "\n"), "\n")}
	}
	system.Register(vm)
	system.SetArguments(vm, args)
	crt.Register(vm)
	registerIO(vm)
	aliasLowercase(vm)
	return vm
}

// aliasLowercase adds a lowercase alias for every builtin so that calls lowered
// to lowercase names (RTL procedures via `uses`) resolve regardless of the
// RTL's PascalCase registration.
func aliasLowercase(vm *ir.VM) {
	keys := make([]string, 0, len(vm.Builtins))
	for k := range vm.Builtins {
		keys = append(keys, k)
	}
	for _, k := range keys {
		low := strings.ToLower(k)
		if _, ok := vm.Builtins[low]; !ok {
			vm.Builtins[low] = vm.Builtins[k]
		}
	}
}

// Run is the convenience runner used by tests and embedders: it executes a
// program and returns its standard output and exit code.
func Run(prog *ir.Program, args []string, input string) (string, int, error) {
	vm := NewVM(prog, args, input)
	vm.Run()
	if vm.RuntimeError != 0 {
		return vm.Output.String(), vm.ExitCode, fmt.Errorf("runtime error %d", vm.RuntimeError)
	}
	return vm.Output.String(), vm.ExitCode, nil
}

// RunInteractive executes a program with live console I/O: output is written to
// stdout as it is produced (so prompts appear before reads) and Read/ReadLn
// consume lines lazily from stdin. It returns the exit code.
func RunInteractive(prog *ir.Program, args []string, stdin io.Reader, stdout io.Writer) (int, error) {
	vm := NewVM(prog, args, "")
	vm.Output.W = stdout
	if stdin != nil {
		vm.Input = &ir.Input{Reader: bufio.NewReader(stdin)}
	}
	vm.Run()
	if vm.RuntimeError != 0 {
		return vm.ExitCode, fmt.Errorf("runtime error %d", vm.RuntimeError)
	}
	return vm.ExitCode, nil
}

// registerIO installs type-aware Write/WriteLn that match TP7 default
// formatting for the common scalar types.
func registerIO(vm *ir.VM) {
	vm.Builtins["write"] = func(vm *ir.VM, args []ir.Value) ir.Value {
		for _, a := range args {
			vm.Output.WriteString(formatWrite(a))
		}
		return ir.Value{Kind: ir.VKNil}
	}
	vm.Builtins["writeln"] = func(vm *ir.VM, args []ir.Value) ir.Value {
		for _, a := range args {
			vm.Output.WriteString(formatWrite(a))
		}
		vm.Output.WriteString("\n")
		return ir.Value{Kind: ir.VKNil}
	}
	vm.Builtins["read"] = readBuiltin
	vm.Builtins["readln"] = readBuiltin
	vm.Builtins["__writefmt"] = func(vm *ir.VM, args []ir.Value) ir.Value {
		return ir.Value{Kind: ir.VKStr, Str: formatField(args)}
	}
	registerFileIO(vm)
}

// formatField renders a Write argument with TP7 field formatting:
// args = [value, width, decimals] where width/decimals are -1 when absent.
func formatField(args []ir.Value) string {
	if len(args) < 3 {
		if len(args) > 0 {
			return formatWrite(args[0])
		}
		return ""
	}
	val := args[0]
	width := int(args[1].Int)
	dec := int(args[2].Int)
	var s string
	if val.Kind == ir.VKReal && dec >= 0 {
		s = strconv.FormatFloat(val.Real, 'f', dec, 64)
	} else {
		s = formatWrite(val)
	}
	if width > len(s) {
		s = strings.Repeat(" ", width-len(s)) + s
	}
	return s
}

// readBuiltin reads one input line and assigns whitespace-separated tokens to
// the reference arguments, parsing each by the target cell's type. A single
// string target captures the whole line.
func readBuiltin(vm *ir.VM, args []ir.Value) ir.Value {
	line := vm.Input.ReadLine()
	fields := strings.Fields(line)
	fi := 0
	for _, ref := range args {
		if ref.Kind != ir.VKPtr || ref.Cell == nil {
			continue
		}
		cell := ref.Cell
		switch cell.Kind {
		case ir.VKStr:
			if len(args) == 1 {
				cell.Str = line
			} else if fi < len(fields) {
				cell.Str = fields[fi]
				fi++
			}
		case ir.VKReal:
			if fi < len(fields) {
				f, _ := strconv.ParseFloat(fields[fi], 64)
				cell.Real = f
				fi++
			}
		case ir.VKChar:
			if len(line) > 0 {
				cell.Ch = line[0]
			}
		default: // integer
			if fi < len(fields) {
				n, _ := strconv.ParseInt(fields[fi], 10, 64)
				cell.Int = n
				cell.Kind = ir.VKInt
				fi++
			}
		}
	}
	return ir.Value{Kind: ir.VKNil}
}

// formatWrite renders a value the way TP7's Write does by default.
func formatWrite(v ir.Value) string {
	switch v.Kind {
	case ir.VKInt:
		return strconv.FormatInt(v.Int, 10)
	case ir.VKBool:
		if v.Bool {
			return "TRUE"
		}
		return "FALSE"
	case ir.VKChar:
		return string([]byte{v.Ch})
	case ir.VKStr:
		return v.Str
	case ir.VKReal:
		return formatRealTP7(v.Real)
	case ir.VKNil:
		return ""
	}
	return v.String()
}

// formatRealTP7 mimics TP7's default Real output: a leading space for
// non-negative values followed by scientific notation with a two-digit
// exponent (e.g. " 3.1400000000E+00").
func formatRealTP7(r float64) string {
	s := strconv.FormatFloat(r, 'E', 10, 64)
	// Go emits E+02 already with at least two exponent digits.
	if r >= 0 {
		s = " " + s
	}
	return s
}
