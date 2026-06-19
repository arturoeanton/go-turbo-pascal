package main

import (
	"path/filepath"
	"testing"
)

// A composed flow as the browser would generate it, with Trace() instrumentation.
const flowProg = `program Flow;
var amount: Currency; approved: Boolean; outcome: string;
begin
  Trace('n1'); outcome := '';
  Trace('n2'); WriteLn('Evaluating');
  Trace('n3'); if amount <= 1000.00 then begin outcome := 'auto-approved'; WriteLn('Auto'); Halt end;
  Trace('n4'); WriteLn('Awaiting approval'); Suspend('approval');
  if not approved then begin outcome := 'rejected'; WriteLn('Rejected'); Halt end;
  Trace('n5'); outcome := 'approved'; WriteLn('Approved');
  Trace('n6'); WriteLn('Result: ', outcome);
end.`

func TestExecuteApprovalThenResume(t *testing.T) {
	// Large amount: runs past the threshold and pauses at the Approval box.
	state, out := execute(flowProg, 2500, false, nil)
	if state == nil || out.Status != "paused" || out.Tag != "approval" {
		t.Fatalf("expected paused at approval, got %+v", out)
	}
	if len(out.Trace) != 4 || out.Trace[3] != "n4" {
		t.Fatalf("pre-pause trace = %v, want [n1 n2 n3 n4]", out.Trace)
	}
	// Approve and resume in a fresh engine.
	_, out2 := execute(flowProg, 2500, true, state)
	if out2.Status != "done" || out2.Outcome != "approved" {
		t.Fatalf("resume: %+v", out2)
	}
	if len(out2.Trace) != 2 || out2.Trace[0] != "n5" {
		t.Fatalf("post-resume trace = %v, want [n5 n6]", out2.Trace)
	}
}

func TestExecuteAutoApprove(t *testing.T) {
	_, out := execute(flowProg, 50, false, nil) // below threshold -> auto-approved, no pause
	if out.Status != "done" || out.Outcome != "auto-approved" {
		t.Fatalf("auto-approve: %+v", out)
	}
}

func TestExecuteReject(t *testing.T) {
	state, _ := execute(flowProg, 2500, false, nil)
	_, out := execute(flowProg, 2500, false, state) // reject
	if out.Status != "done" || out.Outcome != "rejected" {
		t.Fatalf("reject: %+v", out)
	}
}

func TestDBRoundtrip(t *testing.T) {
	initDB(filepath.Join(t.TempDir(), "rad.db"))
	if _, err := db.Exec(`INSERT INTO flows(name,graph,updated) VALUES('demo','{}','now')`); err != nil {
		t.Fatal(err)
	}
	var graph string
	if err := db.QueryRow(`SELECT graph FROM flows WHERE name='demo'`).Scan(&graph); err != nil {
		t.Fatal(err)
	}
	if graph != "{}" {
		t.Fatalf("graph = %q", graph)
	}
}
