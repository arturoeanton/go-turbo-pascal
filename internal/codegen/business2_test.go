package codegen

import "testing"

func TestBizAccounting(t *testing.T) {
	check(t, `program P;
var price, tax, total: Currency;
begin
  price := 100.00;
  tax := Percent(price, 21);      { IVA 21% -> 21.00 }
  total := AddPercent(price, 21); { 121.00 }
  WriteLn(CurrToStr(tax), ' ', CurrToStr(total));
end.`, "21.00 121.00\n")
}

func TestBizRoundTo(t *testing.T) {
	check(t, `program P;
begin
  WriteLn(RoundTo(3.14159, 2):0:2);   { 3.14 }
end.`, "3.14\n")
}

func TestBizStringFormat(t *testing.T) {
	check(t, `program P;
begin
  WriteLn(PadLeft('42', 5, '0'));     { 00042 }
  WriteLn(PadRight('ab', 4, '.'));    { ab.. }
  WriteLn(Replace('a-b-c', '-', '/'));{ a/b/c }
  WriteLn(OnlyDigits('A1B2C3'));      { 123 }
end.`, "00042\nab..\na/b/c\n123\n")
}

func TestBizValidation(t *testing.T) {
	check(t, `program P;
begin
  if IsNumeric('3.14') then WriteLn('num');
  if not IsInteger('3.14') then WriteLn('notint');
  if IsInteger('42') then WriteLn('int');
end.`, "num\nnotint\nint\n")
}

func TestBizSplit(t *testing.T) {
	check(t, `program P;
var parts: array of string; i: Integer;
begin
  parts := Split('a,b,c', ',');
  for i := 0 to High(parts) do WriteLn(parts[i]);
end.`, "a\nb\nc\n")
}

func TestBizDatesAdvanced(t *testing.T) {
	check(t, `program P;
begin
  WriteLn(DayOfWeek('2026-06-18'));        { jueves -> 4 }
  if not IsWeekend('2026-06-18') then WriteLn('habil');
  if IsWeekend('2026-06-20') then WriteLn('finde');  { sabado }
  WriteLn(MonthEnd('2026-02-10'));         { 2026-02-28 }
  WriteLn(Age('2000-06-18', '2026-06-18'));{ 26 }
  WriteLn(AddBusinessDays('2026-06-18', 1)); { viernes 19 }
end.`, "4\nhabil\nfinde\n2026-02-28\n26\n2026-06-19\n")
}
