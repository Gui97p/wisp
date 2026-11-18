package x64

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
)

func constLabel(sym *analyser.Symbol) string {
	return symbolName(sym.Module, sym.Name)
}

func (t *X64Target) constMem(sym *analyser.Symbol, tp analyser.Type) (Mem, error) {
	if sym.Module == "" {
		if mem, ok := t.localConsts[sym]; ok {
			return mem, nil
		}
		mem, err := t.anonymousConst(*sym.Const, tp)
		if err != nil {
			return Mem{}, err
		}
		if t.localConsts == nil {
			t.localConsts = map[*analyser.Symbol]Mem{}
		}
		t.localConsts[sym] = mem
		return mem, nil
	}
	l := constLabel(sym)
	if sym.Module != t.module {
		t.prelude.define(l)
	}
	return label(l, sizeOf(tp)), nil
}

func (t *X64Target) emitConstSymbol(sym *analyser.Symbol, exported bool) error {
	if sym.Const == nil || !sym.Const.Kind.IsAggregate() {
		return nil
	}
	var data strings.Builder
	if err := t.emitConstData(&data, *sym.Const, sym.Type); err != nil {
		return err
	}
	l := constLabel(sym)
	if exported {
		t.rodata.printf("global %s\n", l)
	}
	t.rodata.printf("align 8\n%s:\n%s", l, data.String())
	return nil
}

func (t *X64Target) anonymousConst(value analyser.ConstValue, tp analyser.Type) (Mem, error) {
	t.constCount++
	l := fmt.Sprintf("__wisp_const_%s_%d", symbolName(t.module, "static"), t.constCount)
	var data strings.Builder
	if err := t.emitConstData(&data, value, tp); err != nil {
		return Mem{}, err
	}
	t.rodata.printf("align 8\n%s:\n%s", l, data.String())
	return label(l, sizeOf(tp)), nil
}

func (t *X64Target) emitConstData(out *strings.Builder, value analyser.ConstValue, tp analyser.Type) error {
	size := sizeOf(tp)
	switch u := underlying(tp).(type) {
	case analyser.PrimitiveType:
		if u.Name == "string" {
			dl, dn, err := t.stringData(value.Str)
			if err != nil {
				return err
			}
			emit(out, "dq %s\n\tdq %d\n", dl, dn)
			return nil
		}
		emit(out, "%s ", directive[size])
		switch value.Kind {
		case analyser.ConstBool:
			if value.Bool {
				emitln(out, "1")
			} else {
				emitln(out, "0")
			}
		case analyser.ConstInt:
			if isFloat(tp) {
				emit(out, "%d\n", floatBits(float64(value.Bits()), size))
			} else {
				emit(out, "%d\n", value.Bits())
			}
		case analyser.ConstFloat:
			emit(out, "%d\n", floatBits(value.Float, size))
		default:
			return fmt.Errorf("x86-64: a constant of this kind has no data")
		}
		return nil
	case analyser.ArrayType:
		for _, element := range value.Elems {
			if err := t.emitConstData(out, element, u.Element); err != nil {
				return err
			}
		}
		if missing := int(u.Size) - len(value.Elems); missing > 0 {
			emit(out, "times %d db 0\n", missing*sizeOf(u.Element))
		}
		return nil
	case *analyser.StructType:
		st := structOf(u)
		layout := layoutOf(st)
		pos := 0
		for i, field := range st.Order {
			lField := layout.Fields[field]
			if lField.Offset > pos {
				emit(out, "times %d db 0\n", lField.Offset-pos)
			}
			if i >= len(value.Elems) {
				return fmt.Errorf("x86-64: a struct constant is missing a field")
			}
			if err := t.emitConstData(out, value.Elems[i], lField.Type); err != nil {
				return err
			}
			pos = lField.Offset + lField.Size
		}
		if layout.Size > pos {
			emit(out, "times %d db 0\n", layout.Size-pos)
		}
		return nil
	}
	return fmt.Errorf("x86-64: a constant of type %s has no data", tp.String())
}

func emit(out *strings.Builder, format string, args ...any) {
	fmt.Fprintf(out, "\t"+format, args...)
}

func emitln(out *strings.Builder, text string) {
	out.WriteString(text)
	out.WriteString("\n")
}
