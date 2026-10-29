package lua

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

func (t *LuaTarget) compileArgument(b *strings.Builder, expr *ast.CallExpr, index int) error {
	arg := expr.Args[index]

	if expr.Spread && index == len(expr.Args)-1 {
		b.WriteString("__wisp_spread(")
		if err := t.compileExpression(b, arg); err != nil {
			return err
		}
		b.WriteByte(')')
		return nil
	}

	boxed, ok := t.info.Boxes[arg]
	if !ok {
		text, err := t.compileExprScratch(arg)
		if err != nil {
			return err
		}
		b.WriteString(t.valueText(arg, text))
		return nil
	}

	layout := analyser.Layout(boxed)

	tag, enum := layout.Code, "nil"
	if layout.Named() {
		tag = layout.Identity
	} else if layout.Identity != "" {
		enum = fmt.Sprintf("%q", layout.Identity)
	}

	fmt.Fprintf(b, "__wisp_box(%q, ", tag)
	if err := t.compileExpression(b, arg); err != nil {
		return err
	}
	fmt.Fprintf(b, ", %s)", enum)
	return nil
}

func (t *LuaTarget) compileTypeCase(b *strings.Builder, e *ast.BinaryExpr, tc *analyser.TypeCase) error {
	subject, err := t.compileExprScratch(e.Left)
	if err != nil {
		return err
	}

	layout := analyser.Layout(tc.Type)
	switch {
	case tc.Group:
		name := map[string]string{"int": "__wisp_isint", "uint": "__wisp_isuint", "float": "__wisp_isfloat"}[layout.Code]
		if layout.Code == "float64" {
			name = "__wisp_isfloat"
		}
		fmt.Fprintf(b, "%s(%s)", name, subject)
	case layout.Named():
		fmt.Fprintf(b, "(%s.t == %q)", subject, layout.Identity)
	case layout.Identity != "":
		fmt.Fprintf(b, "(%s.e == %q)", subject, layout.Identity)
	default:
		fmt.Fprintf(b, "(%s.t == %q)", subject, layout.Code)
	}
	return nil
}

func (t *LuaTarget) narrowedIdent(ident *ast.IdentLiteral) bool {
	sym := t.info.Idents[ident]
	if sym == nil {
		return false
	}
	_, narrowed := t.info.Narrowed[sym]
	return narrowed
}
