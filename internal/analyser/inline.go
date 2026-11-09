package analyser

import (
	"slices"

	"github.com/Gui97p/wisp/internal/ast"
)

type InlineCall struct {
	Decl     *ast.FuncDecl
	Body     *ast.BlockStmt
	Receiver *Symbol
	Params   []*Symbol
	ArgIndex []int
	Returns  []Type
	Fallible bool
}

type ConstRefInfo struct {
	Value ConstValue
	Type  Type
	Enum  *EnumValue
}

func (a *Analyser) checkInlineDecl(fd *ast.FuncDecl) {
	for _, p := range fd.Params {
		if p.Const && !fd.Inline {
			a.errorf(fd, "parameter %s is const, and only an inline function can take const parameters", p.Name)
		}
		if fd.Inline && (p.Variadic || p.Any) {
			a.errorf(fd, "an inline function cannot take ...%s yet", p.Name)
		}
	}
	if fd.Inline && fd.Name == "main" && fd.Receiver == nil {
		a.error(fd, "main cannot be inline")
	}
}

func (a *Analyser) errorFile() string {
	if a.siteFile != "" {
		return a.siteFile
	}
	return a.currentFile
}

func (a *Analyser) constantArgument(arg ast.Expression, at ast.Position) (func() ast.Expression, bool) {
	switch arg.(type) {
	case *ast.IntLiteral, *ast.FloatLiteral, *ast.StringLiteral, *ast.CharLiteral, *ast.BoolLiteral:
		return func() ast.Expression {
			return ast.Clone(arg, ast.CloneOptions{At: &at})
		}, true
	}

	value, ok := a.evalConst(arg)
	if !ok {
		return nil, false
	}
	tp := a.info.Types[arg]
	var enum *EnumValue
	if found, isMember := a.info.EnumValues[arg]; isMember {
		enum = &found
	}
	return func() ast.Expression {
		ref := &ast.ConstRef{}
		ref.SetPos(at.Line, at.Col)
		ref.SetEndPos(at.EndLine, at.EndCol)
		a.info.ConstRefs[ref] = ConstRefInfo{Value: value, Type: tp, Enum: enum}
		return ref
	}, true
}

func (a *Analyser) expandInline(expr *ast.CallExpr, ft *FuncType) {
	if !ft.Inline || ft.Decl == nil || ft.Decl.Body == nil || a.info.Inlines[expr] != nil {
		return
	}
	decl := ft.Decl
	if slices.Contains(a.inlineStack, decl) {
		return
	}
	if len(expr.Args) != len(decl.Params) || len(ft.Params) != len(decl.Params) {
		return
	}

	line, col := expr.Position()
	endLine, endCol := expr.EndPosition()
	at := ast.Position{Line: line, Col: col, EndLine: endLine, EndCol: endCol}

	call := &InlineCall{Decl: decl, Returns: ft.Returns, Fallible: ft.Fallible}
	substitutions := map[string]func() ast.Expression{}
	for i, p := range decl.Params {
		if !p.Const {
			call.ArgIndex = append(call.ArgIndex, i)
			continue
		}
		produce, ok := a.constantArgument(expr.Args[i], at)
		if !ok {
			a.errorf(expr.Args[i], "argument %d of %s must be a constant, because %s is a const parameter", i+1, decl.Name, p.Name)
			return
		}
		substitutions[p.Name] = produce
	}

	body := ast.Clone(decl.Body, ast.CloneOptions{
		At: &at,
		Replace: func(node ast.Node) (ast.Node, bool) {
			if id, ok := node.(*ast.IdentLiteral); ok {
				if produce, found := substitutions[id.Value]; found {
					return produce(), true
				}
			}
			return nil, false
		},
	})

	savedScope, savedModuleScope := a.scope, a.moduleScope
	savedModule, savedFile, savedSite := a.module, a.currentFile, a.siteFile
	savedReturns, savedFallible := a.currentReturns, a.currentFallible
	savedLoops, savedYields := a.loopFrames, a.yieldFrames
	savedInline := a.currentInline
	savedUntil := a.untilScope

	if a.siteFile == "" {
		a.siteFile = a.currentFile
	}
	a.scope, a.moduleScope = ft.Scope, ft.Scope
	a.module = ft.Module
	if a.module == "" {
		a.module = savedModule
	}
	a.currentFile = ft.DeclFile
	a.currentReturns, a.currentFallible = ft.Returns, ft.Fallible
	a.loopFrames, a.yieldFrames, a.untilScope = nil, nil, nil
	a.currentInline = call
	a.inlineStack = append(a.inlineStack, decl)

	a.enterScope()
	if decl.Receiver != nil {
		recv := &Symbol{Name: decl.Receiver.Name, Kind: PARAM, Type: ft.Receiver, File: a.currentFile, Used: true}
		if recv.Name != "" {
			a.scope.Define(recv)
		}
		call.Receiver = recv
	}
	for _, i := range call.ArgIndex {
		param := &Symbol{Name: decl.Params[i].Name, Kind: PARAM, Type: ft.Params[i], File: a.currentFile, Used: true}
		a.scope.Define(param)
		call.Params = append(call.Params, param)
	}

	a.checkBlock(body)
	if len(ft.Returns) > 0 && !blockTerminates(body) {
		a.errorAtEnd(expr, "the inline function %s can end without returning", decl.Name)
	}
	a.exitScope()

	a.inlineStack = a.inlineStack[:len(a.inlineStack)-1]
	a.currentInline = savedInline
	a.loopFrames, a.yieldFrames, a.untilScope = savedLoops, savedYields, savedUntil
	a.currentReturns, a.currentFallible = savedReturns, savedFallible
	a.currentFile, a.siteFile = savedFile, savedSite
	a.module = savedModule
	a.scope, a.moduleScope = savedScope, savedModuleScope

	call.Body = body
	a.info.Inlines[expr] = call
}
