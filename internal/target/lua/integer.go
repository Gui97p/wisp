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
	return ok && (p.Name == "float" || p.Name == "float32" || p.Name == "float64")
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

func intBits(tp analyser.Type) (int, bool) {
	bits, signed, ok := analyser.IntegerRange(tp)
	if !ok {
		return 64, true
	}
	return bits, signed
}

func wrapInt(text string, tp analyser.Type) string {
	bits, signed, ok := analyser.IntegerRange(tp)
	if !ok || bits == 64 {
		return text
	}

	modulus := uint64(1) << bits
	if signed {
		half := modulus >> 1
		return fmt.Sprintf("((((%s) + %d) %% %d) - %d)", text, half, modulus, half)
	}
	return fmt.Sprintf("((%s) %% %d)", text, modulus)
}

func integerOp(op, left, right string, result analyser.Type) (string, bool) {
	if !isIntType(result) {
		return "", false
	}

	bits, signed := intBits(result)
	switch op {
	case "&", "|", "^":
		fn := map[string]string{"&": "__wisp_band", "|": "__wisp_bor", "^": "__wisp_bxor"}[op]
		return wrapInt(fmt.Sprintf("%s(%s, %s, %t)", fn, left, right, bits < 64 || signed), result), true
	case "<<":
		return wrapInt(fmt.Sprintf("__wisp_shl(%s, %s, %d)", left, right, bits), result), true
	case ">>":
		if signed {
			return fmt.Sprintf("__wisp_sar(%s, %s)", left, right), true
		}
		return fmt.Sprintf("__wisp_shr(%s, %s)", left, right), true
	case "*":
		if bits == 32 {
			return wrapInt(fmt.Sprintf("__wisp_mulw(%s, %s)", left, right), result), true
		}
	}
	return "", false
}

func complementText(value string, tp analyser.Type) string {
	bits, signed := intBits(tp)
	if signed {
		return fmt.Sprintf("(-(%s) - 1)", value)
	}
	max := uint64(math.MaxUint64)
	if bits < 64 {
		max = uint64(1)<<bits - 1
	}
	return fmt.Sprintf("(%d - (%s))", max, value)
}

func intLiteralText(e *ast.IntLiteral) string {
	switch {
	case e.Unsigned && e.Value < 0:
		return fmt.Sprintf("%d", uint64(e.Value))
	case e.Value == math.MinInt64:
		return "(-9223372036854775808)"
	case e.Value < 0:
		return fmt.Sprintf("(%d)", e.Value)
	}
	return fmt.Sprintf("%d", e.Value)
}

func (t *LuaTarget) compileBinaryExpr(b *strings.Builder, e *ast.BinaryExpr) error {
	if tc, ok := t.info.TypeCases[e]; ok {
		return t.compileTypeCase(b, e, tc)
	}

	if handled, err := t.compileErrorComparison(b, e); handled {
		return err
	}

	if handled, err := t.compilePointerBinary(b, e); handled {
		return err
	}

	if e.Operator == "&&" || e.Operator == "||" {
		return t.compileLogical(b, e)
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

	if text, ok := integerOp(e.Operator, left, right, result); ok {
		b.WriteString(text)
		return nil
	}

	var text string
	switch e.Operator {
	case "!=":
		text = fmt.Sprintf("(%s ~= %s)", left, right)
	case "+":
		if pt, ok := t.info.Types[e.Left].(analyser.PrimitiveType); ok && pt.Name == "string" {
			if rt, ok := t.info.Types[e.Right].(analyser.PrimitiveType); ok && rt.Name == "char" {
				right = fmt.Sprintf("__lua_string.char(%s)", right)
			}
			text = fmt.Sprintf("(%s .. %s)", left, right)
		} else {
			text = wrapInt(fmt.Sprintf("(%s + %s)", left, right), result)
		}
	case "-", "*":
		text = wrapInt(fmt.Sprintf("(%s %s %s)", left, e.Operator, right), result)
	case "/", "%":
		text = t.divisionText(e.Operator, left, right, result)
	case "<", "<=", ">", ">=":
		text = fmt.Sprintf("(%s %s %s)", left, e.Operator, right)
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

	if op == "%" {
		return fmt.Sprintf("__wisp_imod(%s, %s)", left, right)
	}
	return wrapInt(fmt.Sprintf("__wisp_idiv(%s, %s)", left, right), result)
}

func (t *LuaTarget) compileUnaryExpr(b *strings.Builder, e *ast.UnaryExpr) error {
	if e.Operator == "&" {
		return t.compileAddressOf(b, e)
	}

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
		b.WriteString(complementText(value, t.info.Types[e]))
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

	if handled, err := t.compilePointerCast(b, value, src, dst); handled {
		return err
	}

	switch {
	case isCharType(src) && isStringType(dst):
		fmt.Fprintf(b, "string.char(%s)", value)
	case isView(src) && isStringType(dst):
		fmt.Fprintf(b, "__wisp_str(%s)", value)
	case analyser.LosslessWidening(src, dst):
		b.WriteString(value)
	case isFloatType(dst) && isIntType(src):
		fmt.Fprintf(b, "(%s + 0.0)", value)
	case isIntType(dst) && isFloatType(src):
		b.WriteString(wrapInt(fmt.Sprintf("__wisp_f2i(%s)", value), dst))
	case isIntType(dst) && isIntType(src):
		b.WriteString(integerCast(value, src, dst))
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

func (t *LuaTarget) compileIsolated(expr ast.Expression) (text, stmts string, err error) {
	outer := t.pending
	t.pending = nil
	text, err = t.compileExprScratch(expr)
	stmts = strings.Join(t.pending, "")
	t.pending = outer
	return text, stmts, err
}

func (t *LuaTarget) compileLogical(b *strings.Builder, e *ast.BinaryExpr) error {
	left, err := t.compileExprScratch(e.Left)
	if err != nil {
		return err
	}
	right, stmts, err := t.compileIsolated(e.Right)
	if err != nil {
		return err
	}

	op, guard := "and", "%s"
	if e.Operator == "||" {
		op, guard = "or", "not %s"
	}

	if stmts == "" {
		fmt.Fprintf(b, "(%s %s %s)", left, op, right)
		return nil
	}

	result := t.newLabel("sc")
	t.emitPending(fmt.Sprintf("local %s = %s\n", result, left))
	t.emitPending(fmt.Sprintf("if %s then\n%s%s = %s\nend\n", fmt.Sprintf(guard, result), stmts, result, right))
	b.WriteString(result)
	return nil
}

func (t *LuaTarget) compileTernary(b *strings.Builder, e *ast.TernaryExpr) error {
	cond, err := t.compileExprScratch(e.Condition)
	if err != nil {
		return err
	}
	thenText, thenStmts, err := t.compileIsolated(e.Then)
	if err != nil {
		return err
	}
	elseText, elseStmts, err := t.compileIsolated(e.Else)
	if err != nil {
		return err
	}

	if thenStmts == "" && elseStmts == "" {
		if canBeFalsy(t.info.Types[e.Then]) {
			fmt.Fprintf(b, "(function() if %s then return %s else return %s end end)()", cond, thenText, elseText)
			return nil
		}
		fmt.Fprintf(b, "((%s) and (%s) or (%s))", cond, thenText, elseText)
		return nil
	}

	result := t.newLabel("tern")
	t.emitPending(fmt.Sprintf("local %s\nif %s then\n%s%s = %s\nelse\n%s%s = %s\nend\n", result, cond, thenStmts, result, thenText, elseStmts, result, elseText))
	b.WriteString(result)
	return nil
}

func integerCast(value string, src, dst analyser.Type) string {
	dstBits, dstSigned := intBits(dst)
	srcBits, srcSigned := intBits(src)
	if dstBits == 64 && !dstSigned && srcSigned {
		return fmt.Sprintf("__wisp_tou64(%s)", value)
	}
	if dstBits == 64 && dstSigned && !srcSigned && srcBits == 64 {
		return fmt.Sprintf("__wisp_toi64(%s)", value)
	}
	return wrapInt(value, dst)
}
