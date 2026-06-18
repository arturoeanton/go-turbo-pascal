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
