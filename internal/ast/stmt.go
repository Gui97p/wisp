package ast

type BlockStmt struct {
	Statements []Statement
}

func (*BlockStmt) stmt() {}
