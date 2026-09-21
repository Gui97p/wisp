package lua

import (
	"strings"

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
