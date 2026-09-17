package analyser

import "github.com/Gui97p/wisp/internal/ast"

func (a *Analyser) registerTypeAliases() {
	for _, d := range a.program.Declarations {
		td, ok := d.(*ast.TypeDecl)
		if !ok {
			continue
		}

		underlyingType := a.resolveTypeRef(td, a.scope, td.Underlying)
		if underlyingType == nil {
			continue
		}

		nt := NamedType{
			Name:       td.Name,
			Underlying: underlyingType,
		}

		symbol := &Symbol{Name: td.Name, Kind: TYPE, Type: nt}

		if !a.scope.Define(symbol) {
			a.errorAlreadyDeclared(td, TYPE, td.Name)
		}
	}
}

func (a *Analyser) registerConsts() {
	for _, d := range a.program.Declarations {
		cd, ok := d.(*ast.ConstDecl)
		if !ok {
			continue
		}
		a.checkConstDecl(cd)
	}
}

func (a *Analyser) checkConstDecl(decl *ast.ConstDecl) {
	a.checkVarsAndValues(decl, decl.Vars, decl.Values, CONST)
}

func (a *Analyser) checkVarsAndValues(node ast.Node, vars []ast.Param, values []ast.Expression, kind SymbolKind) {
	hasExplicitType := vars[0].Type.Name != ""

	if len(values) == 0 {
		for _, v := range vars {
			t := a.resolveTypeRef(node, a.scope, v.Type)
			if t == nil {
				t = InvalidType{}
			}
			sym := &Symbol{Name: v.Name, Kind: kind, Type: t}
			if !a.scope.Define(sym) {
				a.errorAlreadyDeclared(node, kind, v.Name)
			}

			a.info.VarTypes[node] = append(a.info.VarTypes[node], t)
		}
		return
	}

	valueTypes := a.checkExprList(values)

	if len(valueTypes) != len(vars) {
		a.errorf(node, "expected %d values, got %d", len(vars), len(valueTypes))
	}

	for i, v := range vars {
		var valueType Type
		if i < len(valueTypes) {
			valueType = valueTypes[i]
		} else {
			valueType = InvalidType{}
		}

		var finalType Type
		if hasExplicitType {
			declaredType := a.resolveTypeRef(node, a.scope, v.Type)
			if declaredType == nil {
				declaredType = InvalidType{}
			}
			if _, invalid := valueType.(InvalidType); !invalid {
				if _, declInvalid := declaredType.(InvalidType); !declInvalid {
					if declArr, ok := declaredType.(ArrayType); ok {
						valArr, ok := valueType.(ArrayType)
						switch {
						case !ok:
							a.errorDeclaredAs(node, v.Name, declaredType, valueType)
						case valArr.Size > declArr.Size:
							a.errorf(node, "array %s declared with size %d, got %d elements", v.Name, declArr.Size, valArr.Size)
						case !valArr.Element.Equals(declArr.Element):
							a.errorf(node, "array %s expects element type %s, got %s", v.Name, declArr.Element.String(), valArr.Element.String())
						}
					} else if declSpan, ok := declaredType.(SpanType); ok {
						valArr, ok := valueType.(ArrayType)
						switch {
						case !ok:
							a.errorDeclaredAs(node, v.Name, declaredType, valueType)
						case !valArr.Element.Equals(declSpan.Element):
							a.errorf(node, "span %s expects element type %s, got %s", v.Name, declSpan.Element.String(), valArr.Element.String())
						}
					} else if !valueType.Equals(declaredType) {
						a.errorDeclaredAs(node, v.Name, declaredType, valueType)
					}
				}
			}
			finalType = declaredType
		} else {
			finalType = valueType
		}

		sym := &Symbol{Name: v.Name, Kind: kind, Type: finalType}
		if !a.scope.Define(sym) {
			a.errorAlreadyDeclared(node, kind, v.Name)
		}

		a.info.VarTypes[node] = append(a.info.VarTypes[node], finalType)
	}
}
