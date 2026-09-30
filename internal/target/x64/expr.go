package x64

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

func (t *X64Target) newImm(v int64) (Operand, error) {
	if v != int64(int32(v)) {
		reg := t.ctx.AllocFreeRegister()
		t.text.printft("mov %s, %d\n", t.ctx.GetRegister(reg, 8), v)
		return RegOperand{Reg: reg, Size: 8}, nil
	}
	return Imm(v), nil
}

func (t *X64Target) compileExpr(expr ast.Expression) (Operand, error) {
	switch e := expr.(type) {
	case *ast.IntLiteral:
		return t.newImm(e.Value)
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
		return StrOperand{Label: label, Len: len(decoded)}, nil
	case *ast.IdentLiteral:
		offset, ok := t.ctx.Get(t.info.Idents[e])
		if !ok {
			return nil, fmt.Errorf("x86-64: undefined literal")
		}
		size := t.sizeOf(t.info.Types[e])

		if size > 8 {
			return MemOperand{Offset: offset, Size: size}, nil
		} else {
			freeReg := t.ctx.AllocFreeRegister()
			t.text.printft("mov %s %s, [rbp-%d]\n", sizeLabels[size], t.ctx.GetRegister(freeReg, size), offset)

			return RegOperand{Reg: freeReg, Size: size}, nil
		}

	case *ast.BinaryExpr:
		return t.compileBinaryExpr(e)
	case *ast.CallExpr:
		return t.compileCallExpr(e)
	default:
		return nil, fmt.Errorf("x86-64: unsupported expression %T", expr)
	}
}

func (t *X64Target) compileLValue(expr ast.Expression) (int, error) {
	switch e := expr.(type) {
	case *ast.IdentLiteral:
		offset, ok := t.ctx.Get(t.info.Idents[e])
		if !ok {
			return 0, fmt.Errorf("x86-64: undeclared variable %s", e.Value)
		}
		return offset, nil
	default:
		return 0, fmt.Errorf("x86-64: unsupported assignment target %T", expr)
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
	pushed := t.ctx.Pushed()

	regs := t.ctx.AllocatedRegisters()
	for i := range regs {
		t.push(t.ctx.GetRegister(regs[i], 8))
	}
	align := t.ctx.Pushed()%2 == 1
	if align {
		t.alignStack(-8)
	}

	total := 0
	for _, arg := range expr.Args {
		op, err := t.compileExpr(arg)
		if err != nil {
			return nil, err
		}
		switch o := op.(type) {
		case Imm, RegOperand:
			text, err := t.operandText(op, 8)
			if err != nil {
				return nil, err
			}
			t.push(text)
			total++
		case StrOperand:
			reg := t.ctx.AllocFreeRegister()
			regStr := t.ctx.GetRegister(reg, 8)
			t.text.printft("lea %s, [rel %s]\n", regStr, o.Label)
			t.push(regStr)
			t.ctx.FreeRegister(reg)
			t.push(fmt.Sprint(o.Len))
			total += 2
		case MemOperand:
			if o.Size%8 != 0 {
				return nil, fmt.Errorf("x86-64: invalid parameter with MemOperand")
			}
			for i := 0; i < o.Size-1; i += 8 {
				t.push(t.mem(o.Offset-i, 8))
			}
			total += o.Size / 8
		}
		t.freeOperand(op)
	}

	if total > len(x64context.ParamOrder) {
		return nil, fmt.Errorf("x86-64: more than 6 parameters not supported")
	}
	for r := total - 1; r >= 0; r-- {
		t.pop(t.ctx.GetRegister(x64context.ParamOrder[r], 8))
	}

	sym, err := t.resolveCallee(expr.Name)
	if err != nil {
		return nil, err
	}

	fSym := t.funcSymbol(sym.Module, sym.Name)

	if sym.Module != t.module {
		t.prelude.define(fSym)
	}

	t.text.printft("call %s\n", fSym)
	reg := t.ctx.AllocFreeRegister()
	t.text.printft("mov %s, rax\n", t.ctx.GetRegister(reg, 8))

	if align {
		t.alignStack(8)
	}

	for i := range regs {
		t.pop(t.ctx.GetRegister(regs[len(regs)-i-1], 8))
	}

	if t.ctx.Pushed() != pushed {
		return nil, fmt.Errorf("x86-64: call stack unaligned")
	}

	return RegOperand{Reg: reg, Size: t.sizeOf(t.info.Types[expr])}, nil
}
