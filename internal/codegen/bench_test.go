package codegen

import "testing"

// BenchmarkRunOnly measures the run-only throughput (compile once, run many),
// which is the relevant number for the compile-once / run-many embedding model.
func BenchmarkRunOnly(b *testing.B) {
	src := `program Bench;
var i, s: Integer;
begin
  s := 0;
  for i := 1 to 10000 do
    s := s + i;
end.`
	prog, err := Compile(src, "bench.pas")
	if err != nil {
		b.Fatalf("compile: %v", err)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		vm := NewVM(prog, nil, "")
		vm.Run()
	}
}

// BenchmarkFibRecursive measures recursion/call overhead.
func BenchmarkFibRecursive(b *testing.B) {
	src := `program Bench;
function Fib(n: Integer): Integer;
begin
  if n < 2 then Fib := n else Fib := Fib(n-1) + Fib(n-2);
end;
var r: Integer;
begin
  r := Fib(20);
end.`
	prog, err := Compile(src, "bench.pas")
	if err != nil {
		b.Fatalf("compile: %v", err)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		vm := NewVM(prog, nil, "")
		vm.Run()
	}
}
