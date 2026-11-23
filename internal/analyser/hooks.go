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

	result := a.checkExpr(lowered)
	expr.Lowered = lowered
	return result, true
}

func hookReceiver(t Type) (Type, bool) {
	if _, ok := structValue(t); !ok {
		return nil, false
	}
	return t, true
}

func (a *Analyser) checkOperatorMethodDecl(fd *ast.FuncDecl, recv Type) {
	if fd.Receiver == nil {
		return
	}
	var operator string
	switch fd.Name {
	case "eq":
		operator = "=="
	case "less":
		operator = "<"
	case "index":
		operator = "[]"
	case "slice":
		operator = "[:]"
	default:
		return
	}
	target := recv
	if pointer, ok := recv.(PointerType); ok {
		target = pointer.Element
	}
	switch base := definedBase(target).(type) {
	case PrimitiveType:
		if fd.Name == "index" || fd.Name == "slice" {
			if base.Name != "string" {
				return
			}
		} else if fd.Name == "less" && base.Name == "bool" {
			return
		}
	case PointerType:
		if fd.Name == "less" {
			return
		}
	case NamedType:
		if !base.Enum || fd.Name != "eq" {
			return
		}
	case ArrayType, SpanType:
		if fd.Name == "eq" || fd.Name == "less" {
			return
		}
	default:
		return
	}
	a.errorAtName(fd, "%s already has %s, so %s would never be called: give the method another name", target.String(), operator, fd.Name)
}

func validIndexMethod(ft *FuncType) bool {
	if ft.Fallible || len(ft.Params) != 1 || len(ft.Returns) != 1 {
		return false
	}
	pointer, ok := ft.Returns[0].(PointerType)
	return ok && !pointer.Nullable
}

func (a *Analyser) lowerIndex(expr *ast.IndexExpr, arrType Type) (Type, bool) {
	base, ok := hookReceiver(arrType)
	if !ok {
		return nil, false
	}
	ft := MethodsOf(base)["index"]
	if ft == nil {
		return nil, false
	}
	if !validIndexMethod(ft) {
		a.errorf(expr, "index of %s has to be func (*%s x) index(position) *Element, with one parameter and a pointer that is never null", base.String(), base.String())
		return InvalidType{}, true
	}

	receiver, position := a.pristine(expr.Array), a.pristine(expr.Index)
	name := &ast.MemberExpr{Object: receiver, Field: "index"}
	a.at(expr, name)
	call := &ast.CallExpr{Name: name, Args: []ast.Expression{position}}
	a.at(expr, call)
	deref := &ast.DerefExpr{Value: call}
	a.at(expr, deref)

	result := a.checkExpr(deref)
	expr.Lowered = deref
	return result, true
}

func (a *Analyser) lowerSlice(expr *ast.SliceExpr, arrType Type) (Type, bool) {
	base, ok := hookReceiver(arrType)
	if !ok {
		return nil, false
	}
	ft := MethodsOf(base)["slice"]
	if ft == nil {
		return nil, false
	}
	if ft.Fallible || len(ft.Params) != 2 || len(ft.Returns) != 1 || !isInteger(underlyingOf(ft.Params[0])) || !isInteger(underlyingOf(ft.Params[1])) {
		a.errorf(expr, "slice of %s has to be func (*%s x) slice(int start, int end) Result", base.String(), base.String())
		return InvalidType{}, true
	}

	receiver := a.pristine(expr.Array)
	var start, end ast.Expression
	if expr.Start != nil {
		start = a.pristine(expr.Start)
	} else {
		start = &ast.IntLiteral{Value: 0}
		a.at(expr, start)
	}
	if expr.End != nil {
		end = a.pristine(expr.End)
	} else {
		if MethodsOf(base)["len"] == nil || !isPureOperand(receiver) {
			a.errorf(expr, "a slice without its end needs a len method on %s, and a value held in a variable", base.String())
			return InvalidType{}, true
		}
		size := &ast.MemberExpr{Object: a.pristine(expr.Array), Field: "len"}
		a.at(expr, size)
		sizeCall := &ast.CallExpr{Name: size}
		a.at(expr, sizeCall)
		end = sizeCall
	}

	name := &ast.MemberExpr{Object: receiver, Field: "slice"}
	a.at(expr, name)
	call := &ast.CallExpr{Name: name, Args: []ast.Expression{start, end}}
	a.at(expr, call)

	result := a.checkExpr(call)
	expr.Lowered = call
	return result, true
}
