package ast

type IntLiteral struct {
	Value int64
}

func (*IntLiteral) expr() {}

type FloatLiteral struct {
	Value float64
}

func (*FloatLiteral) expr() {}

type StringLiteral struct {
	Value string
}

func (*StringLiteral) expr() {}

type CharLiteral struct {
	Value byte
}

func (*CharLiteral) expr() {}

type BoolLiteral struct {
	Value int64
}

func (*BoolLiteral) expr() {}
