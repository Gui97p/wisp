package analyser

func alignUp(n, a int) int {
	if a <= 1 {
		return n
	}
	return (n + a - 1) / a * a
}

type FieldLayout struct {
	Offset int
	Size   int
	Type   Type
}

type StructLayout struct {
	Size   int
	Align  int
	Fields map[string]FieldLayout
}

func LayoutOf(st *StructType) *StructLayout {
	if st.layout != nil {
		return st.layout
	}

	l := &StructLayout{Align: 1, Fields: make(map[string]FieldLayout, len(st.Order))}
	pos := 0
	if st.Sum != nil {
		for i, name := range st.Order {
			tp := st.Fields[name]
			l.Align = max(l.Align, AlignOf(tp))
			if i == 0 {
				l.Fields[name] = FieldLayout{Offset: 0, Size: SizeOf(tp), Type: tp}
				pos = SizeOf(tp)
			}
		}
		base, widest := pos, 0
		for i, name := range st.Order {
			if i == 0 {
				continue
			}
			tp := st.Fields[name]
			base = max(base, alignUp(pos, AlignOf(tp)))
			widest = max(widest, SizeOf(tp))
		}
		for i, name := range st.Order {
			if i == 0 {
				continue
			}
			tp := st.Fields[name]
			l.Fields[name] = FieldLayout{Offset: base, Size: SizeOf(tp), Type: tp}
		}
		pos = base + widest
		l.Size = alignUp(pos, l.Align)
		st.layout = l
		return l
	}
	for _, name := range st.Order {
		tp := st.Fields[name]
		align := AlignOf(tp)
		size := SizeOf(tp)
		if st.Packed {
			align = 1
		}
		pos = alignUp(pos, align)
		l.Fields[name] = FieldLayout{Offset: pos, Size: size, Type: tp}
		pos += size
		l.Align = max(l.Align, align)
	}
	l.Size = alignUp(pos, l.Align)

	st.layout = l
	return l
}

func AlignOf(at Type) int {
	switch tp := at.(type) {
	case PrimitiveType:
		switch tp.Name {
		case "int8", "uint8", "bool":
			return 1
		case "int16", "uint16":
			return 2
		case "int32", "uint32", "float32":
			return 4
		case "int", "int64", "uint", "uint64", "float", "float64", "string":
			return 8
		}
	case UntypedIntType, UntypedFloatType:
		return 8
	case PointerType, SpanType:
		return 8
	case ArrayType:
		return AlignOf(tp.Element)
	case *StructType:
		return LayoutOf(tp).Align
	case BoxedType, *InterfaceType:
		return 8
	case NamedType:
		return AlignOf(tp.Underlying)
	}
	return 1
}

func SizeOf(at Type) int {
	switch tp := at.(type) {
	case PrimitiveType:
		switch tp.Name {
		case "int8", "uint8", "bool":
			return 1
		case "int16", "uint16":
			return 2
		case "int32", "uint32", "float32":
			return 4
		case "int", "int64", "uint", "uint64", "float", "float64":
			return 8
		case "string":
			return 16
		}
	case UntypedIntType, UntypedFloatType, PointerType, ErrorType, *FuncType:
		return 8
	case ArrayType:
		return int(tp.Size) * SizeOf(tp.Element)
	case *StructType:
		return LayoutOf(tp).Size
	case SpanType, *InterfaceType:
		return 16
	case BoxedType:
		return 24
	case NamedType:
		return SizeOf(tp.Underlying)
	}

	return 0
}
