package x64

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/diag"
	"github.com/Gui97p/wisp/internal/target"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

type nativeVar struct {
	mem Mem
	tp  analyser.Type
}

func nativeStruct(tp analyser.Type) *analyser.StructType {
	switch v := tp.(type) {
	case *analyser.StructType:
		return v
	case analyser.NamedType:
		return nativeStruct(v.Underlying)
	}
	return nil
}

func isWordPair(tp analyser.Type) bool {
	switch v := tp.(type) {
	case analyser.PrimitiveType:
		return v.Name == "string"
	case analyser.SpanType:
		return true
	case analyser.NamedType:
		return isWordPair(v.Underlying)
	}
	return false
}

func (t *X64Target) nativeOperand(ref string, path []string, addr bool, v nativeVar) (string, error) {
	mem := v.mem
	tp := v.tp

	for _, part := range path {
		switch {
		case nativeStruct(tp) != nil:
			st := nativeStruct(tp)
			f, ok := layoutOf(st).Fields[part]
			if !ok {
				return "", fmt.Errorf("struct %s has no field %s in {%s}", st.Name, part, ref)
			}
			mem = mem.at(f.Offset, f.Size)
			tp = f.Type
		case isWordPair(tp) && (part == "ptr" || part == "len"):
			if part == "ptr" {
				mem = mem.at(0, 8)
			} else {
				mem = mem.at(8, 8)
			}
			tp = analyser.PrimitiveType{Name: "int"}
		default:
			return "", fmt.Errorf("cannot access field %s in {%s}", part, ref)
		}
	}

	text := t.memText(mem)
	if addr {
		return text[strings.Index(text, "["):], nil
	}
	if mem.Size > 8 {
		return "", fmt.Errorf("{%s} is too big for one operand: use {&%s} for its address", ref, ref)
	}
	return text, nil
}

func nativeError(at ast.Position, format string, args ...any) error {
	return &diag.Error{Diagnostic: diag.Diagnostic{
		Message: fmt.Sprintf(format, args...),
		Origin:  "x86-64",
		Line:    at.Line, Col: at.Col, EndLine: at.EndLine, EndCol: at.EndCol,
	}}
}

func (t *X64Target) nativeActive(stmt *ast.NativeStmt) bool {
	return stmt.Backend == "" || target.MatchesName(t.target, stmt.Backend)
}

func (t *X64Target) nativeRegisters(root ast.Node) map[string]bool {
	out := map[string]bool{}
	var walk func(ast.Node)
	walk = func(root ast.Node) {
		ast.Inspect(root, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.NativeStmt:
				if info, ok := t.info.Natives[n]; ok && t.nativeActive(n) {
					for _, name := range info.Registers {
						out[name] = true
					}
				}
				return false
			case *ast.CallExpr:
				if call, ok := t.info.Inlines[n]; ok {
					walk(call.Body)
				}
			case *ast.FuncLiteral:
				return false
			}
			return true
		})
	}
	walk(root)
	return out
}

func (t *X64Target) nativeVariable(ref analyser.NativeRef, at ast.Position) (nativeVar, error) {
	sym := ref.Sym
	size := sizeOf(sym.Type)
	if sym.Global {
		return nativeVar{mem: t.globalMem(sym, size), tp: sym.Type}, nil
	}
	offset, ok := t.ctx.Get(sym)
	if !ok {
		return nativeVar{}, nativeError(at, "variable %s is not declared in this scope", sym.Name)
	}
	if _, inRegister := t.ctx.Register(sym); inRegister {
		return nativeVar{}, nativeError(at, "variable %s lives in a register: the native block needs it in memory", sym.Name)
	}
	return nativeVar{mem: slot(offset, size), tp: sym.Type}, nil
}

func (t *X64Target) nativeRef(ref analyser.NativeRef, addr bool, prefix string, at ast.Position) (string, error) {
	switch ref.Kind {
	case analyser.RefRegister:
		return ref.Reg.Name, nil
	case analyser.RefNumber, analyser.RefExtern:
		return ref.Text, nil
	case analyser.RefLabel:
		return fmt.Sprintf(".%s_%s", prefix, ref.Text), nil
	case analyser.RefFunction:
		return t.funcLabel(ref.Sym), nil
	case analyser.RefConstPart:
		label, size, err := t.stringData(ref.Sym.Const.Str)
		if err != nil {
			return "", nativeError(at, "%s", err)
		}
		if len(ref.Path) == 1 && ref.Path[0] == "len" {
			return fmt.Sprint(size), nil
		}
		return label, nil
	}

	variable, err := t.nativeVariable(ref, at)
	if err != nil {
		return "", err
	}
	text, err := t.nativeOperand(ref.Sym.Name, ref.Path, addr, variable)
	if err != nil {
		return "", nativeError(at, "%s", err)
	}
	return text, nil
}

func (t *X64Target) nativeOperandText(mnemonic string, op ast.NativeOperand, info analyser.NativeOperandInfo, prefix string) (string, error) {
	if !info.Mem {
		text, err := t.nativeRef(info.Ref, info.Addr, prefix, op.At)
		if err == nil && info.Ref.Kind == analyser.RefFunction && strings.EqualFold(mnemonic, "lea") {
			text = "[rel " + text + "]"
		}
		return text, err
	}

	var b strings.Builder
	if info.Size != "" {
		b.WriteString(info.Size)
		b.WriteString(" ")
	}
	b.WriteString("[")
	if info.Rel {
		b.WriteString("rel ")
	}
	for i, term := range info.Terms {
		text, err := t.nativeRef(term.Ref, false, prefix, op.Terms[i].At)
		if err != nil {
			return "", err
		}
		switch {
		case i == 0 && term.Negative:
			b.WriteString("-")
		case i > 0 && term.Negative:
			b.WriteString(" - ")
		case i > 0:
			b.WriteString(" + ")
		}
		if term.Scale != "" {
			b.WriteString(term.Scale)
			b.WriteString("*")
		}
		b.WriteString(text)
	}
	b.WriteString("]")
	return b.String(), nil
}

func (t *X64Target) compileNativeStmt(stmt *ast.NativeStmt) error {
	if !t.nativeActive(stmt) {
		return nil
	}
	info, ok := t.info.Natives[stmt]
	if !ok {
		return nativeError(stmt.BodyAt, "the native block was not analysed")
	}

	prefix := strings.TrimPrefix(t.newLabel("native"), ".")
	for i, instr := range stmt.Instrs {
		if instr.Label != "" {
			t.text.printlnt(fmt.Sprintf(".%s_%s:", prefix, instr.Label))
		}
		if instr.Mnemonic == "" {
			continue
		}

		var line strings.Builder
		for _, p := range instr.Prefixes {
			line.WriteString(p)
			line.WriteString(" ")
		}
		line.WriteString(instr.Mnemonic)

		if strings.EqualFold(instr.Mnemonic, "extern") {
			for _, op := range instr.Operands {
				line.WriteString(" ")
				line.WriteString(op.Path[0])
			}
			t.text.printlnt(line.String() + t.nativeMark(instr.At))
			continue
		}

		for n, op := range instr.Operands {
			text, err := t.nativeOperandText(instr.Mnemonic, op, info.Instrs[i].Operands[n], prefix)
			if err != nil {
				return err
			}
			if n == 0 {
				line.WriteString(" ")
			} else {
				line.WriteString(", ")
			}
			line.WriteString(text)
		}
		t.text.printlnt(line.String() + t.nativeMark(instr.At))
	}
	return nil
}

func (t *X64Target) nativeMark(at ast.Position) string {
	return fmt.Sprintf("%s%d:%d:%d:%d:%s", nativeMarker, at.Line, at.Col, at.EndLine, at.EndCol, t.curFile)
}
