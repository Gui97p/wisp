package lua

import (
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

func (t *LuaTarget) compileDefinition(b *strings.Builder, node ast.Node, vars []ast.Param, values []ast.Expression) error {
	var texts []string

	if len(values) > 0 {
		texts = make([]string, len(values))
		for k, value := range values {
			var text string
			var err error
			if lit, ok := value.(*ast.ArrayLiteral); ok {
				var want *analyser.ArrayType
				if types := t.info.VarTypes[node]; k < len(types) {
					if at, isArray := types[k].(analyser.ArrayType); isArray {
						want = &at
					}
				}
				text, err = t.arrayLiteralText(lit, want)
			} else {
				text, err = t.compileExprScratch(value)
			}
			if err != nil {
				return err
			}
			texts[k] = t.valueText(value, text)
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

	for k := range texts {
		if syms := t.info.VarSymbols[node]; k < len(syms) && t.isBoxed(syms[k]) {
			texts[k] = "{[0]=" + texts[k] + "}"
		}
	}

	b.WriteString("local ")
	for k, variable := range vars {
		if k > 0 {
			b.WriteString(", ")
		}
		b.WriteString(luaIdent(variable.Name))
	}

	b.WriteString(" = ")
	b.WriteString(strings.Join(texts, ", "))
	b.WriteByte('\n')
	return nil
}
