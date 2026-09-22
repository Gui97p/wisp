package analyser

import (
	"github.com/Gui97p/wisp/internal/ast"
)

func (a *Analyser) registerTypeAliases() {
	for _, d := range a.program.Declarations {
		a.currentFile = a.declFiles[d]
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
			Methods:    make(map[string]*FuncType),
		}

		symbol := &Symbol{Name: td.Name, Kind: TYPE, Type: nt}

		if !a.scope.Define(symbol) {
			a.errorAlreadyDeclared(td, TYPE, td.Name)
		}
	}
}

func (a *Analyser) registerConsts() {
	for _, d := range a.program.Declarations {
		a.currentFile = a.declFiles[d]
		cd, ok := d.(*ast.ConstDecl)
		if !ok {
			continue
		}
		a.checkConstDecl(cd)
	}
}

func (a *Analyser) registerMethods() {
	for _, d := range a.program.Declarations {
		a.currentFile = a.declFiles[d]
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Receiver == nil {
			continue
		}

		recvType := a.resolveTypeRef(fd, a.scope, fd.Receiver.Type)
		if recvType == nil {
			continue
		}

		methods := MethodsOf(recvType)
		if methods == nil {
			a.errorf(fd, "cannot declare method on %s", recvType)
			continue
		}
		if _, exists := methods[fd.Name]; exists {
			a.errorf(fd, "method %s already declared for %s", fd.Name, recvType)
			continue
		}

		params := make([]Type, 0, len(fd.Params))
		variadic := false

		for _, p := range fd.Params {
			t := a.resolveTypeRef(fd, a.scope, p.Type)
			if t == nil {
				continue
			}
			params = append(params, t)
			if p.Variadic {
				variadic = true
			}
		}

		returns := make([]Type, 0, len(fd.ReturnTypes))
		for _, r := range fd.ReturnTypes {
			t := a.resolveTypeRef(fd, a.scope, r)
			if t != nil {
				returns = append(returns, t)
			}
		}

		fallibleCount := 0
		for _, t := range returns {
			if _, ok := t.(ErrorUnionType); ok {
				fallibleCount++
			}
		}
		if fallibleCount > 1 {
			a.errorf(fd, "function can only have one fallible (!T) return value")
		}

		methods[fd.Name] = &FuncType{Name: fd.Name, Params: params, Returns: returns, Variadic: variadic}
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
						switch valT := valueType.(type) {
						case ArrayType:
							if !valT.Element.Equals(declSpan.Element) {
								a.errorf(node, "span %s expects element type %s, got %s", v.Name, declSpan.Element.String(), valT.Element.String())
							}
						case SpanType:
							if !valT.Equals(declSpan) {
								a.errorDeclaredAs(node, v.Name, declaredType, valueType)
							}
						default:
							a.errorDeclaredAs(node, v.Name, declaredType, valueType)
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
