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
			Enum:       td.IsEnum,
			Module:     a.module,
		}

		line, col := td.Position()
		symbol := &Symbol{Name: td.Name, Kind: TYPE, Type: nt, Line: line, Col: col, File: a.currentFile}

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

		methods[fd.Name] = &FuncType{Name: fd.Name, Params: params, Returns: returns, Variadic: variadic, Fallible: fd.Fallible}
		a.info.FuncReturns[fd] = returns
	}
}

func (a *Analyser) checkConstDecl(decl *ast.ConstDecl) {
	a.checkVarsAndValues(decl, decl.Vars, decl.Values, CONST)
}

func (a *Analyser) checkVarsAndValues(node ast.Node, vars []ast.Param, values []ast.Expression, kind SymbolKind) {
	hasExplicitType := vars[0].Type.Name != ""
	line, col := node.Position()

	if len(values) == 0 {
		for _, v := range vars {
			t := a.resolveTypeRef(node, a.scope, v.Type)
			if t == nil {
				t = InvalidType{}
			}
			sym := &Symbol{Name: v.Name, Kind: kind, Type: t, Line: line, Col: col, File: a.currentFile}
			if !a.scope.Define(sym) {
				a.errorAlreadyDeclared(node, kind, v.Name)
			}

			a.info.VarTypes[node] = append(a.info.VarTypes[node], t)
			a.info.VarSymbols[node] = append(a.info.VarSymbols[node], sym)
		}
		return
	}

	valueTypes := a.checkExprList(values)
	valueTypes, destructured := a.expandFallible(values, valueTypes, len(vars))
	valueTypes = a.rejectFallible(values, valueTypes)

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
						case !a.arrayElementAssignable(values, len(vars), i, valArr.Element, declArr.Element):
							a.errorf(node, "array %s expects element type %s, got %s", v.Name, declArr.Element.String(), valArr.Element.String())
						default:
							if slot := a.valueSlot(values, len(vars), i); slot != nil {
								a.fixArrayLiteral(*slot, declArr.Element)
							}
						}
					} else if declSpan, ok := declaredType.(SpanType); ok {
						switch valT := valueType.(type) {
						case ArrayType:
							if !a.arrayElementAssignable(values, len(vars), i, valT.Element, declSpan.Element) {
								a.errorf(node, "span %s expects element type %s, got %s", v.Name, declSpan.Element.String(), valT.Element.String())
							} else if slot := a.valueSlot(values, len(vars), i); slot != nil {
								a.fixArrayLiteral(*slot, declSpan.Element)
							}
						case SpanType:
							if !valT.Equals(declSpan) {
								a.errorDeclaredAs(node, v.Name, declaredType, valueType)
							}
						default:
							a.errorDeclaredAs(node, v.Name, declaredType, valueType)
						}
					} else if declMap, isMap := declaredType.(*MapType); isMap && a.mapLiteralAssignable(values, len(vars), i, valueType, declMap) {
						a.fixMapLiteral(values[i], declMap.Key, declMap.Value)
					} else if !a.coerceAt(values, len(vars), i, valueType, declaredType) {
						a.errorDeclaredAs(node, v.Name, declaredType, valueType)
					} else {
						a.checkConstFits(a.valueAt(values, len(vars), i), declaredType)
					}
				}
			}
			finalType = declaredType
		} else if kind == CONST {
			finalType = valueType
		} else {
			if _, untyped := valueType.(UntypedIntType); untyped {
				a.checkConstFits(a.valueAt(values, len(vars), i), PrimitiveType{Name: "int"})
			}
			finalType = defaultType(valueType)
		}

		sym := &Symbol{Name: v.Name, Kind: kind, Type: finalType, Line: line, Col: col, File: a.currentFile}
		if kind == CONST {
			sym.Const = a.constantFor(node, values, len(vars), i, finalType)
		}
		if v.Name == "_" {
			sym.Used = true
		} else {
			if destructured && i == len(vars)-1 {
				sym.MustUse = true
				a.mustUse = append(a.mustUse, sym)
			}
			if !a.scope.Define(sym) {
				a.errorAlreadyDeclared(node, kind, v.Name)
			}
		}

		a.info.VarTypes[node] = append(a.info.VarTypes[node], finalType)
		a.info.VarSymbols[node] = append(a.info.VarSymbols[node], sym)
	}
}

func (a *Analyser) arrayElementAssignable(values []ast.Expression, n, i int, from, to Type) bool {
	if from.Equals(to) {
		return true
	}
	slot := a.valueSlot(values, n, i)
	if slot == nil {
		return false
	}
	if _, isLiteral := (*slot).(*ast.ArrayLiteral); !isLiteral {
		return false
	}
	return implicitConversion(from, to)
}

func (a *Analyser) mapLiteralAssignable(values []ast.Expression, n, i int, from Type, to *MapType) bool {
	slot := a.valueSlot(values, n, i)
	if slot == nil {
		return false
	}
	if _, isLiteral := (*slot).(*ast.MapLiteral); !isLiteral {
		return false
	}
	vm, ok := from.(*MapType)
	if !ok {
		return false
	}
	keyOK := vm.Key.Equals(to.Key) || implicitConversion(vm.Key, to.Key)
	valueOK := vm.Value.Equals(to.Value) || implicitConversion(vm.Value, to.Value)
	return keyOK && valueOK
}
