package codegen

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestUserUnit(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "mathu.pas", `unit MathU;
interface
const Pi100 = 314;
function Cuadrado(n: Integer): Integer;
implementation
function Cuadrado(n: Integer): Integer;
begin
  Cuadrado := n * n;
end;
end.`)
	progPath := filepath.Join(dir, "p.pas")
	prog, err := Compile(`program P;
uses MathU;
begin
  WriteLn(Cuadrado(7));
  WriteLn(Pi100);
end.`, progPath)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	out, _, err := Run(prog, nil, "")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if out != "49\n314\n" {
		t.Fatalf("out=%q", out)
	}
}

func TestUnitInitialization(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "counter.pas", `unit Counter;
interface
var Total: Integer;
implementation
initialization
  Total := 100;
end.`)
	progPath := filepath.Join(dir, "p.pas")
	prog, err := Compile(`program P;
uses Counter;
begin
  Total := Total + 23;
  WriteLn(Total);
end.`, progPath)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	out, _, err := Run(prog, nil, "")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if out != "123\n" {
		t.Fatalf("out=%q", out)
	}
}
