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
	case "panic":
		return t.compilePanic, true
	case "len":
		return t.compileLen, true
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

func (t *LuaTarget) compileLen(b *strings.Builder, args []ast.Expression) error {
	text, err := t.compileExprScratch(args[0])
	if err != nil {
		return err
	}

	tp := t.info.Types[args[0]]
	switch {
	case isStringType(underlying(tp)):
		fmt.Fprintf(b, "#%s", text)
	case isView(tp):
		fmt.Fprintf(b, "(%s).n", text)
	default:
		fmt.Fprintf(b, "__wisp_mlen(%s)", text)
	}
	return nil
}

func (t *LuaTarget) compilePanic(b *strings.Builder, args []ast.Expression) error {
	b.WriteString("__wisp_panic(")

	if err := t.compileExpression(b, args[0]); err != nil {
		return err
	}

	b.WriteString(")\n")
	return nil
}
