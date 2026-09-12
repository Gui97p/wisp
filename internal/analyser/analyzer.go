package analyser

import "github.com/Gui97p/wisp/internal/ast"

type Analyser struct {
	info  *Info
	scope *Scope

	errors []string
}

func NewAnalyzer() *Analyser {
	return &Analyser{
		info:  NewInfo(),
		scope: NewScope(nil),
	}
}

func (a *Analyser) Analyze(program *ast.Program) (*Info, []string) {
	a.registerStructNames(program)
	a.registerStructFields(program)
	a.registerFuncSignatures(program)

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
