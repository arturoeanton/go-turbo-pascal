// Ejemplo: ejecutar scripts no confiables de varios tenants de forma aislada
// y acotada, como haría un motor de reglas de negocio embebido en un SaaS.
//
// Ejecutar con:
//
//	go run ./examples/multitenant
//
// Muestra: vmpas.RunSandboxed + el preset vmpas.Sandboxed() (default-deny con
// techos de pasos/heap/salida/profundidad/tiempo), aislamiento share-nothing
// entre tenants, y cómo un script malicioso se detiene sin colgar el host.
package main

import (
	"fmt"

	"github.com/arturoeanton/go-turbo-pascal/pkg/vmpas"
)

// scripts simula reglas que distintos tenants suben a la plataforma.
var scripts = map[string]string{
	"acme": `program Regla;
var total, descuento: Currency;
begin
  total := 1000.00;
  if total > 500.00 then descuento := Percent(total, 10) else descuento := 0;
  WriteLn('Descuento ACME: ', CurrToStr(descuento));
end.`,

	"globex": `program Regla;
var neto: Currency;
begin
  neto := AddPercent(250.00, 21);   { + IVA 21% }
  WriteLn('Total con IVA: ', CurrToStr(neto));
end.`,

	// Tenant malicioso: bucle infinito. El sandbox lo detiene por tiempo/pasos.
	"evil": `begin while true do begin end; end.`,
}

func main() {
	for _, tenant := range []string{"acme", "globex", "evil"} {
		// Un engine fresco y acotado por cada ejecución: share-nothing.
		out, err := vmpas.RunSandboxed(scripts[tenant], vmpas.Sandboxed())
		fmt.Printf("== tenant %q ==\n", tenant)
		if err != nil {
			fmt.Printf("  detenido por el sandbox: %v\n", err)
		}
		if out != "" {
			fmt.Printf("  salida: %s", out)
		}
	}
}
