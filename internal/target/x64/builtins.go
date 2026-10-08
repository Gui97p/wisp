package x64

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/target"
)

func (t *X64Target) compileBuiltin(name string, expr *ast.CallExpr) (Operand, error) {
	switch name {
	case "emit":
		return t.compileEmitBuiltin(expr)
	case "len":
		return t.compileLenBuiltin(expr)
	case "panic":
		return t.compilePanicBuiltin(expr)
	default:
		return nil, fmt.Errorf("x86-64: builtin %s not supported", name)
	}
}

func (t *X64Target) compileEmitBuiltin(expr *ast.CallExpr) (Operand, error) {
	mod, err := target.Lookup(t.target)
	if err != nil {
		return nil, err
	}
	if mod.RuntimeSource == "" {
		return nil, fmt.Errorf("x86-64: target %s does not provide emit", mod.Name)
	}

	if len(expr.Args) != 1 {
		return nil, fmt.Errorf("x86-64: emit only accepts 1 parameter")
	}

	_, err = t.compileCall("__wisp_emit", true, expr.Args, nil)
	return nil, err
}

func (t *X64Target) compileLenBuiltin(expr *ast.CallExpr) (Operand, error) {
	switch arg := t.info.Types[expr.Args[0]].(type) {
	case analyser.ArrayType:
		return Imm(arg.Size), nil
	case analyser.PrimitiveType, *analyser.SpanType:
		op, err := t.compileExpr(expr.Args[0])
		if err != nil {
			return nil, err
		}

		return t.field(op, 1), nil
	default:
		return nil, fmt.Errorf("x86-64: cannot get length of %s", arg)
	}
}

func (t *X64Target) compilePanicBuiltin(expr *ast.CallExpr) (Operand, error) {
	mod, err := target.Lookup(t.target)
	if err != nil {
		return nil, err
	}
	if mod.RuntimeSource == "" {
		return nil, fmt.Errorf("x86-64: target %s does not provide panic", mod.Name)
	}

	if len(expr.Args) != 1 {
		return nil, fmt.Errorf("x86-64: panic only accepts 1 parameter")
	}

	_, err = t.compileCall("__wisp_panic", true, expr.Args, nil)
	return nil, err
}
