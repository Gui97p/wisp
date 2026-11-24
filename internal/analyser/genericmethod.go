package analyser

import (
	"strconv"

	"github.com/Gui97p/wisp/internal/ast"
)

const hiddenSelf = "__wisp_self"

func (a *Analyser) registerGenericMethod(fd *ast.FuncDecl) {
	if fd.Receiver.Type.Module != "" {
		a.errorAtRef(fd, fd.Receiver.Type, "a method with type parameters is declared on a type of this module")
		return
	}
	if fd.Inline {
		a.errorf(fd, "an inline method cannot have type parameters yet").WithHelp("drop inline, or write an inline function that takes the value")
		return
	}

	params := make([]ast.TypeParam, 0, len(fd.TypeParams)+len(fd.Receiver.Type.Args))
	params = append(params, fd.TypeParams...)
	var owner []ast.TypeParam
	if sym, ok := a.resolveTop(fd.Receiver.Type.Name); ok && sym.Generic != nil {
		owner = sym.Generic.Params
	}
	for i, arg := range fd.Receiver.Type.Args {
		if arg.Type == nil {
			continue
		}
		param := ast.TypeParam{Name: arg.Type.Name}
		if i < len(owner) {
			param = owner[i]
			param.Name = arg.Type.Name
		}
		params = append(params, param)
	}

	receiver := *fd.Receiver
	if receiver.Name == "" {
		receiver.Name = hiddenSelf
	}
	hidden := &ast.FuncDecl{
		Tagged:      fd.Tagged,
		Doc:         fd.Doc,
		Name:        fd.Receiver.Type.Name + "__" + fd.Name,
		TypeParams:  params,
		Params:      append([]ast.Param{receiver}, fd.Params...),
		ReturnTypes: fd.ReturnTypes,
		Fallible:    fd.Fallible,
		Body:        fd.Body,
		Exported:    fd.Exported,
		Reach:       fd.Reach,
		NameAt:      fd.NameAt,
		MethodOf:    fd.Name,
		Selector:    fd.Selector,
		SelectorAt:  fd.SelectorAt,
	}
	line, col := fd.Position()
	hidden.SetPos(line, col)
	endLine, endCol := fd.EndPosition()
	hidden.SetEndPos(endLine, endCol)

	a.registerGenericFunc(hidden)
	if a.mangled == nil {
		a.mangled = map[string]string{}
	}
	a.mangled[hidden.Name] = fd.Name
	if sym, ok := a.resolveTop(hidden.Name); ok {
		if sym.Generic != nil {
			sym.Generic.MethodName = fd.Name
		}
		if a.genericMethodSyms == nil {
			a.genericMethodSyms = map[*ast.FuncDecl]*Symbol{}
		}
		a.genericMethodSyms[fd] = sym
	}
}

func (a *Analyser) resolveTop(name string) (*Symbol, bool) {
	if sym, ok := a.moduleScope.Resolve(name); ok {
		return sym, true
	}
	return nil, false
}

func (a *Analyser) registerGenericMethodEntry(fd *ast.FuncDecl, recvType Type) {
	sym := a.genericMethodSyms[fd]
	if sym == nil {
		return
	}
	methods := MethodsOf(recvType)
	if methods == nil {
		a.errorf(fd, "cannot declare method on %s", recvType)
		return
	}
	if _, exists := methods[fd.Name]; exists {
		a.errorAtName(fd, "method %s already declared for %s", fd.Name, recvType)
		return
	}
	line, _ := fd.Position()
	methods[fd.Name] = &FuncType{
		Name: fd.Name, Receiver: recvType, Exported: fd.Exported,
		InternalDir: reachDir(fd.Reach, a.currentFile), Private: fd.Reach == ast.ReachFile, Module: a.module, DeclFile: a.currentFile, DeclLine: line,
		Decl: fd, GenericMethod: sym,
	}
}

func (a *Analyser) callGenericMethod(expr *ast.CallExpr, member *ast.MemberExpr, objType Type, ft *FuncType, explicit []ast.TypeArg) []Type {
	_, pointerRecv := ft.Receiver.(PointerType)
	_, objectPtr := objType.(PointerType)
	if pointerRecv && !objectPtr && !isAddressable(member.Object) {
		a.errorf(expr, "method %s needs a pointer receiver, but the value is a temporary", member.Field)
	}
	owner, _, ownerExported := methodOwner(objType)
	if ft.Module != a.module && !a.waiveAccess && !(ft.Exported && ownerExported) {
		a.errorf(expr, "method %s of %s is not public", member.Field, owner)
	} else if ft.Private && ft.DeclFile != "" && a.currentFile != "" && ft.DeclFile != a.currentFile {
		a.errorf(expr, "method %s of %s is private to %s", member.Field, owner, ft.DeclFile)
	}

	if a.mangled == nil {
		a.mangled = map[string]string{}
	}
	a.mangled[ft.GenericMethod.Name] = member.Field

	declared := ft.GenericMethod.Generic.Func.Params[1:]
	variadic := len(declared) > 0 && (declared[len(declared)-1].Variadic || declared[len(declared)-1].Any)
	fixed := len(declared)
	if variadic {
		fixed--
	}
	if len(expr.Args) < fixed || (!variadic && len(expr.Args) > fixed) {
		a.evalArgTypes(expr)
		a.errorf(expr, "method %s expects %s, got %d", member.Field, argumentCount(fixed, variadic), len(expr.Args))
		return []Type{InvalidType{}}
	}

	receiver := a.receiverArgument(member.Object, objType, pointerRecv, objectPtr)
	name := &ast.IdentLiteral{Value: ft.GenericMethod.Name}
	name.SetPos(member.FieldAt.Line, member.FieldAt.Col)
	name.SetEndPos(member.FieldAt.EndLine, member.FieldAt.EndCol)
	if member.FieldAt.Line == 0 {
		line, col := member.Position()
		name.SetPos(line, col)
		endLine, endCol := member.EndPosition()
		name.SetEndPos(endLine, endCol)
	}
	expr.Generic = member
	expr.Name = name
	expr.Args = append([]ast.Expression{receiver}, expr.Args...)
	return a.callGeneric(expr, ft.GenericMethod.Type, explicit)
}

func argumentCount(fixed int, variadic bool) string {
	noun := "arguments"
	if fixed == 1 {
		noun = "argument"
	}
	if variadic {
		return "at least " + itoa(fixed) + " " + noun
	}
	return itoa(fixed) + " " + noun
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
