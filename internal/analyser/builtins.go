package analyser

func (a *Analyser) registerBuiltins() {
	intType := PrimitiveType{Name: "int"}
	strType := PrimitiveType{Name: "string"}

	a.universe.Define(&Symbol{Name: "emit", Kind: FUNC, Type: &FuncType{
		Name: "emit", Params: []Type{strType}, Returns: nil, Variadic: true,
	}})
	a.universe.Define(&Symbol{Name: "len", Kind: FUNC, Type: &FuncType{
		Name: "len", Params: []Type{AnyType{}}, Returns: []Type{intType},
	}})
	a.universe.Define(&Symbol{Name: "panic", Kind: FUNC, Type: &FuncType{
		Name: "panic", Params: []Type{strType}, Returns: nil,
	}})

	a.universe.Define(&Symbol{Name: "malloc", Kind: FUNC, Type: &FuncType{
		Name: "malloc", Params: []Type{intType}, Returns: []Type{PointerType{Element: VoidType{}}},
	}})
	a.universe.Define(&Symbol{Name: "realloc", Kind: FUNC, Type: &FuncType{
		Name: "realloc", Params: []Type{PointerType{Element: VoidType{}}, intType}, Returns: []Type{PointerType{Element: VoidType{}}},
	}})
	a.universe.Define(&Symbol{Name: "free", Kind: FUNC, Type: &FuncType{
		Name: "free", Params: []Type{PointerType{Element: VoidType{}}}, Returns: nil,
	}})

	a.universe.Define(&Symbol{Name: "Error", Kind: TYPE, Type: ErrorType{}})
}
