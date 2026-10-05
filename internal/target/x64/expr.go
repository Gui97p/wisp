package x64

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

func (t *X64Target) compileExpr(expr ast.Expression) (Operand, error) {
	switch e := expr.(type) {
	case *ast.IntLiteral:
		if int64(int32(e.Value)) != e.Value {
			reg := t.ctx.AllocFreeRegister()
			if reg == x64context.NoReg {
				return nil, fmt.Errorf("x86-64: no available registers")
			}
			t.text.printft("mov %s, %d\n", t.ctx.GetRegister(reg, 8), e.Value)
			return Reg{Reg: reg, Size: 8}, nil
		}
		return Imm(e.Value), nil
	case *ast.BoolLiteral:
		if e.Value {
			return Imm(1), nil
		} else {
			return Imm(0), nil
		}
	case *ast.CharLiteral:
		return Imm(e.Value), nil
	case *ast.StringLiteral:
		decoded, err := decodeEscapes(e.Value)
		if err != nil {
			return nil, err
		}
		label, ok := t.createData("str", string(decoded))
		if !ok {
			t.rodata.printf("%s db %s\n", label, bytesToAsm(append(decoded, 0)))
		}
		return strLit(label, len(decoded)), nil
	case *ast.IdentLiteral:
		offset, ok := t.ctx.Get(t.info.Idents[e])
		if !ok {
			return nil, fmt.Errorf("x86-64: undefined literal")
		}
		size := sizeOf(t.info.Types[e])

		return slot(offset, size), nil

	case *ast.BinaryExpr:
		return t.compileBinaryExpr(e)
	case *ast.CallExpr:
		return t.compileCallExpr(e)
	default:
		return nil, fmt.Errorf("x86-64: unsupported expression %T", expr)
	}
}

func (t *X64Target) compileLValue(expr ast.Expression) (Mem, error) {
	switch e := expr.(type) {
	case *ast.IdentLiteral:
		offset, ok := t.ctx.Get(t.info.Idents[e])
		if !ok {
			return Mem{}, fmt.Errorf("x86-64: undeclared variable %s", e.Value)
		}
		size := sizeOf(t.info.Types[e])
		return slot(offset, size), nil
	default:
		return Mem{}, fmt.Errorf("x86-64: unsupported assignment target %T", expr)
	}
}

func (t *X64Target) compileBinaryExpr(expr *ast.BinaryExpr) (Operand, error) {
	switch expr.Operator {
	case "+", "-", "*", "/", "%":
		return t.compileArithmetic(expr)
	case "==", "!=", ">", ">=", "<", "<=":
		return t.compileComparison(expr)
	case "&", "|", "^", "<<", ">>":
		return t.compileBitwise(expr)
	case "&&", "||":
		return t.compileLogical(expr)
	default:
		return nil, fmt.Errorf("x86-64: unsupported operator %s", expr.Operator)
	}
}

func (t *X64Target) resolveCallee(expr ast.Expression) (*analyser.Symbol, error) {
	switch e := expr.(type) {
	case *ast.IdentLiteral:
		return t.info.Idents[e], nil
	case *ast.MemberExpr:
		c, err := t.resolveCallee(e.Object)
		if err != nil {
			return nil, err
		}
		tp, ok := c.Type.(*analyser.ModuleType)
		if !ok {
			return c, nil
		}
		return tp.Exports[e.Field], nil
	default:
		return nil, fmt.Errorf("x86-64: invalid callee %s", e)
	}
}

func (t *X64Target) compileCallExpr(expr *ast.CallExpr) (Operand, error) {
	ops, err := t.compileCallValues(expr)
	if err != nil {
		return nil, err
	}

	if len(ops) == 1 {
		return ops[0], nil
	}

	for _, op := range ops {
		t.freeOp(op)
	}
	return nil, nil
}
