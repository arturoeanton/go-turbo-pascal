// Package windos provides the WinDos unit runtime. WinDos is the
// PChar-flavoured equivalent of the Dos unit: every procedure that
// takes a Pascal string in Dos takes a PChar here. The implementation
// uses a global Filesystem handle that can be replaced by the BPGo
// runtime to provide a sandbox or a host view.
package windos

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Filesystem is the same shape as dos.Filesystem but defined here to
// avoid a cycle and allow independent evolution.
type Filesystem interface {
	Stat(name string) (os.FileInfo, error)
	Open(name string) (*os.File, error)
	Create(name string) (*os.File, error)
	Remove(name string) error
	Rename(oldName, newName string) error
	ReadDir(name string) ([]os.DirEntry, error)
	Getenv(name string) string
	Setenv(name, value string) error
	Environ() []string
	WorkingDir(drive byte) string
	ChangeDir(path string) error
	MkDir(path string) error
	RmDir(path string) error
	AbsPath(path string) string
	Exists(name string) bool
}

var FS Filesystem = HostFS{}

// HostFS uses the OS filesystem. Tests can swap FS for a sandbox.
type HostFS struct{}

func (HostFS) Stat(name string) (os.FileInfo, error) { return os.Stat(name) }
func (HostFS) Open(name string) (*os.File, error)    { return os.Open(name) }
func (HostFS) Create(name string) (*os.File, error) {
	return os.Create(name)
}
func (HostFS) Remove(name string) error { return os.Remove(name) }
func (HostFS) Rename(o, n string) error { return os.Rename(o, n) }
func (HostFS) ReadDir(name string) ([]os.DirEntry, error) {
	return os.ReadDir(name)
}
func (HostFS) Getenv(name string) string       { return os.Getenv(name) }
func (HostFS) Setenv(name, value string) error { os.Setenv(name, value); return nil }
func (HostFS) Environ() []string               { return os.Environ() }
func (HostFS) WorkingDir(drive byte) string    { wd, _ := os.Getwd(); return wd }
func (HostFS) ChangeDir(path string) error     { return os.Chdir(path) }
func (HostFS) MkDir(path string) error         { return os.MkdirAll(path, 0o755) }
func (HostFS) RmDir(path string) error         { return os.RemoveAll(path) }
func (HostFS) AbsPath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	wd, _ := os.Getwd()
	return filepath.Clean(filepath.Join(wd, path))
}
func (HostFS) Exists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

// FileExpand returns the absolute, canonical form of a PChar path.
func FileExpand(p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	wd, _ := os.Getwd()
	return filepath.Clean(filepath.Join(wd, p))
}

// FileSearch returns the first match of `p` in the `list` (semicolon
// separated).
func FileSearch(p, list string) string {
	if filepath.IsAbs(p) && FS.Exists(p) {
		return p
	}
	for _, dir := range strings.Split(list, ";") {
		candidate := filepath.Join(strings.TrimSpace(dir), p)
		if FS.Exists(candidate) {
			return candidate
		}
	}
	return ""
}

// FileSplit splits a path into dir, name, ext components.
func FileSplit(p string) (string, string, string) {
	dir := filepath.Dir(p)
	base := filepath.Base(p)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	return dir, name, ext
}

// FileSize returns the file size in bytes (-1 on error).
func FileSize(p string) int64 {
	info, err := FS.Stat(p)
	if err != nil {
		return -1
	}
	return info.Size()
}

// FileExists returns true if the file exists.
func FileExists(p string) bool { return FS.Exists(p) }

// GetCurDir fills a PChar buffer with the current directory.
func GetCurDir(drive byte) string {
	return FS.WorkingDir(drive)
}

// SetCurDir changes the current directory.
func SetCurDir(dir string) error { return FS.ChangeDir(dir) }

// CreateDir creates a directory.
func CreateDir(dir string) error { return FS.MkDir(dir) }

// RemoveDir removes a directory.
func RemoveDir(dir string) error { return FS.RmDir(dir) }

// GetEnvVar reads an environment variable.
func GetEnvVar(name string) string { return FS.Getenv(name) }

// FindFirst/FindNext are implemented as streaming callbacks to a
// caller-supplied handler. The simplest form returns a slice.
func FindFiles(pattern string) ([]string, error) {
	dir := filepath.Dir(pattern)
	base := filepath.Base(pattern)
	entries, err := FS.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		matched, err := filepath.Match(base, e.Name())
		if err != nil {
			continue
		}
		if matched {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	return out, nil
}

// FindFirst opens a search; the caller drives FindNext until it
// returns an error.
type SearchHandle struct {
	files  []string
	pos    int
	filter func(os.FileInfo) bool
}

func (s *SearchHandle) Next() (string, error) {
	for s.pos < len(s.files) {
		f := s.files[s.pos]
		s.pos++
		info, err := FS.Stat(f)
		if err != nil {
			continue
		}
		if s.filter != nil && !s.filter(info) {
			continue
		}
		return f, nil
	}
	return "", errors.New("no more files")
}

// FindFirstResult is a convenience that returns the first match.
func FindFirstResult(pattern string) (string, error) {
	files, err := FindFiles(pattern)
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "", errors.New("no files")
	}
	return files[0], nil
}
