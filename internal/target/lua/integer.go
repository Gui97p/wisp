package lua

import (
	"fmt"
	"math"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

func typedOperand(left, right analyser.Type) analyser.Type {
	switch left.(type) {
	case analyser.UntypedIntType, analyser.UntypedFloatType:
		return right
	}
	return left
}

func isFloatType(tp analyser.Type) bool {
	if nt, ok := tp.(analyser.NamedType); ok {
		tp = nt.Underlying
	}
	p, ok := tp.(analyser.PrimitiveType)
	return ok && (p.Name == "float32" || p.Name == "float64")
}

func isUnsigned64(tp analyser.Type) bool {
	bits, signed, ok := analyser.IntegerRange(tp)
	return ok && bits == 64 && !signed
}

func isSignedInt(tp analyser.Type) bool {
	_, signed, ok := analyser.IntegerRange(tp)
	return ok && signed
}

func isIntType(tp analyser.Type) bool {
	if _, ok := tp.(analyser.UntypedIntType); ok {
		return true
	}
	_, _, ok := analyser.IntegerRange(tp)
	return ok
}

func wrapInt(text string, tp analyser.Type) string {
	bits, signed, ok := analyser.IntegerRange(tp)
	if !ok || bits == 64 {
		return text
	}

	mask := uint64(1)<<bits - 1
	if signed {
		half := uint64(1) << (bits - 1)
		return fmt.Sprintf("((((%s) + %d) & %d) - %d)", text, half, mask, half)
	}
	return fmt.Sprintf("((%s) & %d)", text, mask)
}

func intLiteralText(e *ast.IntLiteral) string {
	switch {
	case e.Value == math.MinInt64:
		return "math.mininteger"
	case e.Value < 0:
		return fmt.Sprintf("(%d)", e.Value)
	}
	return fmt.Sprintf("%d", e.Value)
}

func (t *LuaTarget) compileBinaryExpr(b *strings.Builder, e *ast.BinaryExpr) error {
	if handled, err := t.compileErrorComparison(b, e); handled {
		return err
	}

	left, err := t.compileExprScratch(e.Left)
	if err != nil {
		return err
	}
	right, err := t.compileExprScratch(e.Right)
	if err != nil {
		return err
	}

	result := t.info.Types[e]
	operand := typedOperand(t.info.Types[e.Left], t.info.Types[e.Right])

	var text string
	switch e.Operator {
	case "&&":
		text = fmt.Sprintf("(%s and %s)", left, right)
	case "||":
		text = fmt.Sprintf("(%s or %s)", left, right)
	case "!=":
		text = fmt.Sprintf("(%s ~= %s)", left, right)
	case "^":
		text = fmt.Sprintf("(%s ~ %s)", left, right)
	case "+":
		if pt, ok := t.info.Types[e.Left].(analyser.PrimitiveType); ok && pt.Name == "string" {
			if rt, ok := t.info.Types[e.Right].(analyser.PrimitiveType); ok && rt.Name == "char" {
				right = fmt.Sprintf("string.char(%s)", right)
			}
			text = fmt.Sprintf("(%s .. %s)", left, right)
		} else {
			text = wrapInt(fmt.Sprintf("(%s + %s)", left, right), result)
		}
	case "-", "*":
		text = wrapInt(fmt.Sprintf("(%s %s %s)", left, e.Operator, right), result)
	case "<<":
		text = wrapInt(fmt.Sprintf("(%s << %s)", left, right), result)
	case ">>":
		if isSignedInt(result) {
			text = fmt.Sprintf("__wisp_sar(%s, %s)", left, right)
		} else {
			text = fmt.Sprintf("(%s >> %s)", left, right)
		}
	case "/", "%":
		text = t.divisionText(e.Operator, left, right, result)
	case "<", "<=", ">", ">=":
		text = comparisonText(e.Operator, left, right, operand)
	default:
		text = fmt.Sprintf("(%s %s %s)", left, e.Operator, right)
	}

	b.WriteString(text)
	return nil
}

func (t *LuaTarget) divisionText(op, left, right string, result analyser.Type) string {
	if !isIntType(result) {
		return fmt.Sprintf("(%s %s %s)", left, op, right)
	}

	fn := "__wisp_idiv"
	if isUnsigned64(result) {
		fn = "__wisp_udiv"
	}
	if op == "%" {
		fn = "__wisp_imod"
		if isUnsigned64(result) {
			fn = "__wisp_umod"
		}
		return fmt.Sprintf("%s(%s, %s)", fn, left, right)
	}
	return wrapInt(fmt.Sprintf("%s(%s, %s)", fn, left, right), result)
}

func comparisonText(op, left, right string, operand analyser.Type) string {
	if !isUnsigned64(operand) {
		return fmt.Sprintf("(%s %s %s)", left, op, right)
	}

	switch op {
	case "<":
		return fmt.Sprintf("math.ult(%s, %s)", left, right)
	case "<=":
		return fmt.Sprintf("__wisp_ule(%s, %s)", left, right)
	case ">":
		return fmt.Sprintf("__wisp_ugt(%s, %s)", left, right)
	default:
		return fmt.Sprintf("__wisp_uge(%s, %s)", left, right)
	}
}

func (t *LuaTarget) compileUnaryExpr(b *strings.Builder, e *ast.UnaryExpr) error {
	value, err := t.compileExprScratch(e.Value)
	if err != nil {
		return err
	}

	switch e.Operator {
	case "-":
		b.WriteString(wrapInt(fmt.Sprintf("(-%s)", value), t.info.Types[e]))
	case "!":
		fmt.Fprintf(b, "(not %s)", value)
	case "~":
		b.WriteString(wrapInt(fmt.Sprintf("(~%s)", value), t.info.Types[e]))
	default:
		return fmt.Errorf("lua: unsupported unary operator %s", e.Operator)
	}
	return nil
}

func (t *LuaTarget) compileCastExpr(b *strings.Builder, e *ast.CastExpr) error {
	value, err := t.compileExprScratch(e.Value)
	if err != nil {
		return err
	}

	src := t.info.Types[e.Value]
	dst := t.info.Types[e]

	switch {
	case isCharType(src) && isStringType(dst):
		fmt.Fprintf(b, "string.char(%s)", value)
	case analyser.LosslessWidening(src, dst):
		b.WriteString(value)
	case isFloatType(dst) && isIntType(src):
		if isUnsigned64(src) {
			fmt.Fprintf(b, "__wisp_u2f(%s)", value)
		} else {
			fmt.Fprintf(b, "(%s + 0.0)", value)
		}
	case isIntType(dst) && isFloatType(src):
		b.WriteString(wrapInt(fmt.Sprintf("__wisp_f2i(%s)", value), dst))
	case isIntType(dst) && isIntType(src):
		b.WriteString(wrapInt(value, dst))
	default:
		b.WriteString(value)
	}
	return nil
}

func isCharType(tp analyser.Type) bool {
	p, ok := tp.(analyser.PrimitiveType)
	return ok && p.Name == "char"
}

func isStringType(tp analyser.Type) bool {
	p, ok := tp.(analyser.PrimitiveType)
	return ok && p.Name == "string"
}
