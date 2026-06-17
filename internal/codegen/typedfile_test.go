package codegen

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestTypedFileInteger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nums.dat")
	src := fmt.Sprintf(`program TF;
var
  f: file of Integer;
  i, x, sum: Integer;
begin
  Assign(f, '%s');
  Rewrite(f);
  for i := 1 to 5 do Write(f, i * 10);
  WriteLn('size=', FileSize(f));
  Close(f);

  Assign(f, '%s');
  Reset(f);
  sum := 0;
  while not Eof(f) do
  begin
    Read(f, x);
    sum := sum + x;
  end;
  Close(f);
  WriteLn('sum=', sum);
end.`, path, path)

	got := run(t, src)
	want := "size=5\nsum=150\n" // 10+20+30+40+50
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestTypedFileSeek(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nums.dat")
	src := fmt.Sprintf(`program TF;
var
  f: file of Integer;
  i, x: Integer;
begin
  Assign(f, '%s');
  Rewrite(f);
  for i := 0 to 9 do Write(f, i * i);
  Close(f);

  Assign(f, '%s');
  Reset(f);
  Seek(f, 4);          { quinto registro: 4*4 = 16 }
  Read(f, x);
  WriteLn(x);
  Close(f);
end.`, path, path)

	got := run(t, src)
	if got != "16\n" {
		t.Fatalf("got %q", got)
	}
}
