package lua

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

func luaString(value string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, c := range value {
		switch c {
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		default:
			b.WriteRune(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func errorTable(lit *analyser.ErrorLiteral) string {
	return fmt.Sprintf("{domain=%s, code=%d, message=%s, file=%s, line=%d}",
		luaString(lit.Domain), lit.Code, luaString(lit.Message), luaString(lit.File), lit.Line)
}

func (t *LuaTarget) payloadZeros() ([]string, error) {
	zeros := make([]string, 0, len(t.currentPayloads)+1)
	for _, tp := range t.currentPayloads {
		zero, err := zeroValue(tp)
		if err != nil {
			return nil, err
		}
		zeros = append(zeros, zero)
	}
	return zeros, nil
}

func (t *LuaTarget) compilePropagate(b *strings.Builder, expr *ast.PropagateExpr) error {
	call, err := t.compileExprScratch(expr.Value)
	if err != nil {
		return err
	}

	fallible, ok := t.info.Types[expr.Value].(analyser.FallibleType)
	if !ok {
		return fmt.Errorf("lua: ! applied to a call that cannot fail")
	}

	names := make([]string, len(fallible.Values))
	for i := range names {
		names[i] = t.newLabel("p")
	}
	errName := t.newLabel("perr")

	targets := append(append([]string{}, names...), errName)
	t.emitPending(fmt.Sprintf("local %s = %s\n", strings.Join(targets, ", "), call))

	zeros, err := t.payloadZeros()
	if err != nil {
		return err
	}
	zeros = append(zeros, errName)
	t.emitPending(fmt.Sprintf("if %s ~= nil then return %s end\n", errName, strings.Join(zeros, ", ")))

	b.WriteString(strings.Join(names, ", "))
	return nil
}

func (t *LuaTarget) compileCoalesceExpr(b *strings.Builder, expr *ast.CoalesceExpr) error {
	call, err := t.compileExprScratch(expr.Left)
	if err != nil {
		return err
	}

	fallible, ok := t.info.Types[expr.Left].(analyser.FallibleType)
	if !ok {
		return fmt.Errorf("lua: ?? applied to a call that cannot fail")
	}

	values := make([]string, len(fallible.Values))
	results := make([]string, len(fallible.Values))
	for i := range values {
		values[i] = t.newLabel("co")
		results[i] = t.newLabel("cor")
	}
	errName := t.newLabel("coerr")

	targets := append(append([]string{}, values...), errName)
	t.emitPending(fmt.Sprintf("local %s = %s\n", strings.Join(targets, ", "), call))
	if len(results) > 0 {
		t.emitPending(fmt.Sprintf("local %s\n", strings.Join(results, ", ")))
	}

	outer := t.pending
	t.pending = nil

	var branch strings.Builder
	fmt.Fprintf(&branch, "if %s ~= nil then\n", errName)

	switch {
	case expr.Block != nil:
		block, ok := expr.Block.(*ast.BlockStmt)
		if !ok {
			t.pending = outer
			return fmt.Errorf("lua: coalesce expects a block")
		}

		fmt.Fprintf(&branch, "local %s = %s\n", expr.ErrorBind, errName)
		if err := t.compileStatement(&branch, block); err != nil {
			t.pending = outer
			return err
		}
	default:
		if expr.ErrorBind != "" {
			fmt.Fprintf(&branch, "local %s = %s\n", expr.ErrorBind, errName)
		}

		defaults := []ast.Expression{expr.Default}
		if tuple, isTuple := expr.Default.(*ast.TupleExpr); isTuple {
			defaults = tuple.Elements
		}

		texts := make([]string, len(defaults))
		for i, d := range defaults {
			text, err := t.compileExprScratch(d)
			if err != nil {
				t.pending = outer
				return err
			}
			texts[i] = text
		}
		t.flushPending(&branch)
		fmt.Fprintf(&branch, "%s = %s\n", strings.Join(results, ", "), strings.Join(texts, ", "))
	}

	if len(results) > 0 {
		fmt.Fprintf(&branch, "else\n%s = %s\n", strings.Join(results, ", "), strings.Join(values, ", "))
	} else {
		branch.WriteString("else\n")
	}
	branch.WriteString("end\n")

	t.pending = outer
	t.emitPending(branch.String())

	b.WriteString(strings.Join(results, ", "))
	return nil
}

func (t *LuaTarget) compileErrorComparison(b *strings.Builder, e *ast.BinaryExpr) (bool, error) {
	if e.Operator != "==" && e.Operator != "!=" {
		return false, nil
	}

	_, leftErr := t.info.Types[e.Left].(analyser.ErrorType)
	_, rightErr := t.info.Types[e.Right].(analyser.ErrorType)
	if !leftErr && !rightErr {
		return false, nil
	}

	errSide, otherSide := e.Left, e.Right
	_, leftNull := e.Left.(*ast.NullLiteral)
	if leftNull || (rightErr && !isNullLiteral(e.Right)) {
		errSide, otherSide = e.Right, e.Left
	}

	errText, err := t.compileExprScratch(errSide)
	if err != nil {
		return true, err
	}

	if isNullLiteral(otherSide) {
		op := "=="
		if e.Operator == "!=" {
			op = "~="
		}
		fmt.Fprintf(b, "(%s %s nil)", errText, op)
		return true, nil
	}

	enum, ok := t.info.Types[otherSide].(analyser.NamedType)
	if !ok {
		return true, fmt.Errorf("lua: an Error can only be compared with null or an enum value")
	}

	other, err := t.compileExprScratch(otherSide)
	if err != nil {
		return true, err
	}

	call := fmt.Sprintf("__wisp_is(%s, %s, %s)", errText, luaString(enum.Module+"."+enum.Name), other)
	if e.Operator == "==" {
		b.WriteString(call)
	} else {
		fmt.Fprintf(b, "(not %s)", call)
	}
	return true, nil
}

func (t *LuaTarget) entryErrorCheck() string {
	for _, d := range t.program.Declarations {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Name != "main" || fd.Receiver != nil || !fd.Fallible {
			continue
		}
		return `local __main_result = table.pack(main())
local __main_error = __main_result[__main_result.n]
if __main_error ~= nil then
	io.stderr:write("error: ", __main_error.message, " (", __main_error.file, ":", __main_error.line, ")\n")
	os.exit(1)
end
`
	}
	return "main()\n"
}

func isNullLiteral(expr ast.Expression) bool {
	_, ok := expr.(*ast.NullLiteral)
	return ok
}
