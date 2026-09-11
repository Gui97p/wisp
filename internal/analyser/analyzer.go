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

func (a *Analyser) registerStructNames(program *ast.Program) {
	for _, d := range program.Declarations {
		sd, ok := d.(*ast.StructDecl)
		if !ok {
			continue
		}

		st := &StructType{
			Name:   sd.Name,
			Fields: make(map[string]Type),
			Order:  make([]string, 0, len(sd.Members)),
		}

		symbol := &Symbol{Name: sd.Name, Kind: STRUCT, Type: st}

		if !a.scope.Define(symbol) {
			a.errorf("%s already defined in this scope", sd.Name)
		}
	}
}

func (a *Analyser) registerStructFields(program *ast.Program) {
	for _, d := range program.Declarations {
		sd, ok := d.(*ast.StructDecl)
		if !ok {
			continue
		}

		symbol, _ := a.scope.Resolve(sd.Name)
		st := symbol.Type.(*StructType)

		for _, member := range sd.Members {
			fieldType := a.resolveTypeRef(a.scope, member.Type)
			if fieldType == nil {
				continue
			}

			if _, ok := st.Fields[member.Name]; ok {
				a.errorf("duplicated field %s in struct %s", member.Name, sd.Name)
				continue
			}

			st.Fields[member.Name] = fieldType
			st.Order = append(st.Order, member.Name)
		}
	}
}
