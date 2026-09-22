package analyser

func (a *Analyser) registerBuiltins() {
	intType := PrimitiveType{Name: "int"}
	strType := PrimitiveType{Name: "string"}

	a.scope.Define(&Symbol{Name: "emit", Kind: FUNC, Type: &FuncType{
		Name: "emit", Params: []Type{strType}, Returns: nil, Variadic: true,
	}})
	a.scope.Define(&Symbol{Name: "emitf", Kind: FUNC, Type: &FuncType{
		Name: "emitf", Params: []Type{strType, AnyType{}}, Returns: nil, Variadic: true,
	}})
	a.scope.Define(&Symbol{Name: "len", Kind: FUNC, Type: &FuncType{
		Name: "len", Params: []Type{AnyType{}}, Returns: []Type{intType},
	}})

	a.scope.Define(&Symbol{Name: "malloc", Kind: FUNC, Type: &FuncType{
		Name: "malloc", Params: []Type{intType}, Returns: []Type{PointerType{Element: VoidType{}}},
	}})
	a.scope.Define(&Symbol{Name: "realloc", Kind: FUNC, Type: &FuncType{
		Name: "realloc", Params: []Type{PointerType{Element: VoidType{}}, intType}, Returns: []Type{PointerType{Element: VoidType{}}},
	}})
	a.scope.Define(&Symbol{Name: "free", Kind: FUNC, Type: &FuncType{
		Name: "free", Params: []Type{PointerType{Element: VoidType{}}}, Returns: nil,
	}})

	errorType := &StructType{
		Name: "Error",
		Fields: map[string]Type{
			"code":    intType,
			"message": strType,
		},
		Order:   []string{"code", "message"},
		Methods: make(map[string]*FuncType),
	}
	a.scope.Define(&Symbol{Name: "Error", Kind: STRUCT, Type: errorType})
}
