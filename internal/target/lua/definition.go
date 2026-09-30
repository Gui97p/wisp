package lua

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

func (t *LuaTarget) compileDefinition(b *strings.Builder, node ast.Node, vars []ast.Param, values []ast.Expression) error {
	var texts []string

	if len(values) > 0 {
		texts = make([]string, len(values))
		for k, value := range values {
			text, err := t.compileExprScratch(value)
			if err != nil {
				return err
			}
			if lit, ok := value.(*ast.ArrayLiteral); ok {
				if types := t.info.VarTypes[node]; k < len(types) {
					if at, ok := types[k].(analyser.ArrayType); ok {
						text, err = padArrayLiteral(text, len(lit.Elements), at)
						if err != nil {
							return err
						}
					}
				}
			}
			texts[k] = text
		}
	} else {
		types := t.info.VarTypes[node]
		texts = make([]string, len(vars))
		for k := range vars {
			value, err := zeroValue(types[k])
			if err != nil {
				return err
			}
			texts[k] = value
		}
	}

	t.flushPending(b)

	b.WriteString("local ")
	for k, variable := range vars {
		if k > 0 {
			b.WriteString(", ")
		}
		b.WriteString(variable.Name)
	}

	b.WriteString(" = ")
	b.WriteString(strings.Join(texts, ", "))
	b.WriteByte('\n')
	return nil
}

func padArrayLiteral(text string, given int, at analyser.ArrayType) (string, error) {
	if int64(given) >= at.Size {
		return text, nil
	}

	zero, err := zeroValue(at.Element)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString(strings.TrimSuffix(text, "}"))
	for i := int64(given); i < at.Size; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "[%d]=%s", i, zero)
	}
	b.WriteByte('}')

	return b.String(), nil
}
