package analyser

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/ast"
)

func (a *Analyser) pristine(expr ast.Expression) ast.Expression {
	var options ast.CloneOptions
	options = ast.CloneOptions{
		Replace: func(node ast.Node) (ast.Node, bool) {
			switch n := node.(type) {
			case *ast.CastExpr:
				if n.Implicit {
					return cloneUnwrapped(n.Value, options), true
				}
			case *ast.StructLiteral:
				if n.Implicit {
					return cloneUnwrapped(undoWidening(n), options), true
				}
			case *ast.CallExpr:
				if n.Generic != nil {
					restored := &ast.CallExpr{Name: cloneUnwrapped(n.Generic, options).(ast.Expression), Spread: n.Spread, Range: n.Range}
					for _, arg := range n.Args {
						restored.Args = append(restored.Args, cloneUnwrapped(arg, options).(ast.Expression))
					}
					line, col := n.Position()
					endLine, endCol := n.EndPosition()
					restored.SetPos(line, col)
					restored.SetEndPos(endLine, endCol)
					return restored, true
				}
			}
			return nil, false
		},
	}
	return ast.Clone(expr, options)
}

func isPureOperand(expr ast.Expression) bool {
	switch e := expr.(type) {
	case *ast.IdentLiteral, *ast.IntLiteral, *ast.FloatLiteral, *ast.StringLiteral, *ast.BoolLiteral, *ast.CharLiteral:
		return true
	case *ast.MemberExpr:
		return isPureOperand(e.Object)
	}
	return false
}

func structValue(t Type) (*StructType, bool) {
	st, ok := definedBase(t).(*StructType)
	if !ok || st.Optional != nil || st.Sum != nil {
		return nil, false
	}
	return st, true
}

func operatorMethod(t Type, name string) *FuncType {
	if _, ok := structValue(t); !ok {
		return nil
	}
	return MethodsOf(t)[name]
}

func hasOperatorMethod(t Type, name string, result func(*FuncType) bool) bool {
	ft := operatorMethod(t, name)
	return ft != nil && !ft.Fallible && len(ft.Params) == 1 && ft.Params[0].Equals(t) && len(ft.Returns) == 1 && result(ft)
}

func returnsBool(ft *FuncType) bool {
	return ft.Returns[0].Equals(PrimitiveType{Name: "bool"})
}

func (a *Analyser) lowerOperator(expr *ast.BinaryExpr, left, right Type) (Type, bool) {
	var method string
	switch expr.Operator {
	case "==", "!=":
		method = "eq"
	case "<", "<=", ">", ">=":
		method = "less"
	default:
		return nil, false
	}
	ft := operatorMethod(left, method)
	if ft == nil || !left.Equals(right) {
		return nil, false
	}
	if !hasOperatorMethod(left, method, returnsBool) {
		a.errorf(expr, "%s of %s has to be func (%s a) %s(%s b) bool", method, left.String(), left.String(), method, left.String())
		return InvalidType{}, true
	}

	lhs, rhs := a.pristine(expr.Left), a.pristine(expr.Right)
	at := func(node ast.Node) { a.at(expr, node) }
	call := func(receiver, argument ast.Expression) ast.Expression {
		name := &ast.MemberExpr{Object: receiver, Field: method}
		at(name)
		node := &ast.CallExpr{Name: name, Args: []ast.Expression{argument}}
		at(node)
		return node
	}
	negate := func(value ast.Expression) ast.Expression {
		node := &ast.UnaryExpr{Operator: "!", Value: value}
		at(node)
		return node
	}

	var lowered ast.Expression
	switch expr.Operator {
	case "==":
		lowered = call(lhs, rhs)
	case "!=":
		lowered = negate(call(lhs, rhs))
	case "<":
		lowered = call(lhs, rhs)
	case ">=":
		lowered = negate(call(lhs, rhs))
	case ">", "<=":
		swapped := func(l, r ast.Expression) ast.Expression {
			if expr.Operator == ">" {
				return call(r, l)
			}
			return negate(call(r, l))
		}
		if isPureOperand(lhs) && isPureOperand(rhs) {
			lowered = swapped(lhs, rhs)
			break
		}
		id := a.iterCount
		a.iterCount++
		leftName, rightName := fmt.Sprintf("__wisp_lhs_%d", id), fmt.Sprintf("__wisp_rhs_%d", id)
		ident := func(name string) ast.Expression {
			node := &ast.IdentLiteral{Value: name, Synthetic: true}
			at(node)
			return node
		}
		declare := func(name string, value ast.Expression) ast.Statement {
			decl := &ast.VarStmt{Vars: []ast.Param{{Name: name}}, Values: []ast.Expression{value}}
			at(decl)
			return decl
		}
		yield := &ast.YieldStmt{Values: []ast.Expression{swapped(ident(leftName), ident(rightName))}}
		at(yield)
		block := &ast.BlockStmt{Statements: []ast.Statement{declare(leftName, lhs), declare(rightName, rhs), yield}}
		at(block)
		wrapper := &ast.BlockExpr{Block: block}
		at(wrapper)
		lowered = wrapper
	}

	result := a.checkLowered(lowered)
	expr.Lowered = lowered
	return result, true
}

func (a *Analyser) checkLowered(lowered ast.Expression) Type {
	saved := a.waiveAccess
	if a.instanceContext != nil {
		a.waiveAccess = true
	}
	result := a.checkExpr(lowered)
	a.waiveAccess = saved
	return result
}

func (a *Analyser) checkOperatorMethodDecl(fd *ast.FuncDecl, recv Type) {
	if fd.Receiver == nil || (fd.Name != "eq" && fd.Name != "less") {
		return
	}
	target := recv
	if pointer, ok := recv.(PointerType); ok {
		target = pointer.Element
	}
	operator := "=="
	if fd.Name == "less" {
		operator = "<"
	}
	builtin := false
	switch base := definedBase(target).(type) {
	case PrimitiveType:
		builtin = fd.Name == "eq" || base.Name != "bool"
	case PointerType:
		builtin = fd.Name == "eq"
	case NamedType:
		builtin = base.Enum && fd.Name == "eq"
	}
	if builtin {
		a.errorAtName(fd, "%s already has %s, so %s would never be called: give the method another name", target.String(), operator, fd.Name)
	}
}

func (a *Analyser) lowerIn(expr *ast.InExpr, element Type) (Type, bool) {
	if _, isStruct := structValue(element); !isStruct {
		return nil, false
	}
	if !hasOperatorMethod(element, "eq", returnsBool) {
		a.errorf(expr, "'in' compares the elements with ==, and %s has no eq", element.String()).
			WithHelp("define func (%s a) eq(%s b) bool", element.String(), element.String())
		return PrimitiveType{Name: "bool"}, true
	}

	id := a.iterCount
	a.iterCount++
	foundName, itemName, wantName := fmt.Sprintf("__wisp_found_%d", id), fmt.Sprintf("__wisp_item_%d", id), fmt.Sprintf("__wisp_want_%d", id)
	at := func(node ast.Node) { a.at(expr, node) }
	ident := func(name string) ast.Expression {
		node := &ast.IdentLiteral{Value: name, Synthetic: true}
		at(node)
		return node
	}
	declare := func(name string, value ast.Expression) ast.Statement {
		decl := &ast.VarStmt{Vars: []ast.Param{{Name: name}}, Values: []ast.Expression{value}}
		at(decl)
		return decl
	}
	truth := func(value bool) ast.Expression {
		node := &ast.BoolLiteral{Value: value}
		at(node)
		return node
	}

	collection := a.pristine(expr.Right)
	statements := []ast.Statement{declare(foundName, truth(false)), declare(wantName, a.pristine(expr.Left))}
	if !isPureOperand(collection) {
		collectionName := fmt.Sprintf("__wisp_set_%d", id)
		statements = append(statements, declare(collectionName, collection))
		collection = ident(collectionName)
	}

	same := &ast.BinaryExpr{Left: ident(itemName), Operator: "==", Right: ident(wantName)}
	at(same)
	found := &ast.AssignStmt{Targets: []ast.Expression{ident(foundName)}, Op: "=", Values: []ast.Expression{truth(true)}}
	at(found)
	leave := &ast.BreakStmt{}
	at(leave)
	then := &ast.BlockStmt{Statements: []ast.Statement{found, leave}}
	at(then)
	test := &ast.IfStmt{Condition: same, Then: then}
	at(test)
	body := &ast.BlockStmt{Statements: []ast.Statement{test}}
	at(body)
	loop := &ast.ForStmt{Var: itemName, Range: collection, Body: body}
	at(loop)
	yield := &ast.YieldStmt{Values: []ast.Expression{ident(foundName)}}
	at(yield)

	statements = append(statements, loop, yield)
	block := &ast.BlockStmt{Statements: statements}
	at(block)
	wrapper := &ast.BlockExpr{Block: block}
	at(wrapper)

	result := a.checkLowered(wrapper)
	expr.Lowered = wrapper
	return result, true
}

func hookedOperator(expr ast.Expression) ast.Expression {
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		if e.Lowered != nil {
			return e
		}
		if found := hookedOperator(e.Left); found != nil {
			return found
		}
		return hookedOperator(e.Right)
	case *ast.UnaryExpr:
		return hookedOperator(e.Value)
	case *ast.InExpr:
		if e.Lowered != nil {
			return e
		}
	}
	return nil
}
