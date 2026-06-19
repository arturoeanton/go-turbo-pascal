// Example: run untrusted scripts from multiple tenants in an isolated and
// bounded way, as a business-rules engine embedded in a SaaS would.
//
// Run with:
//
//	go run ./examples/multitenant
//
// Shows: vmpas.RunSandboxed + the vmpas.Sandboxed() preset (default-deny with
// caps on steps/heap/output/depth/time), share-nothing isolation between
// tenants, and how a malicious script is stopped without hanging the host.
package main

import (
	"fmt"

	"github.com/arturoeanton/go-turbo-pascal/pkg/vmpas"
)

// scripts simulates rules that different tenants upload to the platform.
var scripts = map[string]string{
	"acme": `program Regla;
var total, descuento: Currency;
begin
  total := 1000.00;
  if total > 500.00 then descuento := Percent(total, 10) else descuento := 0;
  WriteLn('ACME discount: ', CurrToStr(descuento));
end.`,

	"globex": `program Regla;
var neto: Currency;
begin
  neto := AddPercent(250.00, 21);   { + VAT 21% }
  WriteLn('Total with VAT: ', CurrToStr(neto));
end.`,

	// Malicious tenant: infinite loop. The sandbox stops it by time/steps.
	"evil": `begin while true do begin end; end.`,
}

func main() {
	for _, tenant := range []string{"acme", "globex", "evil"} {
		// A fresh, bounded engine per run: share-nothing.
		out, err := vmpas.RunSandboxed(scripts[tenant], vmpas.Sandboxed())
		fmt.Printf("== tenant %q ==\n", tenant)
		if err != nil {
			fmt.Printf("  stopped by the sandbox: %v\n", err)
		}
		if out != "" {
			fmt.Printf("  output: %s", out)
		}
	}
}
