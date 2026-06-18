// Package cli implements the BPGo command-line front-ends. The same
// driver is used by all five commands (bpgo, tpc, turbo, tdebug,
// bprun); each command instantiates a different set of subcommands.
// The driver supports a TP-style configuration file (TPC.CFG), per
// project compile/buil/clean/reset targets, debug session control and
// the test-compatibility harness.
package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/arturoeanton/go-turbo-pascal/internal/codegen"
)

// Options is the parsed command-line options.
type Options struct {
	Source      string
	Output      string
	UnitDirs    []string
	IncludeDirs []string
	ObjectDirs  []string
	Defines     []string
	Config      string
	Map         bool
	Debug       bool
	Verbose     bool
	Quiet       bool
	Memory      string
	ShowHelp    bool
	ShowVersion bool
	Run         bool
	Build       bool
	Clean       bool
	Reset       bool
	Make        bool
	Standalone  string
	IniFile     string
	Args        []string
}

// ParseOptions parses argv into Options.
func ParseOptions(argv []string) (*Options, error) {
	opts := &Options{Map: true, Config: "TPC.CFG"}
	for i := 0; i < len(argv); i++ {
		a := argv[i]
		switch {
		case a == "-h" || a == "--help" || a == "/?":
			opts.ShowHelp = true
		case a == "-V" || a == "--version":
			opts.ShowVersion = true
		case a == "-v" || a == "--verbose":
			opts.Verbose = true
		case a == "-q" || a == "--quiet":
			opts.Quiet = true
		case a == "-M" || a == "--map":
			opts.Map = true
		case a == "-d" || a == "--debug-info":
			opts.Debug = true
		case a == "-B" || a == "--build":
			opts.Build = true
		case a == "-R" || a == "--run":
			opts.Run = true
		case a == "-Mmake" || a == "--make":
			opts.Make = true
		case a == "-clean":
			opts.Clean = true
		case a == "-reset":
			opts.Reset = true
		case a == "-Cn" || strings.HasPrefix(a, "-C"):
			// TPC.CFG option
		case a == "-U":
			if i+1 < len(argv) {
				opts.UnitDirs = append(opts.UnitDirs, argv[i+1])
				i++
			}
		case strings.HasPrefix(a, "-U"):
			opts.UnitDirs = append(opts.UnitDirs, a[2:])
		case a == "-I":
			if i+1 < len(argv) {
				opts.IncludeDirs = append(opts.IncludeDirs, argv[i+1])
				i++
			}
		case strings.HasPrefix(a, "-I"):
			opts.IncludeDirs = append(opts.IncludeDirs, a[2:])
		case a == "-O":
			if i+1 < len(argv) {
				opts.ObjectDirs = append(opts.ObjectDirs, argv[i+1])
				i++
			}
		case strings.HasPrefix(a, "-O"):
			opts.ObjectDirs = append(opts.ObjectDirs, a[2:])
		case a == "-D":
			if i+1 < len(argv) {
				opts.Defines = append(opts.Defines, argv[i+1])
				i++
			}
		case strings.HasPrefix(a, "-D"):
			opts.Defines = append(opts.Defines, a[2:])
		case strings.HasPrefix(a, "-m"):
			opts.Memory = a[2:]
		case strings.HasPrefix(a, "-do"):
			opts.Standalone = a[3:]
		case strings.HasPrefix(a, "-E"):
			opts.IniFile = a[2:]
		case a == "-E":
			if i+1 < len(argv) {
				opts.IniFile = argv[i+1]
				i++
			}
		case strings.HasPrefix(a, "-o"):
			opts.Output = a[2:]
		case strings.HasPrefix(a, "--"):
			opts.Args = append(opts.Args, a)
		case strings.HasPrefix(a, "-"):
			// unknown: ignore
		default:
			if opts.Source == "" {
				opts.Source = a
			} else {
				opts.Args = append(opts.Args, a)
			}
		}
	}
	return opts, nil
}

// DefaultUsage returns the standard usage message.
func DefaultUsage() string {
	return `Usage: bpgo [options] file [args]

  -h, --help           show help
  -V, --version        show version
  -v, --verbose        verbose output
  -q, --quiet          suppress info output
  -M, --map            generate map file
  -d, --debug-info     include debug info
  -B, --build          build all (project)
  -R, --run            compile and run
  -U<dir>              add unit search path
  -I<dir>              add include search path
  -O<dir>              add object file search path
  -D<name>             define a symbol
  -m<stack,heap>       memory sizes
  -do<file>            write standalone EXE
  -o<file>             output file
  -E<file>             read configuration file
`
}

// Run is the main entry point. The driver dispatches based on opts.
func Run(argv []string, stdin io.Reader, stdout, stderr io.Writer) int {
	opts, err := ParseOptions(argv)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if opts.ShowHelp {
		fmt.Fprint(stdout, DefaultUsage())
		return 0
	}
	if opts.ShowVersion {
		fmt.Fprintln(stdout, "BPGo 0.1.7")
		return 0
	}
	if opts.Source == "" {
		fmt.Fprintln(stderr, "no source file")
		return 1
	}
	if !fileExists(opts.Source) {
		fmt.Fprintln(stderr, "file not found: "+opts.Source)
		return 1
	}
	// Compile and optionally run.
	out, err := CompileAndRun(opts, stdin, stdout, stderr)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if out != "" {
		fmt.Fprint(stdout, out)
	}
	return 0
}

// CompileAndRun compiles the source with the real BPGo compiler and runs the
// resulting program (when opts.Run is true). When running, console I/O streams
// live (stdin/stdout), so it returns "" and the output is already printed.
func CompileAndRun(opts *Options, stdin io.Reader, stdout, stderr io.Writer) (string, error) {
	src, err := os.ReadFile(opts.Source)
	if err != nil {
		return "", err
	}
	prog, err := codegen.Compile(string(src), opts.Source)
	if err != nil {
		return "", fmt.Errorf("compile: %w", err)
	}
	if !opts.Run {
		return "", nil
	}
	code, err := codegen.RunInteractive(prog, opts.Args, stdin, stdout)
	if err != nil {
		return "", err
	}
	if code != 0 {
		return "", fmt.Errorf("exit code %d", code)
	}
	return "", nil
}

func fileExists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

// LoadConfig reads a TPC.CFG-style configuration file.
func LoadConfig(path string) (*Options, error) {
	opts := &Options{Map: true}
	if _, err := os.Stat(path); err != nil {
		return opts, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		switch parts[0] {
		case "-U":
			if len(parts) > 1 {
				opts.UnitDirs = append(opts.UnitDirs, parts[1:]...)
			}
		case "-I":
			if len(parts) > 1 {
				opts.IncludeDirs = append(opts.IncludeDirs, parts[1:]...)
			}
		case "-O":
			if len(parts) > 1 {
				opts.ObjectDirs = append(opts.ObjectDirs, parts[1:]...)
			}
		case "-D":
			if len(parts) > 1 {
				opts.Defines = append(opts.Defines, parts[1:]...)
			}
		case "-M":
			opts.Map = true
		case "-d":
			opts.Debug = true
		case "-m":
			if len(parts) > 1 {
				opts.Memory = parts[1]
			}
		}
	}
	return opts, nil
}

// MergeOptions merges two option sets. Values from src override dst.
func MergeOptions(dst, src *Options) *Options {
	if src == nil {
		return dst
	}
	if src.Source != "" {
		dst.Source = src.Source
	}
	if src.Output != "" {
		dst.Output = src.Output
	}
	dst.UnitDirs = append(dst.UnitDirs, src.UnitDirs...)
	dst.IncludeDirs = append(dst.IncludeDirs, src.IncludeDirs...)
	dst.ObjectDirs = append(dst.ObjectDirs, src.ObjectDirs...)
	dst.Defines = append(dst.Defines, src.Defines...)
	if src.Config != "" && src.Config != "TPC.CFG" {
		dst.Config = src.Config
	}
	if src.Memory != "" {
		dst.Memory = src.Memory
	}
	if src.Debug {
		dst.Debug = true
	}
	if src.Map {
		dst.Map = true
	}
	if src.Standalone != "" {
		dst.Standalone = src.Standalone
	}
	return dst
}

// SourceUnitPaths returns the resolved unit search paths.
func SourceUnitPaths(opts *Options, sourceFile string) []string {
	dirs := append([]string{}, opts.UnitDirs...)
	dirs = append(dirs, filepath.Dir(sourceFile))
	sort.Strings(dirs)
	return dirs
}
