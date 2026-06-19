package vmpas

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

// This file pins the stable public API of pkg/vmpas (see docs/en/api.md). Every
// exported symbol is referenced with its exact signature, so a breaking change
// (renamed/removed symbol, changed signature or struct field) fails to compile
// — the breakage is caught here before it reaches a release. Adding new API is
// fine (it does not break this file); changing existing API is not.
func TestAPIContract(t *testing.T) {
	// Constructors and presets.
	var _ func() *Engine = New
	var _ func(Capabilities) *Engine = NewWith
	var _ func() Capabilities = Restricted
	var _ func() Capabilities = Full
	var _ func() Capabilities = Sandboxed
	var _ func(string, Capabilities) (string, error) = RunSandboxed
	var _ func(*sql.DB) SQLDB = WrapSQLDB

	// Package-level convenience (shared default engine).
	var _ func(string) error = Run
	var _ func(string, any) error = Var
	var _ func(string, any) error = Function
	var _ func(string, any) error = Func
	var _ func(string, any) error = Process
	var _ func(string, any) error = Procedure
	var _ func() string = Output
	var _ func() = Reset
	var _ func(Capabilities) = SetCapabilities

	// Engine methods (method expressions pin the receiver + signature).
	var _ func(*Engine, string, any) error = (*Engine).Var
	var _ func(*Engine, string, any) error = (*Engine).Function
	var _ func(*Engine, string, any) error = (*Engine).Process
	var _ func(*Engine, string) error = (*Engine).Run
	var _ func(*Engine, string) (*Script, error) = (*Engine).Compile
	var _ func(*Engine) string = (*Engine).Output
	var _ func(*Engine, Capabilities) = (*Engine).SetCapabilities
	var _ func(*Engine, SQLDB) = (*Engine).UseDB
	var _ func(*Engine, string) (*CapReport, error) = (*Engine).Analyze
	var _ func(*Engine) []AuditEntry = (*Engine).AuditLog
	var _ func(*Engine, context.Context, string) error = (*Engine).RunContext
	var _ func(*Script, context.Context) error = (*Script).RunContext
	var _ func(*Engine, string, any) error = (*Engine).Get
	var _ func(*Engine, string) (*State, error) = (*Engine).RunDurable
	var _ func(*Engine, string, *State) (*State, error) = (*Engine).ResumeDurable

	// Script methods.
	var _ func(*Script) error = (*Script).Run
	var _ func(*Script) string = (*Script).Output

	// CapReport method.
	var _ func(*CapReport, Capability) bool = (*CapReport).Needs

	// Struct shapes (keyed fields pin names + types).
	var _ = Capabilities{
		FileSystem: true, Network: true, Exec: true, Env: true, Database: true,
		MaxSteps: 0, MaxHeap: 0, MaxOutput: 0, MaxCallDepth: 0,
		MaxDuration: time.Duration(0), Deterministic: true, Seed: 0, Audit: true,
		LiveBindings: true,
	}
	var _ = State{Tag: "", Data: nil, Output: ""}
	var _ = AuditEntry{Capability: CapEnv, Builtin: "", Args: nil}
	var _ = CapReport{Required: nil, Calls: nil}
	var _ error = &RuntimeError{Code: 0, Message: ""}

	// Capability constants.
	var _ Capability = CapFileSystem
	var _ Capability = CapNetwork
	var _ Capability = CapExec
	var _ Capability = CapEnv
	var _ Capability = CapDatabase
}
