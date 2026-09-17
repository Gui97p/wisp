package lua

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

func (t *LuaTarget) compileExpression(b *strings.Builder, expr ast.Expression) error {
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		t.compileExpression(b, e.Left)
		switch e.Operator {
		case "&&":
			b.WriteString(" and ")
		case "||":
			b.WriteString(" or ")
		case "!=":
			b.WriteString("~=")
		default:
			b.WriteString(e.Operator)
		}
		t.compileExpression(b, e.Right)
	case *ast.UnaryExpr:
		switch e.Operator {
		case "-":
			b.WriteString("-")
			t.compileExpression(b, e.Value)
		case "!":
			b.WriteString("not ")
			t.compileExpression(b, e.Value)
		default:
			return fmt.Errorf("lua: unsupported unary operator %s", e.Operator)
		}
	case *ast.CallExpr:
		return t.compileCallExpression(b, e)
	case *ast.MemberExpr:
		t.compileExpression(b, e.Object)
		fmt.Fprintf(b, ".%s", e.Field)
	case *ast.IndexExpr:
		t.compileExpression(b, e.Array)
		b.WriteByte('[')
		t.compileExpression(b, e.Index)
		b.WriteByte(']')
	case *ast.TernaryExpr:
		b.WriteByte('(')
		t.compileExpression(b, e.Condition)
		b.WriteString(") and (")
		t.compileExpression(b, e.Then)
		b.WriteString(") or (")
		t.compileExpression(b, e.Else)
		b.WriteRune(')')
	case *ast.CastExpr:
		t.compileExpression(b, e.Value)

	case *ast.IntLiteral:
		fmt.Fprintf(b, "%d", e.Value)
	case *ast.FloatLiteral:
		fmt.Fprintf(b, "%f", e.Value)
	case *ast.ArrayLiteral:
		b.WriteByte('{')
		for k, v := range e.Elements {
			if k > 0 {
				b.WriteByte(',')
			}
			fmt.Fprintf(b, "[%d]=", k)
			t.compileExpression(b, v)
		}
		b.WriteByte('}')
	case *ast.MapLiteral:
		b.WriteByte('{')
		for k, v := range e.Keys {
			b.WriteByte('[')
			t.compileExpression(b, v)
			b.WriteString("] = ")
			t.compileExpression(b, e.Values[k])
			if k != len(e.Keys)-1 {
				b.WriteByte(',')
			}
		}
		b.WriteByte('}')
	case *ast.StructLiteral:
		return t.compileStructLiteral(b, e)
	case *ast.IdentLiteral:
		b.WriteString(e.Value)
	case *ast.StringLiteral:
		fmt.Fprintf(b, "\"%s\"", e.Value)
	case *ast.CharLiteral:
		fmt.Fprintf(b, "'%c'", e.Value)
	case *ast.BoolLiteral:
		if e.Value {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case *ast.NullLiteral:
		b.WriteString("nil")
	default:
		return fmt.Errorf("lua: unsupported expression %T", expr)
	}
	return nil
}

func (t *LuaTarget) compileStructLiteral(b *strings.Builder, expr *ast.StructLiteral) error {
	st, ok := t.info.Types[expr].(*analyser.StructType)
	if !ok {
		return fmt.Errorf("lua: struct literal %s missing resolved type", expr.Name)
	}

	given := make(map[string]ast.Expression, len(expr.Keys))
	for i, k := range expr.Keys {
		given[k] = expr.Values[i]
	}

	b.WriteByte('{')
	for i, field := range st.Order {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(b, "%s=", field)

		if v, ok := given[field]; ok {
			if err := t.compileExpression(b, v); err != nil {
				return err
			}
			continue
		}

		zv, err := zeroValue(st.Fields[field])
		if err != nil {
			return err
		}
		b.WriteString(zv)
	}
	b.WriteByte('}')

	return nil
}

func (t *LuaTarget) compileCallExpression(b *strings.Builder, expr *ast.CallExpr) error {
	if ident, ok := expr.Name.(*ast.IdentLiteral); ok {
		if _, ok := t.getBuiltin(ident.Value); ok {
			return t.compileBuiltinCall(b, ident.Value, expr.Args)
		}
	}

	t.compileExpression(b, expr.Name)
	b.WriteByte('(')
	for k, v := range expr.Args {
		t.compileExpression(b, v)
		if k != len(expr.Args)-1 {
			b.WriteByte(',')
		}
	}
	b.WriteByte(')')
	return nil
}
