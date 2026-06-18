// Package bench holds comparative benchmarks (BPGo vmpas vs. goja). It lives in
// its own package so the goja dependency never enters pkg/vmpas's import tree.
//
// Run with:
//
//	go test ./internal/bench -bench . -benchmem
//
// Indicative results (Apple M-series, Go 1.26). vmpas trades raw time for a
// far smaller memory footprint and zero external dependencies:
//
//	            time/op      memory/op    allocs/op
//	Sum  vmpas  ~245 µs      ~1.6 KB      12
//	     goja   ~155 µs      ~112 KB      3744
//	Fib  vmpas  ~4.18 ms     ~12.9 KB     65
//	     goja   ~1.51 ms     ~11.4 KB     72
//
// goja is ~1.6x faster on the tight loop and ~2.8x on deep recursion, but
// allocates ~60x more on the loop. The remaining time gap is structural: the
// stack VM copies a 128-byte tagged-union Value on every push/pop. Two cheap
// wins are already in place — call targets are resolved once and cached in the
// instruction (OPCall), and the Value struct was shrunk from 152 to 128 bytes
// by making the set bitmap a pointer. Closing the gap further needs a VM
// redesign (NaN-boxed or typed-register values), which is deliberately out of
// scope here.
package bench
