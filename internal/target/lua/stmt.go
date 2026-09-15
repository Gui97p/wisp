package lua

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

func compileStatement(b *strings.Builder, stmt ast.Statement) error {
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		return compileBlockStatement(b, s)
	case *ast.VarStmt:
		return compileVarStatement(b, s)
	case *ast.ReturnStmt:
		return compileReturnStatement(b, s)
	case *ast.ExpressionStmt:
		return compileExpressionStatement(b, s)
	case *ast.IfStmt:
		return compileIfStatement(b, s)
	// case *ast.ForStmt:
	// 	return compileForStatement(b, s)
	// case *ast.LoopStmt:
	// 	return compileLoopStatement(b, s)
	// case *ast.BreakStmt:
	// 	return compileBreakStatement(b, s)
	// case *ast.ContinueStmt:
	// 	return compileContinueStatement(b, s)
	// case *ast.AssignStmt:
	// 	return compileAssignStatement(b, s)
	// case *ast.IncDecStmt:
	// 	return compileIncDecStatement(b, s)
	default:
		return fmt.Errorf("lua: unsupported statement %T", stmt)
	}
}

func compileBlockStatement(b *strings.Builder, block *ast.BlockStmt) error {
	for _, stmt := range block.Statements {
		if err := compileStatement(b, stmt); err != nil {
			return err
		}
	}
	return nil
}

func compileVarStatement(b *strings.Builder, stmt *ast.VarStmt) error {
	b.WriteString("local ")
	for k, variable := range stmt.Vars {
		b.WriteString(variable.Name)
		if k != len(stmt.Vars)-1 {
			b.WriteByte(',')
		}
	}
	b.WriteString(" = ")
	for k, value := range stmt.Values {
		compileExpression(b, value)
		if k != len(stmt.Values)-1 {
			b.WriteByte(',')
		}
	}
	b.WriteByte('\n')
	return nil
}

func compileReturnStatement(b *strings.Builder, stmt *ast.ReturnStmt) error {
	b.WriteString("return ")
	for k, value := range stmt.Values {
		if err := compileExpression(b, value); err != nil {
			return err
		}
		if k != len(stmt.Values)-1 {
			b.WriteByte(',')
		}
	}

	b.WriteRune('\n')
	return nil
}

func compileExpressionStatement(b *strings.Builder, stmt *ast.ExpressionStmt) error {
	if err := compileExpression(b, stmt.Expr); err != nil {
		return err
	}
	b.WriteRune('\n')
	return nil
}

func compileIfStatement(b *strings.Builder, stmt *ast.IfStmt) error {
	b.WriteString("if ")
	compileExpression(b, stmt.Condition)
	b.WriteString(" then\n")
	compileStatement(b, stmt.Then)

	if stmt.Else != nil {
		b.WriteString("else\n")
		compileStatement(b, stmt.Else)
	}

	b.WriteString("end\n")

	return nil
}
