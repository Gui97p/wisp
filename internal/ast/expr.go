package ast

type BinaryExpr struct {
	Left     Expression
	Operator string
	Right    Expression
}

func (*BinaryExpr) expr() {}

type UnaryExpr struct {
	Value    Expression
	Operator string
}

func (*UnaryExpr) expr() {}
