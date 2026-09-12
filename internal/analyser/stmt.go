package analyser

import "github.com/Gui97p/wisp/internal/ast"

func (a *Analyser) checkBlock(block *ast.BlockStmt) {
	a.info.Scopes[block] = a.scope

	for _, stmt := range block.Statements {
		a.checkStmt(stmt)
	}
}

func (a *Analyser) checkStmt(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.ExpressionStmt:
		a.checkExpr(s.Expr)
	case *ast.BlockStmt:
		a.enterScope()
		a.checkBlock(s)
		a.exitScope()
	}
}
