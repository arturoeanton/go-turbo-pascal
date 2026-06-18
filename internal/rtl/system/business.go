package system

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
)

const isoDate = "2006-01-02"

// registerBusiness installs the business-rules standard library: money
// formatting/parsing, numeric helpers, string predicates and (deterministic)
// date helpers. Dates operate on ISO strings ("YYYY-MM-DD") and never read the
// clock — the current date should be injected by the host, so rule evaluation
// stays reproducible/auditable.
func registerBusiness(vm *ir.VM) {
	// --- money ---
	vm.Builtins["CurrToStr"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		return strVal(toCurr(arg(a, 0)).String())
	}
	vm.Builtins["StrToCurr"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		f, _ := strconv.ParseFloat(strings.TrimSpace(valueToString(arg(a, 0))), 64)
		return ir.Value{Kind: ir.VKCurrency, Int: int64(math.Round(f * ir.CurrencyScale))}
	}

	// --- numbers (preserve the operand kind: int/real/currency) ---
	vm.Builtins["Min"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		if num(arg(a, 1)) < num(arg(a, 0)) {
			return arg(a, 1)
		}
		return arg(a, 0)
	}
	vm.Builtins["Max"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		if num(arg(a, 1)) > num(arg(a, 0)) {
			return arg(a, 1)
		}
		return arg(a, 0)
	}
	vm.Builtins["Clamp"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		x, lo, hi := arg(a, 0), arg(a, 1), arg(a, 2)
		if num(x) < num(lo) {
			return lo
		}
		if num(x) > num(hi) {
			return hi
		}
		return x
	}

	// --- string predicates ---
	vm.Builtins["Contains"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		return boolean(strings.Contains(valueToString(arg(a, 0)), valueToString(arg(a, 1))))
	}
	vm.Builtins["StartsWith"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		return boolean(strings.HasPrefix(valueToString(arg(a, 0)), valueToString(arg(a, 1))))
	}
	vm.Builtins["EndsWith"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		return boolean(strings.HasSuffix(valueToString(arg(a, 0)), valueToString(arg(a, 1))))
	}
	vm.Builtins["IsEmpty"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		return boolean(strings.TrimSpace(valueToString(arg(a, 0))) == "")
	}

	// --- dates (ISO "YYYY-MM-DD", deterministic) ---
	vm.Builtins["DateYear"] = datePart(func(t time.Time) int64 { return int64(t.Year()) })
	vm.Builtins["DateMonth"] = datePart(func(t time.Time) int64 { return int64(t.Month()) })
	vm.Builtins["DateDay"] = datePart(func(t time.Time) int64 { return int64(t.Day()) })
	vm.Builtins["DateAddDays"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		t, err := time.Parse(isoDate, valueToString(arg(a, 0)))
		if err != nil {
			return strVal("")
		}
		return strVal(t.AddDate(0, 0, int(irToInt(arg(a, 1)))).Format(isoDate))
	}
	vm.Builtins["DateDiffDays"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		t1, e1 := time.Parse(isoDate, valueToString(arg(a, 0)))
		t2, e2 := time.Parse(isoDate, valueToString(arg(a, 1)))
		if e1 != nil || e2 != nil {
			return ir.Value{Kind: ir.VKInt}
		}
		return ir.Value{Kind: ir.VKInt, Int: int64(t2.Sub(t1).Hours() / 24)}
	}
	vm.Builtins["DateValid"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		_, err := time.Parse(isoDate, valueToString(arg(a, 0)))
		return boolean(err == nil)
	}
}

func datePart(f func(time.Time) int64) ir.Builtin {
	return func(_ *ir.VM, a []ir.Value) ir.Value {
		t, err := time.Parse(isoDate, valueToString(arg(a, 0)))
		if err != nil {
			return ir.Value{Kind: ir.VKInt}
		}
		return ir.Value{Kind: ir.VKInt, Int: f(t)}
	}
}

// arg returns the i-th argument or a nil value.
func arg(a []ir.Value, i int) ir.Value {
	if i < len(a) {
		return a[i]
	}
	return ir.Value{Kind: ir.VKNil}
}

// num reads any numeric value (int/real/currency) as a float for comparison.
func num(v ir.Value) float64 {
	if v.Kind == ir.VKCurrency {
		return float64(v.Int) / ir.CurrencyScale
	}
	return irToReal(v)
}

// toCurr coerces a numeric value to Currency.
func toCurr(v ir.Value) ir.Value {
	if v.Kind == ir.VKCurrency {
		return v
	}
	return ir.Value{Kind: ir.VKCurrency, Int: int64(math.Round(irToReal(v) * ir.CurrencyScale))}
}

func boolean(b bool) ir.Value { return ir.Value{Kind: ir.VKBool, Bool: b} }
func strVal(s string) ir.Value { return ir.Value{Kind: ir.VKStr, Str: s} }
