package analyser

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/ast"
)

func (a *Analyser) elementOnly(stmt *ast.ForStmt) {
	if stmt.Var2 != "" {
		return
	}
	stmt.Var2, stmt.Var2At = stmt.Var, stmt.VarAt
	stmt.Var, stmt.VarAt = "_", ast.Position{}
}

func iteratorNext(t Type) (*FuncType, bool) {
	next := MethodsOf(t)["next"]
	if next == nil || next.Fallible || len(next.Params) != 0 || len(next.Returns) != 1 {
		return nil, false
	}
	if !isOptionalType(next.Returns[0]) {
		return nil, false
	}
	return next, true
}

func failingNext(t Type) bool {
	next := MethodsOf(t)["next"]
	if next == nil || !next.Fallible || len(next.Params) != 0 || len(next.Returns) != 1 {
		return false
	}
	return isOptionalType(next.Returns[0])
}

func walkable(t Type) bool {
	if _, ok := iteratorNext(t); ok {
		return true
	}
	return failingNext(t)
}

func (a *Analyser) iterates(t Type) bool {
	if MethodsOf(t) == nil {
		return false
	}
	if walkable(t) {
		return true
	}
	iter := MethodsOf(t)["iter"]
	return iter != nil && len(iter.Params) == 0 && len(iter.Returns) == 1 && walkable(iter.Returns[0])
}

func (a *Analyser) emptyLowering(stmt *ast.ForStmt) {
	empty := &ast.BlockStmt{}
	a.synthetic(stmt, empty)
	stmt.Lowered = empty
}

func (a *Analyser) iterableFails(t Type) bool {
	if MethodsOf(t) == nil {
		return false
	}
	if walkable(t) {
		return failingNext(t)
	}
	iter := MethodsOf(t)["iter"]
	if iter == nil || len(iter.Params) != 0 || len(iter.Returns) != 1 {
		return false
	}
	return failingNext(iter.Returns[0])
}

func (a *Analyser) lowerIteratorFor(stmt *ast.ForStmt, rangeType Type, source ast.Expression, failing *ast.PropagateExpr) bool {
	if MethodsOf(rangeType) == nil {
		return false
	}

	direct := true
	iterType := rangeType
	if !walkable(rangeType) {
		iter := MethodsOf(rangeType)["iter"]
		if iter == nil || len(iter.Params) != 0 || len(iter.Returns) != 1 {
			return false
		}
		if !walkable(iter.Returns[0]) {
			return false
		}
		iterType = iter.Returns[0]
		call := &ast.CallExpr{Name: &ast.MemberExpr{Object: source, Field: "iter"}}
		a.at(source, call)
		a.at(source, call.Name)
		source = call
		direct = false
	}

	fails := failingNext(iterType)
	switch {
	case fails && failing == nil:
		a.errorf(stmt.Range, "next of %s can fail, so `for` needs a `!` after it", rangeType.String()).
			WithHelp("write for item in value! { ... }, or loop value.next()! |item| { ... }")
		a.emptyLowering(stmt)
		return true
	case !fails && failing != nil:
		a.errorf(failing, "next of %s cannot fail, so there is nothing for `!` to check", rangeType.String()).
			WithHelp("remove the `!`")
		a.emptyLowering(stmt)
		return true
	}

	id := a.iterCount
	a.iterCount++
	holder := fmt.Sprintf("__wisp_iter_%d", id)
	capture := fmt.Sprintf("__wisp_item_%d", id)
	counter := fmt.Sprintf("__wisp_index_%d", id)

	elemName, elemAt, indexName, indexAt := stmt.Var, stmt.VarAt, "", ast.Position{}
	if stmt.Var2 != "" {
		indexName, indexAt = stmt.Var, stmt.VarAt
		elemName, elemAt = stmt.Var2, stmt.Var2At
	}

	ident := func(name string) *ast.IdentLiteral {
		id := &ast.IdentLiteral{Value: name, Synthetic: true}
		a.synthetic(stmt, id)
		return id
	}
	declare := func(name string, value ast.Expression) *ast.VarStmt {
		decl := &ast.VarStmt{Vars: []ast.Param{{Name: name}}, Values: []ast.Expression{value}}
		a.synthetic(stmt, decl)
		return decl
	}

	held := source
	if _, isPointer := rangeType.(PointerType); direct && !isPointer && isAddressable(source) {
		held = &ast.UnaryExpr{Operator: "&", Value: source}
		a.at(source, held)
	}

	nextCall := &ast.CallExpr{Name: &ast.MemberExpr{Object: ident(holder), Field: "next"}}
	a.synthetic(stmt, nextCall)
	a.synthetic(stmt, nextCall.Name)
	var nextValue ast.Expression = nextCall
	if failing != nil {
		checked := &ast.PropagateExpr{Value: nextCall, Assert: failing.Assert}
		a.at(failing, checked)
		nextValue = checked
	}

	body := stmt.Body
	if indexName != "" && indexName != "_" {
		advance := &ast.IncDecStmt{Target: ident(counter), Op: "++"}
		a.synthetic(stmt, advance)
		index := declare(indexName, ident(counter))
		index.Vars[0].Line, index.Vars[0].Col = indexAt.Line, indexAt.Col
		index.Vars[0].EndLine, index.Vars[0].EndCol = indexAt.EndLine, indexAt.EndCol
		a.place(index, indexAt)
		body.Statements = append([]ast.Statement{index, advance}, body.Statements...)
	}

	item := ident(capture)
	a.place(item, elemAt)
	test := &ast.IfStmt{Condition: item, Capture: elemName, CaptureAt: elemAt, CaptureSrc: item, Then: body}
	a.synthetic(stmt, test)
	leave := &ast.BreakStmt{}
	a.synthetic(stmt, leave)
	otherwise := &ast.BlockStmt{Statements: []ast.Statement{leave}}
	a.synthetic(stmt, otherwise)
	test.Else = otherwise

	step := &ast.BlockStmt{Statements: []ast.Statement{declare(capture, nextValue), test}}
	a.synthetic(stmt, step)
	loopBody := &ast.BlockStmt{Statements: []ast.Statement{step}}
	a.synthetic(stmt, loopBody)
	loop := &ast.LoopStmt{Label: stmt.Label, Body: loopBody}
	a.synthetic(stmt, loop)

	statements := []ast.Statement{declare(holder, held)}
	if indexName != "" && indexName != "_" {
		zero := &ast.IntLiteral{Value: 0}
		a.synthetic(stmt, zero)
		statements = append(statements, declare(counter, zero))
	}
	statements = append(statements, loop)
	outer := &ast.BlockStmt{Statements: statements}
	a.synthetic(stmt, outer)
	stmt.Lowered = outer
	return true
}

func (a *Analyser) lowerPointerFor(stmt *ast.ForStmt, rangeType Type) bool {
	pointer, ok := rangeType.(PointerType)
	if !ok || pointer.Nullable {
		return false
	}
	switch underlyingOf(pointer.Element).(type) {
	case ArrayType, SpanType:
	default:
		return false
	}

	id := a.iterCount
	a.iterCount++
	holder := fmt.Sprintf("__wisp_walk_%d", id)
	hiddenIndex := fmt.Sprintf("__wisp_slot_%d", id)

	elemName, elemAt, indexName, indexAt := stmt.Var, stmt.VarAt, hiddenIndex, ast.Position{}
	if stmt.Var2 != "" {
		indexName, indexAt = stmt.Var, stmt.VarAt
		elemName, elemAt = stmt.Var2, stmt.Var2At
	}

	ident := func(name string) *ast.IdentLiteral {
		node := &ast.IdentLiteral{Value: name, Synthetic: true}
		a.synthetic(stmt, node)
		return node
	}
	collection := func() ast.Expression {
		node := &ast.DerefExpr{Value: ident(holder)}
		a.synthetic(stmt, node)
		return node
	}

	body := stmt.Body
	if elemName != "_" {
		slot := &ast.IndexExpr{Array: collection(), Index: ident(indexName)}
		a.synthetic(stmt, slot)
		address := &ast.UnaryExpr{Operator: "&", Value: slot}
		a.synthetic(stmt, address)
		bind := &ast.VarStmt{Vars: []ast.Param{{Name: elemName, Line: elemAt.Line, Col: elemAt.Col, EndLine: elemAt.EndLine, EndCol: elemAt.EndCol}}, Values: []ast.Expression{address}}
		a.synthetic(stmt, bind)
		a.place(bind, elemAt)
		body.Statements = append([]ast.Statement{bind}, body.Statements...)
	}

	size := &ast.CallExpr{Name: &ast.IdentLiteral{Value: "len"}, Args: []ast.Expression{collection()}}
	a.synthetic(stmt, size)
	a.synthetic(stmt, size.Name)
	one := &ast.IntLiteral{Value: 1}
	a.synthetic(stmt, one)
	last := &ast.BinaryExpr{Left: size, Operator: "-", Right: one}
	a.synthetic(stmt, last)
	zero := &ast.IntLiteral{Value: 0}
	a.synthetic(stmt, zero)
	loop := &ast.ForStmt{Label: stmt.Label, Var: indexName, VarAt: indexAt, Start: zero, End: last, Body: body}
	a.synthetic(stmt, loop)

	hold := &ast.VarStmt{Vars: []ast.Param{{Name: holder}}, Values: []ast.Expression{stmt.Range}}
	a.synthetic(stmt, hold)
	outer := &ast.BlockStmt{Statements: []ast.Statement{hold, loop}}
	a.synthetic(stmt, outer)
	stmt.Lowered = outer
	return true
}
