// Ejemplo: cómo embeber Pascal dentro de un programa Go con pkg/vmpas.
//
// Ejecutar con:
//
//	go run ./examples/embed
//
// Muestra: ejecutar código Pascal, enlazar variables Go (lectura/escritura),
// mapear un struct de Go a un record de Pascal, llamar funciones Go desde
// Pascal y el sandbox de capacidades.
package main

import (
	"fmt"

	"github.com/arturoeanton/go-turbo-pascal/pkg/vmpas"
)

// Punto es un struct de Go que expondremos como un record de Pascal.
type Punto struct {
	X int
	Y int
}

func main() {
	// 1) Ejecutar un programa Pascal y capturar su salida.
	eng := vmpas.New() // sandbox restringido por defecto
	if err := eng.Run(`
		program Saludo;
		var i: Integer;
		begin
		  for i := 1 to 3 do
		    WriteLn('Hola desde Pascal numero ', i);
		end.`); err != nil {
		panic(err)
	}
	fmt.Print(eng.Output())

	// 2) Enlazar una variable Go: Pascal la lee y la escribe.
	eng2 := vmpas.New()
	total := 10
	_ = eng2.Var("total", &total)
	_ = eng2.Run(`for i := 1 to 5 do total := total + i`)
	fmt.Println("total tras el script:", total) // 10 + (1+2+3+4+5) = 25

	// 3) Mapear un struct de Go a un record de Pascal.
	eng3 := vmpas.New()
	p := Punto{X: 3, Y: 4}
	_ = eng3.Var("p", &p)
	_ = eng3.Run(`p.X := p.X * p.X + p.Y * p.Y`) // distancia^2
	fmt.Printf("p tras el script: %+v\n", p)     // {X:25 Y:4}

	// 4) Llamar funciones Go desde Pascal.
	eng4 := vmpas.New()
	_ = eng4.Function("Duplicar", func(n int) int { return n * 2 })
	_ = eng4.Process("Registrar", func(s string) { fmt.Println("[pascal dice]", s) })
	r := 0
	_ = eng4.Var("r", &r)
	_ = eng4.Run(`
		r := Duplicar(21);
		Registrar('resultado calculado')`)
	fmt.Println("r =", r) // 42

	// 5) Sandbox: por defecto el acceso a archivos está bloqueado.
	eng5 := vmpas.New()
	err := eng5.Run(`program T; var f: Text; begin Assign(f, 'x.txt'); end.`)
	fmt.Println("acceso a archivos bloqueado:", err != nil)

	// Con capacidades Full se permite (úsese solo para código de confianza).
	eng6 := vmpas.NewWith(vmpas.Full())
	err = eng6.Run(`program T; var f: Text; begin Assign(f, 'x.txt'); end.`)
	fmt.Println("acceso a archivos permitido (Full):", err == nil)
}
