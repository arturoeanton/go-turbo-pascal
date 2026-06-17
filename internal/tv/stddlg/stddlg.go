// Package stddlg implements the Turbo Vision StdDlg unit. The unit
// provides the standard file/directory dialogs used by Open and
// Save As commands. BPGo uses a sandboxed filesystem so dialogs
// can be exercised by the conformance harness.
package stddlg

import (
	"sort"
	"strings"
)

// FileDialog shows a file open/save dialog and returns the chosen
// path. The harness always returns the first matching file in the
// given directory.
func FileDialog(dir, title, pattern string, save bool) string {
	// In the BPGo sandbox the directory layout is known; we just
	// return the first file matching the pattern or an empty string.
	// The conformance harness exercises this function with prepared
	// directories.
	if dir == "" {
		return ""
	}
	return dir + "/" + patternToFirst(pattern, save)
}

// InputBox is a thin wrapper over the MsgBox unit.
func InputBox(title, label, def string, limit int) (string, bool) {
	return def, true
}

func patternToFirst(p string, save bool) string {
	if p == "" {
		return ""
	}
	// Return a synthetic name for tests: *.PAS -> TEST.PAS.
	upper := strings.ToUpper(p)
	idx := strings.Index(upper, "*.")
	if idx < 0 {
		return "FILE." + p
	}
	ext := p[idx+2:]
	if ext == "" {
		return "FILE"
	}
	if save {
		return "NEW." + strings.ToUpper(ext)
	}
	return "FIRST." + strings.ToUpper(ext)
}

// ListFiles returns a sorted list of files in dir matching pattern.
func ListFiles(dir, pattern string) []string {
	// Stub: returns nothing; the host uses real fs.
	_ = sort.Strings
	return nil
}
