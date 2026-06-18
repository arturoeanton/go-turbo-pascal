// Ejemplo: ejecución durable (pausar / persistir / reanudar) con pkg/vmpas.
//
// Ejecutar con:
//
//	go run ./examples/durable
//
// Una regla de negocio se ejecuta hasta que necesita una aprobación externa.
// En ese punto se pausa con Suspend, el host serializa el estado (lo podría
// guardar en una base de datos y reanudarlo días después, o en otro proceso),
// inyecta la decisión y reanuda exactamente donde quedó. El resultado es
// idéntico a una ejecución ininterrumpida, con estado y salida intactos.
package main

import (
	"fmt"

	"github.com/arturoeanton/go-turbo-pascal/pkg/vmpas"
)

// Una solicitud de gasto: si supera el umbral, la regla pide aprobación.
const rule = `program Aprobacion;
var monto: Currency; aprobado: Boolean; resultado: string;
begin
  WriteLn('Evaluando gasto por ', CurrToStr(monto));
  if monto > 1000.00 then
  begin
    WriteLn('Supera el umbral: requiere aprobacion manual.');
    Suspend('aprobacion-requerida');
    if aprobado then resultado := 'APROBADO' else resultado := 'RECHAZADO';
  end
  else
    resultado := 'APROBADO (automatico)';
  WriteLn('Resultado: ', resultado);
end.`

func main() {
	var monto float64 = 2500.00
	var aprobado bool

	// Determinista para que el estado serializado sea reproducible.
	caps := vmpas.Capabilities{Deterministic: true, Seed: 1}
	eng := vmpas.NewWith(caps)
	eng.Var("monto", &monto)
	eng.Var("aprobado", &aprobado)

	// 1) Arranque: corre hasta que pide aprobación.
	state, err := eng.RunDurable(rule)
	if err != nil {
		panic(err)
	}
	if state == nil {
		fmt.Println("(terminó sin pausar)\n" + eng.Output())
		return
	}
	fmt.Printf("--- pausado en %q ---\n%s\n", state.Tag, state.Output)

	// 2) El estado es portable: serializámoslo (aquí lo mostramos en bytes).
	fmt.Printf("[estado persistido: %d bytes]\n\n", len(state.Data))

	// 3) Llega la decisión humana. La inyectamos por la variable enlazada y
	//    reanudamos en un engine NUEVO (como si fuera otro proceso/instancia).
	aprobado = true
	eng2 := vmpas.NewWith(caps)
	eng2.Var("monto", &monto)
	eng2.Var("aprobado", &aprobado)
	final, err := eng2.ResumeDurable(rule, state)
	if err != nil {
		panic(err)
	}
	if final != nil {
		fmt.Println("(volvió a pausar)")
		return
	}
	fmt.Println("--- reanudado y finalizado ---")
	fmt.Print(eng2.Output())
}
