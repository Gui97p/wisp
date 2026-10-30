package lua

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

func (t *LuaTarget) compileExpression(b *strings.Builder, expr ast.Expression) error {
	return t.locate(expr, t.compileExpressionNode(b, expr))
}

func (t *LuaTarget) compileExpressionNode(b *strings.Builder, expr ast.Expression) error {
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		return t.compileBinaryExpr(b, e)
	case *ast.UnaryExpr:
		return t.compileUnaryExpr(b, e)
	case *ast.DerefExpr:
		value, err := t.compileExprScratch(e.Value)
		if err != nil {
			return err
		}
		fmt.Fprintf(b, "__wisp_get(%s)", value)
	case *ast.CallExpr:
		return t.compileCallExpression(b, e)
	case *ast.MemberExpr:
		if value, ok := t.info.EnumValues[e]; ok {
			fmt.Fprintf(b, "%d", value.Value)
			return nil
		}
		object, err := t.memberObject(e)
		if err != nil {
			return err
		}
		b.WriteString(object)
		b.WriteString(fieldAccess(e.Field))
	case *ast.DotIdent:
		value, ok := t.info.EnumValues[e]
		if !ok {
			return fmt.Errorf("lua: unresolved enum member .%s", e.Name)
		}
		fmt.Fprintf(b, "%d", value.Value)
	case *ast.IndexExpr:
		return t.compileIndex(b, e)
	case *ast.SliceExpr:
		return t.compileSliceExpression(b, e)
	case *ast.CoalesceExpr:
		return t.compileCoalesceExpr(b, e)
	case *ast.SwitchExpr:
		return t.compileSwitchExpression(b, e)
	case *ast.PropagateExpr:
		return t.compilePropagate(b, e)

	case *ast.InExpr:
		return t.compileInExpression(b, e)

	case *ast.TernaryExpr:
		return t.compileTernary(b, e)
	case *ast.CastExpr:
		return t.compileCastExpr(b, e)
	case *ast.ConvExpr:
		return t.compileCastExpr(b, e.Cast)

	case *ast.IntLiteral:
		b.WriteString(intLiteralText(e))
	case *ast.FloatLiteral:
		b.WriteString(floatLiteralText(e.Value))
	case *ast.FuncLiteral:
		return t.compileFuncLiteral(b, e)
	case *ast.ArrayLiteral:
		return t.compileArrayLiteral(b, e)
	case *ast.StructLiteral:
		return t.compileStructLiteral(b, e)
	case *ast.IdentLiteral:
		b.WriteString(luaIdent(e.Value))
		if t.isBoxed(t.info.Idents[e]) {
			b.WriteString("[0]")
		}
		if t.narrowedIdent(e) {
			b.WriteString(".v")
		}
	case *ast.StringLiteral:
		b.WriteString(luaString(e.Value))
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
		return fmt.Errorf("%s is not supported yet", ast.Describe(expr))
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
		b.WriteString(fieldKey(field))

		if v, ok := given[field]; ok {
			text, err := t.compileExprScratch(v)
			if err != nil {
				return err
			}
			b.WriteString(t.valueText(v, text))
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
		b.WriteString(luaIdent(p.Name))
	}
	b.WriteString(")\n")

	prevFallible, prevPayloads, prevDefers := t.currentFallible, t.currentPayloads, t.deferScopes
	t.currentFallible = expr.Fallible
	t.currentPayloads = nil
	t.deferScopes = nil
	if ft, ok := t.info.Types[expr].(*analyser.FuncType); ok {
		t.currentPayloads = ft.Returns
	}
	defer func() {
		t.currentFallible, t.currentPayloads, t.deferScopes = prevFallible, prevPayloads, prevDefers
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
	if lit, ok := t.info.ErrorLiterals[expr]; ok {
		b.WriteString(errorTable(lit))
		return nil
	}

	if expr.Cast != nil {
		return t.compileCastExpr(b, expr.Cast)
	}

	if ident, ok := expr.Name.(*ast.IdentLiteral); ok {
		if tp, isSizeof := t.info.SizeOf[expr]; isSizeof {
			fmt.Fprintf(b, "%d", sizeOf(tp))
			return nil
		}
		if _, ok := t.getBuiltin(ident.Value); ok {
			return t.compileBuiltinCall(b, ident.Value, expr.Args)
		}
	}

	if member, ok := expr.Name.(*ast.MemberExpr); ok {
		objType := t.info.Types[member.Object]
		if methods := analyser.MethodsOf(objType); methods != nil {
			if _, isMethod := methods[member.Field]; isMethod {
				typeName, _ := methodOwnerName(objType)
				prefix := ""
				if mc, ok := t.info.MethodCalls[expr]; ok && mc.Module != "" && ModuleID(mc.Module) != t.id {
					if alias, imported := t.importAlias(mc.Module); imported {
						prefix = luaIdent(alias) + "."
					} else {
						prefix = t.extraModule(mc.Module) + "."
					}
				}
				fmt.Fprintf(b, "%s%s_%s(", prefix, typeName, member.Field)
				receiver := member.Object
				byValue := true
				if mc, ok := t.info.MethodCalls[expr]; ok && mc.Receiver != nil {
					receiver = mc.Receiver
					byValue = !mc.PointerReceiver
				}
				receiverText, err := t.compileExprScratch(receiver)
				if err != nil {
					return err
				}
				if byValue {
					receiverText = t.valueText(receiver, receiverText)
				}
				b.WriteString(receiverText)
				for k := range expr.Args {
					b.WriteByte(',')
					if err := t.compileArgument(b, expr, k); err != nil {
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
	for k := range expr.Args {
		if err := t.compileArgument(b, expr, k); err != nil {
			return err
		}
		if k != len(expr.Args)-1 {
			b.WriteByte(',')
		}
	}
	b.WriteByte(')')
	return nil
}

func (t *LuaTarget) compileSliceExpression(b *strings.Builder, expr *ast.SliceExpr) error {
	if pt, ok := t.info.Types[expr].(analyser.PrimitiveType); ok && pt.Name == "string" {
		b.WriteString("__lua_string.sub(")
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

	array, err := t.compileExprScratch(expr.Array)
	if err != nil {
		return err
	}
	start, end := "nil", "nil"
	if expr.Start != nil {
		if start, err = t.compileExprScratch(expr.Start); err != nil {
			return err
		}
	}
	if expr.End != nil {
		if end, err = t.compileExprScratch(expr.End); err != nil {
			return err
		}
	}

	if isPointer(t.info.Types[expr.Array]) {
		if expr.Start == nil {
			start = "0"
		}
		fmt.Fprintf(b, "__wisp_pslice(%s, %s, %s)", array, start, end)
		return nil
	}
	fmt.Fprintf(b, "__wisp_vslice(%s, %s, %s)", array, start, end)
	return nil
}

func (t *LuaTarget) compileInExpression(b *strings.Builder, expr *ast.InExpr) error {
	switch t.info.Types[expr.Right].(type) {
	case analyser.PrimitiveType:
		b.WriteString("(__lua_string.find(")
		t.compileExpression(b, expr.Right)
		b.WriteString(", ")
		if isChar(t.info.Types[expr.Left]) {
			b.WriteString("__lua_string.char(")
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

func canBeFalsy(tp analyser.Type) bool {
	switch tt := tp.(type) {
	case analyser.PrimitiveType:
		return tt.Name == "bool"
	case analyser.PointerType, analyser.ErrorType, analyser.NullType:
		return true
	}
	return false
}

func (t *LuaTarget) compileArrayLiteral(b *strings.Builder, lit *ast.ArrayLiteral) error {
	var want *analyser.ArrayType
	if at, ok := t.info.Types[lit].(analyser.ArrayType); ok {
		want = &at
	}
	text, err := t.arrayLiteralText(lit, want)
	if err != nil {
		return err
	}
	b.WriteString(text)
	return nil
}

func (t *LuaTarget) arrayLiteralText(lit *ast.ArrayLiteral, want *analyser.ArrayType) (string, error) {
	size := int64(len(lit.Elements))
	var zero string
	if want != nil && want.Size > size {
		size = want.Size
		var err error
		zero, err = zeroValue(want.Element)
		if err != nil {
			return "", err
		}
	}

	var b strings.Builder
	b.WriteString("{t={")
	for k, v := range lit.Elements {
		if k > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "[%d]=", k)
		text, err := t.compileExprScratch(v)
		if err != nil {
			return "", err
		}
		b.WriteString(t.valueText(v, text))
	}
	for k := int64(len(lit.Elements)); k < size; k++ {
		if k > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "[%d]=%s", k, zero)
	}
	fmt.Fprintf(&b, "},o=0,n=%d}", size)
	return b.String(), nil
}

func (t *LuaTarget) compileIndex(b *strings.Builder, e *ast.IndexExpr) error {
	array, err := t.compileExprScratch(e.Array)
	if err != nil {
		return err
	}
	index, err := t.compileExprScratch(e.Index)
	if err != nil {
		return err
	}

	tp := t.info.Types[e.Array]
	switch {
	case isStringType(underlying(tp)):
		fmt.Fprintf(b, "__wisp_sget(%s, %s)", array, index)
	case isView(tp):
		fmt.Fprintf(b, "__wisp_vget(%s, %s)", array, index)
	case isPointer(tp):
		fmt.Fprintf(b, "__wisp_pget(%s, %s)", array, index)
	default:
		fmt.Fprintf(b, "%s[%s]", array, index)
	}
	return nil
}

func floatLiteralText(value float64) string {
	text := strconv.FormatFloat(value, 'g', -1, 64)
	if !strings.ContainsAny(text, ".e") {
		text += ".0"
	}
	if value < 0 {
		return "(" + text + ")"
	}
	return text
}
