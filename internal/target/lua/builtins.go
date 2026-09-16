package lua

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

func (t *LuaTarget) getBuiltin(name string) (func(*strings.Builder, []ast.Expression) error, bool) {
	switch name {
	case "emit":
		return t.compileEmit, true
	case "emitf":
		return t.compileEmitf, true
	case "random":
		return t.compileRandom, true
	default:
		return nil, false
	}
}

func (t *LuaTarget) compileBuiltinCall(b *strings.Builder, name string, args []ast.Expression) error {
	compile, ok := t.getBuiltin(name)
	if !ok {
		return fmt.Errorf("lua: builtin %q is not supported", name)
	}

	return compile(b, args)
}

func (t *LuaTarget) compileEmit(b *strings.Builder, args []ast.Expression) error {
	b.WriteString("print(")

	for i, arg := range args {
		if i > 0 {
			b.WriteString(", ")
		}

		if err := t.compileExpression(b, arg); err != nil {
			return err
		}
	}

	b.WriteString(")\n")
	return nil
}

func (t *LuaTarget) compileEmitf(b *strings.Builder, args []ast.Expression) error {
	b.WriteString("print(string.format(")

	for i, arg := range args {
		if i > 0 {
			b.WriteString(", ")
		}

		if err := t.compileExpression(b, arg); err != nil {
			return err
		}
	}

	b.WriteString("))\n")
	return nil
}

func (*LuaTarget) compileRandom(b *strings.Builder, args []ast.Expression) error {
	b.WriteString("math.random()")
	return nil
}
