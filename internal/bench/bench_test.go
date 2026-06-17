package bench

import (
	"testing"

	"github.com/dop251/goja"

	"github.com/arturoeanton/go-turbo-pascal/pkg/vmpas"
)

// Both engines compile once and run many times on a fresh execution context per
// run, so the comparison is apples-to-apples (compile-once / run-many).

const pascalSum = `begin
  s := 0;
  for i := 1 to 1000 do s := s + i
end.`

const jsSum = `var s = 0; for (var i = 1; i <= 1000; i++) s = s + i;`

const pascalFib = `program F;
function Fib(n: Integer): Integer;
begin
  if n < 2 then Fib := n else Fib := Fib(n-1) + Fib(n-2);
end;
var r: Integer;
begin
  r := Fib(20);
end.`

const jsFib = `function fib(n){ return n < 2 ? n : fib(n-1) + fib(n-2); } var r = fib(20);`

func BenchmarkVMPasSum(b *testing.B) {
	e := vmpas.New()
	s := 0
	e.Var("s", &s)
	sc, err := e.Compile(pascalSum)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = sc.Run()
	}
}

func BenchmarkGojaSum(b *testing.B) {
	prog, err := goja.Compile("sum.js", jsSum, false)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		vm := goja.New()
		if _, err := vm.RunProgram(prog); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVMPasFib(b *testing.B) {
	e := vmpas.New()
	sc, err := e.Compile(pascalFib)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = sc.Run()
	}
}

func BenchmarkGojaFib(b *testing.B) {
	prog, err := goja.Compile("fib.js", jsFib, false)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		vm := goja.New()
		if _, err := vm.RunProgram(prog); err != nil {
			b.Fatal(err)
		}
	}
}
