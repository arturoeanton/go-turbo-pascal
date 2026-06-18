package vmpas

import (
	"strings"

	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
)

// Auditable trace (G2): when Capabilities.Audit is set, every call to a
// capability-gated host builtin (file, network, exec, env, database) is
// recorded in execution order with its arguments. The log is deterministic
// (calls happen in the same order on every run) and pairs with snapshot/resume
// for forensic replay — a tamper-evident record of what a tenant's script did.

// AuditEntry is one recorded capability-relevant action.
type AuditEntry struct {
	Capability Capability // which capability the call exercised
	Builtin    string     // the host builtin invoked (PascalCase)
	Args       []string   // its arguments, rendered for logging
}

// AuditLog returns the capability actions recorded during the most recent run,
// in execution order. It is empty unless Capabilities.Audit was enabled.
func (e *Engine) AuditLog() []AuditEntry {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]AuditEntry, len(e.audit))
	copy(out, e.audit)
	return out
}

// installAudit wraps each capability-gated builtin already registered on vm so
// that every call appends an AuditEntry. It must run before aliasLowercase so
// the lowercase aliases (which codegen calls) point at the wrapped functions.
func (e *Engine) installAudit(vm *ir.VM) {
	if !e.caps.Audit {
		return
	}
	for key, fn := range vm.Builtins {
		cap, ok := builtinCap[strings.ToLower(key)]
		if !ok {
			continue
		}
		inner, name, capability := fn, key, cap
		vm.Builtins[key] = func(rvm *ir.VM, args []ir.Value) ir.Value {
			e.audit = append(e.audit, AuditEntry{
				Capability: capability,
				Builtin:    name,
				Args:       renderArgs(args),
			})
			return inner(rvm, args)
		}
	}
}

// renderArgs formats builtin arguments for the audit log.
func renderArgs(args []ir.Value) []string {
	if len(args) == 0 {
		return nil
	}
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = formatWrite(a)
	}
	return out
}
