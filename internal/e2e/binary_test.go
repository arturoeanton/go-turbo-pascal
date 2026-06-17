package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestBinariesBuild verifies that every BPGo binary builds.
func TestBinariesBuild(t *testing.T) {
	binaries := []string{"bpgo", "pasrun", "pls", "pdap", "tpc", "tdebug", "turbo"}
	for _, bin := range binaries {
		bin := bin
		t.Run(bin, func(t *testing.T) {
			exe := filepath.Join("/tmp", bin)
			_ = exe
			// The test environment is expected to have the binaries
			// already built. We compile each one into /tmp/bin-name.
			out := filepath.Join("/tmp", "bpgo-test-"+bin)
			cmd := exec.Command("go", "build", "-o", out, "./../../cmd/"+bin)
			cmd.Stderr = &bytes.Buffer{}
			cmd.Stdout = &bytes.Buffer{}
			if err := cmd.Run(); err != nil {
				t.Errorf("build %s: %v", bin, err)
			}
			if _, err := os.Stat(out); err != nil {
				t.Errorf("binary %s not created: %v", bin, err)
			}
			os.Remove(out)
		})
	}
}

// TestBinariesHelp runs --help on every binary and verifies the
// output contains the expected banner.
func TestBinariesHelp(t *testing.T) {
	binaries := []string{"bpgo", "tpc", "tdebug"}
	banners := map[string]string{
		"bpgo":   "Usage: bpgo",
		"tpc":    "Usage: bpgo",
		"tdebug": "BPGo source-level debugger",
	}
	for _, bin := range binaries {
		bin := bin
		t.Run(bin, func(t *testing.T) {
			exe := filepath.Join("/tmp", "bpgo-test-"+bin)
			defer os.Remove(exe)
			build := exec.Command("go", "build", "-o", exe, "./../../cmd/"+bin)
			if err := build.Run(); err != nil {
				t.Skipf("build failed: %v", err)
			}
			out, err := exec.Command(exe, "--help").CombinedOutput()
			if err != nil {
				t.Errorf("run --help: %v", err)
			}
			if !strings.Contains(string(out), banners[bin]) {
				t.Errorf("output should contain %q, got %q", banners[bin], string(out))
			}
		})
	}
}

// TestBinariesVersion runs -V on every binary and verifies the
// output contains a version string.
func TestBinariesVersion(t *testing.T) {
	// pasrun/pls/pdap are stdio tools without a -V flag, so they are excluded.
	binaries := []string{"bpgo", "tpc", "tdebug", "turbo"}
	for _, bin := range binaries {
		bin := bin
		t.Run(bin, func(t *testing.T) {
			exe := filepath.Join("/tmp", "bpgo-test-"+bin)
			defer os.Remove(exe)
			build := exec.Command("go", "build", "-o", exe, "./../../cmd/"+bin)
			if err := build.Run(); err != nil {
				t.Skipf("build failed: %v", err)
			}
			out, err := exec.Command(exe, "-V").CombinedOutput()
			if err != nil {
				t.Errorf("run -V: %v", err)
			}
			if !strings.Contains(string(out), "BPGo") && !strings.Contains(string(out), "0.1.0") {
				t.Errorf("output should contain version, got %q", string(out))
			}
		})
	}
}

// TestBPGoMissingFile runs bpgo against a missing file and
// verifies a non-zero exit code.
func TestBPGoMissingFile(t *testing.T) {
	exe := filepath.Join("/tmp", "bpgo-test-bpgo")
	defer os.Remove(exe)
	build := exec.Command("go", "build", "-o", exe, "./../../cmd/bpgo")
	if err := build.Run(); err != nil {
		t.Skipf("build failed: %v", err)
	}
	out, err := exec.Command(exe, "does-not-exist.pas").CombinedOutput()
	if err == nil {
		t.Error("expected non-zero exit for missing file")
	}
	if !strings.Contains(string(out), "not found") {
		t.Errorf("output should mention 'not found', got %q", string(out))
	}
}

// TestBPGoTestCompat runs the conformance harness via the bpgo CLI.
func TestBPGoTestCompat(t *testing.T) {
	exe := filepath.Join("/tmp", "bpgo-test-bpgo")
	defer os.Remove(exe)
	build := exec.Command("go", "build", "-o", exe, "./../../cmd/bpgo")
	if err := build.Run(); err != nil {
		t.Skipf("build failed: %v", err)
	}
	cmd := exec.Command(exe, "test-compat")
	cmd.Dir = "../.."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("test-compat: %v", err)
	}
	if !strings.Contains(string(out), "Conformance") {
		t.Errorf("output should mention Conformance, got %q", string(out))
	}
}
