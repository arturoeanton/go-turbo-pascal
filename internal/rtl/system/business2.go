package system

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
)

// registerBusiness2 installs broader management-system helpers used across
// accounting, inventory/ERP, HR, CMS and DMS: percentages/rounding, string
// formatting/validation and business-day/date helpers. Like the rest of the
// date helpers, everything is deterministic (no clock reads).
func registerBusiness2(vm *ir.VM) {
	// --- accounting / numbers ---
	vm.Builtins["RoundTo"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		p := math.Pow(10, float64(irToInt(arg(a, 1))))
		return ir.Value{Kind: ir.VKReal, Real: math.Round(num(arg(a, 0))*p) / p}
	}
	vm.Builtins["RoundMoney"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		c := toCurr(arg(a, 0))
		// round the 4-decimal scaled value to whole cents (2 decimals)
		return ir.Value{Kind: ir.VKCurrency, Int: roundDiv(c.Int, 100) * 100}
	}
	vm.Builtins["Percent"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		base, pct := arg(a, 0), num(arg(a, 1))
		if base.Kind == ir.VKCurrency {
			return ir.Value{Kind: ir.VKCurrency, Int: int64(math.Round(float64(base.Int) * pct / 100))}
		}
		return ir.Value{Kind: ir.VKReal, Real: num(base) * pct / 100}
	}
	vm.Builtins["AddPercent"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		base, pct := arg(a, 0), num(arg(a, 1))
		if base.Kind == ir.VKCurrency {
			return ir.Value{Kind: ir.VKCurrency, Int: int64(math.Round(float64(base.Int) * (1 + pct/100)))}
		}
		return ir.Value{Kind: ir.VKReal, Real: num(base) * (1 + pct/100)}
	}

	// --- strings: formatting, codes, validation ---
	vm.Builtins["PadLeft"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		return strVal(pad(valueToString(arg(a, 0)), int(irToInt(arg(a, 1))), padChar(a), true))
	}
	vm.Builtins["PadRight"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		return strVal(pad(valueToString(arg(a, 0)), int(irToInt(arg(a, 1))), padChar(a), false))
	}
	vm.Builtins["Replace"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		return strVal(strings.ReplaceAll(valueToString(arg(a, 0)), valueToString(arg(a, 1)), valueToString(arg(a, 2))))
	}
	vm.Builtins["OnlyDigits"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		var b strings.Builder
		for _, r := range valueToString(arg(a, 0)) {
			if r >= '0' && r <= '9' {
				b.WriteRune(r)
			}
		}
		return strVal(b.String())
	}
	vm.Builtins["IsNumeric"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		_, err := strconv.ParseFloat(strings.TrimSpace(valueToString(arg(a, 0))), 64)
		return boolean(err == nil)
	}
	vm.Builtins["IsInteger"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		_, err := strconv.ParseInt(strings.TrimSpace(valueToString(arg(a, 0))), 10, 64)
		return boolean(err == nil)
	}
	vm.Builtins["Split"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		parts := strings.Split(valueToString(arg(a, 0)), valueToString(arg(a, 1)))
		out := make([]ir.Value, len(parts))
		for i, p := range parts {
			out[i] = strVal(p)
		}
		return ir.Value{Kind: ir.VKArray, Array: out}
	}

	// --- dates: business days, HR, accounting periods (ISO, deterministic) ---
	vm.Builtins["DayOfWeek"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		t, err := time.Parse(isoDate, valueToString(arg(a, 0)))
		if err != nil {
			return ir.Value{Kind: ir.VKInt}
		}
		wd := int64(t.Weekday()) // Sunday=0..Saturday=6
		if wd == 0 {
			wd = 7
		}
		return ir.Value{Kind: ir.VKInt, Int: wd} // ISO: Monday=1..Sunday=7
	}
	vm.Builtins["IsWeekend"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		t, err := time.Parse(isoDate, valueToString(arg(a, 0)))
		if err != nil {
			return boolean(false)
		}
		wd := t.Weekday()
		return boolean(wd == time.Saturday || wd == time.Sunday)
	}
	vm.Builtins["MonthEnd"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		t, err := time.Parse(isoDate, valueToString(arg(a, 0)))
		if err != nil {
			return strVal("")
		}
		first := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
		return strVal(first.AddDate(0, 1, -1).Format(isoDate))
	}
	vm.Builtins["Age"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		birth, e1 := time.Parse(isoDate, valueToString(arg(a, 0)))
		asof, e2 := time.Parse(isoDate, valueToString(arg(a, 1)))
		if e1 != nil || e2 != nil {
			return ir.Value{Kind: ir.VKInt}
		}
		years := asof.Year() - birth.Year()
		if asof.Month() < birth.Month() || (asof.Month() == birth.Month() && asof.Day() < birth.Day()) {
			years--
		}
		return ir.Value{Kind: ir.VKInt, Int: int64(years)}
	}
	vm.Builtins["AddBusinessDays"] = func(_ *ir.VM, a []ir.Value) ir.Value {
		t, err := time.Parse(isoDate, valueToString(arg(a, 0)))
		if err != nil {
			return strVal("")
		}
		n := int(irToInt(arg(a, 1)))
		step := 1
		if n < 0 {
			step, n = -1, -n
		}
		for n > 0 {
			t = t.AddDate(0, 0, step)
			if t.Weekday() != time.Saturday && t.Weekday() != time.Sunday {
				n--
			}
		}
		return strVal(t.Format(isoDate))
	}
}

// pad pads s to width with padCh on the left (or right). It never truncates.
func pad(s string, width int, padCh byte, left bool) string {
	if len(s) >= width {
		return s
	}
	fill := strings.Repeat(string(padCh), width-len(s))
	if left {
		return fill + s
	}
	return s + fill
}

// padChar reads the optional 3rd pad-character argument (default space).
func padChar(a []ir.Value) byte {
	if len(a) > 2 {
		if s := valueToString(a[2]); len(s) > 0 {
			return s[0]
		}
	}
	return ' '
}

// roundDiv divides rounding half-away-from-zero.
func roundDiv(x, d int64) int64 {
	if x < 0 {
		return -((-x + d/2) / d)
	}
	return (x + d/2) / d
}
