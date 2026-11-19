package analyser

import (
	"math/big"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

func (a *Analyser) typeOperand(arg ast.Expression) Type {
	switch e := arg.(type) {
	case *ast.IdentLiteral:
		if sym, found := a.scope.Resolve(e.Value); found {
			if sym.Kind == STRUCT || sym.Kind == TYPE {
				return sym.Type
			}
		} else if primitives[e.Value] {
			return PrimitiveType{Name: e.Value}
		}
	case *ast.MemberExpr:
		if module, isIdent := e.Object.(*ast.IdentLiteral); isIdent {
			if sym, found := a.lookupType(a.scope, module.Value, e.Field); found && (sym.Kind == STRUCT || sym.Kind == TYPE) {
				return sym.Type
			}
		}
	case *ast.GenericExpr:
		ref := ast.TypeRef{Args: e.Args}
		switch base := e.Base.(type) {
		case *ast.IdentLiteral:
			ref.Name = base.Value
		case *ast.MemberExpr:
			if module, isIdent := base.Object.(*ast.IdentLiteral); isIdent {
				ref.Module, ref.Name = module.Value, base.Field
			}
		}
		if ref.Name != "" {
			if t := a.resolveTypeRef(arg, a.scope, ref); t != nil {
				if _, invalid := t.(InvalidType); !invalid {
					return t
				}
			}
		}
	}
	return nil
}

func (a *Analyser) complete(t Type) bool {
	switch v := t.(type) {
	case *StructType:
		if v.Optional != nil {
			return a.complete(v.Optional)
		}
		return v.complete
	case ArrayType:
		return a.complete(v.Element)
	case NamedType:
		return a.complete(v.Underlying)
	}
	return true
}

func (a *Analyser) recordLayout(call *ast.CallExpr, t Type, value func() int) {
	if a.complete(t) {
		a.info.LayoutValues[call] = value()
	}
}

func (a *Analyser) checkSizeof(expr *ast.CallExpr) Type {
	intType := PrimitiveType{Name: "int"}

	if len(expr.Args) != 1 {
		a.errorf(expr, "sizeof expects 1 argument, got %d", len(expr.Args))
		a.evalArgTypes(expr)
		return intType
	}

	arg := expr.Args[0]
	measured := a.typeOperand(arg)
	if measured == nil {
		measured = a.checkExpr(arg)
	}

	switch measured.(type) {
	case InvalidType:
		return intType
	case VoidType, ImplicitEnumType:
		a.errorf(expr, "sizeof cannot measure %s", measured.String())
		return intType
	}

	a.info.SizeOf[expr] = measured
	a.recordLayout(expr, measured, func() int { return SizeOf(measured) })
	return intType
}

func (a *Analyser) checkAlignof(expr *ast.CallExpr) Type {
	intType := PrimitiveType{Name: "int"}

	if len(expr.Args) != 1 {
		a.errorf(expr, "alignof expects 1 argument, got %d", len(expr.Args))
		a.evalArgTypes(expr)
		return intType
	}

	measured := a.typeOperand(expr.Args[0])
	if measured == nil {
		measured = a.checkExpr(expr.Args[0])
	}
	switch measured.(type) {
	case InvalidType:
		return intType
	case VoidType, ImplicitEnumType:
		a.errorf(expr, "alignof cannot measure %s", measured.String())
		return intType
	}

	a.recordLayout(expr, measured, func() int { return AlignOf(measured) })
	return intType
}

func fieldPath(expr ast.Expression) ([]string, bool) {
	switch e := expr.(type) {
	case *ast.IdentLiteral:
		return []string{e.Value}, true
	case *ast.MemberExpr:
		base, ok := fieldPath(e.Object)
		return append(base, e.Field), ok
	}
	return nil, false
}

func (a *Analyser) checkOffsetof(expr *ast.CallExpr) Type {
	intType := PrimitiveType{Name: "int"}

	if len(expr.Args) != 2 {
		a.errorf(expr, "offsetof expects a struct type and a field, got %d arguments", len(expr.Args))
		a.evalArgTypes(expr)
		return intType
	}

	owner := a.typeOperand(expr.Args[0])
	if owner == nil {
		a.errorf(expr.Args[0], "offsetof needs the name of a struct as its first argument")
		return intType
	}
	st, ok := underlying(owner).(*StructType)
	if !ok || st.Sum != nil || st.Optional != nil {
		a.errorf(expr.Args[0], "offsetof needs a struct, got %s", owner.String())
		return intType
	}
	path, ok := fieldPath(expr.Args[1])
	if !ok {
		a.errorf(expr.Args[1], "offsetof needs the name of a field, as in offsetof(P, x) or offsetof(P, inner.x)")
		return intType
	}

	offset := 0
	current := st
	for i, name := range path {
		if _, exists := current.Fields[name]; !exists {
			a.errorf(expr.Args[1], "struct %s has no field %s", current.Name, name)
			return intType
		}
		if !a.complete(current) {
			return intType
		}
		field := LayoutOf(current).Fields[name]
		offset += field.Offset
		if i < len(path)-1 {
			next, isStruct := underlying(field.Type).(*StructType)
			if !isStruct {
				a.errorf(expr.Args[1], "%s is not a struct, so it has no field %s", name, path[i+1])
				return intType
			}
			current = next
		}
	}
	a.info.LayoutValues[expr] = offset
	return intType
}

func (a *Analyser) checkStaticAssert(expr *ast.CallExpr) {
	if len(expr.Args) != 2 {
		a.errorf(expr, "staticAssert expects a condition and a message, got %d arguments", len(expr.Args))
		a.evalArgTypes(expr)
		return
	}

	a.checkExprList(expr.Args)
	condition, ok := a.evalConstLoud(expr.Args[0])
	if !ok || condition.Kind != ConstBool {
		a.errorf(expr.Args[0], "the condition of staticAssert must be known at compile time")
		return
	}
	message, ok := a.evalConstLoud(expr.Args[1])
	if !ok || message.Kind != ConstString {
		a.errorf(expr.Args[1], "the message of staticAssert must be a string known at compile time")
		return
	}

	a.info.StaticAsserts[expr] = true
	if !condition.Bool {
		a.errorf(expr, "static assertion failed: %s", message.Str)
	}
}

func (a *Analyser) evalConstLoud(expr ast.Expression) (ConstValue, bool) {
	saved := a.quiet
	a.quiet = false
	defer func() { a.quiet = saved }()
	return a.evalConst(expr)
}

func (a *Analyser) usesLayout(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.CallExpr:
			if id, ok := v.Name.(*ast.IdentLiteral); ok {
				switch id.Value {
				case "sizeof", "alignof", "offsetof":
					if arg, isIdent := v.Args[0].(*ast.IdentLiteral); len(v.Args) == 0 || !isIdent || !primitives[arg.Value] {
						found = true
					}
				}
			}
		case *ast.BlockExpr:
			if v.Static {
				found = true
			}
		case *ast.StructLiteral, *ast.AnonLiteral, *ast.ArrayLiteral:
			found = true
		case *ast.IdentLiteral:
			if a.lateNames[v.Value] {
				found = true
			}
		}
		return !found
	})
	return found
}

func (a *Analyser) registerLateDecls() {
	for _, d := range a.lateDecls {
		a.runLate(d)
	}
	a.lateDecls = nil
}

func (a *Analyser) runLate(d ast.Declaration) {
	if a.lateDone == nil {
		a.lateDone = map[ast.Declaration]bool{}
	}
	if a.lateDone[d] {
		return
	}
	a.lateDone[d] = true

	saved := a.currentFile
	a.currentFile = a.declFiles[d]
	defer func() { a.currentFile = saved }()

	switch decl := d.(type) {
	case *ast.ConstDecl:
		a.measuredStructs(decl)
		a.checkConstDecl(decl)
	case *ast.StaticAssertDecl:
		a.measuredStructs(decl)
		a.checkStaticAssert(decl.Call)
	}
}

func (a *Analyser) measuredStructs(node ast.Node) {
	ast.Inspect(node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 {
			return true
		}
		id, ok := call.Name.(*ast.IdentLiteral)
		if !ok || (id.Value != "sizeof" && id.Value != "alignof" && id.Value != "offsetof") {
			return true
		}
		if arg, isIdent := call.Args[0].(*ast.IdentLiteral); isIdent {
			if sd, found := a.structDecls[arg.Value]; found {
				a.registerFieldsOf(sd)
			}
		}
		return true
	})
}

func (a *Analyser) forceLate(name string) bool {
	if a.lateBusy[name] {
		return false
	}
	if a.lateBusy == nil {
		a.lateBusy = map[string]bool{}
	}
	a.lateBusy[name] = true
	defer delete(a.lateBusy, name)

	for _, d := range a.lateDecls {
		if decl, ok := d.(*ast.ConstDecl); ok && !a.lateDone[d] {
			for _, v := range decl.Vars {
				if v.Name == name {
					a.runLate(d)
					return true
				}
			}
		}
	}
	return false
}

func layoutConst(value int) ConstValue {
	return constInt(big.NewInt(int64(value)))
}

func joinPath(path []string) string {
	return strings.Join(path, ".")
}

func (a *Analyser) constantRoot(expr ast.Expression) (string, bool) {
	if name, ok := a.importedConst(expr); ok {
		return name, true
	}
	root := rootIdentifier(expr)
	if root == nil {
		return "", false
	}
	if _, isDeref := expr.(*ast.DerefExpr); isDeref {
		return "", false
	}
	sym, ok := a.info.Idents[root]
	if !ok || sym.Kind != CONST || sym.Const == nil || !sym.Const.aggregate() {
		return "", false
	}
	return sym.Name, true
}
