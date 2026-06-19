package codegen

import "testing"

func TestCurrencyBasic(t *testing.T) {
	check(t, `program P;
var price, total: Currency;
begin
  price := 19.99;
  total := price + 5.01;
  WriteLn(total);       { 25.00 }
end.`, "25.00\n")
}

func TestCurrencyExactNoFloatError(t *testing.T) {
	// 0.10 + 0.20 = 0.30 exacto (en float daría 0.30000000000000004).
	check(t, `program P;
var a, b: Currency;
begin
  a := 0.10;
  b := 0.20;
  WriteLn(a + b);       { 0.30 }
end.`, "0.30\n")
}

func TestCurrencyTimesQuantity(t *testing.T) {
	check(t, `program P;
var price, total: Currency; qty: Integer;
begin
  price := 2.50;
  qty := 4;
  total := price * qty;
  WriteLn(total);       { 10.00 }
end.`, "10.00\n")
}

func TestCurrencyDivByNumber(t *testing.T) {
	// money / number stays money (an even split), rounded half-up.
	check(t, `program P;
var price, share: Currency;
begin
  price := 10.00;
  share := price / 4;
  WriteLn(share);       { 2.50 }
end.`, "2.50\n")
}

func TestCurrencyDivByZero(t *testing.T) {
	// money / 0 yields zero money rather than panicking.
	check(t, `program P;
var price, share: Currency;
begin
  price := 10.00;
  share := price / 0;
  WriteLn(share);       { 0.00 }
end.`, "0.00\n")
}

func TestCurrencyDivByMoneyRatio(t *testing.T) {
	// money / money is a dimensionless ratio (a real number).
	check(t, `program P;
var a, b: Currency; r: Real;
begin
  a := 10.00;
  b := 4.00;
  r := a / b;
  if r = 2.5 then WriteLn('ratio ok') else WriteLn('bad');
end.`, "ratio ok\n")
}

func TestCurrencyCompare(t *testing.T) {
	check(t, `program P;
var a, b: Currency;
begin
  a := 9.99;
  b := 10.00;
  if a < b then WriteLn('menor');
  if a + 0.01 = b then WriteLn('igual');
end.`, "menor\nigual\n")
}

func TestToCurrencyBuiltin(t *testing.T) {
	check(t, `program P;
var c: Currency; n: Integer;
begin
  n := 7;
  c := ToCurrency(n) / 2;   { 3.50 }
  WriteLn(c);
end.`, "3.50\n")
}
