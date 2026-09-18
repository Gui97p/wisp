package analyser

import "github.com/Gui97p/wisp/internal/ast"

func (a *Analyser) registerStructNames() {
	for _, d := range a.program.Declarations {
		sd, ok := d.(*ast.StructDecl)
		if !ok {
			continue
		}

		st := &StructType{
			Name:    sd.Name,
			Fields:  make(map[string]Type),
			Order:   make([]string, 0, len(sd.Members)),
			Methods: make(map[string]*FuncType),
		}

		symbol := &Symbol{Name: sd.Name, Kind: STRUCT, Type: st}

		if !a.scope.Define(symbol) {
			a.errorAlreadyDeclared(sd, STRUCT, sd.Name)
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
			fieldType := a.resolveTypeRef(sd, a.scope, member.Type)
			if fieldType == nil {
				continue
			}

			if _, ok := st.Fields[member.Name]; ok {
				a.errorf(sd, "duplicated field %s in struct %s", member.Name, sd.Name)
				continue
			}

			st.Fields[member.Name] = fieldType
			st.Order = append(st.Order, member.Name)
		}
	}
}

func (a *Analyser) checkStructLiteral(expr *ast.StructLiteral) Type {
	symbol, ok := a.scope.Resolve(expr.Name)
	if !ok {
		a.errorf(expr, "unknown type %s", expr.Name)
		return InvalidType{}
	}

	st, ok := symbol.Type.(*StructType)
	if !ok {
		a.errorf(expr, "%s is not a struct", expr.Name)
		return InvalidType{}
	}

	seen := make(map[string]bool)
	for i, key := range expr.Keys {
		valueType := a.checkExpr(expr.Values[i])

		fieldType, ok := st.Fields[key]
		if !ok {
			a.errorf(expr, "unknown field %s in struct %s", key, st.Name)
			continue
		}
		if seen[key] {
			a.errorf(expr, "duplicated field %s in struct literal", key)
			continue
		}
		seen[key] = true

		if _, invalid := valueType.(InvalidType); invalid {
			continue
		}
		if !valueType.Equals(fieldType) {
			a.errorf(expr, "field %s expects %s, got %s", key, fieldType.String(), valueType.String())
		}
	}

	return st
}
