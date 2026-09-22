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
		case "^":
			b.WriteString("~")
		case "+":
			if pt, ok := t.info.Types[e.Left].(analyser.PrimitiveType); ok && pt.Name == "string" {
				b.WriteString("..")
			} else {
				b.WriteString(e.Operator)
			}
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
			if _, ok := t.info.Types[e.Value].(analyser.ErrorUnionType); ok {
				return t.compilePropagate(b, e)
			}
			b.WriteString("not ")
			t.compileExpression(b, e.Value)
		case "~":
			b.WriteString("~")
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
		if pt, ok := t.info.Types[e.Array].(analyser.PrimitiveType); ok && pt.Name == "string" {
			b.WriteString("string.byte(")
			if err := t.compileExpression(b, e.Array); err != nil {
				return err
			}
			b.WriteString(", (")
			if err := t.compileExpression(b, e.Index); err != nil {
				return err
			}
			b.WriteString(")+1)")
			break
		}
		t.compileExpression(b, e.Array)
		b.WriteByte('[')
		t.compileExpression(b, e.Index)
		b.WriteByte(']')
	case *ast.SliceExpr:
		return t.compileSliceExpression(b, e)
	case *ast.CoalesceExpr:
		return t.compileCoalesceExpr(b, e)

	case *ast.InExpr:
		return t.compileInExpression(b, e)

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
	case *ast.FuncLiteral:
		return t.compileFuncLiteral(b, e)
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
		fmt.Fprintf(b, "%d", e.Value)
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

func (t *LuaTarget) compileFuncLiteral(b *strings.Builder, expr *ast.FuncLiteral) error {
	b.WriteString("function(")
	for i, p := range expr.Params {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(p.Name)
	}
	b.WriteString(")\n")

	prevCount, prevIdx := t.currentReturnCount, t.currentFallibleIndex
	t.currentReturnCount = len(expr.ReturnTypes)
	t.currentFallibleIndex = -1
	for i, r := range expr.ReturnTypes {
		if r.Fallible {
			t.currentFallibleIndex = i
			break
		}
	}
	defer func() {
		t.currentReturnCount = prevCount
		t.currentFallibleIndex = prevIdx
	}()

	if err := t.compileStatement(b, expr.Block); err != nil {
		return err
	}
	b.WriteString("end")
	return nil
}

func methodOwnerName(t analyser.Type) (string, bool) {
	if pt, ok := t.(analyser.PointerType); ok {
		t = pt.Element
	}
	switch tt := t.(type) {
	case *analyser.StructType:
		return tt.Name, true
	case analyser.NamedType:
		return tt.Name, true
	default:
		return "", false
	}
}

func (t *LuaTarget) compileCallExpression(b *strings.Builder, expr *ast.CallExpr) error {
	if ident, ok := expr.Name.(*ast.IdentLiteral); ok {
		if _, ok := t.getBuiltin(ident.Value); ok {
			return t.compileBuiltinCall(b, ident.Value, expr.Args)
		}
	}

	if member, ok := expr.Name.(*ast.MemberExpr); ok {
		objType := t.info.Types[member.Object]
		if methods := analyser.MethodsOf(objType); methods != nil {
			if _, isMethod := methods[member.Field]; isMethod {
				typeName, _ := methodOwnerName(objType)
				fmt.Fprintf(b, "%s_%s(", typeName, member.Field)
				if err := t.compileExpression(b, member.Object); err != nil {
					return err
				}
				for _, arg := range expr.Args {
					b.WriteByte(',')
					if err := t.compileExpression(b, arg); err != nil {
						return err
					}
				}
				b.WriteByte(')')
				return nil
			}
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

func (t *LuaTarget) compileSliceExpression(b *strings.Builder, expr *ast.SliceExpr) error {
	if pt, ok := t.info.Types[expr].(analyser.PrimitiveType); ok && pt.Name == "string" {
		b.WriteString("string.sub(")
		if err := t.compileExpression(b, expr.Array); err != nil {
			return err
		}
		b.WriteString(", ")
		if expr.Start != nil {
			b.WriteByte('(')
			if err := t.compileExpression(b, expr.Start); err != nil {
				return err
			}
			b.WriteString(")+1")
		} else {
			b.WriteByte('1')
		}
		if expr.End != nil {
			b.WriteString(", ")
			if err := t.compileExpression(b, expr.End); err != nil {
				return err
			}
		}
		b.WriteByte(')')
		return nil
	}

	b.WriteString("__wisp_slice(")
	if err := t.compileExpression(b, expr.Array); err != nil {
		return err
	}
	b.WriteString(", ")
	if expr.Start != nil {
		if err := t.compileExpression(b, expr.Start); err != nil {
			return err
		}
	} else {
		b.WriteByte('0')
	}
	b.WriteString(", ")
	if expr.End != nil {
		if err := t.compileExpression(b, expr.End); err != nil {
			return err
		}
	} else {
		b.WriteString("__wisp_len(")
		if err := t.compileExpression(b, expr.Array); err != nil {
			return err
		}
		b.WriteByte(')')
	}
	b.WriteByte(')')
	return nil
}

func (t *LuaTarget) compilePropagate(b *strings.Builder, expr *ast.UnaryExpr) error {
	box, err := t.compileExprScratch(expr.Value)
	if err != nil {
		return err
	}

	tmp := t.newLabel("prop")
	t.emitPending(fmt.Sprintf("local %s = %s\n", tmp, box))

	parts := make([]string, t.currentReturnCount)
	for i := range parts {
		if i == t.currentFallibleIndex {
			parts[i] = fmt.Sprintf("{value = nil, err = %s.err}", tmp)
		} else {
			parts[i] = "nil"
		}
	}
	t.emitPending(fmt.Sprintf("if %s.err ~= nil then return %s end\n", tmp, strings.Join(parts, ", ")))

	b.WriteString(tmp)
	b.WriteString(".value")
	return nil
}

func (t *LuaTarget) compileCoalesceExpr(b *strings.Builder, expr *ast.CoalesceExpr) error {
	left, err := t.compileExprScratch(expr.Left)
	if err != nil {
		return err
	}

	tmp := t.newLabel("coalesce")
	result := t.newLabel("coalesce_r")
	t.emitPending(fmt.Sprintf("local %s = %s\n", tmp, left))
	t.emitPending(fmt.Sprintf("local %s\n", result))

	outer := t.pending
	t.pending = nil

	var branch strings.Builder
	fmt.Fprintf(&branch, "if %s.err ~= nil then\n", tmp)

	if expr.Default != nil {
		def, err := t.compileExprScratch(expr.Default)
		if err != nil {
			t.pending = outer
			return err
		}
		t.flushPending(&branch)
		fmt.Fprintf(&branch, "%s = %s\n", result, def)
	} else {
		block, ok := expr.Block.(*ast.BlockStmt)
		if !ok {
			t.pending = outer
			return fmt.Errorf("lua: coalesce expects a block")
		}

		fmt.Fprintf(&branch, "local %s = %s.err\n", expr.ErrorBind, tmp)

		t.pushCoalesceResult(result)
		err := t.compileStatement(&branch, block)
		t.popCoalesceResult()
		if err != nil {
			t.pending = outer
			return err
		}
	}

	fmt.Fprintf(&branch, "else\n%s = %s.value\nend\n", result, tmp)

	t.pending = outer
	t.emitPending(branch.String())

	b.WriteString(result)
	return nil
}

func (t *LuaTarget) compileInExpression(b *strings.Builder, expr *ast.InExpr) error {
	switch t.info.Types[expr.Right].(type) {
	case *analyser.MapType:
		b.WriteByte('(')
		t.compileExpression(b, expr.Right)
		b.WriteByte('[')
		t.compileExpression(b, expr.Left)
		b.WriteString("] ~= nil)")
	case analyser.PrimitiveType:
		b.WriteString("(string.find(")
		t.compileExpression(b, expr.Right)
		b.WriteString(", ")
		if isChar(t.info.Types[expr.Left]) {
			b.WriteString("string.char(")
			t.compileExpression(b, expr.Left)
			b.WriteByte(')')
		} else {
			t.compileExpression(b, expr.Left)
		}
		b.WriteString(", 1, true) ~= nil)")
	default:
		b.WriteString("__wisp_contains(")
		t.compileExpression(b, expr.Right)
		b.WriteString(", ")
		t.compileExpression(b, expr.Left)
		b.WriteByte(')')
	}
	return nil
}
