package compile

import (
	"strings"
	"testing"
)

func TestRunVMEmitsWriteLnOutput(t *testing.T) {
	src := `program T;
var
  A, B, S: Integer;
begin
  A := 2;
  B := 3;
  S := A + B;
  WriteLn('Sum = ', S);
end.`
	prog, err := CompileToVM(&CompileConfig{Source: src, SourceFile: "sum.pas"})
	if err != nil {
		t.Fatal(err)
	}
	out, code, err := RunVM(prog, nil)
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out, "Sum = 5") {
		t.Fatalf("output = %q", out)
	}
}
