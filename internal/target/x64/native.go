package x64

import (
	"fmt"
	"github.com/Gui97p/wisp/internal/target"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

type nativeVar struct {
	offset int
	tp     analyser.Type
}

func (t *X64Target) nativeOperand(name, field string, hasField bool, v nativeVar) (string, error) {
	if hasField {
		switch nv := v.tp.(type) {
		case analyser.PrimitiveType:
			if nv.Name != "string" {
				return "", fmt.Errorf("x86-64: cannot access field %s in var %s with type %s", field, name, nv.Name)
			}
		case analyser.SpanType:
		default:
			return "", fmt.Errorf("x86-64: cannot access field %s in var %s", field, name)
		}

		switch field {
		case "ptr":
			return t.mem(v.offset, 8), nil
		case "len":
			return t.mem(v.offset-8, 8), nil
		default:
			return "", fmt.Errorf("x86-64: cannot access field %s in var %s", field, name)
		}
	} else {
		size := t.sizeOf(v.tp)
		if size > 8 {
			return "", fmt.Errorf("x86-64: variable too big. Use {%s.ptr} or {%s.len}", name, name)
		}

		return t.mem(v.offset, size), nil
	}
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
		offset, ok := t.ctx.Get(sym)
		if !ok {
			return fmt.Errorf("x86-64: variable %s not declared on this scope", sym.Name)
		}
		variables[sym.Name] = nativeVar{offset: offset, tp: sym.Type}
	}

	var b strings.Builder
	parts := strings.Split(stmt.Code, ";")

	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	s := strings.Join(parts, "\n\t")

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
		ident := s[i+1 : end]
		name, field, hasField := strings.Cut(ident, ".")
		nativeVar, ok := variables[name]
		if !ok {
			return fmt.Errorf("x86-64: undeclared binding %s", ident)
		}

		inst, err := t.nativeOperand(name, field, hasField, nativeVar)
		if err != nil {
			return err
		}

		b.WriteString(inst)
		i = end + 1
	}

	t.text.printlnt(b.String())
	return nil
}
