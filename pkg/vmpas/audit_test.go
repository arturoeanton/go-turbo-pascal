package vmpas

import "testing"

// With Audit on, every gated host call is recorded in order with its args.
func TestAuditRecordsGatedCalls(t *testing.T) {
	e := NewWith(Capabilities{Env: true, Audit: true})
	if err := e.Run(`WriteLn(GetEnv('HOME'), GetEnv('USER'));`); err != nil {
		t.Fatal(err)
	}
	log := e.AuditLog()
	if len(log) != 2 {
		t.Fatalf("expected 2 audit entries, got %d: %+v", len(log), log)
	}
	if log[0].Capability != CapEnv || log[0].Builtin != "GetEnv" {
		t.Fatalf("entry 0 = %+v", log[0])
	}
	if len(log[0].Args) != 1 || log[0].Args[0] != "HOME" {
		t.Fatalf("entry 0 args = %v, want [HOME]", log[0].Args)
	}
	if log[1].Args[0] != "USER" {
		t.Fatalf("entry 1 args = %v, want [USER]", log[1].Args)
	}
}

// Without Audit the log stays empty (zero overhead, opt-in).
func TestAuditDisabledByDefault(t *testing.T) {
	e := NewWith(Capabilities{Env: true})
	if err := e.Run(`WriteLn(GetEnv('HOME'));`); err != nil {
		t.Fatal(err)
	}
	if len(e.AuditLog()) != 0 {
		t.Fatalf("expected empty audit log, got %+v", e.AuditLog())
	}
}

// The audit log resets between runs (no leakage across tenant requests).
func TestAuditResetsBetweenRuns(t *testing.T) {
	e := NewWith(Capabilities{Env: true, Audit: true})
	_ = e.Run(`WriteLn(GetEnv('A'));`)
	_ = e.Run(`WriteLn(GetEnv('B'));`)
	log := e.AuditLog()
	if len(log) != 1 || log[0].Args[0] != "B" {
		t.Fatalf("expected only the second run's call, got %+v", log)
	}
}
