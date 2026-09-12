package analyser

import (
	"github.com/Gui97p/wisp/internal/ast"
)

type Analyser struct {
	program *ast.Program

	info  *Info
	scope *Scope

	currentReturns []Type
	loopLabels     []string

	errors []string
}

func NewAnalyzer(program *ast.Program) *Analyser {
	return &Analyser{
		program: program,
		info:    NewInfo(),
		scope:   NewScope(nil),
	}
}

func (a *Analyser) Analyze() (*Info, []string) {
	a.registerStructNames()
	a.registerStructFields()
	a.registerFuncSignatures()

	return a.info, a.errors
}

func (a *Analyser) resolveTypeRef(scope *Scope, ref ast.TypeRef) Type {
	var result Type

	if primitives[ref.Name] {
		result = PrimitiveType{Name: ref.Name}
	} else {
		symbol, ok := scope.Resolve(ref.Name)
		if !ok {
			a.errorf("unknown type %s", ref.Name)
			return nil
		}
		if symbol.Kind != STRUCT {
			a.errorf("%s is not a type", ref.Name)
			return nil
		}
		result = symbol.Type
	}

	if ref.IsArray {
		result = ArrayType{Element: result, Size: ref.ArraySize}
	}
	for i := 0; i < ref.PointerDepth; i++ {
		result = PointerType{Element: result}
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

func (a *Analyser) requireBool(t Type, context string) {
	if _, ok := t.(InvalidType); ok {
		return
	}
	if !t.Equals(PrimitiveType{Name: "bool"}) {
		a.errorf("%s must be bool, got %s", context, t.String())
	}
}

func (a *Analyser) requireNumeric(t Type, context string) {
	if _, ok := t.(InvalidType); ok {
		return
	}
	if !isNumeric(t) {
		a.errorf("%s must be numeric, got %s", context, t.String())
	}
}

func (a *Analyser) pushLoop(label string) {
	a.loopLabels = append(a.loopLabels, label)
}

func (a *Analyser) popLoop() {
	a.loopLabels = a.loopLabels[:len(a.loopLabels)-1]
}
