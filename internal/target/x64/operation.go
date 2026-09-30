package x64

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

type Operand interface {
	op()
}

type Imm int64

func (Imm) op() {}

type StrOperand struct {
	Label string
	Len   int
}

func (StrOperand) op() {}

type MemOperand struct {
	Offset int
	Size   int
}

func (MemOperand) op() {}

type RegOperand struct {
	Reg  x64context.Reg
	Size int
}

func (RegOperand) op() {}

func (t *X64Target) ensureRegister(op Operand, size int) x64context.Reg {
	if r, ok := op.(RegOperand); ok {
		return r.Reg
	}

	reg := t.ctx.AllocFreeRegister()
	t.text.printft("mov %s, %d\n", t.ctx.GetRegister(reg, size), op.(Imm))
	return reg
}

func (t *X64Target) operandText(op Operand, size int) (string, error) {
	switch o := op.(type) {
	case Imm:
		return fmt.Sprint(o), nil
	case RegOperand:
		return t.ctx.GetRegister(o.Reg, size), nil
	default:
		return "", fmt.Errorf("x86-64: unsupported operand")
	}
}

func (t *X64Target) storeOperand(op Operand, offset, size int) error {
	switch o := op.(type) {
	case Imm, RegOperand:
		text, err := t.operandText(op, size)
		if err != nil {
			return err
		}
		t.text.printft("mov %s [rbp-%d], %s\n", sizeLabels[size], offset, text)
		return nil
	case StrOperand:
		reg := t.ctx.AllocFreeRegister()
		regStr := t.ctx.GetRegister(reg, 8)

		t.text.printft("lea %s, [rel %s]\n", regStr, o.Label)
		t.text.printft("mov qword [rbp-%d], %s\n", offset, regStr)
		t.text.printft("mov qword [rbp-(%d-8)], %d\n", offset, o.Len)
		t.ctx.FreeRegister(reg)

		return nil
	case MemOperand:
		reg := t.ctx.AllocFreeRegister()
		regStr := t.ctx.GetRegister(reg, 8)

		for i := 0; i < size-1; i += 8 {
			t.text.printft("mov %s, %s\n", regStr, t.mem(o.Offset-i, 8))
			t.text.printft("mov %s, %s\n", t.mem(offset-i, 8), regStr)
		}
		t.ctx.FreeRegister(reg)

		return nil
	default:
		return fmt.Errorf("x86-64: unsupported operand")
	}
}

func (t *X64Target) freeOperand(op Operand) {
	o, ok := op.(RegOperand)
	if !ok {
		return
	}

	t.ctx.FreeRegister(o.Reg)
}

func (t *X64Target) mem(offset, size int) string {
	return fmt.Sprintf("%s [rbp-%d]", sizeLabels[min(size, 8)], offset)
}

var setccOperators = map[string]string{
	"==": "sete",
	"!=": "setne",

	"<":  "setl",
	"<=": "setle",

	">":  "setg",
	">=": "setge",
}

func (t *X64Target) push(operand string) {
	t.text.printft("push %s\n", operand)
	t.ctx.AddPushed(1)
}

func (t *X64Target) pop(operand string) {
	t.text.printft("pop %s\n", operand)
	t.ctx.AddPushed(-1)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
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

var sizeLabels = map[int]string{
	1: "byte",
	2: "word",
	4: "dword",
	8: "qword",
}

func (t *X64Target) sizeOf(at analyser.Type) int {
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
		return int(tp.Size) * t.sizeOf(tp.Element)
	case *analyser.StructType:
		size := 0
		for _, field := range tp.Fields {
			size += t.sizeOf(field)
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

func (t *X64Target) createData(key string, value any) (string, bool) {
	if label, ok := t.dataLabel[value]; ok {
		return label, true
	}

	label := fmt.Sprintf("%s$%s%d", symbolName(t.module), key, t.labelCount)
	t.dataLabel[value] = label
	t.labelCount++
	return label, false
}

func decodeEscapes(s string) ([]byte, error) {
	out := make([]byte, 0, len(s))

	for i := 0; i < len(s); i++ {
		if s[i] != '\\' {
			out = append(out, s[i])
			continue
		}

		i++
		if i >= len(s) {
			return nil, fmt.Errorf("unterminated escape sequence")
		}

		switch s[i] {
		case 'n':
			out = append(out, '\n')
		case 'r':
			out = append(out, '\r')
		case 't':
			out = append(out, '\t')
		case 'b':
			out = append(out, '\b')
		case 'f':
			out = append(out, '\f')
		case 'v':
			out = append(out, '\v')
		case 'a':
			out = append(out, '\a')
		case '\\':
			out = append(out, '\\')
		case '"':
			out = append(out, '"')
		case '\'':
			out = append(out, '\'')
		case '0':
			out = append(out, 0)

		case 'x':
			if i+2 >= len(s) {
				return nil, fmt.Errorf("invalid hexadecimal escape")
			}

			hi, ok1 := hexValue(s[i+1])
			lo, ok2 := hexValue(s[i+2])
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("invalid hexadecimal escape")
			}

			out = append(out, hi<<4|lo)
			i += 2

		default:
			return nil, fmt.Errorf("unknown escape sequence: \\%c", s[i])
		}
	}

	return out, nil
}

func hexValue(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	default:
		return 0, false
	}
}

func bytesToAsm(b []byte) string {
	var sb strings.Builder

	for i, v := range b {
		if i > 0 {
			sb.WriteString(", ")
		}

		sb.WriteString(strconv.Itoa(int(v)))
	}

	return sb.String()
}
