package analyser

import "github.com/Gui97p/wisp/internal/target"

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
	a.universe.Define(&Symbol{Name: "Here", Kind: TYPE, Type: ErrorType{}, Doc: "The place of a call: the first parameter of a function marked #caller, which the compiler fills in at every call. It is an Error with no kind, so `at.file`, `at.line` and `at.message` can be read, and it can be the base of an error."})

	a.registerRange()
	a.registerCTypes()

	for _, symbol := range a.universe.symbols {
		symbol.Builtin = true
	}
}

func (a *Analyser) registerCTypes() {
	long, ulong := "int64", "uint64"
	if target.ActiveOS() == "windows" {
		long, ulong = "int32", "uint32"
	}
	for name, base := range map[string]string{
		"c_char": "int8", "c_schar": "int8", "c_uchar": "uint8",
		"c_short": "int16", "c_ushort": "uint16",
		"c_int": "int32", "c_uint": "uint32",
		"c_long": long, "c_ulong": ulong,
		"c_longlong": "int64", "c_ulonglong": "uint64",
		"c_size": "uint64", "c_ssize": "int64",
		"c_float": "float32", "c_double": "float64",
	} {
		a.universe.Define(&Symbol{Name: name, Kind: TYPE, Type: PrimitiveType{Name: base}, Doc: "The C type of this name for the system being built: " + base + ".", Used: true})
	}
}
