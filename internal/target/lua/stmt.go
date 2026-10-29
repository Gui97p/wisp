package lua

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/target"
)

func (t *LuaTarget) compileStatement(b *strings.Builder, stmt ast.Statement) error {
	return t.locate(stmt, t.compileStatementNode(b, stmt))
}

func (t *LuaTarget) compileStatementNode(b *strings.Builder, stmt ast.Statement) error {
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		return t.compileBlockStatement(b, s)
	case *ast.ConstStmt:
		return t.compileDefinition(b, s, s.Vars, s.Values)
	case *ast.VarStmt:
		return t.compileDefinition(b, s, s.Vars, s.Values)
	case *ast.ReturnStmt:
		return t.compileReturnStatement(b, s)
	case *ast.YieldStmt:
		return t.compileYieldStatement(b, s)
	case *ast.ExpressionStmt:
		return t.compileExpressionStatement(b, s)
	case *ast.IfStmt:
		return t.compileIfStatement(b, s)
	case *ast.ForStmt:
		return t.compileForStatement(b, s)
	case *ast.LoopStmt:
		return t.compileLoopStatement(b, s)
	case *ast.BreakStmt:
		return t.compileBreakStatement(b, s)
	case *ast.ContinueStmt:
		return t.compileContinueStatement(b, s)
	case *ast.NativeStmt:
		return t.compileNativeStatement(b, s)
	case *ast.DeferStmt:
		t.registerDefer(s)
		return nil
	case *ast.AssignStmt:
		return t.compileAssignStatement(b, s)
	case *ast.IncDecStmt:
		return t.compileIncDecStatement(b, s)
	default:
		return fmt.Errorf("%s is not supported yet", ast.Describe(stmt))
	}
}

func (t *LuaTarget) compileBlockStatement(b *strings.Builder, block *ast.BlockStmt) error {
	t.pushDeferScope()
	defer t.popDeferScope()

	for _, stmt := range block.Statements {
		if err := t.compileStatement(b, stmt); err != nil {
			return err
		}
	}

	if len(block.Statements) > 0 {
		switch block.Statements[len(block.Statements)-1].(type) {
		case *ast.ReturnStmt, *ast.BreakStmt, *ast.ContinueStmt, *ast.YieldStmt:
			return nil
		}
	}
	return t.emitDefers(b, len(t.deferScopes)-1, false)
}

func (t *LuaTarget) compileReturnStatement(b *strings.Builder, stmt *ast.ReturnStmt) error {
	texts := make([]string, 0, len(stmt.Values)+len(t.currentPayloads)+1)
	for _, value := range stmt.Values {
		text, err := t.compileExprScratch(value)
		if err != nil {
			return err
		}
		texts = append(texts, t.valueText(value, text))
	}

	switch {
	case t.info.ErrorReturns[stmt]:
		zeros, err := t.payloadZeros()
		if err != nil {
			return err
		}
		texts = append(zeros, texts...)
	case t.currentFallible:
		texts = append(texts, "nil")
	}

	t.flushPending(b)

	if t.hasDefers(0, t.info.ErrorReturns[stmt]) {
		packed := t.newLabel("ret")
		b.WriteString("do\n")
		fmt.Fprintf(b, "local %s = __lua_table.pack(%s)\n", packed, strings.Join(texts, ", "))
		if err := t.emitDefers(b, 0, t.info.ErrorReturns[stmt], packed+"["+packed+".n]"); err != nil {
			return err
		}
		fmt.Fprintf(b, "return __lua_table.unpack(%s, 1, %s.n)\n", packed, packed)
		b.WriteString("end\n")
		return nil
	}

	b.WriteString("return ")
	b.WriteString(strings.Join(texts, ", "))
	b.WriteRune('\n')
	return nil
}

func (t *LuaTarget) compileExpressionStatement(b *strings.Builder, stmt *ast.ExpressionStmt) error {
	text, err := t.compileExprScratch(stmt.Expr)
	if err != nil {
		return err
	}

	t.flushPending(b)

	if _, isCall := stmt.Expr.(*ast.CallExpr); isCall && text != "" {
		b.WriteString(text)
		b.WriteRune('\n')
	}
	return nil
}

func (t *LuaTarget) compileIfStatement(b *strings.Builder, stmt *ast.IfStmt) error {
	cond, err := t.compileExprScratch(stmt.Condition)
	if err != nil {
		return err
	}

	t.flushPending(b)

	var ctx loopContext
	if stmt.IsSwitch {
		ctx = loopContext{BreakLabel: t.newLabel("switch_break")}
		t.pushLoop(ctx)
	}

	b.WriteString("if ")
	b.WriteString(cond)
	b.WriteString(" then\n")
	if err := t.compileStatement(b, stmt.Then); err != nil {
		return err
	}

	if stmt.Else != nil {
		b.WriteString("else\n")
		if err := t.compileStatement(b, stmt.Else); err != nil {
			return err
		}
	}

	b.WriteString("end\n")

	if stmt.IsSwitch {
		t.popLoop()
		fmt.Fprintf(b, "::%s::\n", ctx.BreakLabel)
	}

	return nil
}

func (t *LuaTarget) compileForStatement(b *strings.Builder, stmt *ast.ForStmt) error {
	switch {
	case stmt.Range != nil:
		return t.compileForRange(b, stmt)
	case stmt.Start != nil:
		return t.compileForNumeric(b, stmt)

	default:
		return t.compileForCount(b, stmt)
	}
}

func (t *LuaTarget) compileForCount(b *strings.Builder, stmt *ast.ForStmt) error {
	countVar := t.newLabel("count")
	conditionLabel := t.newLabel("for_condition")

	ctx := loopContext{
		Label:            stmt.Label,
		ContinueLabel:    t.newLabel("for_continue"),
		BreakLabel:       t.newLabel("for_break"),
		SupportsContinue: true,
	}

	t.pushLoop(ctx)
	defer t.popLoop()

	end, err := t.compileExprScratch(stmt.End)
	if err != nil {
		return err
	}
	t.flushPending(b)

	fmt.Fprintf(b, "local %s = %s\n", countVar, end)

	fmt.Fprintf(b, "::%s::\n", conditionLabel)

	fmt.Fprintf(
		b,
		"if %s <= 0 then goto %s end\n",
		countVar,
		ctx.BreakLabel,
	)

	b.WriteString("do\n")

	if err := t.compileStatement(b, stmt.Body); err != nil {
		return err
	}

	b.WriteString("end\n")

	fmt.Fprintf(b, "::%s::\n", ctx.ContinueLabel)
	fmt.Fprintf(b, "%s = %s - 1\n", countVar, countVar)
	fmt.Fprintf(b, "goto %s\n", conditionLabel)
	fmt.Fprintf(b, "::%s::\n", ctx.BreakLabel)

	return nil
}

func (t *LuaTarget) compileForRange(b *strings.Builder, stmt *ast.ForStmt) error {
	if pt, ok := t.info.Types[stmt.Range].(analyser.PrimitiveType); ok && pt.Name == "string" {
		return t.compileForRangeString(b, stmt)
	}
	if isView(t.info.Types[stmt.Range]) {
		return t.compileForRangeView(b, stmt)
	}

	rangeVar := t.newLabel("range")
	keyVar := t.newLabel("range_key")
	valueVar := t.newLabel("range_value")

	conditionLabel := t.newLabel("for_condition")

	ctx := loopContext{
		Label:            stmt.Label,
		ContinueLabel:    t.newLabel("for_continue"),
		BreakLabel:       t.newLabel("for_break"),
		SupportsContinue: true,
	}

	t.pushLoop(ctx)
	defer t.popLoop()

	rangeText, err := t.compileExprScratch(stmt.Range)
	if err != nil {
		return err
	}
	t.flushPending(b)

	fmt.Fprintf(b, "local %s = %s\n", rangeVar, rangeText)

	fmt.Fprintf(b, "local %s = nil\n", keyVar)
	fmt.Fprintf(b, "local %s = nil\n", valueVar)

	if luaIdent(stmt.Var) != "" {
		fmt.Fprintf(b, "local %s = nil\n", luaIdent(stmt.Var))
	}

	if luaIdent(stmt.Var2) != "" {
		fmt.Fprintf(b, "local %s = nil\n", luaIdent(stmt.Var2))
	}

	fmt.Fprintf(b, "::%s::\n", conditionLabel)

	fmt.Fprintf(
		b,
		"%s, %s = next(%s, %s)\n",
		keyVar,
		valueVar,
		rangeVar,
		keyVar,
	)

	fmt.Fprintf(
		b,
		"if %s == nil then goto %s end\n",
		keyVar,
		ctx.BreakLabel,
	)

	b.WriteString("do\n")

	if luaIdent(stmt.Var) != "" {
		if luaIdent(stmt.Var2) != "" {
			fmt.Fprintf(b, "%s = %s\n", luaIdent(stmt.Var), keyVar)
			fmt.Fprintf(b, "%s = %s\n", luaIdent(stmt.Var2), valueVar)
		} else {
			fmt.Fprintf(b, "%s = %s\n", luaIdent(stmt.Var), keyVar)
		}
	}

	if err := t.compileStatement(b, stmt.Body); err != nil {
		return err
	}

	b.WriteString("end\n")

	fmt.Fprintf(b, "::%s::\n", ctx.ContinueLabel)
	fmt.Fprintf(b, "goto %s\n", conditionLabel)

	fmt.Fprintf(b, "::%s::\n", ctx.BreakLabel)

	return nil
}

func (t *LuaTarget) compileForRangeString(b *strings.Builder, stmt *ast.ForStmt) error {
	rangeVar := t.newLabel("range")
	lenVar := t.newLabel("range_len")
	idxVar := t.newLabel("range_idx")

	conditionLabel := t.newLabel("for_condition")

	ctx := loopContext{
		Label:            stmt.Label,
		ContinueLabel:    t.newLabel("for_continue"),
		BreakLabel:       t.newLabel("for_break"),
		SupportsContinue: true,
	}

	t.pushLoop(ctx)
	defer t.popLoop()

	rangeText, err := t.compileExprScratch(stmt.Range)
	if err != nil {
		return err
	}
	t.flushPending(b)

	fmt.Fprintf(b, "local %s = %s\n", rangeVar, rangeText)
	fmt.Fprintf(b, "local %s = #%s\n", lenVar, rangeVar)
	fmt.Fprintf(b, "local %s = 0\n", idxVar)

	fmt.Fprintf(b, "::%s::\n", conditionLabel)

	fmt.Fprintf(
		b,
		"if %s >= %s then goto %s end\n",
		idxVar,
		lenVar,
		ctx.BreakLabel,
	)

	b.WriteString("do\n")

	if luaIdent(stmt.Var) != "" {
		fmt.Fprintf(b, "local %s = %s\n", luaIdent(stmt.Var), idxVar)
	}
	if luaIdent(stmt.Var2) != "" {
		fmt.Fprintf(b, "local %s = __lua_string.byte(%s, %s + 1)\n", luaIdent(stmt.Var2), rangeVar, idxVar)
	}

	if err := t.compileStatement(b, stmt.Body); err != nil {
		return err
	}

	b.WriteString("end\n")

	fmt.Fprintf(b, "::%s::\n", ctx.ContinueLabel)
	fmt.Fprintf(b, "%s = %s + 1\n", idxVar, idxVar)
	fmt.Fprintf(b, "goto %s\n", conditionLabel)

	fmt.Fprintf(b, "::%s::\n", ctx.BreakLabel)

	return nil
}

func (t *LuaTarget) compileForRangeView(b *strings.Builder, stmt *ast.ForStmt) error {
	rangeVar := t.newLabel("range")
	lenVar := t.newLabel("range_len")
	idxVar := t.newLabel("range_idx")

	conditionLabel := t.newLabel("for_condition")

	ctx := loopContext{
		Label:            stmt.Label,
		ContinueLabel:    t.newLabel("for_continue"),
		BreakLabel:       t.newLabel("for_break"),
		SupportsContinue: true,
	}

	t.pushLoop(ctx)
	defer t.popLoop()

	rangeText, err := t.compileExprScratch(stmt.Range)
	if err != nil {
		return err
	}
	t.flushPending(b)

	fmt.Fprintf(b, "local %s = %s\n", rangeVar, rangeText)
	fmt.Fprintf(b, "local %s = %s.n\n", lenVar, rangeVar)
	fmt.Fprintf(b, "local %s = 0\n", idxVar)

	fmt.Fprintf(b, "::%s::\n", conditionLabel)

	fmt.Fprintf(
		b,
		"if %s >= %s then goto %s end\n",
		idxVar,
		lenVar,
		ctx.BreakLabel,
	)

	b.WriteString("do\n")

	if luaIdent(stmt.Var) != "" {
		fmt.Fprintf(b, "local %s = %s\n", luaIdent(stmt.Var), idxVar)
	}
	if luaIdent(stmt.Var2) != "" {
		element := fmt.Sprintf("%s.t[%s.o + %s]", rangeVar, rangeVar, idxVar)
		if elem := viewElement(t.info.Types[stmt.Range]); elem != nil && isAggregate(elem) {
			element = t.aggregateFns(elem).copyName + "(" + element + ")"
		}
		fmt.Fprintf(b, "local %s = %s\n", luaIdent(stmt.Var2), element)
	}

	if err := t.compileStatement(b, stmt.Body); err != nil {
		return err
	}

	b.WriteString("end\n")

	fmt.Fprintf(b, "::%s::\n", ctx.ContinueLabel)
	fmt.Fprintf(b, "%s = %s + 1\n", idxVar, idxVar)
	fmt.Fprintf(b, "goto %s\n", conditionLabel)

	fmt.Fprintf(b, "::%s::\n", ctx.BreakLabel)

	return nil
}

func (t *LuaTarget) compileForNumeric(b *strings.Builder, stmt *ast.ForStmt) error {
	conditionLabel := t.newLabel("for_condition")

	ctx := loopContext{
		Label:            stmt.Label,
		ContinueLabel:    t.newLabel("for_continue"),
		BreakLabel:       t.newLabel("for_break"),
		SupportsContinue: true,
	}

	t.pushLoop(ctx)
	defer t.popLoop()

	start := "1"
	if stmt.Start != nil {
		text, err := t.compileExprScratch(stmt.Start)
		if err != nil {
			return err
		}
		t.flushPending(b)
		start = text
	}
	fmt.Fprintf(b, "local %s = %s\n", luaIdent(stmt.Var), start)

	end, err := t.compileExprScratch(stmt.End)
	if err != nil {
		return err
	}
	t.flushPending(b)
	endVar := t.newLabel("for_end")
	fmt.Fprintf(b, "local %s = %s\n", endVar, end)

	step := "1"
	if stmt.Step != nil {
		text, err := t.compileExprScratch(stmt.Step)
		if err != nil {
			return err
		}
		t.flushPending(b)
		step = text
	}
	stepVar := t.newLabel("for_step")
	fmt.Fprintf(b, "local %s = %s\n", stepVar, step)
	if stmt.Step != nil {
		fmt.Fprintf(b, "if %s <= 0 then __wisp_panic(\"range step must be positive\") end\n", stepVar)
	}

	upVar := t.newLabel("for_up")
	fmt.Fprintf(b, "local %s = %s <= %s\n", upVar, luaIdent(stmt.Var), endVar)

	fmt.Fprintf(b, "::%s::\n", conditionLabel)
	fmt.Fprintf(b, "if %s then\nif %s > %s then goto %s end\nelse\nif %s < %s then goto %s end\nend\n",
		upVar, luaIdent(stmt.Var), endVar, ctx.BreakLabel, luaIdent(stmt.Var), endVar, ctx.BreakLabel)

	b.WriteString("do\n")

	if err := t.compileStatement(b, stmt.Body); err != nil {
		return err
	}

	b.WriteString("end\n")

	fmt.Fprintf(b, "::%s::\n", ctx.ContinueLabel)
	fmt.Fprintf(b, "if %s then %s = %s + %s else %s = %s - %s end\n", upVar, luaIdent(stmt.Var), luaIdent(stmt.Var), stepVar, luaIdent(stmt.Var), luaIdent(stmt.Var), stepVar)

	fmt.Fprintf(b, "goto %s\n", conditionLabel)
	fmt.Fprintf(b, "::%s::\n", ctx.BreakLabel)

	return nil
}

func (t *LuaTarget) compileLoopStatement(b *strings.Builder, stmt *ast.LoopStmt) error {
	conditionLabel := t.newLabel("loop_condition")

	ctx := loopContext{
		Label:            stmt.Label,
		ContinueLabel:    t.newLabel("loop_continue"),
		BreakLabel:       t.newLabel("loop_break"),
		SupportsContinue: true,
	}

	t.pushLoop(ctx)
	defer t.popLoop()

	fmt.Fprintf(b, "::%s::\n", conditionLabel)

	if stmt.Condition != nil {
		cond, err := t.compileExprScratch(stmt.Condition)
		if err != nil {
			return err
		}
		t.flushPending(b)
		fmt.Fprintf(b, "if not (%s) then goto %s end\n", cond, ctx.BreakLabel)
	}

	b.WriteString("do\n")

	if err := t.compileStatement(b, stmt.Body); err != nil {
		return err
	}

	inside := stmt.UntilCondition != nil && t.info.UntilSeesBody[stmt]
	if inside {
		cond, err := t.compileExprScratch(stmt.UntilCondition)
		if err != nil {
			return err
		}
		t.flushPending(b)
		fmt.Fprintf(b, "if %s then goto %s end\n", cond, ctx.BreakLabel)
	}

	b.WriteString("end\n")

	fmt.Fprintf(b, "::%s::\n", ctx.ContinueLabel)

	if stmt.UntilCondition != nil && !inside {
		cond, err := t.compileExprScratch(stmt.UntilCondition)
		if err != nil {
			return err
		}
		t.flushPending(b)
		fmt.Fprintf(b, "if %s then goto %s end\n", cond, ctx.BreakLabel)
	}

	fmt.Fprintf(b, "goto %s\n", conditionLabel)
	fmt.Fprintf(b, "::%s::\n", ctx.BreakLabel)

	return nil
}

func (t *LuaTarget) compileBreakStatement(b *strings.Builder, stmt *ast.BreakStmt) error {
	var loop *loopContext

	if stmt.Label == "" {
		loop = t.currentLoop()
	} else {
		loop = t.findLoop(stmt.Label)
	}

	if loop == nil {
		return fmt.Errorf("break: no matching loop")
	}

	if err := t.emitDefers(b, loop.DeferDepth, false); err != nil {
		return err
	}
	fmt.Fprintf(b, "goto %s\n", loop.BreakLabel)

	return nil
}

func (t *LuaTarget) compileContinueStatement(b *strings.Builder, stmt *ast.ContinueStmt) error {
	var loop *loopContext

	if stmt.Label == "" {
		loop = t.currentContinuable()
	} else {
		loop = t.findLoop(stmt.Label)
	}

	if loop == nil {
		return fmt.Errorf("continue: no matching loop")
	}

	if err := t.emitDefers(b, loop.DeferDepth, false); err != nil {
		return err
	}
	fmt.Fprintf(b, "goto %s\n", loop.ContinueLabel)

	return nil
}

func (t *LuaTarget) compileNativeStatement(b *strings.Builder, stmt *ast.NativeStmt) error {
	if stmt.Backend != "" && !target.MatchesName("lua", stmt.Backend) {
		return nil
	}
	_, err := fmt.Fprintln(b, stmt.Code)
	return err
}

func (t *LuaTarget) compileAssignStatement(b *strings.Builder, stmt *ast.AssignStmt) error {
	places := make([]placeSpec, len(stmt.Targets))
	for i, target := range stmt.Targets {
		place, err := t.compilePlace(target)
		if err != nil {
			return err
		}
		places[i] = place
	}

	values := make([]string, len(stmt.Values))
	for i, value := range stmt.Values {
		text, err := t.compileExprScratch(value)
		if err != nil {
			return err
		}
		values[i] = text
	}

	t.flushPending(b)

	if stmt.Op != "=" {
		t.stabilize(b, &places[0])
		value := t.compoundText(stmt, places[0].get(), values[0])
		b.WriteString(places[0].set(value))
		return nil
	}

	allText := true
	for _, place := range places {
		if place.kind != placeText || place.into != "" {
			allText = false
		}
	}

	if allText {
		targets := make([]string, len(places))
		for i, place := range places {
			targets[i] = place.text
		}
		fmt.Fprintf(b, "%s = %s\n", strings.Join(targets, ", "), strings.Join(values, ", "))
		return nil
	}

	if len(places) == 1 {
		b.WriteString(places[0].set(values[0]))
		return nil
	}

	for i := range values {
		if i < len(stmt.Values) {
			values[i] = t.valueText(stmt.Values[i], values[i])
		}
	}

	temps := make([]string, len(places))
	for i := range temps {
		temps[i] = t.newLabel("assign")
	}
	fmt.Fprintf(b, "local %s = %s\n", strings.Join(temps, ", "), strings.Join(values, ", "))
	for i, place := range places {
		b.WriteString(place.set(temps[i]))
	}
	return nil
}

func (t *LuaTarget) compoundText(stmt *ast.AssignStmt, target, value string) string {
	targetType := t.info.Types[stmt.Targets[0]]
	op := strings.TrimSuffix(stmt.Op, "=")

	if pointee(targetType) != nil {
		if op == "-" {
			return fmt.Sprintf("__wisp_padd(%s, -(%s))", target, value)
		}
		return fmt.Sprintf("__wisp_padd(%s, %s)", target, value)
	}

	switch op {
	case "+":
		if pt, ok := targetType.(analyser.PrimitiveType); ok && pt.Name == "string" {
			if vt, ok := t.info.Types[stmt.Values[0]].(analyser.PrimitiveType); ok && vt.Name == "char" {
				value = fmt.Sprintf("__lua_string.char(%s)", value)
			}
			return fmt.Sprintf("(%s .. %s)", target, value)
		}
		return wrapInt(fmt.Sprintf("(%s + %s)", target, value), targetType)
	case "-", "*":
		return wrapInt(fmt.Sprintf("(%s %s %s)", target, op, value), targetType)
	case "<<":
		return wrapInt(fmt.Sprintf("(%s << %s)", target, value), targetType)
	case ">>":
		if isSignedInt(targetType) {
			return fmt.Sprintf("__wisp_sar(%s, %s)", target, value)
		}
		return fmt.Sprintf("(%s >> %s)", target, value)
	case "/", "%":
		return t.divisionText(op, target, value, targetType)
	case "^":
		return fmt.Sprintf("(%s ~ %s)", target, value)
	}
	return fmt.Sprintf("(%s %s %s)", target, op, value)
}

func (t *LuaTarget) compileIncDecStatement(b *strings.Builder, stmt *ast.IncDecStmt) error {
	place, err := t.compilePlace(stmt.Target)
	if err != nil {
		return err
	}

	t.flushPending(b)
	t.stabilize(b, &place)

	targetType := t.info.Types[stmt.Target]
	if pointee(targetType) != nil {
		step := "1"
		if stmt.Op[0] == '-' {
			step = "-1"
		}
		b.WriteString(place.set(fmt.Sprintf("__wisp_padd(%s, %s)", place.get(), step)))
		return nil
	}

	text := wrapInt(fmt.Sprintf("(%s %c 1)", place.get(), stmt.Op[0]), targetType)
	b.WriteString(place.set(text))
	return nil
}

func (t *LuaTarget) compileYieldStatement(b *strings.Builder, stmt *ast.YieldStmt) error {
	if len(t.coalesceFrames) == 0 {
		return fmt.Errorf("lua: yield outside a ?? handler")
	}
	frame := t.coalesceFrames[len(t.coalesceFrames)-1]

	texts := make([]string, len(stmt.Values))
	for i, value := range stmt.Values {
		text, err := t.compileExprScratch(value)
		if err != nil {
			return err
		}
		texts[i] = text
	}

	if len(frame.results) == 0 || (len(texts) == 1 && texts[0] == "") {
		t.flushPending(b)
		return t.finishYield(b, frame)
	}

	t.flushPending(b)
	fmt.Fprintf(b, "%s = %s\n", strings.Join(frame.results, ", "), strings.Join(texts, ", "))
	return t.finishYield(b, frame)
}

func (t *LuaTarget) finishYield(b *strings.Builder, frame coalesceFrame) error {
	if err := t.emitDefers(b, frame.deferDepth, false); err != nil {
		return err
	}
	fmt.Fprintf(b, "goto %s\n", frame.label)
	return nil
}
