package x64

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

type binaryOp struct {
	immediate string
	register  string
}

var binaryOps = map[string]binaryOp{
	"+": {
		immediate: "add rax, %d",
		register:  "add rax, rcx",
	},
	"-": {
		immediate: "sub rax, %d",
		register:  "sub rax, rcx",
	},
	"*": {
		immediate: "imul rax, %d",
		register:  "imul rax, rcx",
	},
}

func compileBinaryExpression(b *strings.Builder, expr *ast.BinaryExpr) error {
	if err := compileExpression(b, expr.Left); err != nil {
		return err
	}

	cmd, ok := binaryOps[expr.Operator]
	if !ok {
		return fmt.Errorf("x86-64: unsupported binary operator %q", expr.Operator)
	}

	switch right := expr.Right.(type) {
	case *ast.IntLiteral:
		fmt.Fprintf(b, fmt.Sprintf("\t%s\n", cmd.immediate), right.Value)
	default:
		b.WriteString("\tpush rax\n")

		if err := compileExpression(b, expr.Right); err != nil {
			return err
		}

		b.WriteString("\tmov rcx, rax\n")
		b.WriteString("\tpop rax\n")
		fmt.Fprintf(b, "\t%s\n", cmd.register)
	}

	return nil
}
