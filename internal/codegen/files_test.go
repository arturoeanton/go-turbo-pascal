package codegen

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestTextFileIO(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.txt")
	src := fmt.Sprintf(`program F;
var
  f: Text;
  s: String;
  i: Integer;
begin
  Assign(f, '%s');
  Rewrite(f);
  WriteLn(f, 'uno');
  WriteLn(f, 'dos');
  WriteLn(f, 'tres');
  Close(f);

  Assign(f, '%s');
  Reset(f);
  i := 0;
  while not Eof(f) do
  begin
    ReadLn(f, s);
    WriteLn('got: ', s);
    i := i + 1;
  end;
  Close(f);
  WriteLn('n=', i);
end.`, path, path)

	prog, err := Compile(src, "f.pas")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	out, _, err := Run(prog, nil, "")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	want := "got: uno\ngot: dos\ngot: tres\nn=3\n"
	if out != want {
		t.Fatalf("out=%q want %q", out, want)
	}
}
