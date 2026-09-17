package lua

import (
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

func (t *LuaTarget) compileDefinition(b *strings.Builder, node ast.Node, vars []ast.Param, values []ast.Expression) error {
	b.WriteString("local ")
	for k, variable := range vars {
		if k > 0 {
			b.WriteString(", ")
		}
		b.WriteString(variable.Name)
	}

	b.WriteString(" = ")
	if len(values) > 0 {
		for k, value := range values {
			if k > 0 {
				b.WriteString(", ")
			}

			if err := t.compileExpression(b, value); err != nil {
				return err
			}
		}
	} else {
		types := t.info.VarTypes[node]
		for k := range vars {
			if k > 0 {
				b.WriteString(", ")
			}

			value, err := zeroValue(types[k])
			if err != nil {
				return err
			}

			b.WriteString(value)
		}
	}
	b.WriteByte('\n')
	return nil
}
