package x64

import (
	"fmt"
	"slices"

	"github.com/Gui97p/wisp/internal/analyser"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

var setccOperators = map[string]string{
	"==": "e",
	"!=": "ne",

	"<":  "l",
	"<=": "le",

	">":  "g",
	">=": "ge",
}

var usetccOperators = map[string]string{
	"==": "e",
	"!=": "ne",

	"<":  "b",
	"<=": "be",

	">":  "a",
	">=": "ae",
}

var ucomiOperators = map[string]string{
	"==": "sete",
	"!=": "setne",

	"<":  "seta",
	"<=": "setae",

	">":  "seta",
	">=": "setae",
}

var inverseCondition = map[string]string{
	"e": "ne", "ne": "e",
	"l": "ge", "ge": "l",
	"le": "g", "g": "le",
	"b": "ae", "ae": "b",
	"be": "a", "a": "be",
}

var castOperators = map[int]map[int]string{
	1: {
		2: "movsx",
		4: "movsx",
		8: "movsx",
	},
	2: {
		4: "movsx",
		8: "movsx",
	},
	4: {
		8: "movsxd",
	},
}

var ucastOperators = map[int]map[int]string{
	1: {
		2: "movzx",
		4: "movzx",
		8: "movzx",
	},
	2: {
		4: "movzx",
		8: "movzx",
	},
	4: {
		8: "mov",
	},
}

var mathOperators = map[string]string{
	"+": "add",
	"-": "sub",
	"*": "mul",
	"/": "div",
	"&": "and",
	"|": "or",
	"^": "xor",
}

var unaryOperators = map[string]string{
	"-": "neg",
	"~": "not",
}

var incDecOperators = map[string]string{
	"++": "inc",
	"--": "dec",
}

var sizeLabels = map[int]string{
	1: "byte",
	2: "word",
	4: "dword",
	8: "qword",
}

var directive = map[int]string{
	1: "db",
	2: "dw",
	4: "dd",
	8: "dq",
}

var repRegisters = []x64context.Register{x64context.AX, x64context.CX, x64context.SI, x64context.DI}

const REP_THRESHOLD = 1024

func (t *X64Target) pushRep(except ...x64context.Register) (regs []x64context.Register) {
	for _, r := range repRegisters {
		if t.ctx.IsAllocated(r) && !slices.Contains(except, r) {
			t.push(r)
			regs = append(regs, r)
		}
	}
	return
}

func (t *X64Target) popRep(regs []x64context.Register) {
	for _, r := range slices.Backward(regs) {
		t.pop(r)
	}
}

func alignUp(n, a int) int {
	if a <= 1 {
		return n
	}
	return (n + a - 1) / a * a
}

type fieldLayout = analyser.FieldLayout

type structLayout = analyser.StructLayout

func layoutOf(st *analyser.StructType) *structLayout {
	return analyser.LayoutOf(st)
}

func alignOf(at analyser.Type) int {
	return analyser.AlignOf(at)
}

func sizeOf(at analyser.Type) int {
	return analyser.SizeOf(at)
}

func structOf(tp analyser.Type) *analyser.StructType {
	switch t := tp.(type) {
	case *analyser.StructType:
		return t
	case analyser.NamedType:
		return structOf(t.Underlying)
	case analyser.PointerType:
		return structOf(t.Element)
	}
	return nil
}

func arrayOf(tp analyser.Type) (analyser.ArrayType, bool) {
	switch t := tp.(type) {
	case analyser.ArrayType:
		return t, true
	case analyser.NamedType:
		return arrayOf(t.Underlying)
	}
	return analyser.ArrayType{}, false
}

func (t *X64Target) newLabel(key string) string {
	if _, ok := t.labels[key]; !ok {
		t.labels[key] = 0
	}

	label := fmt.Sprintf(".%s%d", key, t.labels[key])
	t.labels[key]++
	return label
}

func (t *X64Target) alignStack(n int) {
	if n == 0 {
		return
	}
	inst := "add"
	if n < 0 {
		inst = "sub"
	}
	t.ctx.AddPushed(-n / 8)
	t.text.printft("%s rsp, %d\n", inst, abs(n))
}

func (t *X64Target) createData(key string, value any) (string, bool) {
	if label, ok := t.dataLabel[value]; ok {
		return label, true
	}

	label := fmt.Sprintf("%s$%s%d", symbolName(t.module), key, t.labelCount)
	t.dataLabel[value] = label
	t.labelCount++
	return label, false
}

func (t *X64Target) stringData(s string) (string, int, error) {
	op, err := t.resolveStringLiteral(s)
	if err != nil {
		return "", 0, err
	}
	w := op.(Wide)
	return w.Words[0].(Addr).Of.Label, int(w.Words[1].(Imm)), nil
}
