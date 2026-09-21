package analyser

import (
	"slices"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/diag"
)

type Analyser struct {
	program *ast.Program

	info  *Info
	scope *Scope

	currentReturns []Type
	loopLabels     []string

	errors diag.List
}

func NewAnalyser(program *ast.Program) *Analyser {
	return &Analyser{
		program: program,
		info:    NewInfo(),
		scope:   NewScope(nil),
	}
}

func (a *Analyser) Analyze() *Info {
	a.registerStructNames()
	a.registerTypeAliases()
	a.registerMethods()
	a.registerStructFields()
	a.registerBuiltins()
	a.registerFuncSignatures()
	a.registerConsts()
	a.checkMainFunc()
	a.checkFuncBodies()

	return a.info
}

func (a *Analyser) resolveTypeRef(node ast.Node, scope *Scope, ref ast.TypeRef) Type {
	if ref.IsFunc {
		params := make([]Type, len(ref.FuncParams))
		for i, p := range ref.FuncParams {
			params[i] = a.resolveTypeRef(node, scope, p)
		}
		returns := make([]Type, len(ref.FuncReturns))
		for i, r := range ref.FuncReturns {
			returns[i] = a.resolveTypeRef(node, scope, r)
		}
		return &FuncType{Params: params, Returns: returns}
	}

	if ref.IsMap {
		keyType := a.resolveTypeRef(node, scope, *ref.MapKey)
		valueType := a.resolveTypeRef(node, scope, *ref.MapValue)
		if keyType == nil || valueType == nil {
			return nil
		}

		var result Type = &MapType{Key: keyType, Value: valueType}
		for i := 0; i < ref.PointerDepth; i++ {
			result = PointerType{Element: result}
		}
		if ref.Fallible {
			result = ErrorUnionType{Payload: result}
		}
		return result
	}

	var result Type

	if primitives[ref.Name] {
		result = PrimitiveType{Name: ref.Name}
	} else {
		symbol, ok := scope.Resolve(ref.Name)
		if !ok {
			a.errorf(node, "unknown type %s", ref.Name)
			return nil
		}
		switch symbol.Kind {
		case STRUCT, TYPE:
			result = symbol.Type
		default:
			a.errorf(node, "%s is not a type", ref.Name)
			return nil
		}
	}

	for _, v := range slices.Backward(ref.Dims) {
		if v.IsSpan {
			result = SpanType{Element: result}
		} else {
			result = ArrayType{Element: result, Size: v.Size}
		}
	}

	for i := 0; i < ref.PointerDepth; i++ {
		result = PointerType{Element: result}
	}

	if ref.Fallible {
		result = ErrorUnionType{Payload: result}
	}

	return result
}

func (a *Analyser) enterScope() *Scope {
	a.scope = NewScope(a.scope)
	return a.scope
}

func (a *Analyser) exitScope() {
	a.scope = a.scope.parent
}

func (a *Analyser) requireBool(node ast.Node, t Type, context string) {
	if _, ok := t.(InvalidType); ok {
		return
	}
	if !t.Equals(PrimitiveType{Name: "bool"}) {
		a.errorf(node, "%s must be bool, got %s", context, t.String())
	}
}

func (a *Analyser) requireNumeric(node ast.Node, t Type, context string) {
	if _, ok := t.(InvalidType); ok {
		return
	}
	if !isNumeric(t) {
		a.errorf(node, "%s must be numeric, got %s", context, t.String())
	}
}

func (a *Analyser) pushLoop(label string) {
	a.loopLabels = append(a.loopLabels, label)
}

func (a *Analyser) popLoop() {
	a.loopLabels = a.loopLabels[:len(a.loopLabels)-1]
}

func rootIdentifier(expr ast.Expression) *ast.IdentLiteral {
	switch e := expr.(type) {
	case *ast.IdentLiteral:
		return e
	case *ast.MemberExpr:
		return rootIdentifier(e.Object)
	case *ast.IndexExpr:
		return rootIdentifier(e.Array)
	case *ast.UnaryExpr:
		if e.Operator == "*" {
			return rootIdentifier(e.Value)
		}
	}
	return nil
}
