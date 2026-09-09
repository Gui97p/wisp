package ast

type ExpressionStmt struct {
	Expr Expression
}

func (*ExpressionStmt) stmt() {}

type BlockStmt struct {
	Statements []Statement
}

func (*BlockStmt) stmt() {}

type VarStmt struct {
	Name      string
	Type      string
	Value     Expression
	IsPointer bool
}

func (*VarStmt) stmt() {}

type ReturnStmt struct {
	Values []Expression
}

func (*ReturnStmt) stmt() {}

type IfStmt struct {
	Condition Expression
	Then      *BlockStmt
	Else      Statement
}

func (*IfStmt) stmt() {}

type ForStmt struct {
	Label string
	Var   string
	Start Expression
	End   Expression
	Step  Expression
	Body  *BlockStmt
}

func (*ForStmt) stmt() {}

type LoopStmt struct {
	Condition Expression
	Body      *BlockStmt
	Value     Expression
}

func (*LoopStmt) stmt() {}

type BreakStmt struct {
	Label string
}

func (*BreakStmt) stmt() {}

type ContinueStmt struct {
	Label string
}

func (*ContinueStmt) stmt() {}

type AssignStmt struct {
	Name  string
	Op    string
	Value Expression
}

func (*AssignStmt) stmt() {}
