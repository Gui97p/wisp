package ast

type Param struct {
	Name      string
	Type      string
	IsPointer bool
}

type FuncDecl struct {
	Name        string
	Params      []Param
	Body        *BlockStmt
	ReturnTypes []string
}

func (*FuncDecl) decl() {}

type StructDecl struct {
	Name   string
	Params []Param
}

func (*StructDecl) decl() {}
