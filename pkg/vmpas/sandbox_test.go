package vmpas

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestSandboxEnvGatedByCapability(t *testing.T) {
	os.Setenv("VMPAS_TEST_ENV", "hello")
	defer os.Unsetenv("VMPAS_TEST_ENV")

	// Denied by default: GetEnv is not registered, so it is unknown.
	if err := New().Run(`program T; var s: string; begin s := GetEnv('VMPAS_TEST_ENV'); end.`); err == nil {
		t.Fatal("expected GetEnv to be blocked under the default sandbox")
	}

	// Granted: GetEnv returns the value.
	e := NewWith(Capabilities{Env: true})
	var s string
	if err := e.Var("s", &s); err != nil {
		t.Fatal(err)
	}
	if err := e.Run(`s := GetEnv('VMPAS_TEST_ENV')`); err != nil {
		t.Fatalf("run with Env cap: %v", err)
	}
	if s != "hello" {
		t.Fatalf("GetEnv = %q, want %q", s, "hello")
	}
}

func TestSandboxExecGatedByCapability(t *testing.T) {
	if err := New().Run(`program T; var n: Integer; begin n := Exec('true'); end.`); err == nil {
		t.Fatal("expected Exec to be blocked under the default sandbox")
	}

	e := NewWith(Capabilities{Exec: true})
	var code int
	if err := e.Var("code", &code); err != nil {
		t.Fatal(err)
	}
	if err := e.Run(`code := Exec('exit 7')`); err != nil {
		t.Fatalf("run with Exec cap: %v", err)
	}
	if code != 7 {
		t.Fatalf("Exec exit code = %d, want 7", code)
	}
}

func TestSandboxNetworkGatedByCapability(t *testing.T) {
	// Only verify the gate: HttpGet must be unknown under the default sandbox.
	if err := New().Run(`program T; var s: string; begin s := HttpGet('http://x'); end.`); err == nil {
		t.Fatal("expected HttpGet to be blocked under the default sandbox")
	}
	// Granted: HttpGet resolves (a failed request just yields an empty string).
	e := NewWith(Capabilities{Network: true})
	if err := e.Run(`program T; var s: string; begin s := HttpGet('http://127.0.0.1:1/'); end.`); err != nil {
		t.Fatalf("run with Network cap: %v", err)
	}
}

func TestSandboxStepLimit(t *testing.T) {
	e := NewWith(Capabilities{MaxSteps: 5000})
	err := e.Run(`program T; var i, x: Integer; begin x := 0; while true do x := x + 1; end.`)
	if err == nil {
		t.Fatal("expected the step limit to halt an infinite loop")
	}
}

func TestSandboxHeapLimit(t *testing.T) {
	e := NewWith(Capabilities{MaxHeap: 10})
	err := e.Run(`program T;
type PNode = ^TNode; TNode = record next: PNode; end;
var p: PNode; i: Integer;
begin
  for i := 1 to 1000 do begin New(p); end;
end.`)
	if err == nil {
		t.Fatal("expected the heap limit to halt unbounded allocation")
	}
	if !strings.Contains(err.Error(), "203") {
		t.Fatalf("expected heap-overflow (203), got %v", err)
	}
}

func TestSandboxTimeLimit(t *testing.T) {
	e := NewWith(Capabilities{MaxSteps: 1 << 30, MaxDuration: 50 * time.Millisecond})
	start := time.Now()
	err := e.Run(`program T; var x: Integer; begin x := 0; while true do x := x + 1; end.`)
	if err == nil {
		t.Fatal("expected the time limit to halt an infinite loop")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("time limit did not trip promptly: %v", elapsed)
	}
}
