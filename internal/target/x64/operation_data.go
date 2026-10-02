package x64

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/analyser"
)

var setccOperators = map[string]string{
	"==": "sete",
	"!=": "setne",

	"<":  "setl",
	"<=": "setle",

	">":  "setg",
	">=": "setge",
}

var sizeLabels = map[int]string{
	1: "byte",
	2: "word",
	4: "dword",
	8: "qword",
}

func sizeOf(at analyser.Type) int {
	switch tp := at.(type) {
	case analyser.PrimitiveType:
		switch tp.Name {
		case "int8", "uint8", "bool", "char":
			return 1
		case "int16", "uint16":
			return 2
		case "int32", "uint32", "float32":
			return 4
		case "int", "int64", "uint", "uint64", "float64":
			return 8
		case "string":
			return 16
		}
	case analyser.UntypedIntType, analyser.UntypedFloatType:
		return 8
	case analyser.PointerType:
		return 8
	case analyser.ArrayType:
		return int(tp.Size) * sizeOf(tp.Element)
	case *analyser.StructType:
		size := 0
		for _, field := range tp.Fields {
			size += sizeOf(field)
		}
		return size
	case analyser.SpanType:
		return 16
	}

	return 0
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
