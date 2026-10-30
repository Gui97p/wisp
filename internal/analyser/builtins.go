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

	a.universe.Define(&Symbol{Name: "sizeof", Kind: FUNC, Type: &FuncType{
		Name: "sizeof", Params: []Type{AnyType{}}, Returns: []Type{intType},
	}})

	a.universe.Define(&Symbol{Name: "Error", Kind: TYPE, Type: ErrorType{}})
}
