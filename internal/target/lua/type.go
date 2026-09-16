package lua

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
)

var zeroValues = map[string]string{
	"int":   "0",
	"int8":  "0",
	"int16": "0",
	"int32": "0",
	"int64": "0",

	"uint":   "0",
	"uint8":  "0",
	"uint16": "0",
	"uint32": "0",
	"uint64": "0",

	"float32": "0.0",
	"float64": "0.0",

	"char":   "0",
	"string": `""`,
	"bool":   "false",
}

func zeroValue(t analyser.Type) (string, error) {
	switch t := t.(type) {
	case analyser.PrimitiveType:
		value, ok := zeroValues[t.Name]
		if !ok {
			return "", fmt.Errorf("lua: no zero value for type %s", t.Name)
		}

		return value, nil

	case analyser.UntypedIntType:
		return "0", nil

	case analyser.UntypedFloatType:
		return "0.0", nil

	case analyser.PointerType:
		return "nil", nil

	case analyser.NullType:
		return "nil", nil

	case *analyser.MapType:
		return "{}", nil

	case analyser.SpanType:
		return "nil", nil

	case *analyser.StructType:
		var b strings.Builder

		b.WriteString("{")

		for i, field := range t.Order {
			if i > 0 {
				b.WriteString(", ")
			}

			value, err := zeroValue(t.Fields[field])
			if err != nil {
				return "", err
			}

			fmt.Fprintf(&b, "%s = %s", field, value)
		}

		b.WriteString("}")

		return b.String(), nil
	case analyser.ArrayType:
		var b strings.Builder

		b.WriteByte('{')

		for i := int64(0); i < t.Size; i++ {
			if i > 0 {
				b.WriteString(", ")
			}

			value, err := zeroValue(t.Element)
			if err != nil {
				return "", err
			}

			b.WriteString(value)
		}

		b.WriteByte('}')

		return b.String(), nil

	default:
		return "", fmt.Errorf("lua: no zero value for type %s", t.String())
	}
}
