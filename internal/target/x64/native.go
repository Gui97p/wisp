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

func (t *X64Target) nativeConstant(parts []string, addr bool) (string, bool, error) {
	if addr || t.info.Module == nil {
		return "", false, nil
	}
	sym, rest := analyser.NativeConstantRef(t.info.Module, parts)
	if sym == nil || sym.Kind != analyser.CONST || sym.Const == nil {
		return "", false, nil
	}

	if sym.Const.Kind == analyser.ConstString {
		if len(rest) != 1 || (rest[0] != "ptr" && rest[0] != "len") {
			return "", false, fmt.Errorf("the string constant %s has the parts ptr and len: write {%s.ptr} or {%s.len}", sym.Name, sym.Name, sym.Name)
		}
		label, size, err := t.stringData(sym.Const.Str)
		if err != nil {
			return "", false, err
		}
		if rest[0] == "ptr" {
			return label, true, nil
		}
		return fmt.Sprint(size), true, nil
	}

	if len(rest) != 0 {
		return "", false, fmt.Errorf("the constant %s has no part %s", sym.Name, rest[0])
	}
	value, ok := analyser.NativeConstant(sym)
	return value, ok, nil
}

func nativeError(at ast.Position, format string, args ...any) error {
	return &diag.Error{Diagnostic: diag.Diagnostic{
		Message: fmt.Sprintf(format, args...),
		Origin:  "x86-64",
		Line:    at.Line, Col: at.Col, EndLine: at.EndLine, EndCol: at.EndCol,
	}}
}

func nativeRefAt(stmt *ast.NativeStmt, index, length int) ast.Position {
	before := stmt.Code[:index]
	line := stmt.CodeAt.Line + strings.Count(before, "\n")
	col := stmt.CodeAt.Col + 1 + index
	if nl := strings.LastIndex(before, "\n"); nl >= 0 {
		col = index - nl
	}
	return ast.Position{Line: line, Col: col, EndLine: line, EndCol: col + length - 1}
}

func (t *X64Target) compileNativeStmt(stmt *ast.NativeStmt) error {
	if stmt.Backend != "" {
		if !target.MatchesName(t.target, stmt.Backend) {
			return nil
		}
	}

	variables := map[string]nativeVar{}
	for _, binding := range stmt.Bindings {
		sym := t.info.Idents[binding.Var]
		size := sizeOf(sym.Type)
		if sym.Global {
			variables[sym.Name] = nativeVar{mem: t.globalMem(sym, size), tp: sym.Type}
			continue
		}
		offset, ok := t.ctx.Get(sym)
		if !ok {
			line, col := binding.Var.Position()
			endLine, endCol := binding.Var.EndPosition()
			return nativeError(ast.Position{Line: line, Col: col, EndLine: endLine, EndCol: endCol}, "variable %s not declared on this scope", sym.Name)
		}
		variables[sym.Name] = nativeVar{mem: slot(offset, size), tp: sym.Type}
	}

	var b strings.Builder
	code := stmt.Code

	for i := 0; i < len(code); {
		if code[i] != '{' {
			b.WriteByte(code[i])
			i++
			continue
		}

		end := strings.IndexByte(code[i+1:], '}')
		if end == -1 {
			return nativeError(nativeRefAt(stmt, i, 1), "invalid binding definition: missing }")
		}

		end += i + 1
		ref := code[i+1 : end]
		at := nativeRefAt(stmt, i, end-i+1)
		addr := strings.HasPrefix(ref, "&")
		ref = strings.TrimPrefix(ref, "&")
		parts := strings.Split(ref, ".")
		variable, ok := variables[parts[0]]
		if !ok {
			value, isConst, err := t.nativeConstant(parts, addr)
			if err != nil {
				return nativeError(at, "%s", err)
			}
			if isConst {
				b.WriteString(value)
				i = end + 1
				continue
			}
			return nativeError(at, "undeclared binding %s", ref)
		}

		operand, err := t.nativeOperand(ref, parts[1:], addr, variable)
		if err != nil {
			return nativeError(at, "%s", err)
		}

		b.WriteString(operand)
		i = end + 1
	}

	separator := ";"
	if stmt.Raw {
		separator = "\n"
	}
	parts := strings.Split(b.String(), separator)

	lines := parts[:0]
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" || !stmt.Raw {
			lines = append(lines, part)
		}
	}

	t.text.printlnt(strings.Join(lines, "\n\t"))
	return nil
}
