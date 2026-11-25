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

func (t *X64Target) emitInterfaceData(expr ast.Expression) error {
	err := fmt.Errorf("x86-64: unsupported global initializer")
	cast, ok := expr.(*ast.CastExpr)
	if !ok {
		return err
	}
	conv, ok := t.info.IfaceConversions[cast]
	if !ok {
		return err
	}
	address, ok := cast.Value.(*ast.UnaryExpr)
	if !ok {
		return err
	}
	id, ok := address.Value.(*ast.IdentLiteral)
	if !ok {
		return err
	}
	sym := t.info.Idents[id]
	if sym == nil {
		return err
	}
	object := symbolName(sym.Module, sym.EmitName())
	if sym.Module != t.module {
		t.prelude.define(object)
	}
	t.data.printft("dq %s\n\tdq %s\n", object, t.vtableLabel(conv))
	return nil
}

func (t *X64Target) ifaceDescriptor(it *analyser.InterfaceType) string {
	sym := "__wisp_iface_" + re.ReplaceAllString(analyser.TypeKey(it.Module, it.Name), "_")
	if it.Module != t.module {
		t.prelude.define(sym)
	}
	return sym
}

func (t *X64Target) dynamicTable(key string) string {
	var pairs []string
	for _, impl := range t.info.DynamicImpls {
		module, implKey, ok := analyser.DescriptorOf(impl.From)
		if !ok || module != t.module || implKey != key {
			continue
		}
		pairs = append(pairs, t.ifaceDescriptor(impl.Iface), t.vtableLabel(impl))
	}
	if len(pairs) == 0 {
		return "0"
	}
	label, existed := t.createData("dyn", "\x00dyn|"+key)
	if !existed {
		t.rodata.printf("align 8\n%s: dq %d, %s\n", label, len(pairs)/2, strings.Join(pairs, ", "))
	}
	return label
}

func (t *X64Target) dynamicVtable(box Mem, it *analyser.InterfaceType) (Operand, error) {
	desc := t.ctx.AllocFreeRegister()
	t.loadWord(desc, box.at(0, 8))
	skip := t.newLabel("dyn_named")
	t.text.printft("cmp %s, %d\n", t.ctx.GetRegister(desc, 8), analyser.BoxCode("pointer"))
	t.text.printft("jne %s\n", skip)
	t.loadWord(desc, box.at(16, 8))
	t.text.printf("%s:\n", skip)

	void := analyser.PointerType{Element: analyser.VoidType{}}
	descArg := t.hiddenArg(void, Reg{desc, 8})
	t.ctx.FreeRegister(desc)
	ifaceArg := t.hiddenArg(void, Addr{Of: label(t.ifaceDescriptor(it), 8)})

	ops, err := t.compileCall(t.useDyn(), false, []ast.Expression{descArg, ifaceArg}, []analyser.Type{void}, false, false)
	if err != nil {
		return nil, err
	}
	return ops[0], nil
}

func (t *X64Target) compileDynamicCase(expr *ast.BinaryExpr, tc *analyser.TypeCase) (Operand, error) {
	op, err := t.compileExpr(expr.Left)
	if err != nil {
		return nil, err
	}
	box := op.(Mem)

	vtable, err := t.dynamicVtable(box, tc.Dynamic)
	if err != nil {
		return nil, err
	}
	reg := t.ctx.AllocFreeRegister()
	t.text.printft("xor %s, %s\n", t.ctx.GetRegister(reg, 4), t.ctx.GetRegister(reg, 4))
	t.text.printft("test %s, %s\n", t.opText(vtable, 8), t.opText(vtable, 8))
	t.text.printft("setne %s\n", t.ctx.GetRegister(reg, 1))
	t.freeOp(vtable)
	t.freeOp(box)
	return Reg{reg, 1}, nil
}

func (t *X64Target) dynamicValue(box Mem, it *analyser.InterfaceType) (Operand, error) {
	vtable, err := t.dynamicVtable(box, it)
	if err != nil {
		return nil, err
	}
	object := slot(t.ctx.Reserve(16), 16)
	t.storeWord(object.at(8, 8), vtable)
	t.freeOp(vtable)

	data := t.ctx.AllocFreeRegister()
	dataStr := t.ctx.GetRegister(data, 8)
	other := t.newLabel("dyn_data")
	done := t.newLabel("dyn_data_done")
	t.loadWord(data, box.at(0, 8))
	t.text.printft("cmp %s, %d\n", dataStr, analyser.BoxCode("pointer"))
	t.text.printft("jne %s\n", other)
	t.loadWord(data, box.at(8, 8))
	t.text.printft("jmp %s\n", done)
	t.text.printf("%s:\n", other)
	t.loadWord(data, Addr{Of: box.at(8, 8)})
	t.text.printf("%s:\n", done)
	t.storeWord(object.at(0, 8), Reg{data, 8})
	t.ctx.FreeRegister(data)
	return object, nil
}
