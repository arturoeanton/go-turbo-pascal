// Package turbo3 implements the Turbo3 unit runtime. The Turbo3 unit
// exposes legacy Turbo Pascal 3 text file variables (Kbd, Lst, Aux,
// AuxIn, Con) and a few constants. BPGo provides them as Text file
// variables and seeds the runtime with the standard device mappings.
package turbo3

import (
	"github.com/arturoeanton/go-turbo-pascal/internal/rtl/printer"
)

// File IDs for the standard Turbo3 file variables.
const (
	KbdID   = 100
	LstID   = 101
	AuxID   = 102
	AuxInID = 103
	ConID   = 104
)

// Init assigns the standard Turbo3 file variables to logical file
// IDs. Programs that want the legacy semantics can read from these
// IDs through the System unit.
func Init() {
	printer.Open("")
	// Kbd, Con etc. are pre-assigned logical IDs; the System unit
	// recognises them and routes I/O to stdin/stdout/lst/aux.
}
