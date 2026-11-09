package analyser

import (
	"math/big"
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
	fixed := inlineBindings(decl.Body)
	for i, p := range decl.Params {
		if !p.Const {
			if produce, ok := a.propagatedArgument(expr.Args[i], ft.Params[i], p.Name, fixed, at); ok {
				substitutions[p.Name] = produce
				continue
			}
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
	if ft.Module != "" && ft.Module != savedModule {
		a.checkInlineVisibility(expr, decl, body, ft.Module)
	}
}

type bindings struct {
	changed map[string]bool
	defined map[string]bool
}

func inlineBindings(body *ast.BlockStmt) bindings {
	b := bindings{changed: map[string]bool{}, defined: map[string]bool{}}
	target := func(expr ast.Expression) {
		if id, ok := expr.(*ast.IdentLiteral); ok {
			b.changed[id.Value] = true
		}
	}
	define := func(names ...string) {
		for _, name := range names {
			if name != "" {
				b.defined[name] = true
			}
		}
	}
	ast.Inspect(body, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.AssignStmt:
			for _, t := range n.Targets {
				target(t)
			}
		case *ast.IncDecStmt:
			target(n.Target)
		case *ast.UnaryExpr:
			if n.Operator == "&" {
				target(n.Value)
			}
		case *ast.NativeStmt:
			for _, binding := range n.Bindings {
				if binding.Var != nil {
					b.changed[binding.Var.Value] = true
				}
			}
		case *ast.VarStmt:
			for _, v := range n.Vars {
				define(v.Name)
			}
		case *ast.ConstStmt:
			for _, v := range n.Vars {
				define(v.Name)
			}
		case *ast.ForStmt:
			define(n.Var, n.Var2)
		case *ast.FuncLiteral:
			for _, p := range n.Params {
				define(p.Name)
			}
		case *ast.CoalesceExpr:
			define(n.ErrorBind)
		case *ast.DeferStmt:
			define(n.ErrorBind)
		}
		return true
	})
	return b
}

func (a *Analyser) propagatedArgument(arg ast.Expression, tp Type, name string, fixed bindings, at ast.Position) (func() ast.Expression, bool) {
	if fixed.changed[name] || fixed.defined[name] {
		return nil, false
	}
	base := underlying(tp)
	if p, ok := base.(PrimitiveType); ok && p.Name != "bool" && !isInteger(base) {
		return nil, false
	}
	if _, isPrimitive := base.(PrimitiveType); !isPrimitive {
		if named, isNamed := tp.(NamedType); !isNamed || !named.Enum {
			return nil, false
		}
	}

	a.quiet = true
	value, ok := a.evalConst(arg)
	a.quiet = false
	if !ok || (value.Kind != ConstInt && value.Kind != ConstBool) {
		return nil, false
	}
	if value.Kind == ConstInt && value.Int.Sign() == 0 {
		return nil, false
	}
	converted, ok := a.convertConst(value, tp)
	if !ok {
		return nil, false
	}

	var enum *EnumValue
	if found, isMember := a.info.EnumValues[arg]; isMember {
		enum = &found
	}
	return func() ast.Expression {
		ref := &ast.ConstRef{}
		ref.SetPos(at.Line, at.Col)
		ref.SetEndPos(at.EndLine, at.EndCol)
		a.info.ConstRefs[ref] = ConstRefInfo{Value: converted, Type: tp, Enum: enum}
		return ref
	}, true
}

func (a *Analyser) checkInlineVisibility(site *ast.CallExpr, decl *ast.FuncDecl, body *ast.BlockStmt, module string) {
	info, ok := a.modules[module]
	if !ok {
		return
	}
	reported := map[string]bool{}
	ast.Inspect(body, func(node ast.Node) bool {
		id, ok := node.(*ast.IdentLiteral)
		if !ok {
			return true
		}
		sym := a.info.Idents[id]
		if sym == nil || sym.Module != module || reported[sym.Name] {
			return true
		}
		if sym.Kind != FUNC && !(sym.Kind == VAR && sym.Global) {
			return true
		}
		if exported, found := info.Exports[sym.Name]; found && exported == sym {
			return true
		}
		reported[sym.Name] = true
		a.errorf(site, "inline function %s uses %s, which is private to module %s, so it cannot be expanded here", decl.Name, sym.Name, module)
		return true
	})
}

func (a *Analyser) placeholderBody(fd *ast.FuncDecl, ft *FuncType, body *ast.BlockStmt) *ast.BlockStmt {
	values := map[string]ConstRefInfo{}
	for i, p := range fd.Params {
		if !p.Const || i >= len(ft.Params) {
			continue
		}
		if info, ok := placeholderFor(ft.Params[i]); ok {
			values[p.Name] = info
		}
	}
	return ast.Clone(body, ast.CloneOptions{
		Replace: func(node ast.Node) (ast.Node, bool) {
			id, ok := node.(*ast.IdentLiteral)
			if !ok {
				return nil, false
			}
			info, found := values[id.Value]
			if !found {
				return nil, false
			}
			ref := &ast.ConstRef{}
			line, col := id.Position()
			endLine, endCol := id.EndPosition()
			ref.SetPos(line, col)
			ref.SetEndPos(endLine, endCol)
			a.info.ConstRefs[ref] = info
			return ref, true
		},
	})
}

func placeholderFor(tp Type) (ConstRefInfo, bool) {
	if named, ok := tp.(NamedType); ok && named.Enum {
		if len(named.MemberOrder) == 0 {
			return ConstRefInfo{}, false
		}
		first := named.MemberOrder[0]
		value := named.Members[first]
		return ConstRefInfo{Value: constInt(big.NewInt(value)), Type: tp, Enum: &EnumValue{Type: named, Name: first, Value: value}}, true
	}
	base := underlying(tp)
	switch {
	case isInteger(base):
		return ConstRefInfo{Value: constInt(big.NewInt(1)), Type: tp}, true
	case isFloat(base):
		return ConstRefInfo{Value: ConstValue{Kind: ConstFloat, Float: 1}, Type: tp}, true
	}
	if p, ok := base.(PrimitiveType); ok {
		switch p.Name {
		case "bool":
			return ConstRefInfo{Value: ConstValue{Kind: ConstBool, Bool: true}, Type: tp}, true
		case "string":
			return ConstRefInfo{Value: ConstValue{Kind: ConstString}, Type: tp}, true
		}
	}
	return ConstRefInfo{}, false
}
