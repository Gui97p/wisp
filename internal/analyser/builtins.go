package analyser

func (a *Analyser) registerBuiltins() {
	a.universe.Define(&Symbol{Name: "Comparable", Kind: INTERFACE, Type: &InterfaceType{Name: "Comparable", builtin: "comparable", key: "Comparable"}, Used: true})
	a.universe.Define(&Symbol{Name: "Orderable", Kind: INTERFACE, Type: &InterfaceType{Name: "Orderable", builtin: "orderable", key: "Orderable"}, Used: true})
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

	a.universe.Define(&Symbol{Name: "alignof", Kind: FUNC, Type: &FuncType{
		Name: "alignof", Params: []Type{AnyType{}}, Returns: []Type{intType},
	}})
	a.universe.Define(&Symbol{Name: "offsetof", Kind: FUNC, Type: &FuncType{
		Name: "offsetof", Params: []Type{AnyType{}, AnyType{}}, Returns: []Type{intType},
	}})
	a.universe.Define(&Symbol{Name: "staticAssert", Kind: FUNC, Type: &FuncType{
		Name: "staticAssert", Params: []Type{PrimitiveType{Name: "bool"}, strType}, Returns: nil,
	}})

	for _, name := range []string{"truncate", "saturate"} {
		a.universe.Define(&Symbol{Name: name, Kind: FUNC, Type: &FuncType{
			Name: name, Params: []Type{AnyType{}, AnyType{}}, Returns: []Type{intType},
		}})
	}

	a.universe.Define(&Symbol{Name: "Error", Kind: TYPE, Type: ErrorType{}})

	a.registerRange()

	for _, symbol := range a.universe.symbols {
		symbol.Builtin = true
	}
}
