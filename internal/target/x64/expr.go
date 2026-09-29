package x64

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/ast"
)

func (t *X64Target) compileExpr(expr ast.Expression) (Operand, error) {
	switch e := expr.(type) {
	case *ast.IntLiteral:
		return Imm(e.Value), nil
	case *ast.BoolLiteral:
		if e.Value {
			return Imm(1), nil
		} else {
			return Imm(0), nil
		}
	case *ast.CharLiteral:
		return Imm(e.Value), nil
	case *ast.IdentLiteral:
		offset, ok := t.ctx.Get(t.info.Idents[e])
		if !ok {
			return nil, fmt.Errorf("x86-64: undefined literal")
		}

		freeReg := t.ctx.AllocFreeRegister()
		size := t.sizeOf(t.info.Types[e])
		t.text.printft("mov %s, [rbp-%d]\n", t.ctx.GetRegister(freeReg, size), offset)

		return RegOperand{Reg: freeReg, Size: size}, nil

	case *ast.BinaryExpr:
		return t.compileBinaryExpr(e)
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
	left, err := t.compileExpr(expr.Left)
	if err != nil {
		return nil, err
	}
	right, err := t.compileExpr(expr.Right)
	if err != nil {
		return nil, err
	}

	size := t.sizeOf(t.info.Types[expr.Left])
	leftReg := t.ensureRegister(left, size)
	reg := t.ctx.GetRegister(leftReg, size)
	opText := t.operandText(right, size)
	if r, ok := right.(RegOperand); ok {
		t.ctx.FreeRegister(r.Reg)
	}
	switch expr.Operator {
	case "+":
		t.text.printft("add %s, %s\n", reg, opText)
	case "-":
		t.text.printft("sub %s, %s\n", reg, opText)
	case "*":
		t.text.printft("imul %s, %s\n", reg, opText)
	case "==", "!=", "<", "<=", ">", ">=":

		size := t.sizeOf(t.info.Types[expr])
		resultReg := t.ctx.AllocFreeRegister()
		resultRegStr := t.ctx.GetRegister(resultReg, size)

		t.text.printft("xor %s, %s\n", resultRegStr, resultRegStr)
		t.text.printft("cmp %s, %s\n", reg, opText)
		t.text.printft("%s %s\n", setccOperators[expr.Operator], resultRegStr)

		return RegOperand{Reg: resultReg, Size: size}, nil
	}
	return RegOperand{Reg: leftReg, Size: size}, nil
}
