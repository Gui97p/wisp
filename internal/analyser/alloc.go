package analyser

import "github.com/Gui97p/wisp/internal/ast"

var allocMethods = map[string]string{
	"make":      "alloc",
	"makeDirty": "allocDirty",
	"create":    "alloc",
	"free":      "free",
	"resize":    "realloc",
}

func (a *Analyser) sizeofOf(near ast.Node, typeArg ast.Expression, elem Type) *ast.CallExpr {
	name := &ast.IdentLiteral{Value: "sizeof"}
	a.at(near, name)
	call := &ast.CallExpr{Name: name, Args: []ast.Expression{typeArg}}
	a.at(near, call)
	a.info.Idents[name] = a.universe.symbols["sizeof"]
	a.info.SizeOf[call] = elem
	a.info.Types[call] = PrimitiveType{Name: "int"}
	a.info.CallReturns[call] = []Type{PrimitiveType{Name: "int"}}
	return call
}

func (a *Analyser) checkAllocBuiltin(expr *ast.CallExpr, name string) []Type {
	want := 3
	if name == "create" || name == "free" {
		want = 2
	}
	usages := map[string]string{
		"make":      "make(allocator, Type, count)",
		"makeDirty": "makeDirty(allocator, Type, count)",
		"create":    "create(allocator, Type)",
		"free":      "free(allocator, value)",
		"resize":    "resize(allocator, slice, count)",
	}
	if len(expr.Args) != want {
		a.errorf(expr, "%s takes %d arguments, as in %s, got %d", name, want, usages[name], len(expr.Args))
		a.evalValueArgs(expr.Args)
		return []Type{InvalidType{}}
	}

	allocator := a.checkExpr(expr.Args[0])
	if _, bad := allocator.(InvalidType); bad {
		a.evalValueArgs(expr.Args[1:])
		return []Type{InvalidType{}}
	}
	method := allocMethods[name]
	if methods := MethodsOf(allocator); methods == nil || methods[method] == nil {
		a.errorf(expr.Args[0], "%s needs an allocator, a value with the method %s, and %s has none", name, method, allocator.String())
		a.evalValueArgs(expr.Args[1:])
		return []Type{InvalidType{}}
	}

	lowered := &ast.CallExpr{Name: &ast.MemberExpr{Object: expr.Args[0], Field: method}}
	a.at(expr, lowered)
	a.at(expr, lowered.Name)

	if name == "resize" {
		return a.checkResize(expr, lowered, allocator)
	}

	if name == "free" {
		lowered.Args = []ast.Expression{expr.Args[1]}
		expr.Lowered = lowered
		return a.checkCallExpr(lowered)
	}

	elem, ok := a.typeArgument(expr.Args[1])
	if !ok {
		if id, isIdent := expr.Args[1].(*ast.IdentLiteral); isIdent {
			a.errorf(id, "unknown type %s", id.Value)
		} else {
			a.errorf(expr.Args[1], "%s takes a type name second, as in %s(allocator, int)", name, name)
		}
		if len(expr.Args) > 2 {
			a.checkExpr(expr.Args[2])
		}
		return []Type{InvalidType{}}
	}
	if _, isVoid := elem.(VoidType); isVoid {
		a.errorf(expr.Args[1], "%s cannot take void", name)
		return []Type{InvalidType{}}
	}
	size := a.sizeofOf(expr, expr.Args[1], elem)

	if name == "create" {
		lowered.Args = []ast.Expression{size}
		expr.Lowered = lowered
		results := a.checkCallExpr(lowered)
		if len(results) != 1 {
			return []Type{InvalidType{}}
		}
		if _, isFallible := results[0].(FallibleType); !isFallible {
			return results
		}
		return []Type{FallibleType{Values: []Type{PointerType{Element: elem}}}}
	}

	count := expr.Args[2]
	if counted := a.checkExpr(count); !isInteger(underlyingOf(counted)) {
		if _, bad := counted.(InvalidType); !bad {
			a.errorf(count, "the count of %s has to be an integer, got %s", name, counted.String())
		}
		return []Type{InvalidType{}}
	}
	bytes := &ast.BinaryExpr{Left: count, Operator: "*", Right: size}
	a.at(count, bytes)
	lowered.Args = []ast.Expression{bytes}
	expr.Lowered = lowered
	results := a.checkCallExpr(lowered)
	if len(results) != 1 {
		return []Type{InvalidType{}}
	}
	if _, isFallible := results[0].(FallibleType); !isFallible {
		a.errorf(expr, "the method %s of %s has to be fallible, as in `func (...) alloc(int n) !*void`", method, allocator.String())
		return []Type{InvalidType{}}
	}
	a.info.Makes[expr] = &MakeInfo{Elem: elem, Count: count, Size: size}
	return []Type{FallibleType{Values: []Type{SpanType{Element: elem}}}}
}

func (a *Analyser) evalValueArgs(args []ast.Expression) {
	for _, arg := range args {
		if _, isType := a.typeArgument(arg); !isType {
			a.checkExpr(arg)
		}
	}
}

func (a *Analyser) checkResize(expr, lowered *ast.CallExpr, allocator Type) []Type {
	slice := expr.Args[1]
	span, ok := a.checkExpr(slice).(SpanType)
	if !ok {
		if _, bad := a.info.Types[slice].(InvalidType); !bad {
			a.errorf(slice, "resize takes a slice, got %s", a.info.Types[slice].String())
		}
		a.checkExpr(expr.Args[2])
		return []Type{InvalidType{}}
	}

	count := expr.Args[2]
	if counted := a.checkExpr(count); !isInteger(underlyingOf(counted)) {
		if _, bad := counted.(InvalidType); !bad {
			a.errorf(count, "the count of resize has to be an integer, got %s", counted.String())
		}
		return []Type{InvalidType{}}
	}

	measured := func() *ast.CallExpr {
		first := &ast.IndexExpr{Array: slice, Index: a.intLiteral(slice, 0)}
		a.at(slice, first)
		return a.sizeofOf(expr, first, span.Element)
	}
	length := &ast.CallExpr{Name: &ast.IdentLiteral{Value: "len"}, Args: []ast.Expression{slice}}
	a.at(slice, length)
	a.at(slice, length.Name)
	oldBytes := &ast.BinaryExpr{Left: length, Operator: "*", Right: measured()}
	a.at(slice, oldBytes)
	newBytes := &ast.BinaryExpr{Left: count, Operator: "*", Right: measured()}
	a.at(count, newBytes)

	lowered.Args = []ast.Expression{slice, oldBytes, newBytes}
	expr.Lowered = lowered
	results := a.checkCallExpr(lowered)
	if len(results) != 1 {
		return []Type{InvalidType{}}
	}
	if _, isFallible := results[0].(FallibleType); !isFallible {
		a.errorf(expr, "the method realloc of %s has to be fallible, as in `func (...) realloc(?*void p, int old, int n) !*void`", allocator.String())
		return []Type{InvalidType{}}
	}
	a.info.Resizes[expr] = &ResizeInfo{Elem: span.Element, Slice: slice, Count: count}
	return []Type{FallibleType{Values: []Type{SpanType{Element: span.Element}}}}
}
