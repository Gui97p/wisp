package analyser

import "github.com/Gui97p/wisp/internal/ast"

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
	a.checkVarsAndValues(decl.Vars, decl.Values, CONST, decl)
}

func (a *Analyser) checkVarsAndValues(vars []ast.Param, values []ast.Expression, kind SymbolKind, node ast.Node) {
	hasExplicitType := vars[0].Type.Name != ""

	if len(values) == 0 {
		for _, v := range vars {
			t := a.resolveTypeRef(a.scope, v.Type)
			if t == nil {
				t = InvalidType{}
			}
			sym := &Symbol{Name: v.Name, Kind: kind, Type: t}
			if !a.scope.Define(sym) {
				a.errorf("variable %s already declared in this scope", v.Name)
			}

			a.info.VarTypes[node] = append(a.info.VarTypes[node], t)
		}
		return
	}

	valueTypes := a.checkExprList(values)

	if len(valueTypes) != len(vars) {
		a.errorf("expected %d values, got %d", len(vars), len(valueTypes))
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
			declaredType := a.resolveTypeRef(a.scope, v.Type)
			if declaredType == nil {
				declaredType = InvalidType{}
			}
			if _, invalid := valueType.(InvalidType); !invalid {
				if _, declInvalid := declaredType.(InvalidType); !declInvalid {
					if declArr, ok := declaredType.(ArrayType); ok {
						valArr, ok := valueType.(ArrayType)
						switch {
						case !ok:
							a.errorf("variable %s declared as %s, got %s", v.Name, declaredType.String(), valueType.String())
						case valArr.Size > declArr.Size:
							a.errorf("array %s declared with size %d, got %d elements", v.Name, declArr.Size, valArr.Size)
						case !valArr.Element.Equals(declArr.Element):
							a.errorf("array %s expects element type %s, got %s", v.Name, declArr.Element.String(), valArr.Element.String())
						}
					} else if declSpan, ok := declaredType.(SpanType); ok {
						valArr, ok := valueType.(ArrayType)
						switch {
						case !ok:
							a.errorf("variable %s declared as %s, got %s", v.Name, declaredType.String(), valueType.String())
						case !valArr.Element.Equals(declSpan.Element):
							a.errorf("span %s expects element type %s, got %s", v.Name, declSpan.Element.String(), valArr.Element.String())
						}
					} else if !valueType.Equals(declaredType) {
						a.errorf("variable %s declared as %s, got %s", v.Name, declaredType.String(), valueType.String())
					}
				}
			}
			finalType = declaredType
		} else {
			finalType = valueType
		}

		sym := &Symbol{Name: v.Name, Kind: kind, Type: finalType}
		if !a.scope.Define(sym) {
			a.errorf("variable %s already declared in this scope", v.Name)
		}

		a.info.VarTypes[node] = append(a.info.VarTypes[node], finalType)
	}
}
