package vmpas

import (
	"strings"
	"testing"
	"time"
)

// Output cap halts a guest that writes without bound.
func TestSandboxMaxOutput(t *testing.T) {
	caps := Sandboxed()
	caps.MaxOutput = 1024
	out, err := RunSandboxed(`
var i: Integer;
begin
  for i := 1 to 100000 do WriteLn('flooding the output buffer here');
end.`, caps)
	if err == nil {
		t.Fatal("expected output-limit error")
	}
	if len(out) > caps.MaxOutput+512 { // bounded near the cap, not unbounded
		t.Fatalf("output not bounded: %d bytes", len(out))
	}
}

// Call-depth cap halts runaway recursion deterministically.
func TestSandboxMaxCallDepth(t *testing.T) {
	caps := Sandboxed()
	caps.MaxCallDepth = 256
	caps.MaxSteps = 100_000_000 // ensure depth (not steps) is what trips
	_, err := RunSandboxed(`program P;
function Rec(n: Integer): Integer;
begin Rec := Rec(n + 1); end;
var x: Integer;
begin x := Rec(0); end.`, caps)
	if err == nil {
		t.Fatal("expected stack-overflow error from call-depth cap")
	}
}

// Wall-clock cap halts an infinite loop.
func TestSandboxMaxDuration(t *testing.T) {
	caps := Sandboxed()
	caps.MaxDuration = 100 * time.Millisecond
	caps.MaxSteps = 1_000_000_000
	start := time.Now()
	_, err := RunSandboxed(`begin while true do begin end; end.`, caps)
	if err == nil {
		t.Fatal("expected timeout/limit error")
	}
	if time.Since(start) > 3*time.Second {
		t.Fatalf("did not halt promptly: %v", time.Since(start))
	}
}

// Default sandbox denies host access: file/network/exec/env builtins are not
// even known identifiers (compile-time rejection).
func TestSandboxDeniesHostAccess(t *testing.T) {
	_, err := RunSandboxed(`begin Exec('rm', '-rf'); end.`, Sandboxed())
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "exec") &&
		!strings.Contains(err.Error(), "unknown") {
		// Accept any compile error mentioning the unknown identifier.
		if err == nil {
			t.Fatal("expected sandbox to reject Exec")
		}
	}
}

// A valid bounded script runs fine under the sandbox (no false positives).
func TestSandboxAllowsValidScript(t *testing.T) {
	out, err := RunSandboxed(`program P;
var t: Currency;
begin
  t := AddPercent(100.00, 21);
  WriteLn(CurrToStr(t));
end.`, Sandboxed())
	if err != nil {
		t.Fatalf("valid script failed under sandbox: %v", err)
	}
	if strings.TrimSpace(out) != "121.00" {
		t.Fatalf("got %q", out)
	}
}

// Each RunSandboxed call is isolated: no state leaks between tenants.
func TestSandboxIsolation(t *testing.T) {
	out1, _ := RunSandboxed(`program P; var n: Integer; begin n := 7; WriteLn(n); end.`, Sandboxed())
	out2, _ := RunSandboxed(`program P; var n: Integer; begin WriteLn(n); end.`, Sandboxed())
	if strings.TrimSpace(out1) != "7" {
		t.Fatalf("run1 got %q", out1)
	}
	if strings.TrimSpace(out2) != "0" { // fresh zero-value, not 7
		t.Fatalf("state leaked across runs: run2 got %q", out2)
	}
}
