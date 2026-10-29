package lua

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

func (t *LuaTarget) collectBoxed() {
	t.boxed = map[*analyser.Symbol]bool{}

	var visit func(v reflect.Value)
	visit = func(v reflect.Value) {
		switch v.Kind() {
		case reflect.Ptr:
			if v.IsNil() {
				return
			}
			switch n := v.Interface().(type) {
			case *ast.UnaryExpr:
				if n.Operator == "&" {
					t.markBoxed(n.Value)
				}
			case *ast.CallExpr:
				if mc, ok := t.info.MethodCalls[n]; ok && mc.Receiver != nil {
					if u, isUnary := mc.Receiver.(*ast.UnaryExpr); isUnary && u.Operator == "&" {
						t.markBoxed(u.Value)
					}
				}
			}
			visit(v.Elem())
		case reflect.Interface:
			if !v.IsNil() {
				visit(v.Elem())
			}
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				visit(v.Field(i))
			}
		case reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				visit(v.Index(i))
			}
		}
	}
	visit(reflect.ValueOf(t.program))
}

func (t *LuaTarget) markBoxed(target ast.Expression) {
	id, ok := target.(*ast.IdentLiteral)
	if !ok {
		return
	}
	sym := t.info.Idents[id]
	if sym == nil || (sym.Kind != analyser.VAR && sym.Kind != analyser.PARAM) || sym.Type == nil || isAggregate(sym.Type) {
		return
	}
	t.boxed[sym] = true
}

func (t *LuaTarget) isBoxed(sym *analyser.Symbol) bool {
	return sym != nil && t.boxed[sym]
}

func (t *LuaTarget) compileAddressOf(b *strings.Builder, e *ast.UnaryExpr) error {
	switch target := e.Value.(type) {
	case *ast.IdentLiteral:
		sym := t.info.Idents[target]
		text, err := t.compileExprScratch(target)
		if err != nil {
			return err
		}
		if t.isBoxed(sym) {
			fmt.Fprintf(b, "{t=%s,o=0}", strings.TrimSuffix(text, "[0]"))
			return nil
		}
		if isAggregate(t.info.Types[target]) {
			fmt.Fprintf(b, "{t={%s},o=1}", text)
			return nil
		}
		return fmt.Errorf("lua: cannot take the address of %s", target.Value)
	case *ast.MemberExpr:
		object, err := t.memberObject(target)
		if err != nil {
			return err
		}
		fmt.Fprintf(b, "{t=%s,o=%q}", object, target.Field)
		return nil
	case *ast.IndexExpr:
		array, err := t.compileExprScratch(target.Array)
		if err != nil {
			return err
		}
		index, err := t.compileExprScratch(target.Index)
		if err != nil {
			return err
		}
		if isPointer(t.info.Types[target.Array]) {
			fmt.Fprintf(b, "__wisp_padd(%s, %s)", array, index)
			return nil
		}
		fmt.Fprintf(b, "__wisp_vref(%s, %s)", array, index)
		return nil
	case *ast.DerefExpr:
		return t.compileExpression(b, target.Value)
	}
	return fmt.Errorf("lua: cannot take the address of this expression")
}

func (t *LuaTarget) memberObject(e *ast.MemberExpr) (string, error) {
	object, err := t.compileExprScratch(e.Object)
	if err != nil {
		return "", err
	}
	tp := t.info.Types[e.Object]
	switch {
	case isPointer(tp):
		return fmt.Sprintf("__wisp_get(%s)", object), nil
	case isErrorType(tp):
		return fmt.Sprintf("__wisp_nn(%s)", object), nil
	}
	return object, nil
}

func isErrorType(tp analyser.Type) bool {
	_, ok := underlying(tp).(analyser.ErrorType)
	return ok
}

func (t *LuaTarget) compilePointerBinary(b *strings.Builder, e *ast.BinaryExpr) (bool, error) {
	lt, rt := t.info.Types[e.Left], t.info.Types[e.Right]
	if !isPointer(lt) && !isPointer(rt) {
		return false, nil
	}

	left, err := t.compileExprScratch(e.Left)
	if err != nil {
		return true, err
	}
	right, err := t.compileExprScratch(e.Right)
	if err != nil {
		return true, err
	}

	_, leftNull := e.Left.(*ast.NullLiteral)
	_, rightNull := e.Right.(*ast.NullLiteral)

	switch e.Operator {
	case "==", "!=":
		var text string
		if leftNull || rightNull {
			text = fmt.Sprintf("(%s == %s)", left, right)
		} else {
			text = fmt.Sprintf("__wisp_peq(%s, %s)", left, right)
		}
		if e.Operator == "!=" {
			text = fmt.Sprintf("(not %s)", text)
		}
		b.WriteString(text)
	case "<":
		fmt.Fprintf(b, "__wisp_plt(%s, %s)", left, right)
	case "<=":
		fmt.Fprintf(b, "__wisp_ple(%s, %s)", left, right)
	case ">":
		fmt.Fprintf(b, "__wisp_plt(%s, %s)", right, left)
	case ">=":
		fmt.Fprintf(b, "__wisp_ple(%s, %s)", right, left)
	case "+":
		if isPointer(lt) {
			fmt.Fprintf(b, "__wisp_padd(%s, %s)", left, right)
		} else {
			fmt.Fprintf(b, "__wisp_padd(%s, %s)", right, left)
		}
	case "-":
		if isPointer(lt) && isPointer(rt) {
			fmt.Fprintf(b, "__wisp_pdiff(%s, %s)", left, right)
		} else {
			fmt.Fprintf(b, "__wisp_padd(%s, -(%s))", left, right)
		}
	default:
		return true, fmt.Errorf("lua: operator %s on a pointer is not supported", e.Operator)
	}
	return true, nil
}

func (t *LuaTarget) compilePointerCast(b *strings.Builder, value string, src, dst analyser.Type) (bool, error) {
	switch {
	case isPointer(dst) && isPointer(src):
		if _, void := pointee(src).(analyser.VoidType); void {
			if _, toVoid := pointee(dst).(analyser.VoidType); !toVoid {
				zero, err := zeroValue(pointee(dst))
				if err != nil {
					return true, err
				}
				fmt.Fprintf(b, "__wisp_cast(%s, function() return %s end, %d)", value, zero, max(sizeOf(pointee(dst)), 1))
				return true, nil
			}
		}
		if !sameLayout(pointee(src), pointee(dst)) {
			return true, fmt.Errorf("lua: reading memory as another type needs raw memory, which the Lua target does not have")
		}
		b.WriteString(value)
		return true, nil
	case isPointer(dst) && isView(src):
		fmt.Fprintf(b, "__wisp_vptr(%s)", value)
		return true, nil
	case isPointer(dst) && isStringType(underlying(src)):
		fmt.Fprintf(b, "__wisp_bytes(%s)", value)
		return true, nil
	case isIntType(dst) && isPointer(src):
		fmt.Fprintf(b, "__wisp_addr(%s, %d)", value, max(sizeOf(pointee(src)), 1))
		return true, nil
	case isPointer(dst) && isIntType(src):
		fmt.Fprintf(b, "__wisp_fromaddr(%s, %d)", value, max(sizeOf(pointee(dst)), 1))
		return true, nil
	}
	return false, nil
}

func sameLayout(a, b analyser.Type) bool {
	if _, ok := a.(analyser.VoidType); ok {
		return true
	}
	if _, ok := b.(analyser.VoidType); ok {
		return true
	}
	if a.Equals(b) {
		return true
	}
	return isIntType(a) && isIntType(b) && sizeOf(a) == sizeOf(b)
}
