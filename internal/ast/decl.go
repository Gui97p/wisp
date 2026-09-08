package ast

type Param struct {
	Name      string
	Type      string
	IsPointer bool
}

type ReturnType struct {
	Type      string
	IsPointer bool
}

type FuncDecl struct {
	Name        string
	Params      []Param
	Body        *BlockStmt
	ReturnTypes []ReturnType
}

func (*FuncDecl) decl() {}

type StructDecl struct {
	Name    string
	Members []Param
}

func (*StructDecl) decl() {}
