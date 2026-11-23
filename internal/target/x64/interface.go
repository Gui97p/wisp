package x64

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

func (t *X64Target) descriptorText(tp analyser.Type) string {
	if tag, ok := analyser.InstanceTag(tp); ok {
		return strconv.FormatInt(tag, 10)
	}
	mod, key, _ := analyser.DescriptorOf(tp)
	sym := "__wisp_type_" + re.ReplaceAllString(key, "_")
	if mod != t.module {
		t.prelude.define(sym)
	}
	return sym
}

func (t *X64Target) vtableLabel(conv *analyser.IfaceConversion) string {
	key := "\x00vtable|" + analyser.BoxKey(conv.From.Element) + "|" + conv.Iface.Key()
	if tag, ok := analyser.InstanceTag(conv.From); ok {
		key += "|" + strconv.FormatInt(tag, 10)
	}
	label, existed := t.createData("vt", key)
	if existed {
		return label
	}

	words := []string{t.descriptorText(conv.From)}
	for _, entry := range conv.Entries {
		sym := t.funcSymbol(entry.Module, fmt.Sprintf("%s_%s", entry.Owner, entry.Name))
		if entry.Module != t.module {
			t.prelude.define(sym)
		}
		words = append(words, sym)
	}
	t.rodata.printf("align 8\n%s: dq %s\n", label, strings.Join(words, ", "))
	return label
}

func (t *X64Target) compileInterfaceConversion(expr *ast.CastExpr, conv *analyser.IfaceConversion) (Operand, error) {
	op, err := t.compileExpr(expr.Value)
	if err != nil {
		return nil, err
	}
	table := t.vtableLabel(conv)
	return Wide{Words: []Operand{op, Addr{Of: label(table, 8)}}, Size: 16}, nil
}

func (t *X64Target) compileInterfaceCall(expr *ast.CallExpr, mc *analyser.MethodCall) ([]Operand, error) {
	op, err := t.compileExpr(mc.Receiver)
	if err != nil {
		return nil, err
	}
	object := slot(t.ctx.Reserve(16), 16)
	t.store(op, object)
	t.freeOp(op)

	reg := t.ctx.AllocFreeRegister()
	regStr := t.ctx.GetRegister(reg, 8)
	t.loadWord(reg, object.at(8, 8))
	t.text.printft("mov %s, [%s + %d]\n", regStr, regStr, 8*(1+mc.Slot))
	target := slot(t.ctx.Reserve(8), 8)
	t.storeWord(target, Reg{reg, 8})
	t.ctx.FreeRegister(reg)

	data := t.hiddenArg(analyser.PointerType{Element: analyser.VoidType{}}, object.at(0, 8))
	args := append([]ast.Expression{data}, expr.Args...)
	return t.compileCall(t.memText(target), false, args, t.info.CallReturns[expr], false, false)
}

func (t *X64Target) compileInterfaceTypeCase(expr *ast.BinaryExpr, tc *analyser.TypeCase) (Operand, error) {
	op, err := t.compileExpr(expr.Left)
	if err != nil {
		return nil, err
	}
	subject := op.(Mem)

	result := t.ctx.AllocFreeRegister()
	t.text.printft("xor %s, %s\n", t.ctx.GetRegister(result, 4), t.ctx.GetRegister(result, 4))

	tag := t.ctx.AllocFreeRegister()
	tagStr := t.ctx.GetRegister(tag, 8)
	t.loadWord(tag, subject.at(8, 8))
	t.text.printft("mov %s, [%s]\n", tagStr, tagStr)

	want := t.ctx.AllocFreeRegister()
	text := t.descriptorText(tc.Type)
	if _, err := strconv.ParseInt(text, 10, 64); err == nil {
		t.text.printft("mov %s, %s\n", t.ctx.GetRegister(want, 8), text)
	} else {
		t.text.printft("lea %s, [rel %s]\n", t.ctx.GetRegister(want, 8), text)
	}
	t.text.printft("cmp %s, %s\n", tagStr, t.ctx.GetRegister(want, 8))
	t.text.printft("sete %s\n", t.ctx.GetRegister(result, 1))

	t.ctx.FreeRegister(want)
	t.ctx.FreeRegister(tag)
	t.freeOp(subject)
	return Reg{result, 1}, nil
}

func (t *X64Target) compileInterfaceData(expr *ast.CastExpr) (Operand, error) {
	op, err := t.compileExpr(expr.Value)
	if err != nil {
		return nil, err
	}
	object := slot(t.ctx.Reserve(16), 16)
	t.store(op, object)
	t.freeOp(op)

	reg := t.ctx.AllocFreeRegister()
	t.loadWord(reg, object.at(0, 8))
	return Reg{reg, 8}, nil
}
