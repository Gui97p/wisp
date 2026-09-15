package x64

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

func compileExpression(b *strings.Builder, expr ast.Expression) error {
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		return compileBinaryExpression(b, e)
	case *ast.IntLiteral:
		return compileIntLiteral(b, e)
	default:
		return fmt.Errorf("x86-64: unsupported expression %T", expr)
	}
}

func compileBinaryExpression(b *strings.Builder, expr *ast.BinaryExpr) error {
	if err := compileExpression(b, expr.Left); err != nil {
		return fmt.Errorf("x86-64: failed to compile expression")
	}

	switch right := expr.Right.(type) {
	case *ast.IntLiteral:
		return compileBinaryImmediate(b, expr.Operator, right)
	default:
		b.WriteString("\tpush rax\n")

		if err := compileExpression(b, expr.Right); err != nil {
			return err
		}

		b.WriteString("\tpop rcx\n")

		switch expr.Operator {
		case "+":
			b.WriteString("\tadd rax, rcx\n")
		}
	}
	return nil
}

func compileBinaryImmediate(b *strings.Builder, operator string, right *ast.IntLiteral) error {
	switch operator {
	case "+":
		fmt.Fprintf(b, "\tadd rax, %d\n", right.Value)

	case "-":
		fmt.Fprintf(b, "\tsub rax, %d\n", right.Value)

	default:
		return fmt.Errorf("x86-64: failed to compile expression")
	}

	return nil
}
