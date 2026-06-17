package vmpas_test

import (
	"fmt"

	"github.com/arturoeanton/go-turbo-pascal/pkg/vmpas"
)

func Example() {
	eng := vmpas.New()
	v1 := 0
	f1 := func(x int) int { return x + 10 }
	p1 := func(x int) { fmt.Println("p1", x) }

	_ = eng.Var("v1", &v1)
	_ = eng.Function("f1", f1)
	_ = eng.Process("p1", p1)
	_ = eng.Run("v1 := f1(5); p1(v1)")

	fmt.Println(v1)
	// Output:
	// p1 15
	// 15
}
