package lua

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

func compileExpression(b *strings.Builder, expr ast.Expression) error {
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		compileExpression(b, e.Left)
		b.WriteString(e.Operator)
		compileExpression(b, e.Right)
	case *ast.UnaryExpr:
		switch e.Operator {
		case "-":
			b.WriteString("-")
			compileExpression(b, e.Value)
		case "!":
			b.WriteString("not ")
			compileExpression(b, e.Value)
		default:
			return fmt.Errorf("lua: unsupported unary operator %s", e.Operator)
		}
	case *ast.CallExpr:
		compileExpression(b, e.Name)
		b.WriteByte('(')
		for k, v := range e.Args {
			compileExpression(b, v)
			if k != len(e.Args)-1 {
				b.WriteByte(',')
			}
		}
		b.WriteByte(')')

	case *ast.IntLiteral:
		fmt.Fprintf(b, "%d", e.Value)
	case *ast.FloatLiteral:
		fmt.Fprintf(b, "%f", e.Value)
	case *ast.ArrayLiteral:
		b.WriteByte('{')
		for k, v := range e.Elements {
			compileExpression(b, v)
			if k != len(e.Elements)-1 {
				b.WriteByte(',')
			}
		}
		b.WriteByte('}')
	case *ast.MapLiteral:
		b.WriteByte('{')
		for k, v := range e.Keys {
			b.WriteByte('[')
			compileExpression(b, v)
			b.WriteString("] = ")
			compileExpression(b, e.Values[k])
			if k != len(e.Keys)-1 {
				b.WriteByte(',')
			}
		}
		b.WriteByte('}')
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
