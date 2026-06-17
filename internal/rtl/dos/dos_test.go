package dos

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestPackUnpackTime(t *testing.T) {
	dt := DateTime{Year: 2024, Month: 12, Day: 25, Hour: 13, Min: 45, Sec: 30}
	d, t2 := PackTime(dt)
	out := UnpackTime(d, t2)
	if out.Year != 2024 || out.Month != 12 || out.Day != 25 || out.Hour != 13 || out.Min != 45 {
		t.Errorf("roundtrip: %+v", out)
	}
}

func TestGetDateAndTime(t *testing.T) {
	now := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	y, m, d, dow := GetDate(now)
	if y != 2024 || m != 6 || d != 15 {
		t.Errorf("date: %d %d %d", y, m, d)
	}
	if dow < 1 || dow > 7 {
		t.Errorf("dow out of range: %d", dow)
	}
}

func TestSandboxFS(t *testing.T) {
	root := t.TempDir()
	fs := NewSandboxFS(root)
	if err := fs.Setenv("FOO", "bar"); err != nil {
		t.Fatal(err)
	}
	if fs.Getenv("FOO") != "bar" {
		t.Error("env not set")
	}
	envs := fs.Environ()
	if len(envs) == 0 || !strings.Contains(envs[0], "FOO=bar") {
		t.Errorf("env list: %v", envs)
	}
	if err := fs.MkDir("sub"); err != nil {
		t.Fatal(err)
	}
	if !fs.Exists("sub") {
		t.Error("sub not created")
	}
	if err := fs.ChangeDir("sub"); err != nil {
		t.Fatal(err)
	}
	if fs.WorkingDir(0) == "" {
		t.Error("empty cwd")
	}
}

func TestSandboxFSReadDir(t *testing.T) {
	root := t.TempDir()
	fs := NewSandboxFS(root)
	f, err := fs.Create("test.txt")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("hi")
	f.Close()
	entries, err := fs.ReadDir("/")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range entries {
		if e.Name() == "test.txt" {
			found = true
		}
	}
	if !found {
		t.Error("test.txt not found")
	}
}

func TestEnvCountAndStr(t *testing.T) {
	fs := NewSandboxFS(t.TempDir())
	fs.Setenv("A", "1")
	fs.Setenv("B", "2")
	fs.Setenv("C", "3")
	if EnvCount(fs) != 3 {
		t.Errorf("EnvCount = %d", EnvCount(fs))
	}
	got := []string{EnvStr(fs, 1), EnvStr(fs, 2), EnvStr(fs, 3)}
	sort.Strings(got)
	if got[0] != "A=1" || got[1] != "B=2" || got[2] != "C=3" {
		t.Errorf("EnvStr: %v", got)
	}
}

func TestFindExact(t *testing.T) {
	root := t.TempDir()
	fs := NewSandboxFS(root)
	f, _ := fs.Create("hello.txt")
	f.Close()
	var rec SearchRec
	ok, err := Find(fs, filepath.Join("hello.txt"), 0, &rec)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("find failed")
	}
	if rec.Name != "hello.txt" {
		t.Errorf("name = %q", rec.Name)
	}
}

func TestFindWildcard(t *testing.T) {
	root := t.TempDir()
	fs := NewSandboxFS(root)
	fs.Create("a.txt")
	fs.Create("b.txt")
	fs.Create("c.log")
	var rec SearchRec
	ok, _ := Find(fs, filepath.Join("*.txt"), 0, &rec)
	if !ok {
		t.Fatal("find failed")
	}
	if !strings.HasSuffix(rec.Name, ".txt") {
		t.Errorf("name = %q", rec.Name)
	}
}

func TestDispatchUnsupported(t *testing.T) {
	err := DispatchIntr(0x99, &Registers{})
	if err == nil {
		t.Error("expected error for unsupported interrupt")
	}
}

func TestDispatchSupported(t *testing.T) {
	// int 21h, AH=4Ch should return nil (exit).
	err := DispatchIntr(0x21, &Registers{AH: 0x4C})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestAbsPath(t *testing.T) {
	fs := NewSandboxFS(t.TempDir())
	if !filepath.IsAbs(fs.AbsPath("a/b")) {
		t.Errorf("AbsPath should return absolute: %s", fs.AbsPath("a/b"))
	}
}

func TestEnvAndCwd(t *testing.T) {
	fs := NewSandboxFS(t.TempDir())
	fs.Setenv("X", "y")
	if GetEnv(fs, "X") != "y" {
		t.Error("GetEnv failed")
	}
}

func TestWorkingDirDefault(t *testing.T) {
	fs := NewSandboxFS(t.TempDir())
	if fs.WorkingDir(0) == "" {
		t.Error("WorkingDir should return at least root")
	}
}

func TestChDirToFile(t *testing.T) {
	fs := NewSandboxFS(t.TempDir())
	f, _ := fs.Create("f")
	f.Close()
	if err := fs.ChangeDir("f"); err == nil {
		t.Error("expected error changing to a file")
	}
	_ = os.Remove
}
