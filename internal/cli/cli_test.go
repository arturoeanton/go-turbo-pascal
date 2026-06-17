package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestParseOptions(t *testing.T) {
	opts, err := ParseOptions([]string{"hello.pas", "-v", "-U", "units", "-D", "DEBUG"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Source != "hello.pas" {
		t.Errorf("Source: %q", opts.Source)
	}
	if !opts.Verbose {
		t.Error("Verbose")
	}
	if len(opts.UnitDirs) != 1 || opts.UnitDirs[0] != "units" {
		t.Errorf("UnitDirs: %v", opts.UnitDirs)
	}
	if len(opts.Defines) != 1 || opts.Defines[0] != "DEBUG" {
		t.Errorf("Defines: %v", opts.Defines)
	}
}

func TestParseOptionsHelp(t *testing.T) {
	opts, _ := ParseOptions([]string{"--help"})
	if !opts.ShowHelp {
		t.Error("ShowHelp")
	}
}

func TestParseOptionsVersion(t *testing.T) {
	opts, _ := ParseOptions([]string{"-V"})
	if !opts.ShowVersion {
		t.Error("ShowVersion")
	}
}

func TestParseOptionsRun(t *testing.T) {
	opts, _ := ParseOptions([]string{"-R", "hello.pas", "arg1"})
	if !opts.Run {
		t.Error("Run")
	}
	if opts.Source != "hello.pas" {
		t.Errorf("Source: %q", opts.Source)
	}
	if len(opts.Args) != 1 || opts.Args[0] != "arg1" {
		t.Errorf("Args: %v", opts.Args)
	}
}

func TestParseOptionsOutput(t *testing.T) {
	opts, _ := ParseOptions([]string{"-ohello.exe", "hello.pas"})
	if opts.Output != "hello.exe" {
		t.Errorf("Output: %q", opts.Output)
	}
}

func TestParseOptionsMemory(t *testing.T) {
	opts, _ := ParseOptions([]string{"-m16384,0,655360", "hello.pas"})
	if opts.Memory != "16384,0,655360" {
		t.Errorf("Memory: %q", opts.Memory)
	}
}

func TestParseOptionsBuild(t *testing.T) {
	opts, _ := ParseOptions([]string{"-B", "hello.pas"})
	if !opts.Build {
		t.Error("Build")
	}
}

func TestParseOptionsIncludes(t *testing.T) {
	opts, _ := ParseOptions([]string{"-Iinc1", "-Iinc2", "hello.pas"})
	if len(opts.IncludeDirs) != 2 {
		t.Errorf("IncludeDirs: %v", opts.IncludeDirs)
	}
}

func TestDefaultUsage(t *testing.T) {
	if !strings.Contains(DefaultUsage(), "bpgo") {
		t.Error("DefaultUsage should mention bpgo")
	}
}

func TestRunNoSource(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Run([]string{}, &bytes.Buffer{}, &out, &errOut)
	if code == 0 {
		t.Error("expected non-zero exit")
	}
	if !strings.Contains(errOut.String(), "no source") {
		t.Errorf("stderr: %q", errOut.String())
	}
}

func TestRunHelp(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Run([]string{"--help"}, &bytes.Buffer{}, &out, &errOut)
	if code != 0 {
		t.Errorf("code: %d", code)
	}
	if !strings.Contains(out.String(), "bpgo") {
		t.Errorf("out: %q", out.String())
	}
}

func TestRunVersion(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Run([]string{"-V"}, &bytes.Buffer{}, &out, &errOut)
	if code != 0 {
		t.Errorf("code: %d", code)
	}
	if !strings.Contains(out.String(), "BPGo") {
		t.Errorf("out: %q", out.String())
	}
}

func TestRunMissingFile(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Run([]string{"does-not-exist.pas"}, &bytes.Buffer{}, &out, &errOut)
	if code == 0 {
		t.Error("expected non-zero exit")
	}
}

func TestLoadConfig(t *testing.T) {
	tmp := t.TempDir() + "/TPC.CFG"
	content := "# comment\n-U units\n-I includes\n-D DEBUG\n-M\n"
	if err := writeFile(tmp, content); err != nil {
		t.Fatal(err)
	}
	opts, err := LoadConfig(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if len(opts.UnitDirs) != 1 || opts.UnitDirs[0] != "units" {
		t.Errorf("UnitDirs: %v", opts.UnitDirs)
	}
	if len(opts.IncludeDirs) != 1 || opts.IncludeDirs[0] != "includes" {
		t.Errorf("IncludeDirs: %v", opts.IncludeDirs)
	}
	if len(opts.Defines) != 1 || opts.Defines[0] != "DEBUG" {
		t.Errorf("Defines: %v", opts.Defines)
	}
	if !opts.Map {
		t.Error("Map")
	}
}

func TestLoadConfigMissing(t *testing.T) {
	opts, err := LoadConfig("/non/existent/TPC.CFG")
	if err != nil {
		t.Fatal(err)
	}
	if !opts.Map {
		t.Error("Map default true")
	}
}

func TestMergeOptions(t *testing.T) {
	a := &Options{Source: "a.pas", UnitDirs: []string{"x"}}
	b := &Options{Source: "b.pas", UnitDirs: []string{"y"}, Memory: "1024,0,4096"}
	c := MergeOptions(a, b)
	if c.Source != "b.pas" {
		t.Errorf("Source: %q", c.Source)
	}
	if len(c.UnitDirs) != 2 {
		t.Errorf("UnitDirs: %v", c.UnitDirs)
	}
	if c.Memory != "1024,0,4096" {
		t.Errorf("Memory: %q", c.Memory)
	}
}

func TestSourceUnitPaths(t *testing.T) {
	opts := &Options{UnitDirs: []string{"a"}}
	dirs := SourceUnitPaths(opts, "src/hello.pas")
	if len(dirs) < 2 {
		t.Errorf("dirs: %v", dirs)
	}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
