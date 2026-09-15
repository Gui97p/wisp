package analyser

func (a *Analyser) registerBuiltins() {
	intType := PrimitiveType{Name: "int"}
	strType := PrimitiveType{Name: "string"}
	floatType := PrimitiveType{Name: "float64"}

	a.scope.Define(&Symbol{Name: "emit", Kind: FUNC, Type: &FuncType{
		Name: "emit", Params: []Type{strType}, Returns: nil,
	}})
	a.scope.Define(&Symbol{Name: "emitf", Kind: FUNC, Type: &FuncType{
		Name: "emitf", Params: []Type{strType}, Returns: nil,
	}})

	a.scope.Define(&Symbol{Name: "random", Kind: FUNC, Type: &FuncType{
		Name: "random", Params: nil, Returns: []Type{floatType},
	}})

	a.scope.Define(&Symbol{Name: "malloc", Kind: FUNC, Type: &FuncType{
		Name: "malloc", Params: []Type{intType}, Returns: []Type{PointerType{Element: VoidType{}}},
	}})
	a.scope.Define(&Symbol{Name: "free", Kind: FUNC, Type: &FuncType{
		Name: "free", Params: []Type{PointerType{Element: VoidType{}}}, Returns: nil,
	}})
}
