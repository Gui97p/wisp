package ast

type Node interface{}

type Declaration interface {
	Node
	decl()
}

type Statement interface {
	Node
	stmt()
}

type Expression interface {
	Node
	expr()
}

type Program struct {
	Declarations []Declaration
}
