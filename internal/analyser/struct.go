package analyser

import "github.com/Gui97p/wisp/internal/ast"

func (a *Analyser) registerStructNames() {
	for _, d := range a.program.Declarations {
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
			a.errorf("%s struct already defined in this scope", sd.Name)
		}
	}
}

func (a *Analyser) registerStructFields() {
	for _, d := range a.program.Declarations {
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

func (a *Analyser) checkStructConstruction(expr *ast.CallExpr, st *StructType) []Type {
	argTypes := a.evalArgTypes(expr)

	if len(argTypes) != len(st.Order) {
		a.errorf("struct %s expects %d fields, got %d", st.Name, len(st.Order), len(argTypes))
	}

	for i, at := range argTypes {
		if _, ok := at.(InvalidType); ok {
			continue
		}
		fieldName := st.Order[i]
		fieldType := st.Fields[fieldName]
		if !at.Equals(fieldType) {
			a.errorf("field %d (%s) from %s expected %s, got %s", i+1, fieldName, st.Name, fieldType.String(), at.String())
		}
	}

	return []Type{st}
}
