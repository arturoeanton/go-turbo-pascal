package codegen8086

import (
	"strings"
	"testing"

	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
)

func TestGenerateSimple(t *testing.T) {
	p := &ir.Program{Entry: "main"}
	p.Modules = []*ir.Module{{Name: "main", Funcs: map[string]*ir.Function{}}}
	fn := ir.NewFunction("main")
	fn.Emit(ir.Instr{Op: ir.OPPushInt, A: 1})
	fn.Emit(ir.Instr{Op: ir.OPPushInt, A: 2})
	fn.Emit(ir.Instr{Op: ir.OPBinary, S: "+"})
	fn.Emit(ir.Instr{Op: ir.OPPop})
	fn.Emit(ir.Instr{Op: ir.OPHalt})
	p.Modules[0].Funcs["main"] = fn
	out, err := Generate(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Modules) != 1 {
		t.Errorf("expected 1 module, got %d", len(out.Modules))
	}
	if len(out.Modules[0].Funcs) != 1 {
		t.Errorf("expected 1 func, got %d", len(out.Modules[0].Funcs))
	}
	body := out.Modules[0].Funcs[0].Body
	if len(body) == 0 {
		t.Error("expected non-empty body")
	}
	if !strings.Contains(out.String(), "module main") {
		t.Error("output should contain module name")
	}
}

func TestGenerateCall(t *testing.T) {
	p := &ir.Program{Entry: "main"}
	p.Modules = []*ir.Module{{Name: "main", Funcs: map[string]*ir.Function{}}}
	main := ir.NewFunction("main")
	main.Emit(ir.Instr{Op: ir.OPCall, S: "helper", A: 0})
	main.Emit(ir.Instr{Op: ir.OPHalt})
	helper := ir.NewFunction("helper")
	helper.Emit(ir.Instr{Op: ir.OPReturn})
	p.Modules[0].Funcs["main"] = main
	p.Modules[0].Funcs["helper"] = helper
	out, err := Generate(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Modules[0].Funcs) != 2 {
		t.Errorf("expected 2 funcs, got %d", len(out.Modules[0].Funcs))
	}
}

func TestLink(t *testing.T) {
	p := &ir.Program{Entry: "main"}
	p.Modules = []*ir.Module{{Name: "main", Funcs: map[string]*ir.Function{}}}
	fn := ir.NewFunction("main")
	fn.Emit(ir.Instr{Op: ir.OPHalt})
	p.Modules[0].Funcs["main"] = fn
	out, err := Generate(p)
	if err != nil {
		t.Fatal(err)
	}
	code, _, ip, cs, _ := out.Link("main")
	if len(code) == 0 {
		t.Error("expected non-empty code segment")
	}
	_ = ip
	_ = cs
}

func TestEncodeInstr(t *testing.T) {
	b := encodeInstr(Instr{Op: "mov", Dst: RegAX("ax"), Src: Imm(5)})
	if len(b) < 1 {
		t.Error("encoded instruction should have bytes")
	}
}

func TestOperandText(t *testing.T) {
	if operandText(RegAX("ax")) != "ax" {
		t.Error("RegAX")
	}
	if operandText(Imm(5)) != "5" {
		t.Error("Imm")
	}
	if operandText(Lab("foo")) != "foo" {
		t.Error("Lab")
	}
}
