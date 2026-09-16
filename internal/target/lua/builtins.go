package lua

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

func getBuiltin(name string) (func(*strings.Builder, []ast.Expression) error, bool) {
	switch name {
	case "emit":
		return compileEmit, true
	case "emitf":
		return compileEmitf, true
	case "random":
		return compileRandom, true
	default:
		return nil, false
	}
}

func compileBuiltinCall(b *strings.Builder, name string, args []ast.Expression) error {
	compile, ok := getBuiltin(name)
	if !ok {
		return fmt.Errorf("lua: builtin %q is not supported", name)
	}

	return compile(b, args)
}

func compileEmit(b *strings.Builder, args []ast.Expression) error {
	b.WriteString("print(")

	for i, arg := range args {
		if i > 0 {
			b.WriteString(", ")
		}

		if err := compileExpression(b, arg); err != nil {
			return err
		}
	}

	b.WriteString(")\n")
	return nil
}

func compileEmitf(b *strings.Builder, args []ast.Expression) error {
	b.WriteString("print(string.format(")

	for i, arg := range args {
		if i > 0 {
			b.WriteString(", ")
		}

		if err := compileExpression(b, arg); err != nil {
			return err
		}
	}

	b.WriteString("))\n")
	return nil
}

func compileRandom(b *strings.Builder, args []ast.Expression) error {
	b.WriteString("math.random()")
	return nil
}
