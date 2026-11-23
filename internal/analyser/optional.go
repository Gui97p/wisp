package analyser

import (
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/diag"
)

const (
	optionalValue   = "$value"
	optionalPresent = "$present"
)

var optionalStructs = map[string]*StructType{}

func (a *Analyser) optionalOf(node ast.Node, elem Type) Type {
	result, reason := tryOptionalOf(elem)
	if reason != "" {
		a.error(node, reason)
		return InvalidType{}
	}
	return result
}

func tryOptionalOf(elem Type) (Type, string) {
	switch t := elem.(type) {
	case InvalidType:
		return elem, ""
	case VoidType:
		return nil, "void cannot be optional"
	case ErrorType:
		return nil, "an Error is already null when there is no error: ?Error is not needed"
	case PointerType:
		if t.Nullable {
			return nil, "this pointer is already optional"
		}
		t.Nullable = true
		return t, ""
	case *FuncType:
		if t.Nullable {
			return nil, "this function type is already optional"
		}
		copied := *t
		copied.Nullable = true
		return &copied, ""
	case NamedType:
		if _, isFunc := t.Underlying.(*FuncType); isFunc {
			if t.Nullable {
				return nil, "this function type is already optional"
			}
			t.Nullable = true
			return t, ""
		}
	case *StructType:
		if t.Optional != nil {
			return nil, "this type is already optional"
		}
	}
	return optionalStruct(elem), ""
}

func optionalStruct(elem Type) *StructType {
	key := elem.String()
	if st, ok := optionalStructs[key]; ok && st.Optional.Equals(elem) && elem.Equals(st.Optional) {
		return st
	}
	st := &StructType{
		Name:     "?" + key,
		Fields:   map[string]Type{optionalValue: elem, optionalPresent: PrimitiveType{Name: "bool"}},
		Order:    []string{optionalValue, optionalPresent},
		Methods:  map[string]*FuncType{},
		Optional: elem,
	}
	optionalStructs[key] = st
	return st
}

func optionalElement(t Type) (Type, bool) {
	switch o := t.(type) {
	case *StructType:
		if o.Optional != nil {
			return o.Optional, true
		}
	case PointerType:
		if o.Nullable {
			o.Nullable = false
			return o, true
		}
	case *FuncType:
		if o.Nullable {
			copied := *o
			copied.Nullable = false
			return &copied, true
		}
	case NamedType:
		if _, isFunc := o.Underlying.(*FuncType); isFunc && o.Nullable {
			o.Nullable = false
			return o, true
		}
	}
	return nil, false
}

func isOptionalType(t Type) bool {
	_, ok := optionalElement(t)
	return ok
}

func isOptionalStruct(t Type) (*StructType, bool) {
	st, ok := t.(*StructType)
	if !ok || st.Optional == nil {
		return nil, false
	}
	return st, true
}

func undoWidening(lit *ast.StructLiteral) ast.Node {
	for i, key := range lit.Keys {
		if key == optionalValue {
			return lit.Values[i]
		}
	}
	null := &ast.NullLiteral{}
	line, col := lit.Position()
	null.SetPos(line, col)
	endLine, endCol := lit.EndPosition()
	null.SetEndPos(endLine, endCol)
	return null
}

func (a *Analyser) coerceOptional(slot *ast.Expression, from, to Type) (handled, ok bool) {
	if st, isStruct := isOptionalStruct(to); isStruct {
		if _, isNull := from.(NullType); isNull {
			if _, literal := (*slot).(*ast.NullLiteral); !literal {
				return false, false
			}
			a.widenAbsent(slot, st)
			return true, true
		}
		if other, same := isOptionalStruct(from); same {
			_ = other
			return false, false
		}
		if !a.coerce(slot, from, st.Optional) {
			return false, false
		}
		a.widenPresent(slot, st)
		return true, true
	}

	element, nullable := optionalElement(to)
	if !nullable {
		return false, false
	}
	if _, isNull := from.(NullType); isNull {
		return false, false
	}
	fp, fromPointer := from.(PointerType)
	tp, toPointer := element.(PointerType)
	switch {
	case fromPointer && toPointer && !fp.Nullable:
		_, fromVoid := fp.Element.(VoidType)
		_, toVoid := tp.Element.(VoidType)
		if fromVoid || toVoid || fp.Element.Equals(tp.Element) {
			a.insertCast(slot, to)
			return true, true
		}
	case !toPointer && from.Equals(element):
		a.insertCast(slot, to)
		return true, true
	}
	return false, false
}

func (a *Analyser) widenAbsent(slot *ast.Expression, st *StructType) {
	inner := *slot
	lit := &ast.StructLiteral{Implicit: true}
	line, col := inner.Position()
	lit.SetPos(line, col)
	endLine, endCol := inner.EndPosition()
	lit.SetEndPos(endLine, endCol)
	a.info.Types[lit] = st
	*slot = lit
}

func (a *Analyser) widenPresent(slot *ast.Expression, st *StructType) {
	inner := *slot
	line, col := inner.Position()
	endLine, endCol := inner.EndPosition()

	present := &ast.BoolLiteral{Value: true}
	present.SetPos(line, col)
	present.SetEndPos(endLine, endCol)
	a.info.Types[present] = PrimitiveType{Name: "bool"}

	lit := &ast.StructLiteral{Implicit: true}
	lit.Keys = []string{optionalValue, optionalPresent}
	lit.KeyAt = []ast.Position{{}, {}}
	lit.Values = []ast.Expression{inner, present}
	lit.SetPos(line, col)
	lit.SetEndPos(endLine, endCol)
	a.info.Types[lit] = st
	*slot = lit
}

func (a *Analyser) at(node ast.Node, built ast.Node) {
	line, col := node.Position()
	built.SetPos(line, col)
	endLine, endCol := node.EndPosition()
	built.SetEndPos(endLine, endCol)
}

func (a *Analyser) synthetic(node ast.Node, built ast.Node) {
	line, col := node.Position()
	built.SetPos(line, col)
	built.SetEndPos(line, col-1)
}

func (a *Analyser) place(node ast.Node, at ast.Position) {
	if at.Line == 0 {
		return
	}
	node.SetPos(at.Line, at.Col)
	node.SetEndPos(at.EndLine, at.EndCol)
}

func (a *Analyser) optionalMember(object ast.Expression, field string) *ast.MemberExpr {
	member := &ast.MemberExpr{Object: object, Field: field}
	a.at(object, member)
	return member
}

func (a *Analyser) boolLiteral(near ast.Node, value bool) *ast.BoolLiteral {
	lit := &ast.BoolLiteral{Value: value}
	a.at(near, lit)
	return lit
}

func (a *Analyser) negate(near ast.Node, value ast.Expression) *ast.UnaryExpr {
	not := &ast.UnaryExpr{Operator: "!", Value: value}
	a.at(near, not)
	return not
}

func (a *Analyser) rejectOptionalUse(node ast.Node, t Type, what string) bool {
	if !isOptionalType(t) {
		return false
	}
	a.errorf(node, "%s can be null, so it cannot be used for %s", t.String(), what).
		WithHelp("unwrap it first: `x ?? default`, `x!!` (stops if it is null) or `if x |v| { ... }`")
	return true
}

func (a *Analyser) checkOptionalComparison(expr *ast.BinaryExpr, left, right Type) (Type, bool) {
	leftOpt, leftIsOpt := isOptionalStruct(left)
	rightOpt, rightIsOpt := isOptionalStruct(right)
	_, leftNull := left.(NullType)
	_, rightNull := right.(NullType)

	switch {
	case leftIsOpt && rightNull:
		if !a.requireStored(expr, expr.Left) {
			return InvalidType{}, true
		}
		a.presenceTest(expr, expr.Left)
	case rightIsOpt && leftNull:
		if !a.requireStored(expr, expr.Right) {
			return InvalidType{}, true
		}
		a.presenceTest(expr, expr.Right)
	case leftIsOpt && rightIsOpt:
		if !leftOpt.Equals(rightOpt) {
			return InvalidType{}, false
		}
		if !a.requireStable(expr, expr.Left) || !a.requireStable(expr, expr.Right) {
			return InvalidType{}, true
		}
		a.pairTest(expr)
	case leftIsOpt && !rightNull:
		if !a.requireStable(expr, expr.Left) {
			return InvalidType{}, true
		}
		if !a.coerce(&expr.Right, right, leftOpt.Optional) {
			return InvalidType{}, false
		}
		a.valueTest(expr, expr.Left, expr.Right, false)
	case rightIsOpt && !leftNull:
		if !a.requireStable(expr, expr.Right) {
			return InvalidType{}, true
		}
		if !a.coerce(&expr.Left, left, rightOpt.Optional) {
			return InvalidType{}, false
		}
		a.valueTest(expr, expr.Right, expr.Left, true)
	case isOptionalType(left) && !leftIsOpt && !rightNull && !isOptionalType(right):
		if !a.coerce(&expr.Right, right, left) {
			return nil, false
		}
	case isOptionalType(right) && !rightIsOpt && !leftNull && !isOptionalType(left):
		if !a.coerce(&expr.Left, left, right) {
			return nil, false
		}
	default:
		return nil, false
	}
	return a.checkBinaryExpr(expr), true
}

func (a *Analyser) requireStable(expr *ast.BinaryExpr, operand ast.Expression) bool {
	if isAddressable(operand) {
		return true
	}
	a.error(expr, "comparing an optional with a value reads it twice: store the optional in a variable first").
		WithHelp("or compare with null, or unwrap it with ??")
	return false
}

func (a *Analyser) requireStored(expr *ast.BinaryExpr, operand ast.Expression) bool {
	if isAddressable(operand) {
		return true
	}
	a.error(expr, "a null test needs the optional in memory: store the result in a variable first").
		WithHelp("or use `if f() |v| { ... }` or `f() ?? default`")
	return false
}

func (a *Analyser) presenceTest(expr *ast.BinaryExpr, optional ast.Expression) {
	present := a.optionalMember(optional, optionalPresent)
	equal := expr.Operator == "!="
	expr.Left = present
	expr.Right = a.boolLiteral(expr, equal)
	expr.Operator = "=="
}

func (a *Analyser) valueTest(expr *ast.BinaryExpr, optional, value ast.Expression, valueFirst bool) {
	present := a.optionalMember(optional, optionalPresent)
	inner := a.optionalMember(optional, optionalValue)
	same := &ast.BinaryExpr{Operator: expr.Operator}
	a.at(expr, same)
	if valueFirst {
		same.Left, same.Right = value, inner
	} else {
		same.Left, same.Right = inner, value
	}
	if expr.Operator == "==" {
		expr.Left, expr.Operator, expr.Right = present, "&&", same
		return
	}
	expr.Left, expr.Operator, expr.Right = a.negate(expr, present), "||", same
}

func (a *Analyser) pairTest(expr *ast.BinaryExpr) {
	leftPresent := a.optionalMember(expr.Left, optionalPresent)
	rightPresent := a.optionalMember(expr.Right, optionalPresent)
	leftValue := a.optionalMember(expr.Left, optionalValue)
	rightValue := a.optionalMember(expr.Right, optionalValue)

	same := &ast.BinaryExpr{Left: leftPresent, Operator: "==", Right: rightPresent}
	a.at(expr, same)
	values := &ast.BinaryExpr{Left: leftValue, Operator: "==", Right: rightValue}
	a.at(expr, values)
	absent := a.negate(expr, a.optionalMember(expr.Left, optionalPresent))
	either := &ast.BinaryExpr{Left: absent, Operator: "||", Right: values}
	a.at(expr, either)
	both := &ast.BinaryExpr{Left: same, Operator: "&&", Right: either}
	a.at(expr, both)

	if expr.Operator == "==" {
		expr.Left, expr.Operator, expr.Right = both.Left, both.Operator, both.Right
		return
	}
	expr.Left, expr.Operator, expr.Right = a.negate(expr, both), "==", a.boolLiteral(expr, true)
}

func (a *Analyser) checkNeverNull(expr *ast.BinaryExpr, left, right Type) bool {
	_, leftNull := left.(NullType)
	_, rightNull := right.(NullType)
	other := left
	switch {
	case rightNull && !leftNull:
	case leftNull && !rightNull:
		other = right
	default:
		return false
	}
	if !canBeNullKind(other) || isOptionalType(other) {
		return false
	}
	a.errorf(expr, "%s is never null, so comparing it with null is always %t", other.String(), expr.Operator == "!=").
		WithHelp("write the type as ?%s if it can be null", other.String())
	return true
}

func canBeNullKind(t Type) bool {
	switch o := t.(type) {
	case PointerType, *FuncType:
		return true
	case NamedType:
		_, isFunc := o.Underlying.(*FuncType)
		return isFunc
	}
	return false
}

func (a *Analyser) checkOptionalCoalesce(expr *ast.CoalesceExpr, element Type) []Type {
	if expr.ErrorBind != "" {
		d := a.errorf(expr, "an optional has no error to bind: |%s| is only for calls that can fail", expr.ErrorBind)
		d.WithHelp("write `x ?? value`, `x ?? return` or a block that ends in yield")
		return []Type{InvalidType{}}
	}
	if !a.checkHandlerBlock(expr, []Type{element}) {
		return []Type{InvalidType{}}
	}
	return []Type{element}
}

func OptionalElement(t Type) (Type, bool) {
	return optionalElement(t)
}

func OptionalStruct(t Type) (*StructType, bool) {
	return isOptionalStruct(t)
}

func OptionalFields() (value, present string) {
	return optionalValue, optionalPresent
}

func (a *Analyser) copyIdent(src *ast.IdentLiteral) *ast.IdentLiteral {
	ident := &ast.IdentLiteral{Value: src.Value, Synthetic: true}
	a.at(src, ident)
	return ident
}

func (a *Analyser) lowerCapture(stmt *ast.IfStmt) {
	src := stmt.CaptureSrc
	name, at := stmt.Capture, stmt.CaptureAt
	stmt.Capture, stmt.CaptureSrc = "", nil

	sourceType := a.checkExpr(src)
	if _, bad := sourceType.(InvalidType); bad {
		return
	}
	if !isOptionalType(sourceType) {
		a.errorf(src, "|%s| needs an optional value (?T), got %s", name, sourceType.String()).
			WithHelp("a value that cannot be null is used directly")
		return
	}

	if _, isStruct := isOptionalStruct(sourceType); isStruct {
		stmt.Condition = a.optionalMember(a.copyIdent(src), optionalPresent)
	} else {
		test := &ast.BinaryExpr{Left: a.copyIdent(src), Operator: "!=", Right: &ast.NullLiteral{}}
		a.at(src, test)
		a.at(src, test.Right)
		stmt.Condition = test
	}

	unwrap := &ast.PropagateExpr{Value: a.copyIdent(src), Assert: true, Unchecked: true}
	a.at(src, unwrap)

	bind := &ast.VarStmt{}
	bind.Vars = append(bind.Vars, ast.Param{Name: name, Line: at.Line, Col: at.Col, EndLine: at.EndLine, EndCol: at.EndCol})
	bind.Values = append(bind.Values, unwrap)
	a.at(src, bind)
	stmt.Then.Statements = append([]ast.Statement{bind}, stmt.Then.Statements...)
}

func (a *Analyser) isPanicCall(expr ast.Expression) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	id, ok := call.Name.(*ast.IdentLiteral)
	if !ok || id.Value != "panic" {
		return false
	}
	sym, found := a.scope.Resolve("panic")
	return found && sym.Builtin && sym.Name == "panic"
}

func zeroProblem(t Type) string {
	switch o := t.(type) {
	case *InterfaceType:
		return "an interface value always holds something"
	case PointerType:
		if !o.Nullable {
			return t.String() + " cannot be null"
		}
	case *FuncType:
		if !o.Nullable {
			return "a function value cannot be null"
		}
	case NamedType:
		if _, isFunc := o.Underlying.(*FuncType); isFunc {
			if !o.Nullable {
				return o.Name + " cannot be null"
			}
			return ""
		}
		return zeroProblem(o.Underlying)
	case ArrayType:
		return zeroProblem(o.Element)
	case *StructType:
		if o.Optional != nil {
			return ""
		}
		if o.Sum != nil {
			if first := o.Sum.Variants[0]; first.Payload != nil {
				return zeroProblem(first.Payload)
			}
			return ""
		}
		for _, name := range o.Order {
			if why := zeroProblem(o.Fields[name]); why != "" {
				return "field " + name + " of " + o.Name + ": " + why
			}
		}
	}
	return ""
}

func (a *Analyser) requireZeroValue(node ast.Node, name string, t Type) {
	why := zeroProblem(t)
	if why == "" {
		return
	}
	a.errorf(node, "%s has no value to start with: %s", name, why).
		WithHelp("give it a value, or make the type optional (?T), whose first value is null")
}

func (a *Analyser) unifyOptional(left, right *ast.Expression, lt, rt Type) (Type, bool) {
	_, leftNull := lt.(NullType)
	_, rightNull := rt.(NullType)
	switch {
	case leftNull && rightNull:
		return nil, false
	case leftNull:
		if isOptionalType(rt) {
			return rt, a.coerce(left, lt, rt)
		}
		opt, reason := tryOptionalOf(defaultType(rt))
		if reason != "" || opt == nil {
			return nil, false
		}
		return opt, a.coerce(left, lt, opt) && a.coerce(right, rt, opt)
	case rightNull:
		if isOptionalType(lt) {
			return lt, a.coerce(right, rt, lt)
		}
		opt, reason := tryOptionalOf(defaultType(lt))
		if reason != "" || opt == nil {
			return nil, false
		}
		return opt, a.coerce(right, rt, opt) && a.coerce(left, lt, opt)
	}
	if isOptionalType(lt) && !isOptionalType(rt) {
		if element, _ := optionalElement(lt); element.Equals(rt) {
			return lt, a.coerce(right, rt, lt)
		}
	}
	if isOptionalType(rt) && !isOptionalType(lt) {
		if element, _ := optionalElement(rt); element.Equals(lt) {
			return rt, a.coerce(left, lt, rt)
		}
	}
	return nil, false
}

func mismatchHelp(got, want Type) string {
	if _, isNull := got.(NullType); isNull {
		if canBeNullKind(want) && !isOptionalType(want) {
			return "a " + want.String() + " is never null: write the type as ?" + want.String() + " if it can be absent"
		}
		return ""
	}
	if element, optional := optionalElement(got); optional && element.Equals(want) {
		return "the value can be null: unwrap it with `x ?? default`, `x!!` or `if x |v| { ... }`"
	}
	return ""
}

func (a *Analyser) helpMismatch(d *diag.Diagnostic, got, want Type) *diag.Diagnostic {
	if help := mismatchHelp(got, want); help != "" {
		d.WithHelp("%s", help)
	}
	return d
}

func (a *Analyser) argumentMismatch(expr ast.Node, i int, ft *FuncType, got Type) {
	d := a.errorf(expr, "argument %d from %s: expected %s, got %s", i+1, ft.label(), ft.Params[i].String(), got.String())
	if help := mismatchHelp(got, ft.Params[i]); help != "" {
		d.WithHelp("%s", help)
		return
	}
	d.WithHelp("%s", signatureHelp(ft))
}

func absentOnError(t Type) Type {
	if !canBeNullKind(t) || isOptionalType(t) {
		return t
	}
	if opt, reason := tryOptionalOf(t); reason == "" && opt != nil {
		return opt
	}
	return t
}

func (a *Analyser) checkBlockExpr(expr *ast.BlockExpr) Type {
	errorsBefore := a.errorCount()
	a.enterScope()
	frame := yieldFrame{infer: true, label: expr.Label}
	if a.blockHint != nil {
		frame.infer = false
		frame.values = []Type{a.blockHint}
		a.blockHint = nil
	}
	a.yieldFrames = append(a.yieldFrames, frame)
	a.checkBlock(expr.Block)
	frame = a.yieldFrames[len(a.yieldFrames)-1]
	a.yieldFrames = a.yieldFrames[:len(a.yieldFrames)-1]
	a.exitScope()

	if len(frame.values) == 0 {
		a.error(expr, "this block never yields a value").
			WithHelp("end it with `yield value;`")
		return InvalidType{}
	}
	if !a.blockDiverges(expr.Block) {
		a.error(expr, "the block must end with yield, return, break or continue")
		return InvalidType{}
	}
	if expr.Static && a.errorCount() == errorsBefore {
		a.runStatic(expr, frame.values[0])
	}
	return frame.values[0]
}

type coverage struct {
	enum    NamedType
	sum     *StructType
	covered map[int64]bool
	known   bool
}

func (c *coverage) missing() []string {
	var out []string
	if c.sum != nil {
		for _, v := range c.sum.Sum.Variants {
			if !c.covered[v.Tag] {
				out = append(out, "."+v.Name)
			}
		}
		return out
	}
	for _, name := range c.enum.MemberOrder {
		if !c.covered[c.enum.Members[name]] {
			out = append(out, "."+name)
		}
	}
	return out
}

func (a *Analyser) checkExhaustive(head *ast.IfStmt) {
	if !head.IsSwitch || head.SwitchDefault {
		return
	}
	c := &coverage{covered: map[int64]bool{}}
	for cur := head; cur != nil; {
		a.collectCovered(cur.Condition, c)
		next, _ := cur.Else.(*ast.BlockStmt)
		cur = nil
		if next != nil && len(next.Statements) == 1 {
			if link, ok := next.Statements[0].(*ast.IfStmt); ok && link.SwitchLink {
				cur = link
			}
		}
	}
	if !c.known {
		return
	}
	missing := c.missing()
	if len(missing) == 0 {
		return
	}
	a.errorRange(switchKeyword(head), "this switch does not cover %s", strings.Join(missing, ", ")).
		WithHelp("add a case for each, or a default if the rest does not matter")
}

func (a *Analyser) coversEverything(arms []*ast.SwitchArm) bool {
	c := &coverage{covered: map[int64]bool{}}
	for _, arm := range arms {
		if arm.Condition != nil {
			a.collectCovered(arm.Condition, c)
		}
	}
	return c.known && len(c.missing()) == 0
}

func (a *Analyser) collectCovered(cond ast.Expression, c *coverage) {
	bin, ok := cond.(*ast.BinaryExpr)
	if !ok {
		return
	}
	if test := a.info.SumTests[bin]; test != nil {
		c.sum, c.known = test.Sum, true
		c.covered[test.Variant.Tag] = true
		return
	}
	if member, isMember := bin.Left.(*ast.MemberExpr); isMember && member.Field == sumTag && bin.Operator == "==" {
		if tag, isTag := bin.Right.(*ast.IntLiteral); isTag {
			if sum, ok := sumOf(a.info.Types[member.Object]); ok {
				c.sum, c.known = sum, true
				c.covered[tag.Value] = true
			}
		}
		return
	}
	switch bin.Operator {
	case "||":
		a.collectCovered(bin.Left, c)
		a.collectCovered(bin.Right, c)
	case "==":
		named, isNamed := a.info.Types[bin.Left].(NamedType)
		if !isNamed || !named.Enum {
			return
		}
		c.enum, c.known = named, true
		if value, ok := a.info.EnumValues[bin.Right]; ok {
			c.covered[value.Value] = true
		}
	}
}

func (a *Analyser) sumBinding(test *SumTest, bind *ast.TypeBind) ast.Statement {
	if test.Variant.Payload == nil {
		line, col := bind.Line, bind.Col
		a.errors.Add(a.currentFile, line, col, bind.EndLine, bind.EndCol, "variant %s has no data to name", test.Variant.Name)
		return nil
	}
	subject := test.Subject
	if id, ok := subject.(*ast.IdentLiteral); ok {
		subject = a.copyIdent(id)
	}
	var value ast.Expression = a.optionalMember(subject, test.Variant.Field)
	if _, isPointer := a.info.Types[test.Subject].(PointerType); isPointer {
		ref := &ast.UnaryExpr{Operator: "&", Value: value}
		a.at(test.Subject, ref)
		value = ref
	}
	decl := &ast.VarStmt{}
	decl.Vars = append(decl.Vars, ast.Param{Name: bind.Name, Line: bind.Line, Col: bind.Col, EndLine: bind.EndLine, EndCol: bind.EndCol})
	decl.Values = append(decl.Values, value)
	a.at(test.Subject, decl)
	return decl
}

func (a *Analyser) bindSumCase(stmt *ast.IfStmt) {
	bin, ok := stmt.Condition.(*ast.BinaryExpr)
	if !ok || bin.Bind == nil {
		return
	}
	test := a.info.SumTests[bin]
	if test == nil {
		return
	}
	bind := bin.Bind
	bin.Bind = nil
	if decl := a.sumBinding(test, bind); decl != nil {
		stmt.Then.Statements = append([]ast.Statement{decl}, stmt.Then.Statements...)
	}
}

func (a *Analyser) bindSumArm(arm *ast.SwitchArm) {
	bin, ok := arm.Condition.(*ast.BinaryExpr)
	if !ok || bin.Bind == nil {
		return
	}
	test := a.info.SumTests[bin]
	if test == nil {
		return
	}
	bind := bin.Bind
	bin.Bind = nil
	decl := a.sumBinding(test, bind)
	if decl == nil {
		return
	}
	if arm.Exit != nil {
		block := &ast.BlockStmt{Statements: []ast.Statement{decl, arm.Exit}}
		a.at(arm, block)
		arm.Exit = block
		return
	}
	if len(arm.Values) != 1 {
		a.errorf(arm, "a case that names its data gives exactly one value, got %d", len(arm.Values))
		return
	}
	give := &ast.YieldStmt{Values: arm.Values}
	a.at(arm, give)
	body := &ast.BlockStmt{Statements: []ast.Statement{decl, give}}
	a.at(arm, body)
	block := &ast.BlockExpr{Block: body}
	a.at(arm, block)
	arm.Values = []ast.Expression{block}
}

func (a *Analyser) typeArgument(arg ast.Expression) (Type, bool) {
	switch e := arg.(type) {
	case *ast.IdentLiteral:
		if sym, found := a.scope.Resolve(e.Value); found {
			if sym.Kind == STRUCT || sym.Kind == TYPE {
				sym.Used = true
				return sym.Type, true
			}
		} else if primitives[e.Value] {
			return PrimitiveType{Name: e.Value}, true
		}
	case *ast.MemberExpr:
		if module, isIdent := e.Object.(*ast.IdentLiteral); isIdent {
			if sym, found := a.lookupType(a.scope, module.Value, e.Field); found && (sym.Kind == STRUCT || sym.Kind == TYPE) {
				return sym.Type, true
			}
		}
	}
	return nil, false
}

func (a *Analyser) checkWidthBuiltin(expr *ast.CallExpr, name string) Type {
	if len(expr.Args) != 2 {
		a.errorf(expr, "%s expects a type and a value, got %d arguments", name, len(expr.Args))
		a.evalArgTypes(expr)
		return InvalidType{}
	}

	target, ok := a.typeArgument(expr.Args[0])
	if !ok || !isInteger(underlyingOf(target)) {
		a.errorf(expr.Args[0], "%s takes an integer type first, as in %s(byte, x)", name, name)
		a.checkExpr(expr.Args[1])
		return InvalidType{}
	}

	from := a.checkExpr(expr.Args[1])
	if _, bad := from.(InvalidType); bad {
		return target
	}
	if !isInteger(underlyingOf(from)) && !(name == "saturate" && isFloat(underlyingOf(from))) {
		a.errorf(expr.Args[1], "%s cannot take a %s", name, from.String()).
			WithHelp("it takes an integer%s", map[bool]string{true: " or a floating point number", false: ""}[name == "saturate"])
		return target
	}
	switch from.(type) {
	case UntypedIntType:
		a.fixUntyped(expr.Args[1], PrimitiveType{Name: "int"})
	case UntypedFloatType:
		a.fixUntyped(expr.Args[1], PrimitiveType{Name: "float"})
	}

	cast := &ast.CastExpr{Value: expr.Args[1], Truncate: name == "truncate", Saturate: name == "saturate"}
	a.at(expr, cast)
	a.info.Types[cast] = target
	expr.Cast = cast
	return target
}

func switchKeyword(node ast.Node) ast.Position {
	line, col := node.Position()
	return ast.Position{Line: line, Col: col, EndLine: line, EndCol: col + len("switch") - 1}
}
