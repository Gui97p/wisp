package analyser

func (a *Analyser) registerBuiltins() {
	intType := PrimitiveType{Name: "int"}
	strType := PrimitiveType{Name: "string"}

	a.prelude.Define(&Symbol{Name: "emitf", Kind: FUNC, Type: &FuncType{
		Name: "emitf", Params: []Type{strType, AnyType{}}, Returns: nil, Variadic: true,
	}})
	a.prelude.Define(&Symbol{Name: "len", Kind: FUNC, Type: &FuncType{
		Name: "len", Params: []Type{AnyType{}}, Returns: []Type{intType},
	}})

	a.prelude.Define(&Symbol{Name: "malloc", Kind: FUNC, Type: &FuncType{
		Name: "malloc", Params: []Type{intType}, Returns: []Type{PointerType{Element: VoidType{}}},
	}})
	a.prelude.Define(&Symbol{Name: "realloc", Kind: FUNC, Type: &FuncType{
		Name: "realloc", Params: []Type{PointerType{Element: VoidType{}}, intType}, Returns: []Type{PointerType{Element: VoidType{}}},
	}})
	a.prelude.Define(&Symbol{Name: "free", Kind: FUNC, Type: &FuncType{
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
	a.prelude.Define(&Symbol{Name: "Error", Kind: STRUCT, Type: errorType})
}
