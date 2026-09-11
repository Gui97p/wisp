package analyser

type Type interface {
	String() string
	Equals(Type) bool
}

type PrimitiveType struct {
	Name string
}

type PointerType struct {
	Element Type
}

type ArrayType struct {
	Element Type
	Size    int64
}

type StructType struct {
	Name   string
	Fields map[string]Type
	Order  []string
}

type FuncType struct {
	Params  []Type
	Returns []Type
}
