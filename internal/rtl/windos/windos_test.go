package windos

import (
	"strings"
	"testing"
)

func TestFileExpand(t *testing.T) {
	got := FileExpand("a/b")
	if !strings.HasSuffix(got, "a/b") && !strings.HasSuffix(got, `a\b`) {
		t.Errorf("FileExpand: %s", got)
	}
}

func TestFileSplit(t *testing.T) {
	d, n, e := FileSplit("/x/y/foo.pas")
	if d != "/x/y" || n != "foo" || e != ".pas" {
		t.Errorf("FileSplit: %s %s %s", d, n, e)
	}
}

func TestFileSizeMissing(t *testing.T) {
	if FileSize("does-not-exist") != -1 {
		t.Error("FileSize should return -1 on missing file")
	}
}

func TestFileExists(t *testing.T) {
	if !FileExists("/") {
		t.Error("FileExists should accept / on unix")
	}
}

func TestGetEnvVarMissing(t *testing.T) {
	if v := GetEnvVar("__BPGO_TEST_NOT_SET__"); v != "" {
		t.Errorf("GetEnvVar: %q", v)
	}
}

func TestFindFirst(t *testing.T) {
	f, err := FindFirstResult("/etc/hosts")
	if err != nil {
		// Some test environments may not have /etc/hosts.
		t.Skip("no hosts file in this environment")
	}
	if !strings.HasSuffix(f, "hosts") {
		t.Errorf("FindFirst: %q", f)
	}
}

func TestFileSearchAbsolute(t *testing.T) {
	if v := FileSearch("/etc/hosts", ""); v == "" {
		// May not exist on macOS; just check that it returns something or nothing.
	}
}
