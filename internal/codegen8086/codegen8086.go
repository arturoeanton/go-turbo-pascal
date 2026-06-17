// Package codegen8086 implements the 8086/286 code generator. The
// generator walks the IR produced by internal/ir and produces textual
// 8086 assembly for the BPGo dos16 backend. The output is intended to
// be assembled with a downstream tool (TLINK, NASM or BPGo's own
// linker) into a DOS MZ executable. The generator targets the small
// memory model: near calls within the code segment, far calls across
// units, and a single data segment. The stack frame layout matches
// TP7/BP7: saved BP, return address, parameters, locals.
package codegen8086

import (
	"fmt"
	"sort"
	"strings"

	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
)

// Target represents the destination of an assembly instruction.
type Target int

const (
	TReg Target = iota
	TMem
	TImm
	TLab
)

// Instr is a single 8086 instruction.
type Instr struct {
	Op    string
	Dst   Operand
	Src   Operand
	Label string
}

// Operand describes an instruction operand.
type Operand struct {
	Kind   Target
	Reg    string
	Disp   int
	Label  string
	Imm    int64
	Size   int // 0=word default, 1=byte
	IsAddr bool
}

func RegAX(o string) Operand { return Operand{Kind: TReg, Reg: o} }
func Mem(o string, disp int) Operand {
	return Operand{Kind: TMem, Reg: o, Disp: disp}
}
func Imm(v int64) Operand     { return Operand{Kind: TImm, Imm: v} }
func Lab(name string) Operand { return Operand{Kind: TLab, Label: name} }

// Func is the assembly representation of a function.
type Func struct {
	Name   string
	Body   []Instr
	Locals map[string]int // name -> offset from BP
}

// Module is a single 8086 assembly module (one per unit).
type Module struct {
	Name    string
	Funcs   []*Func
	Globals []string
	SegCode []byte
	SegData []byte
}

// Program is a collection of modules.
type Program struct {
	Modules []*Module
}

// Generate walks the IR and produces 8086 assembly text.
func Generate(p *ir.Program) (*Program, error) {
	out := &Program{}
	for _, m := range p.Modules {
		mod := &Module{Name: m.Name, Globals: sortedKeys(m.Globals)}
		for _, fn := range sortedFuncs(m.Funcs) {
			f := generateFunc(m.Name, fn)
			mod.Funcs = append(mod.Funcs, f)
		}
		out.Modules = append(out.Modules, mod)
	}
	return out, nil
}

func sortedKeys(globals []ir.Global) []string {
	out := make([]string, len(globals))
	for i, g := range globals {
		out[i] = g.Name
	}
	sort.Strings(out)
	return out
}

func sortedFuncs(m map[string]*ir.Function) []*ir.Function {
	out := make([]*ir.Function, 0, len(m))
	for _, f := range m {
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// generateFunc emits one Func from an IR Function.
func generateFunc(moduleName string, fn *ir.Function) *Func {
	f := &Func{Name: moduleName + "." + fn.Name, Locals: map[string]int{}}
	// Prologue: push BP; mov BP, SP; sub SP, locals.
	f.Body = append(f.Body, Instr{Op: "PROC", Label: f.Name})
	f.Body = append(f.Body, Instr{Op: "push", Dst: RegAX("bp")})
	f.Body = append(f.Body, Instr{Op: "mov", Dst: RegAX("bp"), Src: RegAX("sp")})
	// Reserve local space (16 bytes default per local).
	locals := len(fn.Locals) * 2
	if locals > 0 {
		f.Body = append(f.Body, Instr{Op: "sub", Dst: RegAX("sp"), Src: Imm(int64(locals))})
	}
	for i, n := range fn.Locals {
		f.Locals[n] = -(i + 1) * 2
	}
	for _, p := range fn.Params {
		f.Locals[p] = 4 + 2*indexOf(fn.Params, p) // return addr at 2, old BP at 0
	}
	// Emit body.
	for i, ins := range fn.Code {
		asm := translateInstr(i, ins, fn, f)
		f.Body = append(f.Body, asm...)
	}
	// Epilogue: leave; ret.
	f.Body = append(f.Body, Instr{Op: "leave"})
	f.Body = append(f.Body, Instr{Op: "ret"})
	f.Body = append(f.Body, Instr{Op: "ENDP", Label: f.Name})
	return f
}

func indexOf(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}

// translateInstr converts one IR instruction to a slice of 8086 asm
// instructions. Some IR ops expand to multiple 8086 instructions.
func translateInstr(idx int, ins ir.Instr, fn *ir.Function, f *Func) []Instr {
	switch ins.Op {
	case ir.OPNoop:
		return nil
	case ir.OPEnter, ir.OPLeave:
		return nil
	case ir.OPHalt:
		return []Instr{{Op: "mov", Dst: RegAX("ah"), Src: Imm(0x4C)}, {Op: "int", Dst: Imm(0x21)}}
	case ir.OPError:
		return []Instr{{Op: "mov", Dst: RegAX("ax"), Src: Ins(ins.A)}, {Op: "call", Dst: Lab("__bpgo_runtime_error")}}
	case ir.OPPushInt:
		return []Instr{{Op: "push", Dst: Imm(ins.A)}}
	case ir.OPPushReal:
		// Real is a 6-byte TP value; push 6 bytes.
		return []Instr{{Op: "sub", Dst: RegAX("sp"), Src: Imm(6)}, {Op: fmt.Sprintf("mov dword ptr [bp-%d], %d", idx*4+4, int64(ins.R)), Dst: Imm(0)}}
	case ir.OPPushStr:
		return []Instr{{Op: "push", Dst: Lab("__str_" + labelSuffix(ins.S))}}
	case ir.OPPushBool:
		v := int64(0)
		if ins.A != 0 {
			v = 1
		}
		return []Instr{{Op: "push", Dst: Imm(v)}}
	case ir.OPPushNil:
		return []Instr{{Op: "push", Dst: Imm(0)}}
	case ir.OPLoadGlobal:
		return []Instr{{Op: "push", Dst: Mem("word ptr ["+ins.S+"]", 0)}}
	case ir.OPStoreGlobal:
		return []Instr{{Op: "pop", Dst: RegAX("ax")}, {Op: "mov", Dst: Mem("word ptr ["+ins.S+"]", 0), Src: RegAX("ax")}}
	case ir.OPLoadLocal:
		off, ok := f.Locals[nameForLocal(fn, int(ins.A))]
		if !ok {
			off = int(ins.A) * 2
		}
		return []Instr{{Op: "mov", Dst: RegAX("ax"), Src: Mem("word ptr [bp"+signed(off)+"]", 0)}, {Op: "push", Dst: RegAX("ax")}}
	case ir.OPStoreLocal:
		off, ok := f.Locals[nameForLocal(fn, int(ins.A))]
		if !ok {
			off = int(ins.A) * 2
		}
		return []Instr{{Op: "pop", Dst: RegAX("ax")}, {Op: "mov", Dst: Mem("word ptr [bp"+signed(off)+"]", 0), Src: RegAX("ax")}}
	case ir.OPJump:
		return []Instr{{Op: "jmp", Dst: Lab(labelFor(ins.A))}}
	case ir.OPJumpIfFalse:
		return []Instr{{Op: "pop", Dst: RegAX("ax")}, {Op: "test", Dst: RegAX("ax"), Src: RegAX("ax")}, {Op: "jz", Dst: Lab(labelFor(ins.A))}}
	case ir.OPJumpIfTrue:
		return []Instr{{Op: "pop", Dst: RegAX("ax")}, {Op: "test", Dst: RegAX("ax"), Src: RegAX("ax")}, {Op: "jnz", Dst: Lab(labelFor(ins.A))}}
	case ir.OPReturn:
		return []Instr{{Op: "leave"}, {Op: "ret"}}
	case ir.OPBinary:
		return binaryInstr(ins.S)
	case ir.OPCompare:
		return compareInstr(ins.S)
	case ir.OPUnary:
		return unaryInstr(ins.S)
	case ir.OPCallBuiltin:
		return []Instr{{Op: "call", Dst: Lab("__builtin_" + ins.S)}}
	case ir.OPCall:
		return []Instr{{Op: "call", Dst: Lab(ins.S)}}
	case ir.OPPop:
		return []Instr{{Op: "pop", Dst: RegAX("ax")}}
	case ir.OPDup:
		return []Instr{{Op: "mov", Dst: RegAX("ax"), Src: Mem("word ptr [sp]", 0)}, {Op: "push", Dst: RegAX("ax")}}
	case ir.OPNew:
		return []Instr{{Op: "call", Dst: Lab("__bpgo_new")}}
	case ir.OPDeref:
		return []Instr{{Op: "pop", Dst: RegAX("ax")}, {Op: "mov", Dst: RegAX("ax"), Src: Mem("word ptr [ax]", 0)}, {Op: "push", Dst: RegAX("ax")}}
	case ir.OPIndex:
		return []Instr{{Op: "pop", Dst: RegAX("bx")}, {Op: "pop", Dst: RegAX("ax")}, {Op: "mov", Dst: RegAX("ax"), Src: Mem(fmt.Sprintf("word ptr [ax+bx*%d]", elemSize(ins)), 0)}, {Op: "push", Dst: RegAX("ax")}}
	case ir.OPIndexStore:
		return []Instr{{Op: "pop", Dst: RegAX("cx")}, {Op: "pop", Dst: RegAX("bx")}, {Op: "pop", Dst: RegAX("ax")}, {Op: "mov", Dst: Mem(fmt.Sprintf("word ptr [ax+bx*%d]", elemSize(ins)), 0), Src: RegAX("cx")}}
	case ir.OPIn:
		return []Instr{{Op: "pop", Dst: RegAX("bx")}, {Op: "pop", Dst: RegAX("ax")}, {Op: "call", Dst: Lab("__bpgo_in_set")}}
	case ir.OPMkSet:
		return []Instr{{Op: "call", Dst: Lab("__bpgo_mkset")}}
	case ir.OPMkString:
		return []Instr{{Op: "call", Dst: Lab("__bpgo_mkstring")}}
	case ir.OPMkArray:
		return []Instr{{Op: "call", Dst: Lab("__bpgo_mkarray")}}
	}
	return []Instr{{Op: fmt.Sprintf("; unknown op %d", ins.Op)}}
}

func labelSuffix(s string) string {
	// Hash string to short label
	var h uint32 = 5381
	for _, c := range s {
		h = ((h << 5) + h) + uint32(c)
	}
	return fmt.Sprintf("%08x", h)
}

func labelFor(pc int64) string { return fmt.Sprintf("L%d", pc) }

func nameForLocal(fn *ir.Function, idx int) string {
	if idx < 0 || idx >= len(fn.Locals) {
		return ""
	}
	return fn.Locals[idx]
}

func signed(i int) string {
	if i < 0 {
		return fmt.Sprintf("%d", i)
	}
	return fmt.Sprintf("+%d", i)
}

func elemSize(ins ir.Instr) int { return 2 } // default word

func Ins(v int64) Operand { return Imm(v) }

func binaryInstr(op string) []Instr {
	switch op {
	case "+":
		return []Instr{{Op: "pop", Dst: RegAX("bx")}, {Op: "pop", Dst: RegAX("ax")}, {Op: "add", Dst: RegAX("ax"), Src: RegAX("bx")}, {Op: "push", Dst: RegAX("ax")}}
	case "-":
		return []Instr{{Op: "pop", Dst: RegAX("bx")}, {Op: "pop", Dst: RegAX("ax")}, {Op: "sub", Dst: RegAX("ax"), Src: RegAX("bx")}, {Op: "push", Dst: RegAX("ax")}}
	case "*":
		return []Instr{{Op: "pop", Dst: RegAX("bx")}, {Op: "pop", Dst: RegAX("ax")}, {Op: "imul", Dst: RegAX("bx")}, {Op: "push", Dst: RegAX("ax")}}
	case "/", "div":
		return []Instr{{Op: "pop", Dst: RegAX("bx")}, {Op: "pop", Dst: RegAX("ax")}, {Op: "cwd"}, {Op: "idiv", Dst: RegAX("bx")}, {Op: "push", Dst: RegAX("ax")}}
	case "mod":
		return []Instr{{Op: "pop", Dst: RegAX("bx")}, {Op: "pop", Dst: RegAX("ax")}, {Op: "cwd"}, {Op: "idiv", Dst: RegAX("bx")}, {Op: "push", Dst: RegAX("dx")}}
	case "and":
		return []Instr{{Op: "pop", Dst: RegAX("bx")}, {Op: "pop", Dst: RegAX("ax")}, {Op: "and", Dst: RegAX("ax"), Src: RegAX("bx")}, {Op: "push", Dst: RegAX("ax")}}
	case "or":
		return []Instr{{Op: "pop", Dst: RegAX("bx")}, {Op: "pop", Dst: RegAX("ax")}, {Op: "or", Dst: RegAX("ax"), Src: RegAX("bx")}, {Op: "push", Dst: RegAX("ax")}}
	case "xor":
		return []Instr{{Op: "pop", Dst: RegAX("bx")}, {Op: "pop", Dst: RegAX("ax")}, {Op: "xor", Dst: RegAX("ax"), Src: RegAX("bx")}, {Op: "push", Dst: RegAX("ax")}}
	case "shl":
		return []Instr{{Op: "pop", Dst: RegAX("cx")}, {Op: "pop", Dst: RegAX("ax")}, {Op: "shl", Dst: RegAX("ax"), Src: RegAX("cl")}, {Op: "push", Dst: RegAX("ax")}}
	case "shr":
		return []Instr{{Op: "pop", Dst: RegAX("cx")}, {Op: "pop", Dst: RegAX("ax")}, {Op: "shr", Dst: RegAX("ax"), Src: RegAX("cl")}, {Op: "push", Dst: RegAX("ax")}}
	}
	return []Instr{{Op: "; unsupported binop " + op}}
}

func compareInstr(op string) []Instr {
	switch op {
	case "=":
		return []Instr{{Op: "pop", Dst: RegAX("bx")}, {Op: "pop", Dst: RegAX("ax")}, {Op: "cmp", Dst: RegAX("ax"), Src: RegAX("bx")}, {Op: "sete", Dst: RegAX("al")}, {Op: "movzx", Dst: RegAX("ax"), Src: RegAX("al")}, {Op: "push", Dst: RegAX("ax")}}
	case "<>":
		return []Instr{{Op: "pop", Dst: RegAX("bx")}, {Op: "pop", Dst: RegAX("ax")}, {Op: "cmp", Dst: RegAX("ax"), Src: RegAX("bx")}, {Op: "setne", Dst: RegAX("al")}, {Op: "movzx", Dst: RegAX("ax"), Src: RegAX("al")}, {Op: "push", Dst: RegAX("ax")}}
	case "<":
		return []Instr{{Op: "pop", Dst: RegAX("bx")}, {Op: "pop", Dst: RegAX("ax")}, {Op: "cmp", Dst: RegAX("ax"), Src: RegAX("bx")}, {Op: "setl", Dst: RegAX("al")}, {Op: "movzx", Dst: RegAX("ax"), Src: RegAX("al")}, {Op: "push", Dst: RegAX("ax")}}
	case "<=":
		return []Instr{{Op: "pop", Dst: RegAX("bx")}, {Op: "pop", Dst: RegAX("ax")}, {Op: "cmp", Dst: RegAX("ax"), Src: RegAX("bx")}, {Op: "setle", Dst: RegAX("al")}, {Op: "movzx", Dst: RegAX("ax"), Src: RegAX("al")}, {Op: "push", Dst: RegAX("ax")}}
	case ">":
		return []Instr{{Op: "pop", Dst: RegAX("bx")}, {Op: "pop", Dst: RegAX("ax")}, {Op: "cmp", Dst: RegAX("ax"), Src: RegAX("bx")}, {Op: "setg", Dst: RegAX("al")}, {Op: "movzx", Dst: RegAX("ax"), Src: RegAX("al")}, {Op: "push", Dst: RegAX("ax")}}
	case ">=":
		return []Instr{{Op: "pop", Dst: RegAX("bx")}, {Op: "pop", Dst: RegAX("ax")}, {Op: "cmp", Dst: RegAX("ax"), Src: RegAX("bx")}, {Op: "setge", Dst: RegAX("al")}, {Op: "movzx", Dst: RegAX("ax"), Src: RegAX("al")}, {Op: "push", Dst: RegAX("ax")}}
	}
	return []Instr{{Op: "; unsupported cmp " + op}}
}

func unaryInstr(op string) []Instr {
	switch op {
	case "-":
		return []Instr{{Op: "pop", Dst: RegAX("ax")}, {Op: "neg", Dst: RegAX("ax")}, {Op: "push", Dst: RegAX("ax")}}
	case "not":
		return []Instr{{Op: "pop", Dst: RegAX("ax")}, {Op: "test", Dst: RegAX("ax"), Src: RegAX("ax")}, {Op: "setz", Dst: RegAX("al")}, {Op: "movzx", Dst: RegAX("ax"), Src: RegAX("al")}, {Op: "push", Dst: RegAX("ax")}}
	}
	return []Instr{{Op: "; unsupported unary " + op}}
}

// String returns a textual representation of the program for tests.
func (p *Program) String() string {
	var b strings.Builder
	for _, m := range p.Modules {
		fmt.Fprintf(&b, "module %s\n", m.Name)
		if len(m.Globals) > 0 {
			b.WriteString("  globals:\n")
			for _, g := range m.Globals {
				fmt.Fprintf(&b, "    %s\n", g)
			}
		}
		for _, f := range m.Funcs {
			fmt.Fprintf(&b, "  %s\n", f.Name)
			for _, ins := range f.Body {
				fmt.Fprintf(&b, "    %s %s %s\n", ins.Op, operandText(ins.Dst), operandText(ins.Src))
			}
		}
	}
	return b.String()
}

func operandText(o Operand) string {
	switch o.Kind {
	case TReg:
		return o.Reg
	case TMem:
		return fmt.Sprintf("[%s%+d]", o.Reg, o.Disp)
	case TImm:
		return fmt.Sprintf("%d", o.Imm)
	case TLab:
		return o.Label
	}
	return ""
}

// Link produces a flat list of segments ready to be encoded as MZ
// image. It lays out the code segment first, then data, and computes
// the CS:IP for the entry point.
func (p *Program) Link(entry string) (codeSeg, dataSeg []byte, ip uint16, cs uint16, relocs []Relocation) {
	// Find entry function.
	var entryFn *Func
	for _, m := range p.Modules {
		for _, f := range m.Funcs {
			if f.Name == entry {
				entryFn = f
			}
		}
	}
	if entryFn == nil {
		// Use the first function.
		for _, m := range p.Modules {
			if len(m.Funcs) > 0 {
				entryFn = m.Funcs[0]
				break
			}
		}
	}
	// Compute code segment bytes.
	for _, m := range p.Modules {
		for _, f := range m.Funcs {
			for _, ins := range f.Body {
				codeSeg = append(codeSeg, encodeInstr(ins)...)
			}
		}
	}
	// Compute IP relative to entry function.
	if entryFn != nil {
		ip = 0 // simplified
		cs = 0
		_ = entryFn
	}
	return
}

// Relocation is a placeholder for a code relocation.
type Relocation struct {
	Offset int
	Label  string
}

// encodeInstr returns 0..N bytes for an 8086 instruction. This is a
// simplified assembler: it emits placeholder opcodes (1 byte) followed
// by operands. The output is not executable but is a stable
// representation useful for golden tests and link debugging.
func encodeInstr(ins Instr) []byte {
	if ins.Label != "" && (ins.Op == "PROC" || ins.Op == "ENDP") {
		return []byte{} // labels are 0 bytes
	}
	out := []byte{opCode(ins.Op)}
	out = append(out, encodeOperand(ins.Dst)...)
	out = append(out, encodeOperand(ins.Src)...)
	return out
}

func encodeOperand(o Operand) []byte {
	if o.Kind == TImm {
		// 16-bit immediate.
		return []byte{byte(o.Imm), byte(o.Imm >> 8)}
	}
	return nil
}

func opCode(name string) byte {
	switch name {
	case "mov":
		return 0x89
	case "push":
		return 0x50
	case "pop":
		return 0x58
	case "add":
		return 0x01
	case "sub":
		return 0x29
	case "imul":
		return 0xF7
	case "idiv":
		return 0xF7
	case "cwd":
		return 0x99
	case "and":
		return 0x21
	case "or":
		return 0x09
	case "xor":
		return 0x31
	case "shl":
		return 0xD1
	case "shr":
		return 0xD1
	case "cmp":
		return 0x39
	case "jmp":
		return 0xEB
	case "jz", "je":
		return 0x74
	case "jnz", "jne":
		return 0x75
	case "jl", "jnge":
		return 0x7C
	case "jle", "jng":
		return 0x7E
	case "jg", "jnle":
		return 0x7F
	case "jge", "jnl":
		return 0x7D
	case "call":
		return 0xE8
	case "ret":
		return 0xC3
	case "leave":
		return 0xC9
	case "int":
		return 0xCD
	case "test":
		return 0x85
	case "neg":
		return 0xF7
	case "setz", "sete", "setne", "setl", "setle", "setg", "setge", "movzx":
		return 0xFE // 2-byte opcode, simplified
	case "":
		return 0x90 // nop
	}
	return 0xFF // unknown
}
