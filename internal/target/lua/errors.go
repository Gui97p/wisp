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

	if expr.Assert {
		t.emitPending(fmt.Sprintf("if %s ~= nil then __wisp_unwrap(%s) end\n", errName, errName))
		b.WriteString(strings.Join(names, ", "))
		return nil
	}

	zeros, err := t.payloadZeros()
	if err != nil {
		return err
	}
	zeros = append(zeros, errName)
	if t.hasDefers(0, true) {
		var exit strings.Builder
		fmt.Fprintf(&exit, "if %s ~= nil then\n", errName)
		if err := t.emitDefers(&exit, 0, true, errName); err != nil {
			return err
		}
		fmt.Fprintf(&exit, "return %s\nend\n", strings.Join(zeros, ", "))
		t.emitPending(exit.String())
	} else {
		t.emitPending(fmt.Sprintf("if %s ~= nil then return %s end\n", errName, strings.Join(zeros, ", ")))
	}

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

	block, ok := expr.Block.(*ast.BlockStmt)
	if !ok {
		return fmt.Errorf("lua: ?? expects a block")
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

	if expr.ErrorBind != "" {
		fmt.Fprintf(&branch, "local %s = %s\n", expr.ErrorBind, errName)
	}
	endLabel := t.newLabel("coend")
	t.coalesceFrames = append(t.coalesceFrames, coalesceFrame{results: results, label: endLabel, deferDepth: len(t.deferScopes)})
	err = t.compileStatement(&branch, block)
	t.coalesceFrames = t.coalesceFrames[:len(t.coalesceFrames)-1]
	if err != nil {
		t.pending = outer
		return err
	}

	if len(results) > 0 {
		fmt.Fprintf(&branch, "else\n%s = %s\n", strings.Join(results, ", "), strings.Join(values, ", "))
	} else {
		branch.WriteString("else\n")
	}
	branch.WriteString("end\n")
	fmt.Fprintf(&branch, "::%s::\n", endLabel)

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
	prelude, call := "", "main()"
	for _, d := range t.program.Declarations {
		fd, ok := d.(*ast.FuncDecl)
		if ok && fd.Name == "main" && fd.Receiver == nil && len(fd.Params) == 1 {
			prelude = "local __args = {}\nfor i = 1, #arg do\n\t__args[i - 1] = arg[i]\nend\n"
			call = "main(__args)"
		}
	}
	for _, d := range t.program.Declarations {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Name != "main" || fd.Receiver != nil || !fd.Fallible {
			continue
		}
		return prelude + `local __main_result = table.pack(` + call + `)
local __main_error = __main_result[__main_result.n]
if __main_error ~= nil then
	io.stderr:write("error: ", __main_error.message, " (", __main_error.file, ":", __main_error.line, ")\n")
	os.exit(1)
end
`
	}
	return prelude + call + "\n"
}

func isNullLiteral(expr ast.Expression) bool {
	_, ok := expr.(*ast.NullLiteral)
	return ok
}

func (t *LuaTarget) compileSwitchExpression(b *strings.Builder, expr *ast.SwitchExpr) error {
	values := t.info.SwitchValues[expr]
	results := make([]string, len(values))
	for i := range results {
		results[i] = t.newLabel("sw")
	}
	if len(results) > 0 {
		t.emitPending(fmt.Sprintf("local %s\n", strings.Join(results, ", ")))
	}

	outer := t.pending
	t.pending = nil

	var branch strings.Builder
	for _, stmt := range expr.Prelude {
		if err := t.compileStatement(&branch, stmt); err != nil {
			t.pending = outer
			return err
		}
	}

	arms := make([]*ast.SwitchArm, 0, len(expr.Arms)+1)
	for i := range expr.Arms {
		arms = append(arms, &expr.Arms[i])
	}
	arms = append(arms, expr.Default)

	opened := 0
	for _, arm := range arms {
		if arm.Condition != nil {
			cond, err := t.compileExprScratch(arm.Condition)
			if err != nil {
				t.pending = outer
				return err
			}
			t.flushPending(&branch)
			fmt.Fprintf(&branch, "if %s then\n", cond)
			opened++
		}

		if err := t.compileSwitchArm(&branch, arm, results); err != nil {
			t.pending = outer
			return err
		}

		if arm.Condition != nil {
			branch.WriteString("else\n")
		}
	}
	for range opened {
		branch.WriteString("end\n")
	}

	t.pending = outer
	t.emitPending(branch.String())

	b.WriteString(strings.Join(results, ", "))
	return nil
}

func (t *LuaTarget) compileSwitchArm(b *strings.Builder, arm *ast.SwitchArm, results []string) error {
	if arm.Exit != nil {
		return t.compileStatement(b, arm.Exit)
	}

	texts := make([]string, len(arm.Values))
	for i, value := range arm.Values {
		text, err := t.compileExprScratch(value)
		if err != nil {
			return err
		}
		texts[i] = text
	}
	t.flushPending(b)
	fmt.Fprintf(b, "%s = %s\n", strings.Join(results, ", "), strings.Join(texts, ", "))
	return nil
}
