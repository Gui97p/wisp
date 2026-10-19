package x64

import (
	"fmt"
	"strings"

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
				return "", fmt.Errorf("x86-64: struct %s has no field %s in {%s}", st.Name, part, ref)
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
			return "", fmt.Errorf("x86-64: cannot access field %s in {%s}", part, ref)
		}
	}

	text := t.memText(mem)
	if addr {
		return text[strings.Index(text, "["):], nil
	}
	if mem.Size > 8 {
		return "", fmt.Errorf("x86-64: {%s} is too big for one operand: use {&%s} for its address", ref, ref)
	}
	return text, nil
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
			return fmt.Errorf("x86-64: variable %s not declared on this scope", sym.Name)
		}
		variables[sym.Name] = nativeVar{mem: slot(offset, size), tp: sym.Type}
	}

	var b strings.Builder
	separator := ";"
	if stmt.Raw {
		separator = "\n"
	}
	parts := strings.Split(stmt.Code, separator)

	lines := parts[:0]
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" || !stmt.Raw {
			lines = append(lines, part)
		}
	}

	s := strings.Join(lines, "\n\t")

	for i := 0; i < len(s); {
		if s[i] != '{' {
			b.WriteByte(s[i])
			i++
			continue
		}

		end := strings.IndexByte(s[i+1:], '}')
		if end == -1 {
			return fmt.Errorf("x86-64: invalid binding definition")
		}

		end += i + 1
		ref := s[i+1 : end]
		addr := strings.HasPrefix(ref, "&")
		ref = strings.TrimPrefix(ref, "&")
		parts := strings.Split(ref, ".")
		nativeVar, ok := variables[parts[0]]
		if !ok {
			return fmt.Errorf("x86-64: undeclared binding %s", ref)
		}

		operand, err := t.nativeOperand(ref, parts[1:], addr, nativeVar)
		if err != nil {
			return err
		}

		b.WriteString(operand)
		i = end + 1
	}

	t.text.printlnt(b.String())
	return nil
}
