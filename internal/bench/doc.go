// Package bench holds comparative benchmarks (BPGo vmpas vs. goja). It lives in
// its own package so the goja dependency never enters pkg/vmpas's import tree.
//
// Run with:
//
//	go test ./internal/bench -bench . -benchmem
package bench
