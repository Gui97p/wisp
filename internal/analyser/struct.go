package analyser

import "github.com/Gui97p/wisp/internal/ast"

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
			a.errorf("%s struct already defined in this scope", sd.Name)
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
