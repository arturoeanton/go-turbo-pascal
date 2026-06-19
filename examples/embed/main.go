// Example: how to embed Pascal inside a Go program with pkg/vmpas.
//
// Run with:
//
//	go run ./examples/embed
//
// Shows: running Pascal code, binding Go variables (read/write),
// mapping a Go struct to a Pascal record, calling Go functions from
// Pascal, and the capability sandbox.
package main

import (
	"fmt"

	"github.com/arturoeanton/go-turbo-pascal/pkg/vmpas"
)

// Punto is a Go struct we'll expose as a Pascal record.
type Punto struct {
	X int
	Y int
}

func main() {
	// 1) Run a Pascal program and capture its output.
	eng := vmpas.New() // restricted sandbox by default
	if err := eng.Run(`
		program Saludo;
		var i: Integer;
		begin
		  for i := 1 to 3 do
		    WriteLn('Hello from Pascal number ', i);
		end.`); err != nil {
		panic(err)
	}
	fmt.Print(eng.Output())

	// 2) Bind a Go variable: Pascal reads and writes it.
	eng2 := vmpas.New()
	total := 10
	_ = eng2.Var("total", &total)
	_ = eng2.Run(`for i := 1 to 5 do total := total + i`)
	fmt.Println("total after the script:", total) // 10 + (1+2+3+4+5) = 25

	// 3) Map a Go struct to a Pascal record.
	eng3 := vmpas.New()
	p := Punto{X: 3, Y: 4}
	_ = eng3.Var("p", &p)
	_ = eng3.Run(`p.X := p.X * p.X + p.Y * p.Y`) // distance^2
	fmt.Printf("p after the script: %+v\n", p)   // {X:25 Y:4}

	// 4) Call Go functions from Pascal.
	eng4 := vmpas.New()
	_ = eng4.Function("Duplicar", func(n int) int { return n * 2 })
	_ = eng4.Process("Registrar", func(s string) { fmt.Println("[pascal says]", s) })
	r := 0
	_ = eng4.Var("r", &r)
	_ = eng4.Run(`
		r := Duplicar(21);
		Registrar('computed result')`)
	fmt.Println("r =", r) // 42

	// 5) Sandbox: file access is blocked by default.
	eng5 := vmpas.New()
	err := eng5.Run(`program T; var f: Text; begin Assign(f, 'x.txt'); end.`)
	fmt.Println("file access blocked:", err != nil)

	// With Full capabilities it is allowed (use only for trusted code).
	eng6 := vmpas.NewWith(vmpas.Full())
	err = eng6.Run(`program T; var f: Text; begin Assign(f, 'x.txt'); end.`)
	fmt.Println("file access allowed (Full):", err == nil)
}
