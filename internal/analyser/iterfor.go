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
	if next == nil || len(next.Params) != 0 || len(next.Returns) != 1 {
		return nil, false
	}
	if !isOptionalType(next.Returns[0]) {
		return nil, false
	}
	return next, true
}

func (a *Analyser) lowerIteratorFor(stmt *ast.ForStmt, rangeType Type) bool {
	if MethodsOf(rangeType) == nil {
		return false
	}

	source := stmt.Range
	direct := true
	if _, ok := iteratorNext(rangeType); !ok {
		if next := MethodsOf(rangeType)["next"]; next != nil {
			if _, fallible := next.Returns[0].(FallibleType); len(next.Returns) == 1 && fallible {
				a.errorf(stmt.Range, "next of %s can fail, so `for` cannot walk it", rangeType.String()).
					WithHelp("use loop value.next()! |item| { ... } for an iterator that can fail")
				return false
			}
		}
		iter := MethodsOf(rangeType)["iter"]
		if iter == nil || len(iter.Params) != 0 || len(iter.Returns) != 1 {
			return false
		}
		if _, ok := iteratorNext(iter.Returns[0]); !ok {
			return false
		}
		call := &ast.CallExpr{Name: &ast.MemberExpr{Object: source, Field: "iter"}}
		a.at(source, call)
		a.at(source, call.Name)
		source = call
		direct = false
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
		a.at(stmt, id)
		return id
	}
	declare := func(name string, value ast.Expression) *ast.VarStmt {
		decl := &ast.VarStmt{Vars: []ast.Param{{Name: name}}, Values: []ast.Expression{value}}
		a.at(stmt, decl)
		return decl
	}

	held := source
	if _, isPointer := rangeType.(PointerType); direct && !isPointer && isAddressable(source) {
		held = &ast.UnaryExpr{Operator: "&", Value: source}
		a.at(source, held)
	}

	nextCall := &ast.CallExpr{Name: &ast.MemberExpr{Object: ident(holder), Field: "next"}}
	a.at(stmt, nextCall)
	a.at(stmt, nextCall.Name)

	body := stmt.Body
	if indexName != "" && indexName != "_" {
		advance := &ast.IncDecStmt{Target: ident(counter), Op: "++"}
		a.at(stmt, advance)
		index := declare(indexName, ident(counter))
		index.Vars[0].Line, index.Vars[0].Col = indexAt.Line, indexAt.Col
		index.Vars[0].EndLine, index.Vars[0].EndCol = indexAt.EndLine, indexAt.EndCol
		body.Statements = append([]ast.Statement{index, advance}, body.Statements...)
	}

	item := ident(capture)
	test := &ast.IfStmt{Condition: item, Capture: elemName, CaptureAt: elemAt, CaptureSrc: item, Then: body}
	a.at(stmt, test)
	leave := &ast.BreakStmt{}
	a.at(stmt, leave)
	otherwise := &ast.BlockStmt{Statements: []ast.Statement{leave}}
	a.at(stmt, otherwise)
	test.Else = otherwise

	step := &ast.BlockStmt{Statements: []ast.Statement{declare(capture, nextCall), test}}
	a.at(stmt, step)
	loopBody := &ast.BlockStmt{Statements: []ast.Statement{step}}
	a.at(stmt, loopBody)
	loop := &ast.LoopStmt{Label: stmt.Label, Body: loopBody}
	a.at(stmt, loop)

	statements := []ast.Statement{declare(holder, held)}
	if indexName != "" && indexName != "_" {
		zero := &ast.IntLiteral{Value: 0}
		a.at(stmt, zero)
		statements = append(statements, declare(counter, zero))
	}
	statements = append(statements, loop)
	outer := &ast.BlockStmt{Statements: statements}
	a.at(stmt, outer)
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
		a.at(stmt, node)
		return node
	}
	collection := func() ast.Expression {
		node := &ast.DerefExpr{Value: ident(holder)}
		a.at(stmt, node)
		return node
	}

	body := stmt.Body
	if elemName != "_" {
		slot := &ast.IndexExpr{Array: collection(), Index: ident(indexName)}
		a.at(stmt, slot)
		address := &ast.UnaryExpr{Operator: "&", Value: slot}
		a.at(stmt, address)
		bind := &ast.VarStmt{Vars: []ast.Param{{Name: elemName, Line: elemAt.Line, Col: elemAt.Col, EndLine: elemAt.EndLine, EndCol: elemAt.EndCol}}, Values: []ast.Expression{address}}
		a.at(stmt, bind)
		body.Statements = append([]ast.Statement{bind}, body.Statements...)
	}

	size := &ast.CallExpr{Name: &ast.IdentLiteral{Value: "len"}, Args: []ast.Expression{collection()}}
	a.at(stmt, size)
	a.at(stmt, size.Name)
	one := &ast.IntLiteral{Value: 1}
	a.at(stmt, one)
	last := &ast.BinaryExpr{Left: size, Operator: "-", Right: one}
	a.at(stmt, last)
	zero := &ast.IntLiteral{Value: 0}
	a.at(stmt, zero)
	loop := &ast.ForStmt{Label: stmt.Label, Var: indexName, VarAt: indexAt, Start: zero, End: last, Body: body}
	a.at(stmt, loop)

	hold := &ast.VarStmt{Vars: []ast.Param{{Name: holder}}, Values: []ast.Expression{stmt.Range}}
	a.at(stmt, hold)
	outer := &ast.BlockStmt{Statements: []ast.Statement{hold, loop}}
	a.at(stmt, outer)
	stmt.Lowered = outer
	return true
}
