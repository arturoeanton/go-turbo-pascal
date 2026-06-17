package printer

import (
	"os"
	"testing"
)

func osReadFile(p string) ([]byte, error) { return os.ReadFile(p) }

func TestPrinterWrite(t *testing.T) {
	if err := Open(""); err != nil {
		t.Fatal(err)
	}
	if err := WriteString("hello"); err != nil {
		t.Fatal(err)
	}
	if err := WriteLn(" world"); err != nil {
		t.Fatal(err)
	}
	got := Output()
	if got != "hello world\r\n" {
		t.Errorf("got %q", got)
	}
	_ = Close()
}

func TestPrinterFile(t *testing.T) {
	tmp := t.TempDir() + "/lst.txt"
	if err := Open(tmp); err != nil {
		t.Fatal(err)
	}
	WriteString("printer content")
	Close()
	// Re-read the file to verify.
	data, err := osReadFile(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "printer content" {
		t.Errorf("got %q", string(data))
	}
}
