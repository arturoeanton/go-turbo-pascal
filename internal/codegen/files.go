package codegen

import (
	"encoding/binary"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
)

// typedRecSize is the on-disk size of one typed-file record (scalar element).
const typedRecSize = 8

// openFile is a file opened by the running program. Reads and writes go through
// the raw *os.File so text lines and binary typed records can share a position.
type openFile struct {
	name string
	f    *os.File
}

// fileManager owns the open files for one VM run, backing the TP7 file builtins.
//
// NOTE: this touches the real filesystem and is only registered by the pasrun
// CLI / codegen.Run. The embeddable pkg/vmpas gates file access through its
// capability sandbox.
type fileManager struct {
	files map[int64]*openFile
	next  int64
}

func (fm *fileManager) handle(ref ir.Value) *openFile {
	h := int64(0)
	if ref.Kind == ir.VKPtr {
		if ref.Cell != nil {
			h = ref.Cell.Int
		}
	} else {
		h = ref.Int
	}
	return fm.files[h]
}

// readLine reads one line (up to '\n') directly from the file.
func readLine(f *os.File) (string, bool) {
	var b []byte
	tmp := make([]byte, 1)
	for {
		n, err := f.Read(tmp)
		if n > 0 {
			if tmp[0] == '\n' {
				return strings.TrimRight(string(b), "\r"), true
			}
			b = append(b, tmp[0])
		}
		if err != nil {
			return strings.TrimRight(string(b), "\r"), len(b) > 0
		}
	}
}

func registerFileIO(vm *ir.VM) {
	fm := &fileManager{files: map[int64]*openFile{}, next: 1}
	nilv := ir.Value{Kind: ir.VKNil}

	vm.Builtins["assign"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		if len(a) < 2 || a[0].Kind != ir.VKPtr || a[0].Cell == nil {
			return nilv
		}
		h := fm.next
		fm.next++
		a[0].Cell.Kind = ir.VKInt
		a[0].Cell.Int = h
		fm.files[h] = &openFile{name: textOf(a[1])}
		return nilv
	}
	open := func(a []ir.Value, opener func(name string) (*os.File, error)) ir.Value {
		if of := fm.handle(arg0(a)); of != nil {
			if of.f != nil {
				of.f.Close()
			}
			if f, err := opener(of.name); err == nil {
				of.f = f
			}
		}
		return nilv
	}
	vm.Builtins["reset"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		return open(a, func(n string) (*os.File, error) { return os.Open(n) })
	}
	vm.Builtins["rewrite"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		return open(a, func(n string) (*os.File, error) { return os.Create(n) })
	}
	vm.Builtins["append"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		return open(a, func(n string) (*os.File, error) {
			return os.OpenFile(n, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		})
	}
	vm.Builtins["close"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		if of := fm.handle(arg0(a)); of != nil && of.f != nil {
			of.f.Close()
			of.f = nil
		}
		return nilv
	}
	vm.Builtins["erase"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		if of := fm.handle(arg0(a)); of != nil {
			os.Remove(of.name)
		}
		return nilv
	}

	// Text I/O: args[0] is the file reference, the rest are values/targets.
	fwrite := func(newline bool) ir.Builtin {
		return func(vm *ir.VM, a []ir.Value) ir.Value {
			if len(a) == 0 {
				return nilv
			}
			of := fm.handle(a[0])
			if of == nil || of.f == nil {
				return nilv
			}
			for _, v := range a[1:] {
				of.f.WriteString(formatWrite(v))
			}
			if newline {
				of.f.WriteString("\n")
			}
			return nilv
		}
	}
	vm.Builtins["__fwrite"] = fwrite(false)
	vm.Builtins["__fwriteln"] = fwrite(true)

	fread := func(vm *ir.VM, a []ir.Value) ir.Value {
		if len(a) == 0 {
			return nilv
		}
		of := fm.handle(a[0])
		if of == nil || of.f == nil {
			return nilv
		}
		line, _ := readLine(of.f)
		assignTokens(line, a[1:])
		return nilv
	}
	vm.Builtins["__fread"] = fread
	vm.Builtins["__freadln"] = fread

	// Typed (binary) I/O: one fixed-size scalar record per call.
	vm.Builtins["__fwritetyped"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		if len(a) < 2 {
			return nilv
		}
		of := fm.handle(a[0])
		if of == nil || of.f == nil {
			return nilv
		}
		var buf [typedRecSize]byte
		v := a[1]
		if v.Kind == ir.VKReal {
			binary.LittleEndian.PutUint64(buf[:], math.Float64bits(v.Real))
		} else {
			binary.LittleEndian.PutUint64(buf[:], uint64(scalarInt(v)))
		}
		of.f.Write(buf[:])
		return nilv
	}
	vm.Builtins["__freadtyped"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		if len(a) < 2 || a[1].Kind != ir.VKPtr || a[1].Cell == nil {
			return nilv
		}
		of := fm.handle(a[0])
		if of == nil || of.f == nil {
			return nilv
		}
		var buf [typedRecSize]byte
		if _, err := of.f.Read(buf[:]); err != nil {
			return nilv
		}
		bits := binary.LittleEndian.Uint64(buf[:])
		cell := a[1].Cell
		switch cell.Kind {
		case ir.VKReal:
			cell.Real = math.Float64frombits(bits)
		case ir.VKChar:
			cell.Ch = byte(bits)
		case ir.VKBool:
			cell.Bool = bits != 0
		default:
			cell.Kind = ir.VKInt
			cell.Int = int64(bits)
		}
		return nilv
	}

	vm.Builtins["eof"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		of := fm.handle(arg0(a))
		if of == nil || of.f == nil {
			return ir.Value{Kind: ir.VKBool, Bool: true}
		}
		pos, _ := of.f.Seek(0, 1)
		fi, err := of.f.Stat()
		if err != nil {
			return ir.Value{Kind: ir.VKBool, Bool: true}
		}
		return ir.Value{Kind: ir.VKBool, Bool: pos >= fi.Size()}
	}
	vm.Builtins["seek"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		if len(a) >= 2 {
			if of := fm.handle(a[0]); of != nil && of.f != nil {
				of.f.Seek(scalarInt(a[1])*typedRecSize, 0)
			}
		}
		return nilv
	}
	vm.Builtins["filepos"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		if of := fm.handle(arg0(a)); of != nil && of.f != nil {
			pos, _ := of.f.Seek(0, 1)
			return ir.Value{Kind: ir.VKInt, Int: pos / typedRecSize}
		}
		return ir.Value{Kind: ir.VKInt}
	}
	vm.Builtins["filesize"] = func(vm *ir.VM, a []ir.Value) ir.Value {
		if of := fm.handle(arg0(a)); of != nil && of.f != nil {
			if fi, err := of.f.Stat(); err == nil {
				return ir.Value{Kind: ir.VKInt, Int: fi.Size() / typedRecSize}
			}
		}
		return ir.Value{Kind: ir.VKInt}
	}
}

func arg0(args []ir.Value) ir.Value {
	if len(args) > 0 {
		return args[0]
	}
	return ir.Value{}
}

func textOf(v ir.Value) string {
	switch v.Kind {
	case ir.VKStr:
		return v.Str
	case ir.VKChar:
		return string([]byte{v.Ch})
	}
	return ""
}

// scalarInt extracts an integer ordinal from a scalar value.
func scalarInt(v ir.Value) int64 {
	switch v.Kind {
	case ir.VKInt:
		return v.Int
	case ir.VKChar:
		return int64(v.Ch)
	case ir.VKBool:
		if v.Bool {
			return 1
		}
	case ir.VKReal:
		return int64(v.Real)
	}
	return 0
}

// assignTokens splits a line and writes its whitespace-separated tokens into
// the reference cells, parsing each by the cell's type (a single string target
// captures the whole line).
func assignTokens(line string, refs []ir.Value) {
	fields := strings.Fields(line)
	fi := 0
	for _, ref := range refs {
		if ref.Kind != ir.VKPtr || ref.Cell == nil {
			continue
		}
		cell := ref.Cell
		switch cell.Kind {
		case ir.VKStr:
			if len(refs) == 1 {
				cell.Str = line
			} else if fi < len(fields) {
				cell.Str = fields[fi]
				fi++
			}
		case ir.VKReal:
			if fi < len(fields) {
				f, _ := strconv.ParseFloat(fields[fi], 64)
				cell.Real = f
				fi++
			}
		case ir.VKChar:
			if len(line) > 0 {
				cell.Ch = line[0]
			}
		default:
			if fi < len(fields) {
				n, _ := strconv.ParseInt(fields[fi], 10, 64)
				cell.Int = n
				cell.Kind = ir.VKInt
				fi++
			}
		}
	}
}
