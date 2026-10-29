package lua

import "github.com/Gui97p/wisp/internal/analyser"

func underlying(tp analyser.Type) analyser.Type {
	for {
		nt, ok := tp.(analyser.NamedType)
		if !ok {
			return tp
		}
		tp = nt.Underlying
	}
}

func isView(tp analyser.Type) bool {
	switch underlying(tp).(type) {
	case analyser.ArrayType, analyser.SpanType:
		return true
	}
	return false
}

func isPointer(tp analyser.Type) bool {
	_, ok := underlying(tp).(analyser.PointerType)
	return ok
}

func isAggregate(tp analyser.Type) bool {
	switch underlying(tp).(type) {
	case analyser.ArrayType, *analyser.StructType:
		return true
	}
	return false
}

func pointee(tp analyser.Type) analyser.Type {
	if pt, ok := underlying(tp).(analyser.PointerType); ok {
		return pt.Element
	}
	return nil
}

func isMap(tp analyser.Type) bool {
	_, ok := underlying(tp).(*analyser.MapType)
	return ok
}

func viewElement(tp analyser.Type) analyser.Type {
	switch v := underlying(tp).(type) {
	case analyser.ArrayType:
		return v.Element
	case analyser.SpanType:
		return v.Element
	}
	return nil
}
