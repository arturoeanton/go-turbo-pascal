package codegen

import "testing"

func TestBizMoneyStr(t *testing.T) {
	check(t, `program P;
var c: Currency;
begin
  c := StrToCurr('19.99');
  WriteLn(CurrToStr(c + 0.01));   { 20.00 }
end.`, "20.00\n")
}

func TestBizMinMaxClamp(t *testing.T) {
	check(t, `program P;
begin
  WriteLn(Min(3, 7));        { 3 }
  WriteLn(Max(3, 7));        { 7 }
  WriteLn(Clamp(15, 0, 10)); { 10 }
  WriteLn(Clamp(-2, 0, 10)); { 0 }
  WriteLn(Clamp(5, 0, 10));  { 5 }
end.`, "3\n7\n10\n0\n5\n")
}

func TestBizStringPredicates(t *testing.T) {
	check(t, `program P;
begin
  if Contains('hello world', 'world') then WriteLn('c');
  if StartsWith('foobar', 'foo') then WriteLn('s');
  if EndsWith('foobar', 'bar') then WriteLn('e');
  if IsEmpty('   ') then WriteLn('empty');
end.`, "c\ns\ne\nempty\n")
}

func TestBizDates(t *testing.T) {
	check(t, `program P;
begin
  WriteLn(DateYear('2026-06-18'));
  WriteLn(DateMonth('2026-06-18'));
  WriteLn(DateDay('2026-06-18'));
  WriteLn(DateAddDays('2026-06-18', 14));
  WriteLn(DateDiffDays('2026-06-01', '2026-06-18'));
  if DateValid('2026-02-30') then WriteLn('valida') else WriteLn('invalida');
end.`, "2026\n6\n18\n2026-07-02\n17\ninvalida\n")
}

func TestBizMoneyMinMax(t *testing.T) {
	// Min/Max preserve Currency.
	check(t, `program P;
var a, b: Currency;
begin
  a := 9.99; b := 12.50;
  WriteLn(CurrToStr(Min(a, b)));   { 9.99 }
  WriteLn(CurrToStr(Max(a, b)));   { 12.50 }
end.`, "9.99\n12.50\n")
}
