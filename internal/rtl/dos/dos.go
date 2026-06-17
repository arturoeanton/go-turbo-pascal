// Package dos implements the Dos unit runtime. The Dos unit provides
// DOS-level services: date/time, file attributes, directory operations,
// environment variables, interrupt dispatch and the PSP/environment
// block abstraction. The implementation uses an in-memory sandbox by
// default; a real host filesystem adapter is used when the BPGo
// runtime is started with Filesystem = HostFS.
package dos

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// Filesystem abstracts the DOS file model.
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
	DiskFree(drive byte) int64
	DiskSize(drive byte) int64
	WorkingDir(drive byte) string
	ChangeDir(path string) error
	MkDir(path string) error
	RmDir(path string) error
	AbsPath(path string) string
	Exists(name string) bool
	IsAttr(name string, attr uint16) bool
	SetAttr(name string, attr uint16) error
}

// SandboxFS is the in-memory filesystem used by tests and the
// BPGo runner. It maps drive letters to a root directory under
// RootPath.
type SandboxFS struct {
	mu       sync.Mutex
	RootPath string
	Env      map[string]string
	Cwd      string
	attrs    map[string]uint16
}

func NewSandboxFS(root string) *SandboxFS {
	fs := &SandboxFS{
		RootPath: root,
		Env:      map[string]string{},
		Cwd:      string(filepath.Separator),
		attrs:    map[string]uint16{},
	}
	if root != "" {
		_ = os.MkdirAll(root, 0o755)
	}
	return fs
}

func (fs *SandboxFS) resolve(name string) string {
	if filepath.IsAbs(name) {
		return filepath.Join(fs.RootPath, name)
	}
	return filepath.Join(fs.RootPath, fs.Cwd, name)
}

func (fs *SandboxFS) Stat(name string) (os.FileInfo, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return os.Stat(fs.resolve(name))
}

func (fs *SandboxFS) Open(name string) (*os.File, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return os.Open(fs.resolve(name))
}

func (fs *SandboxFS) Create(name string) (*os.File, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	p := fs.resolve(name)
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	return os.Create(p)
}

func (fs *SandboxFS) Remove(name string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return os.Remove(fs.resolve(name))
}

func (fs *SandboxFS) Rename(oldName, newName string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return os.Rename(fs.resolve(oldName), fs.resolve(newName))
}

func (fs *SandboxFS) ReadDir(name string) ([]os.DirEntry, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return os.ReadDir(fs.resolve(name))
}

func (fs *SandboxFS) Getenv(name string) string {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if v, ok := fs.Env[name]; ok {
		return v
	}
	return ""
}

func (fs *SandboxFS) Setenv(name, value string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.Env[name] = value
	return nil
}

func (fs *SandboxFS) Environ() []string {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	out := make([]string, 0, len(fs.Env))
	for k, v := range fs.Env {
		out = append(out, k+"="+v)
	}
	sort.Strings(out)
	return out
}

func (fs *SandboxFS) DiskFree(drive byte) int64 {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if runtime.GOOS == "windows" {
		// Best effort; not always available.
		return 0
	}
	var p string
	if drive == 0 {
		p = "/"
	} else {
		p = "/"
	}
	// Use a no-op syscall; on host we don't have a portable disk-free API.
	_ = p
	return 0
}

func (fs *SandboxFS) DiskSize(drive byte) int64 {
	return 0
}

func (fs *SandboxFS) WorkingDir(drive byte) string {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.Cwd
}

func (fs *SandboxFS) ChangeDir(path string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if !filepath.IsAbs(path) {
		path = filepath.Join(fs.Cwd, path)
	}
	cleaned := filepath.Clean(path)
	info, err := os.Stat(filepath.Join(fs.RootPath, cleaned))
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory")
	}
	fs.Cwd = cleaned
	return nil
}

func (fs *SandboxFS) MkDir(path string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return os.MkdirAll(fs.resolve(path), 0o755)
}

func (fs *SandboxFS) RmDir(path string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return os.RemoveAll(fs.resolve(path))
}

func (fs *SandboxFS) AbsPath(path string) string {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if !filepath.IsAbs(path) {
		return filepath.Clean(filepath.Join(fs.Cwd, path))
	}
	return filepath.Clean(path)
}

func (fs *SandboxFS) Exists(name string) bool {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	_, err := os.Stat(fs.resolve(name))
	return err == nil
}

func (fs *SandboxFS) IsAttr(name string, attr uint16) bool {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if runtime.GOOS == "windows" {
		info, err := os.Stat(fs.resolve(name))
		if err != nil {
			return false
		}
		return (attrToOS(info) & attr) != 0
	}
	return false
}

func (fs *SandboxFS) SetAttr(name string, attr uint16) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.attrs[name] = attr
	return nil
}

func attrToOS(info os.FileInfo) uint16 {
	m := info.Mode()
	var a uint16
	if m&0o200 == 0 {
		a |= 0x01 // readonly
	}
	if m.IsDir() {
		a |= 0x10
	}
	return a
}

// Registers struct corresponds to the TP7 Registers type.
type Registers struct {
	AX, BX, CX, DX uint16
	SI, DI, BP     uint16
	DS, ES         uint16
	Flags          uint16
	AL, AH         byte
	BL, BH         byte
	CL, CH         byte
	DL, DH         byte
}

// DateTime is the BP7 DateTime record.
type DateTime struct {
	Year  uint16
	Month uint16
	Day   uint16
	Hour  uint16
	Min   uint16
	Sec   uint16
	Hund  uint16
}

// PackTime encodes a DateTime into a DOS time/date pair.
func PackTime(dt DateTime) (date, time uint16) {
	year := int(dt.Year) - 1980
	if year < 0 {
		year = 0
	}
	if year > 127 {
		year = 127
	}
	date = uint16((year << 9)) | uint16((int(dt.Month)&0xF)<<5) | uint16(dt.Day&0x1F)
	hour := int(dt.Hour) & 0x1F
	min := int(dt.Min) & 0x3F
	sec := int(dt.Sec) / 2
	time = uint16((hour << 11) | (min << 5) | sec)
	return
}

// UnpackTime decodes a DOS time/date pair into a DateTime.
func UnpackTime(date, time uint16) DateTime {
	year := int((date>>9)&0x7F) + 1980
	month := (date >> 5) & 0x0F
	day := date & 0x1F
	hour := (time >> 11) & 0x1F
	min := (time >> 5) & 0x3F
	sec := (time & 0x1F) * 2
	return DateTime{
		Year:  uint16(year),
		Month: month,
		Day:   day,
		Hour:  hour,
		Min:   min,
		Sec:   uint16(sec),
	}
}

// GetDate returns the current date in TP7 conventions (year, month, day, day-of-week).
func GetDate(now time.Time) (y, m, d, dow uint16) {
	y = uint16(now.Year())
	m = uint16(now.Month())
	d = uint16(now.Day())
	dow = uint16(now.Weekday() + 1) // Sunday=1 in DOS
	if dow > 7 {
		dow = 7
	}
	return
}

// GetTime returns the current time (hour, minute, second, hundredths).
func GetTime(now time.Time) (h, m, s, hund uint16) {
	h = uint16(now.Hour())
	m = uint16(now.Minute())
	s = uint16(now.Second())
	hund = uint16(now.Nanosecond() / 10_000_000)
	return
}

// SearchRec is the BP7 SearchRec record used by FindFirst/FindNext.
type SearchRec struct {
	Fill [21]byte
	Attr uint16
	Time uint16
	Date uint16
	Size int64
	Name string
}

// Find searches a path with optional wildcards. Returns false when no
// more files match.
func Find(fs Filesystem, path string, attr uint16, rec *SearchRec) (bool, error) {
	pattern := filepath.Base(path)
	dir := filepath.Dir(path)
	if !strings.ContainsAny(pattern, "*?[") {
		// Exact filename; check existence.
		info, err := fs.Stat(path)
		if err != nil {
			return false, err
		}
		*rec = SearchRec{Attr: attr, Name: info.Name(), Size: info.Size(), Date: 0, Time: 0}
		return true, nil
	}
	entries, err := fs.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, e := range entries {
		matched, err := filepath.Match(pattern, e.Name())
		if err != nil || !matched {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		*rec = SearchRec{Attr: attr, Name: info.Name(), Size: info.Size(), Date: 0, Time: 0}
		return true, nil
	}
	return false, nil
}

// EnvCount returns the number of environment variables.
func EnvCount(fs Filesystem) int {
	return len(fs.Environ())
}

// EnvStr returns the n-th environment variable (1-based).
func EnvStr(fs Filesystem, n int) string {
	envs := fs.Environ()
	if n < 1 || n > len(envs) {
		return ""
	}
	return envs[n-1]
}

// GetEnv returns the value of an environment variable.
func GetEnv(fs Filesystem, name string) string {
	return fs.Getenv(name)
}

// File attribute constants.
const (
	AttrReadOnly  = 0x01
	AttrHidden    = 0x02
	AttrSystem    = 0x04
	AttrVolumeID  = 0x08
	AttrDirectory = 0x10
	AttrArchive   = 0x20
	AttrAnyFile   = 0x3F
)

// Intr is a stub that returns a documented error when an unsupported
// interrupt is invoked. Subset support is provided for common DOS
// services; the implementation dispatches the request to the sandbox.
type IntrHandler func(intno byte, regs *Registers) error

var intrHandlers = map[byte]IntrHandler{}

func RegisterIntr(intno byte, h IntrHandler) {
	intrHandlers[intno] = h
}

func DispatchIntr(intno byte, regs *Registers) error {
	if h, ok := intrHandlers[intno]; ok {
		return h(intno, regs)
	}
	return fmt.Errorf("unsupported interrupt %02xh", intno)
}

// DefaultHandlers installs a small set of DOS/BIOS interrupt handlers
// that return documented "unsupported" errors.
func DefaultHandlers() {
	RegisterIntr(0x21, func(intno byte, regs *Registers) error {
		// int 21h subset: AH=4Ch (exit), AH=09h (print string)
		switch regs.AH {
		case 0x4C:
			// Exit
			return nil
		case 0x09:
			// Print string at DS:DX. Caller has placed the string at
			// that address; we just no-op.
			return nil
		}
		return fmt.Errorf("unsupported int 21h function %02xh", regs.AH)
	})
	RegisterIntr(0x10, func(intno byte, regs *Registers) error {
		return fmt.Errorf("unsupported int 10h function %02xh", regs.AH)
	})
	RegisterIntr(0x16, func(intno byte, regs *Registers) error {
		return fmt.Errorf("unsupported int 16h function %02xh", regs.AH)
	})
	RegisterIntr(0x33, func(intno byte, regs *Registers) error {
		return fmt.Errorf("unsupported int 33h function %02xh", regs.AH)
	})
}

func init() {
	DefaultHandlers()
}
