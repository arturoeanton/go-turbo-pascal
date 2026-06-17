package codegen

import "testing"

func TestCrtUnitEmitsAnsi(t *testing.T) {
	// uses Crt enables the Crt builtins; they emit ANSI escape sequences.
	got := run(t, `program C;
uses Crt;
begin
  ClrScr;
  GotoXY(10, 5);
  TextColor(4);       { rojo -> ANSI 31 }
  Write('X');
  NormVideo;
end.`)
	want := "\x1b[2J\x1b[H" + "\x1b[5;10H" + "\x1b[31m" + "X" + "\x1b[0m"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestCrtRequiresUses(t *testing.T) {
	// Without `uses Crt`, ClrScr must be an unknown identifier at compile time.
	if _, err := Compile(`program C; begin ClrScr; end.`, "c.pas"); err == nil {
		t.Fatal("expected a compile error calling ClrScr without `uses Crt`")
	}
}
