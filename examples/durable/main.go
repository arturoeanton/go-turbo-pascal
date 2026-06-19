// Example: durable execution (pause / persist / resume) with pkg/vmpas.
//
// Run with:
//
//	go run ./examples/durable
//
// A business rule runs until it needs an external approval. At that point
// it pauses with Suspend, the host serializes the state (it could store it
// in a database and resume it days later, or in another process), injects
// the decision and resumes exactly where it left off. The result is
// identical to an uninterrupted run, with state and output intact.
package main

import (
	"fmt"

	"github.com/arturoeanton/go-turbo-pascal/pkg/vmpas"
)

// A spending request: if it exceeds the threshold, the rule asks for approval.
const rule = `program Aprobacion;
var monto: Currency; aprobado: Boolean; resultado: string;
begin
  WriteLn('Evaluating expense of ', CurrToStr(monto));
  if monto > 1000.00 then
  begin
    WriteLn('Exceeds the threshold: requires manual approval.');
    Suspend('aprobacion-requerida');
    if aprobado then resultado := 'APPROVED' else resultado := 'REJECTED';
  end
  else
    resultado := 'APPROVED (automatic)';
  WriteLn('Result: ', resultado);
end.`

func main() {
	var monto float64 = 2500.00
	var aprobado bool

	// Deterministic so the serialized state is reproducible.
	caps := vmpas.Capabilities{Deterministic: true, Seed: 1}
	eng := vmpas.NewWith(caps)
	eng.Var("monto", &monto)
	eng.Var("aprobado", &aprobado)

	// 1) Start: run until it asks for approval.
	state, err := eng.RunDurable(rule)
	if err != nil {
		panic(err)
	}
	if state == nil {
		fmt.Println("(finished without pausing)\n" + eng.Output())
		return
	}
	fmt.Printf("--- paused at %q ---\n%s\n", state.Tag, state.Output)

	// 2) The state is portable: serialize it (here we show it as bytes).
	fmt.Printf("[persisted state: %d bytes]\n\n", len(state.Data))

	// 3) The human decision arrives. We inject it through the bound variable
	//    and resume in a NEW engine (as if it were another process/instance).
	aprobado = true
	eng2 := vmpas.NewWith(caps)
	eng2.Var("monto", &monto)
	eng2.Var("aprobado", &aprobado)
	final, err := eng2.ResumeDurable(rule, state)
	if err != nil {
		panic(err)
	}
	if final != nil {
		fmt.Println("(paused again)")
		return
	}
	fmt.Println("--- resumed and finished ---")
	fmt.Print(eng2.Output())
}
