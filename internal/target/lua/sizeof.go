package lua

import "github.com/Gui97p/wisp/internal/analyser"

func alignUp(n, a int) int {
	if a <= 1 {
		return n
	}
	return (n + a - 1) / a * a
}

func sizeOf(at analyser.Type) int {
	switch tp := at.(type) {
	case analyser.PrimitiveType:
		switch tp.Name {
		case "int8", "uint8", "bool", "char":
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
	case analyser.UntypedIntType, analyser.UntypedFloatType, analyser.PointerType, analyser.ErrorType:
		return 8
	case analyser.ArrayType:
		return int(tp.Size) * sizeOf(tp.Element)
	case *analyser.StructType:
		size, _ := structLayout(tp)
		return size
	case analyser.SpanType:
		return 16
	case analyser.BoxedType:
		return 24
	case analyser.NamedType:
		return sizeOf(tp.Underlying)
	}
	return 0
}

func alignOf(at analyser.Type) int {
	switch tp := at.(type) {
	case analyser.PrimitiveType:
		switch tp.Name {
		case "int8", "uint8", "bool", "char":
			return 1
		case "int16", "uint16":
			return 2
		case "int32", "uint32", "float32":
			return 4
		}
		return 8
	case analyser.ArrayType:
		return alignOf(tp.Element)
	case *analyser.StructType:
		_, align := structLayout(tp)
		return align
	case analyser.NamedType:
		return alignOf(tp.Underlying)
	}
	return 8
}

func structLayout(st *analyser.StructType) (size, align int) {
	align = 1
	pos := 0
	for _, name := range st.Order {
		tp := st.Fields[name]
		fieldAlign := alignOf(tp)
		pos = alignUp(pos, fieldAlign) + sizeOf(tp)
		align = max(align, fieldAlign)
	}
	return alignUp(pos, align), align
}
