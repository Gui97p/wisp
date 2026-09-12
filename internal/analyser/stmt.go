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
		if call, ok := s.Expr.(*ast.CallExpr); ok {
			a.checkCallExpr(call)
		} else {
			a.checkExpr(s.Expr)
		}
	// case *ast.VarStmt:
	// case *ast.ReturnStmt:
	// case *ast.IfStmt:
	// case *ast.ForStmt:
	// case *ast.LoopStmt:
	// case *ast.AssignStmt:
	// case *ast.IncDecStmt:
	// case *ast.MultiVarStmt:
	// case *ast.BreakStmt, *ast.ContinueStmt:
	case *ast.BlockStmt:
		a.enterScope()
		a.checkBlock(s)
		a.exitScope()
	}
}
