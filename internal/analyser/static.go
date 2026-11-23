package analyser

import (
	"fmt"
	"math/big"
	"slices"

	"github.com/Gui97p/wisp/internal/ast"
)

const staticStepLimit = 2_000_000

type staticFlow int

const (
	flowNext staticFlow = iota
	flowBreak
	flowContinue
	flowYield
	flowReturn
)

type staticRun struct {
	a      *Analyser
	flow   staticFlow
	label  string
	result *ConstValue
	failed bool
}

func (r *staticRun) fail(node ast.Node, format string, args ...any) staticFlow {
	if !r.failed {
		r.failed = true
		r.a.errorf(node, "%s", fmt.Sprintf(format, args...)).
			WithHelp("a static block can use numbers, booleans, strings, arrays, structs, control flow and ordinary functions, but not pointers, memory or types as values")
	}
	return flowReturn
}

func (a *Analyser) runStatic(expr *ast.BlockExpr, tp Type) {
	if a.staticVars != nil {
		return
	}
	if _, done := a.info.Statics[expr]; done {
		return
	}

	saved := a.quiet
	a.quiet = false
	a.staticVars = map[*Symbol]ConstValue{}
	a.staticSteps = 0
	defer func() {
		a.quiet = saved
		a.staticVars = nil
	}()

	r := &staticRun{a: a}
	flow := r.block(expr.Block)
	if !r.failed && (flow != flowYield || r.result == nil) {
		r.fail(expr, "this static block ended without a value")
	}
	if r.failed {
		if zero, ok := zeroValue(tp); ok {
			a.info.Statics[expr] = zero
		}
		return
	}
	if _, bad := tp.(InvalidType); bad {
		return
	}
	a.info.Statics[expr] = *r.result
}

func (r *staticRun) step(node ast.Node) bool {
	r.a.staticSteps++
	if r.a.staticSteps > staticStepLimit {
		r.fail(node, "the static block takes too long: it ran more than %d steps", staticStepLimit)
		return false
	}
	return true
}

func (r *staticRun) value(expr ast.Expression) (ConstValue, bool) {
	before := len(r.a.errors)
	value, ok := r.a.evalConst(expr)
	if !ok && !r.failed {
		if len(r.a.errors) > before {
			r.failed = true
		} else if outer := r.outerVariable(expr); outer != nil {
			r.fail(outer, "%s is a variable of the surrounding code: a static block runs at compile time, and only sees its own variables and constants", outer.Value)
		} else if hooked := hookedOperator(expr); hooked != nil {
			r.fail(hooked, "this operator goes through a method of a type, and a static block runs only constants and ordinary functions")
		} else {
			r.fail(expr, "this cannot be evaluated at compile time")
		}
	}
	return value, ok
}

func (r *staticRun) block(b *ast.BlockStmt) staticFlow {
	if b == nil {
		return flowNext
	}
	for _, stmt := range b.Statements {
		if flow := r.stmt(stmt); flow != flowNext {
			if flow == flowBreak && b.Label != "" && r.label == b.Label {
				r.label = ""
				return flowNext
			}
			return flow
		}
	}
	return flowNext
}

func (r *staticRun) fits(node ast.Node, tp Type, value ConstValue) bool {
	if value.Kind != ConstInt || tp == nil {
		return true
	}
	if bits, signed, ok := IntegerRange(tp); ok && !fitsBig(value.Int, bits, signed) {
		r.fail(node, "the value %s does not fit in %s", value.Int.String(), tp.String())
		return false
	}
	return true
}

func (r *staticRun) declare(stmt ast.Statement, symbols []*Symbol, types []Type, values []ast.Expression) staticFlow {
	for i, sym := range symbols {
		var value ConstValue
		var ok bool
		if i < len(values) {
			value, ok = r.value(values[i])
			if !ok {
				return flowReturn
			}
			if i < len(types) {
				if converted, can := r.a.convertConst(value, types[i]); can {
					value = converted
				}
				if !r.fits(stmt, types[i], value) {
					return flowReturn
				}
			}
		} else if i < len(types) {
			value, ok = zeroValue(types[i])
			if !ok {
				return r.fail(stmt, "a variable of type %s cannot be used at compile time", types[i].String())
			}
		}
		r.a.staticVars[sym] = value.clone()
	}
	return flowNext
}

type staticSlot struct {
	get func() ConstValue
	set func(ConstValue)
	tp  Type
}

func (r *staticRun) slot(target ast.Expression) (staticSlot, bool) {
	a := r.a
	switch e := target.(type) {
	case *ast.IdentLiteral:
		sym := a.info.Idents[e]
		if sym == nil {
			r.fail(target, "this name is not a variable of the static block")
			return staticSlot{}, false
		}
		if _, exists := a.staticVars[sym]; !exists {
			r.fail(target, "%s is not a variable of this static block", sym.Name)
			return staticSlot{}, false
		}
		return staticSlot{
			get: func() ConstValue { return a.staticVars[sym] },
			set: func(v ConstValue) { a.staticVars[sym] = v },
			tp:  sym.Type,
		}, true
	case *ast.IndexExpr:
		base, ok := r.slot(e.Array)
		if !ok {
			return staticSlot{}, false
		}
		index, ok := r.value(e.Index)
		if !ok {
			return staticSlot{}, false
		}
		if index.Kind != ConstInt || !index.Int.IsInt64() {
			r.fail(e.Index, "the index must be an integer")
			return staticSlot{}, false
		}
		i := index.Int.Int64()
		current := base.get()
		if current.Kind != ConstArray {
			r.fail(e, "only an element of an array can be assigned in a static block")
			return staticSlot{}, false
		}
		if i < 0 || i >= int64(len(current.Elems)) {
			r.fail(e, "index %d is out of range: the array has %d elements", i, len(current.Elems))
			return staticSlot{}, false
		}
		return staticSlot{
			get: func() ConstValue { return base.get().Elems[i] },
			set: func(v ConstValue) { base.get().Elems[i] = v },
			tp:  a.info.Types[e],
		}, true
	case *ast.MemberExpr:
		base, ok := r.slot(e.Object)
		if !ok {
			return staticSlot{}, false
		}
		current := base.get()
		st, isStruct := underlying(current.Type).(*StructType)
		if current.Kind != ConstStruct || !isStruct {
			r.fail(e, "only a field of a struct can be assigned in a static block")
			return staticSlot{}, false
		}
		for i, name := range st.Order {
			if name == e.Field {
				return staticSlot{
					get: func() ConstValue { return base.get().Elems[i] },
					set: func(v ConstValue) { base.get().Elems[i] = v },
					tp:  st.Fields[name],
				}, true
			}
		}
		r.fail(e, "%s has no field %s", st.Name, e.Field)
		return staticSlot{}, false
	}
	r.fail(target, "only a variable, an element or a field can be assigned in a static block")
	return staticSlot{}, false
}

func (r *staticRun) assign(stmt ast.Statement, target ast.Expression, op string, valueExpr ast.Expression) staticFlow {
	slot, ok := r.slot(target)
	if !ok {
		return flowReturn
	}

	var value ConstValue
	if op == "=" {
		value, ok = r.value(valueExpr)
		if !ok {
			return flowReturn
		}
	} else {
		bin := &ast.BinaryExpr{Left: target, Operator: op[:len(op)-1], Right: valueExpr}
		bin.SetPos(target.Position())
		bin.SetEndPos(valueExpr.EndPosition())
		r.a.info.Types[bin] = slot.tp
		value, ok = r.value(bin)
		if !ok {
			return flowReturn
		}
	}
	if converted, can := r.a.fitConst(value, slot.tp); can {
		value = converted
	}
	if !r.fits(stmt, slot.tp, value) {
		return flowReturn
	}
	slot.set(value.clone())
	return flowNext
}

func (r *staticRun) loopBody(body *ast.BlockStmt, label string) (stop bool, flow staticFlow) {
	flow = r.block(body)
	switch flow {
	case flowBreak:
		if r.label == "" || r.label == label {
			r.label = ""
			return true, flowNext
		}
		return true, flowBreak
	case flowContinue:
		if r.label == "" || r.label == label {
			r.label = ""
			return false, flowNext
		}
		return true, flowContinue
	case flowYield, flowReturn:
		return true, flow
	}
	return false, flowNext
}

func (r *staticRun) stmt(stmt ast.Statement) staticFlow {
	if !r.step(stmt) {
		return flowReturn
	}
	a := r.a

	switch s := stmt.(type) {
	case *ast.BlockStmt:
		return r.block(s)
	case *ast.VarStmt:
		return r.declare(s, a.info.VarSymbols[s], a.info.VarTypes[s], s.Values)
	case *ast.ConstStmt:
		return r.declare(s, a.info.VarSymbols[s], a.info.VarTypes[s], s.Values)
	case *ast.AssignStmt:
		if len(s.Targets) != 1 || len(s.Values) != 1 {
			return r.fail(s, "only a single assignment can be evaluated at compile time")
		}
		return r.assign(s, s.Targets[0], s.Op, s.Values[0])
	case *ast.IncDecStmt:
		one := &ast.IntLiteral{Value: 1}
		one.SetPos(s.Position())
		a.info.Types[one] = PrimitiveType{Name: "int"}
		op := "+="
		if s.Op == "--" {
			op = "-="
		}
		return r.assign(s, s.Target, op, one)
	case *ast.ExpressionStmt:
		call, ok := s.Expr.(*ast.CallExpr)
		if !ok {
			return r.fail(s, "this statement cannot be evaluated at compile time")
		}
		if a.info.StaticAsserts[call] {
			condition, ok := a.evalConst(call.Args[0])
			if !ok {
				return r.fail(call, "this condition is not known at compile time")
			}
			if condition.Kind == ConstBool && !condition.Bool {
				message, _ := a.evalConst(call.Args[1])
				return r.fail(call, "static assertion failed: %s", message.Str)
			}
			return flowNext
		}
		if a.isPanicCall(call) {
			a.staticPanic(call)
			r.failed = true
			return flowReturn
		}
		if _, ok := r.value(call); !ok {
			return flowReturn
		}
		return flowNext
	case *ast.IfStmt:
		if s.Capture != "" {
			return r.fail(s, "an if with a capture cannot be evaluated at compile time")
		}
		flow := r.ifChain(s)
		if s.IsSwitch && flow == flowBreak && r.label == "" {
			return flowNext
		}
		return flow
	case *ast.ForStmt:
		return r.forLoop(s)
	case *ast.LoopStmt:
		return r.loop(s)
	case *ast.BreakStmt:
		r.label = s.Label
		return flowBreak
	case *ast.ContinueStmt:
		r.label = s.Label
		return flowContinue
	case *ast.YieldStmt:
		if len(s.Values) != 1 {
			return r.fail(s, "a static block gives exactly one value")
		}
		value, ok := r.value(s.Values[0])
		if !ok {
			return flowReturn
		}
		r.result = &value
		if s.Target != "" {
			r.label = s.Target
		}
		return flowYield
	case *ast.ReturnStmt:
		if len(s.Values) != 1 {
			return r.fail(s, "a function used at compile time returns exactly one value")
		}
		value, ok := r.value(s.Values[0])
		if !ok {
			return flowReturn
		}
		r.result = &value
		return flowReturn
	}
	return r.fail(stmt, "%s cannot be evaluated at compile time", ast.Describe(stmt))
}

func (r *staticRun) ifChain(s *ast.IfStmt) staticFlow {
	condition, ok := r.value(s.Condition)
	if !ok {
		return flowReturn
	}
	if condition.Kind != ConstBool {
		return r.fail(s.Condition, "the condition must be a boolean")
	}
	if condition.Bool {
		return r.block(s.Then)
	}
	if s.Else != nil {
		return r.stmt(s.Else)
	}
	return flowNext
}

func (r *staticRun) forLoop(s *ast.ForStmt) staticFlow {
	if s.Range != nil {
		return r.forRange(s)
	}
	if s.Start == nil || s.End == nil || s.Var2 != "" {
		return r.fail(s, "only a counting loop, as in for i in 0..9, or a walk over an array or a string can be evaluated at compile time")
	}
	start, ok := r.value(s.Start)
	if !ok {
		return flowReturn
	}
	end, ok := r.value(s.End)
	if !ok {
		return flowReturn
	}
	step := big.NewInt(1)
	if s.Step != nil {
		value, ok := r.value(s.Step)
		if !ok {
			return flowReturn
		}
		step = value.Int
	}
	if start.Kind != ConstInt || end.Kind != ConstInt || step == nil || step.Sign() == 0 {
		return r.fail(s, "the bounds and the step of the loop must be non-zero integers known at compile time")
	}

	var sym *Symbol
	if s.Var != "" {
		if syms := r.a.info.VarSymbols[s]; len(syms) > 0 {
			sym = syms[0]
		}
	}
	for i := new(big.Int).Set(start.Int); ; i.Add(i, step) {
		if step.Sign() > 0 && i.Cmp(end.Int) > 0 || step.Sign() < 0 && i.Cmp(end.Int) < 0 {
			return flowNext
		}
		if !r.step(s) {
			return flowReturn
		}
		if sym != nil {
			r.a.staticVars[sym] = constInt(new(big.Int).Set(i))
		}
		stop, flow := r.loopBody(s.Body, s.Label)
		if stop {
			return flow
		}
	}
}

func (r *staticRun) forRange(s *ast.ForStmt) staticFlow {
	whole, ok := r.value(s.Range)
	if !ok {
		return flowReturn
	}
	var count int
	switch whole.Kind {
	case ConstArray:
		count = len(whole.Elems)
	case ConstString:
		count = len(whole.Str)
	default:
		return r.fail(s.Range, "only an array or a string can be walked at compile time")
	}
	whole = whole.clone()
	symbols := r.a.info.VarSymbols[s]
	for i := 0; i < count; i++ {
		if !r.step(s) {
			return flowReturn
		}
		if s.Var != "" && len(symbols) > 0 {
			r.a.staticVars[symbols[0]] = constInt(big.NewInt(int64(i)))
		}
		if s.Var2 != "" && len(symbols) > 1 {
			if whole.Kind == ConstArray {
				r.a.staticVars[symbols[1]] = whole.Elems[i].clone()
			} else {
				r.a.staticVars[symbols[1]] = constInt(big.NewInt(int64(whole.Str[i])))
			}
		}
		stop, flow := r.loopBody(s.Body, s.Label)
		if stop {
			return flow
		}
	}
	return flowNext
}

func (r *staticRun) loop(s *ast.LoopStmt) staticFlow {
	for {
		if !r.step(s) {
			return flowReturn
		}
		if s.Condition != nil {
			condition, ok := r.value(s.Condition)
			if !ok {
				return flowReturn
			}
			if condition.Kind != ConstBool {
				return r.fail(s.Condition, "the condition must be a boolean")
			}
			if !condition.Bool {
				return flowNext
			}
		}
		stop, flow := r.loopBody(s.Body, s.Label)
		if stop {
			return flow
		}
		if s.UntilCondition != nil {
			condition, ok := r.value(s.UntilCondition)
			if !ok {
				return flowReturn
			}
			if condition.Kind == ConstBool && condition.Bool {
				return flowNext
			}
		}
	}
}

const staticDepthLimit = 200

func (a *Analyser) staticPanic(call *ast.CallExpr) {
	message := "no message"
	if len(call.Args) == 1 {
		if value, ok := a.evalConst(call.Args[0]); ok && value.Kind == ConstString {
			message = value.Str
		}
	}
	a.errorf(call, "the static block stopped: %s", message)
}

func (a *Analyser) staticCall(call *ast.CallExpr) (ConstValue, bool) {
	if a.isPanicCall(call) {
		a.staticPanic(call)
		return ConstValue{}, false
	}

	ft := a.calleeFunc(call)
	if ft == nil || ft.Decl == nil || ft.Decl.Body == nil || ft.Decl.Extern || ft.Decl.Native {
		a.errorf(call, "only a function written in Wisp, with a body, can be called at compile time")
		return ConstValue{}, false
	}
	if ft.Decl.Receiver != nil {
		a.errorf(call, "a method cannot be called at compile time yet")
		return ConstValue{}, false
	}
	if len(ft.Returns) != 1 || ft.Fallible {
		a.errorf(call, "a function used at compile time returns exactly one value and cannot fail")
		return ConstValue{}, false
	}

	a.staticDepth++
	defer func() { a.staticDepth-- }()
	if a.staticDepth > staticDepthLimit {
		a.errorf(call, "the recursion of %s is too deep for compile time: more than %d calls inside each other", ft.label(), staticDepthLimit)
		return ConstValue{}, false
	}

	inline := a.info.Inlines[call]
	if inline == nil {
		decl := ft.Decl
		reusable := !call.Spread && len(call.Args) == len(decl.Params) && !slices.ContainsFunc(decl.Params, func(p ast.Param) bool { return p.Const || p.Variadic })
		if reusable {
			inline = a.staticExpansions[decl]
		}
		if inline == nil {
			a.expandInlineWith(call, ft, true)
			inline = a.info.Inlines[call]
			if reusable && inline != nil {
				a.staticExpansions[decl] = inline
			}
		}
	}
	if inline == nil {
		a.errorf(call, "this call cannot be expanded at compile time")
		return ConstValue{}, false
	}

	values := make([]ConstValue, len(inline.Params))
	for k := range inline.Params {
		value, ok := a.evalConst(call.Args[inline.ArgIndex[k]])
		if !ok {
			a.errorf(call.Args[inline.ArgIndex[k]], "this argument is not known at compile time")
			return ConstValue{}, false
		}
		values[k] = value.clone()
	}

	callerVars := a.staticVars
	a.staticVars = make(map[*Symbol]ConstValue, len(values)+4)
	defer func() { a.staticVars = callerVars }()
	for k, param := range inline.Params {
		a.staticVars[param] = values[k]
	}

	r := &staticRun{a: a}
	flow := r.block(inline.Body)
	if r.failed {
		return ConstValue{}, false
	}
	if flow != flowReturn || r.result == nil {
		a.errorf(call, "the function %s ended without returning a value at compile time", inline.Decl.Name)
		return ConstValue{}, false
	}
	return *r.result, true
}

func (a *Analyser) evalSwitchExpr(e *ast.SwitchExpr) (ConstValue, bool) {
	r := &staticRun{a: a}
	for _, stmt := range e.Prelude {
		if flow := r.stmt(stmt); r.failed || flow != flowNext {
			return ConstValue{}, false
		}
	}
	arm := func(arm *ast.SwitchArm) (ConstValue, bool) {
		if arm.Exit != nil {
			a.errorf(arm, "a switch arm that leaves cannot be evaluated at compile time")
			return ConstValue{}, false
		}
		if len(arm.Values) != 1 {
			a.errorf(arm, "a switch arm gives exactly one value at compile time")
			return ConstValue{}, false
		}
		return a.evalConst(arm.Values[0])
	}
	for i := range e.Arms {
		condition, ok := a.evalConst(e.Arms[i].Condition)
		if !ok || condition.Kind != ConstBool {
			a.errorf(e.Arms[i].Condition, "this case cannot be tested at compile time")
			return ConstValue{}, false
		}
		if condition.Bool {
			return arm(&e.Arms[i])
		}
	}
	if e.Default != nil {
		return arm(e.Default)
	}
	a.errorf(e, "no case of this switch matches at compile time")
	return ConstValue{}, false
}

func (r *staticRun) outerVariable(expr ast.Expression) *ast.IdentLiteral {
	var found *ast.IdentLiteral
	ast.Inspect(expr, func(n ast.Node) bool {
		id, ok := n.(*ast.IdentLiteral)
		if !ok || found != nil {
			return found == nil
		}
		sym := r.a.info.Idents[id]
		if sym == nil || (sym.Kind != VAR && sym.Kind != PARAM) {
			return true
		}
		if _, known := r.a.staticVars[sym]; !known {
			found = id
		}
		return true
	})
	return found
}
