package analyser

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

func (a *Analyser) registerInterfaceNames() {
	for _, d := range a.program.Declarations {
		decl, ok := d.(*ast.InterfaceDecl)
		if !ok {
			continue
		}
		a.currentFile = a.declFiles[d]

		if len(decl.TypeParams) > 0 {
			a.registerGenericInterface(decl)
			continue
		}

		it := &InterfaceType{Name: decl.Name, Module: a.module, Exported: decl.Exported, Doc: decl.Doc, key: qualifiedName(a.module, decl.Name)}
		file := a.currentFile
		scope := a.moduleScope
		it.fill = func() { a.fillInterface(it, decl, scope, file) }

		line, col := decl.Position()
		symbol := &Symbol{Name: decl.Name, Kind: INTERFACE, Type: it, Line: line, Col: col, File: a.currentFile, Module: a.module, Doc: decl.Doc, Used: true}
		if !a.scope.Define(symbol) {
			a.errorAlreadyDeclared(decl, INTERFACE, decl.Name)
		}
	}
}

func (a *Analyser) registerGenericInterface(decl *ast.InterfaceDecl) {
	g := &GenericDecl{Name: decl.Name, Module: a.module, File: a.currentFile, Scope: a.moduleScope, Params: decl.TypeParams, Interface: decl, Exported: decl.Exported}
	line, col := decl.Position()
	symbol := &Symbol{Name: decl.Name, Kind: INTERFACE, Type: GenericType{Decl: g}, Line: line, Col: col, File: a.currentFile, Module: a.module, Doc: decl.Doc, Generic: g, Used: true}
	g.Symbol = symbol
	if !a.scope.Define(symbol) {
		a.errorAlreadyDeclared(decl, INTERFACE, decl.Name)
	}
}

func (a *Analyser) fillInterfaces() {
	for _, d := range a.program.Declarations {
		if decl, ok := d.(*ast.InterfaceDecl); ok && len(decl.TypeParams) == 0 {
			if symbol, found := a.scope.Resolve(decl.Name); found {
				if it, isInterface := symbol.Type.(*InterfaceType); isInterface {
					it.ensure()
				}
			}
		}
	}
}

func (a *Analyser) instantiateInterface(node ast.Node, g *GenericDecl, args []TypeValue) Type {
	key := g.key(args)
	if it, ok := a.ifaceInstances[key]; ok {
		return it
	}
	if a.instanceDepth >= maxInstanceDepth {
		a.errorf(node, "the instantiation of %s is nested too deep", shortName(g.display(args))).
			WithHelp("an interface that contains itself with a bigger type never ends")
		return InvalidType{}
	}
	if !a.checkBounds(node, g, args) {
		return InvalidType{}
	}

	it := &InterfaceType{Name: g.display(args), Module: g.Module, Exported: g.Exported, Doc: g.Interface.Doc, Instance: &InstanceInfo{Decl: g, Args: args}, key: key}
	a.ifaceInstances[key] = it
	scope := a.bindScope(g.Scope, paramNames(g.Params), g.Params, args, 0)
	file := g.File
	it.fill = func() {
		a.instanceDepth++
		a.fillInterface(it, g.Interface, scope, file)
		a.instanceDepth--
	}
	return it
}

func (a *Analyser) fillInterface(it *InterfaceType, decl *ast.InterfaceDecl, scope *Scope, file string) {
	savedFile := a.currentFile
	a.currentFile = file
	defer func() { a.currentFile = savedFile }()

	seen := map[string]bool{}
	for _, m := range decl.Methods {
		if seen[m.Name] {
			a.errorRange(m.NameAt, "duplicated method %s in interface %s", m.Name, decl.Name)
			continue
		}
		seen[m.Name] = true

		sig := MethodSig{Name: m.Name, Doc: m.Doc, Fallible: m.Fallible}
		valid := true
		for _, p := range m.Params {
			at := ast.Position{Line: p.Line, Col: p.Col, EndLine: p.EndLine, EndCol: p.EndCol}
			if p.Any || p.Variadic {
				a.errorRange(at, "an interface method cannot take a variadic parameter yet")
				valid = false
				continue
			}
			t := a.resolveTypeRef(decl, scope, p.Type)
			if t == nil {
				valid = false
				continue
			}
			sig.Params = append(sig.Params, t)
			sig.ParamNames = append(sig.ParamNames, p.Name)
		}
		for _, r := range m.ReturnTypes {
			t := a.resolveTypeRef(decl, scope, r)
			if t == nil {
				valid = false
				continue
			}
			sig.Returns = append(sig.Returns, t)
		}
		if valid {
			it.Methods = append(it.Methods, sig)
		}
	}

	if decl.Constraint != nil {
		it.Expr = a.resolveConstraint(decl, decl.Constraint, scope, false)
	}
}

func (a *Analyser) resolveConstraint(node ast.Node, c *ast.Constraint, scope *Scope, underAnd bool) *ConstraintExpr {
	switch c.Kind {
	case ast.ConstraintOr:
		left := a.resolveConstraint(node, c.Left, scope, false)
		right := a.resolveConstraint(node, c.Right, scope, false)
		if left == nil || right == nil {
			return nil
		}
		if !left.typeSet() || !right.typeSet() {
			a.errorRange(c.At, "| joins types, and %s has methods: join interfaces with &", c.String())
			return nil
		}
		return &ConstraintExpr{Kind: ast.ConstraintOr, Left: left, Right: right}
	case ast.ConstraintAnd:
		left := a.resolveConstraint(node, c.Left, scope, true)
		right := a.resolveConstraint(node, c.Right, scope, true)
		if left == nil || right == nil {
			return nil
		}
		return &ConstraintExpr{Kind: ast.ConstraintAnd, Left: left, Right: right}
	case ast.ConstraintNot:
		inner := a.resolveConstraint(node, c.Left, scope, false)
		if inner == nil {
			return nil
		}
		if !underAnd {
			a.errorRange(c.At, "! only excludes types from what & already selected, as in Integer & !byte")
			return nil
		}
		if !inner.typeSet() {
			a.errorRange(c.At, "! excludes types, and %s has methods", c.Left.String())
			return nil
		}
		return &ConstraintExpr{Kind: ast.ConstraintNot, Left: inner}
	}

	ref := *c.Type
	if len(ref.Layers) == 0 && !ref.IsFunc {
		if symbol, found := a.lookupType(scope, ref.Module, ref.Name); found && symbol.Kind == INTERFACE {
			return a.interfaceAtom(node, c, symbol, ref, scope)
		}
	}
	t := a.resolveTypeRef(node, scope, ref)
	if t == nil {
		return nil
	}
	if _, invalid := t.(InvalidType); invalid {
		return nil
	}
	return &ConstraintExpr{Kind: ast.ConstraintType, Type: t}
}

func (a *Analyser) interfaceAtom(node ast.Node, c *ast.Constraint, symbol *Symbol, ref ast.TypeRef, scope *Scope) *ConstraintExpr {
	symbol.Used = true
	var t Type
	if g := symbol.Generic; g != nil {
		if len(ref.Args) == 0 {
			a.errorRange(c.At, "%s is generic: write %s<...>", qualified(ref.Module, ref.Name), qualified(ref.Module, ref.Name))
			return nil
		}
		t = a.instantiateStructRef(node, g, ref.Args, scope)
	} else {
		if len(ref.Args) > 0 {
			a.errorRange(c.At, "%s takes no type arguments", qualified(ref.Module, ref.Name))
			return nil
		}
		t = symbol.Type
	}
	it, isInterface := t.(*InterfaceType)
	if !isInterface {
		return nil
	}
	if it.filling {
		a.errorRange(c.At, "interface %s contains itself", it.Name)
		return nil
	}
	it.ensure()
	return &ConstraintExpr{Kind: ast.ConstraintType, Iface: it}
}

func (a *Analyser) checkBounds(node ast.Node, g *GenericDecl, args []TypeValue) bool {
	ok := true
	var scope *Scope
	for i, param := range g.Params {
		if param.Constraint == nil || param.Const || i >= len(args) || args[i].Type == nil {
			continue
		}
		if _, invalid := args[i].Type.(InvalidType); invalid {
			continue
		}
		if scope == nil {
			scope = a.bindScope(g.Scope, paramNames(g.Params), g.Params, args, 0)
		}
		savedFile := a.currentFile
		a.currentFile = g.File
		expr := a.resolveConstraint(g.node(), param.Constraint, scope, false)
		a.currentFile = savedFile
		if expr == nil {
			ok = false
			continue
		}
		if reason := a.satisfies(args[i].Type, expr); reason != "" {
			a.errorf(node, "%s does not satisfy %s, the constraint of %s in %s", args[i].Type, expr, param.Name, g.Name).
				WithHelp("%s", reason)
			ok = false
		}
	}
	return ok
}

type implEntry struct {
	iface *InterfaceType
	ref   ast.TypeRef
}

func (a *Analyser) newImplSet(refs []ast.TypeRef, typ Type, node ast.Node, scope *Scope, args []TypeValue) *ImplSet {
	if len(refs) == 0 {
		return nil
	}
	set := &ImplSet{refs: refs, node: node, typ: typ, file: a.currentFile}
	for _, arg := range args {
		if invalid, ok := arg.Type.(InvalidType); ok && invalid.Name != "" {
			set.skip = true
		}
	}
	set.init = func() {
		savedFile := a.currentFile
		a.currentFile = set.file
		defer func() { a.currentFile = savedFile }()
		for _, ref := range refs {
			if it := a.resolveImplemented(node, ref, scope); it != nil {
				set.explicit = append(set.explicit, implEntry{it, ref})
				set.add(it)
			}
		}
	}
	return set
}

func (a *Analyser) resolveImplemented(node ast.Node, ref ast.TypeRef, scope *Scope) *InterfaceType {
	symbol, found := a.lookupType(scope, ref.Module, ref.Name)
	if !found {
		a.errorAtRef(node, ref, "unknown interface %s", qualified(ref.Module, ref.Name))
		return nil
	}
	if symbol.Kind != INTERFACE {
		a.errorAtRef(node, ref, "%s is not an interface", qualified(ref.Module, ref.Name))
		return nil
	}
	symbol.Used = true

	var t Type
	if g := symbol.Generic; g != nil {
		if len(ref.Args) == 0 {
			a.errorAtRef(node, ref, "%s is generic: write %s<...>", qualified(ref.Module, ref.Name), qualified(ref.Module, ref.Name))
			return nil
		}
		t = a.instantiateStructRef(node, g, ref.Args, scope)
	} else {
		if len(ref.Args) > 0 {
			a.errorAtRef(node, ref, "%s takes no type arguments", qualified(ref.Module, ref.Name))
			return nil
		}
		t = symbol.Type
	}
	it, isInterface := t.(*InterfaceType)
	if !isInterface {
		return nil
	}
	if it.ConstraintOnly() {
		a.errorAtRef(node, ref, "%s is a constraint, not a contract: a type does not implement it", it.Name).
			WithHelp("it names types, or uses | or !, so a type satisfies it by being one of them")
		return nil
	}
	return it
}

func (a *Analyser) verifyImpl(set *ImplSet) {
	if set == nil || set.verified {
		return
	}
	set.verified = true
	set.ensure()
	if set.skip {
		return
	}

	savedFile := a.currentFile
	a.currentFile = set.file
	defer func() { a.currentFile = savedFile }()

	for _, entry := range set.explicit {
		problems := a.methodProblems(set.typ, entry.iface)
		if entry.iface.Exported && typeExported(set.typ) {
			methods := MethodsOf(set.typ)
			for _, want := range entry.iface.AllMethods() {
				if have, ok := methods[want.Name]; ok && !have.Exported {
					problems = append(problems, "method "+want.Name+" must be public, because both "+elementName(set.typ)+" and the interface "+entry.iface.Name+" are public")
				}
			}
		}
		if len(problems) > 0 {
			a.errorAtRef(set.node, entry.ref, "%s declares implements %s, but it does not", elementName(set.typ), entry.iface.Name).
				WithHelp("%s", strings.Join(problems, "\n"))
		}
	}
}

func (a *Analyser) checkImplements() {
	for _, d := range a.program.Declarations {
		a.currentFile = a.declFiles[d]
		switch decl := d.(type) {
		case *ast.StructDecl:
			if len(decl.TypeParams) == 0 && len(decl.Implements) > 0 {
				a.verifyDeclared(decl.Name)
			}
		case *ast.TypeDecl:
			if len(decl.TypeParams) == 0 && len(decl.Implements) > 0 {
				a.verifyDeclared(decl.Name)
			}
		}
	}
}

func (a *Analyser) verifyDeclared(name string) {
	if symbol, ok := a.scope.Resolve(name); ok {
		a.verifyImpl(implSetOf(symbol.Type))
	}
}

func (a *Analyser) verifyPendingImpls() bool {
	if len(a.pendingImpls) == 0 {
		return false
	}
	pending := a.pendingImpls
	a.pendingImpls = nil
	for _, set := range pending {
		a.verifyImpl(set)
	}
	return true
}

func typeExported(t Type) bool {
	if p, ok := t.(PointerType); ok {
		t = p.Element
	}
	switch v := t.(type) {
	case *StructType:
		return v.Exported
	case NamedType:
		return v.Exported
	}
	return false
}

type instanceContext struct {
	gen   *GenericDecl
	args  []TypeValue
	exprs []*ConstraintExpr
	done  bool
}

func (a *Analyser) boundGuarantees(recv Type, name string) bool {
	ctx := a.instanceContext
	if ctx == nil || ctx.gen == nil || len(ctx.args) != len(ctx.gen.Params) {
		return false
	}
	if !ctx.done {
		ctx.done = true
		ctx.exprs = make([]*ConstraintExpr, len(ctx.gen.Params))
		scope := a.bindScope(ctx.gen.Scope, paramNames(ctx.gen.Params), ctx.gen.Params, ctx.args, 0)
		savedFile := a.currentFile
		savedErrors := len(a.errors)
		a.currentFile = ctx.gen.File
		for i, param := range ctx.gen.Params {
			if param.Constraint != nil && !param.Const {
				ctx.exprs[i] = a.resolveConstraint(ctx.gen.node(), param.Constraint, scope, false)
			}
		}
		a.currentFile = savedFile
		a.errors = a.errors[:savedErrors]
	}
	for i, expr := range ctx.exprs {
		if expr == nil || ctx.args[i].Type == nil || typeKey(unwrapPointer(ctx.args[i].Type)) != typeKey(unwrapPointer(recv)) {
			continue
		}
		for _, m := range expr.requiredMethods() {
			if m.Name == name {
				return true
			}
		}
	}
	return false
}

func unwrapPointer(t Type) Type {
	if p, ok := t.(PointerType); ok {
		return p.Element
	}
	return t
}

func (a *Analyser) checkInterfaceTypeCase(expr *ast.BinaryExpr, held *InterfaceType) Type {
	boolType := PrimitiveType{Name: "bool"}

	if _, isGroup := a.typeGroup(expr.Right); isGroup {
		a.errorf(expr.Right, "signed and unsigned name integers, and %s holds a pointer to a type of its own", held.Name)
		return boolType
	}

	t, ok := a.labelType(expr.Right)
	if !ok {
		return InvalidType{}
	}

	pointer, isPointer := t.(PointerType)
	if !isPointer || pointer.Nullable {
		a.errorf(expr.Right, "a case on %s names a pointer to a type, as in case *Buffer b: it holds a pointer", held.Name)
		return InvalidType{}
	}
	if reason := a.satisfiesInterface(pointer, held); reason != "" {
		a.errorf(expr.Right, "%s can never be what %s holds", t, held.Name).WithHelp("%s", reason)
		return InvalidType{}
	}

	a.info.TypeCases[expr] = &TypeCase{Type: t, Iface: held}
	return boolType
}

func (a *Analyser) vtableEntries(node ast.Node, pointer PointerType, it *InterfaceType) ([]VtableEntry, bool) {
	concrete := pointer.Element
	key := typeKey(concrete) + "|" + it.key + "|" + a.module
	if entries, ok := a.vtables[key]; ok {
		return entries, true
	}

	methods := MethodsOf(concrete)
	var entries []VtableEntry
	for _, m := range it.AllMethods() {
		ft := methods[m.Name]
		if ft == nil {
			return nil, false
		}
		if ft.Inline {
			a.errorf(node, "method %s of %s is inline, so it has no address to put in %s", m.Name, elementName(pointer), it.Name).
				WithHelp("an inline method works as a constraint, not behind an interface value")
			return nil, false
		}

		owner, _, _ := methodOwner(concrete)
		if ft.Template != nil {
			owner = ft.Template.Owner
			ft = a.instanceOfMethod(node, ft)
		}
		if _, byPointer := ft.Receiver.(PointerType); byPointer {
			entries = append(entries, VtableEntry{Owner: owner, Module: ft.Module, Name: m.Name})
			continue
		}
		entries = append(entries, a.valueReceiverWrapper(node, concrete, owner, m.Name, ft))
	}
	a.vtables[key] = entries
	return entries, true
}

func (a *Analyser) valueReceiverWrapper(node ast.Node, concrete Type, owner, name string, ft *FuncType) VtableEntry {
	wrapperName := "__iface_" + name
	key := owner + "." + wrapperName + "|" + a.module
	if entry, ok := a.ifaceWrappers[key]; ok {
		return entry
	}
	entry := VtableEntry{Owner: owner, Module: a.module, Name: wrapperName}
	a.ifaceWrappers[key] = entry

	place := func(target ast.Node) {
		line, col := node.Position()
		endLine, endCol := node.EndPosition()
		switch v := target.(type) {
		case *ast.IdentLiteral:
			v.SetPos(line, col)
			v.SetEndPos(endLine, endCol)
		case *ast.DerefExpr:
			v.SetPos(line, col)
			v.SetEndPos(endLine, endCol)
		case *ast.MemberExpr:
			v.SetPos(line, col)
			v.SetEndPos(endLine, endCol)
		case *ast.CallExpr:
			v.SetPos(line, col)
			v.SetEndPos(endLine, endCol)
		case *ast.PropagateExpr:
			v.SetPos(line, col)
			v.SetEndPos(endLine, endCol)
		case *ast.ReturnStmt:
			v.SetPos(line, col)
			v.SetEndPos(endLine, endCol)
		case *ast.ExpressionStmt:
			v.SetPos(line, col)
			v.SetEndPos(endLine, endCol)
		case *ast.BlockStmt:
			v.SetPos(line, col)
			v.SetEndPos(endLine, endCol)
		}
	}

	params := make([]ast.Param, len(ft.Params))
	names := make([]string, len(ft.Params))
	args := make([]ast.Expression, len(ft.Params))
	for i := range ft.Params {
		names[i] = fmt.Sprintf("__arg%d", i)
		params[i] = ast.Param{Name: names[i]}
		ident := &ast.IdentLiteral{Value: names[i]}
		place(ident)
		args[i] = ident
	}

	receiver := &ast.IdentLiteral{Value: "__recv"}
	place(receiver)
	deref := &ast.DerefExpr{Value: receiver}
	place(deref)
	member := &ast.MemberExpr{Object: deref, Field: name}
	place(member)
	var call ast.Expression = &ast.CallExpr{Name: member, Args: args}
	place(call)
	if ft.Fallible {
		call = &ast.PropagateExpr{Value: call}
		place(call)
	}

	var body ast.Statement
	if len(ft.Returns) == 0 {
		body = &ast.ExpressionStmt{Expr: call}
	} else {
		body = &ast.ReturnStmt{Values: []ast.Expression{call}}
	}
	place(body)
	block := &ast.BlockStmt{Statements: []ast.Statement{body}}
	place(block)

	decl := &ast.FuncDecl{
		Name:        wrapperName,
		Receiver:    &ast.Param{Name: "__recv", Type: ast.TypeRef{Name: owner}},
		Params:      params,
		Body:        block,
		ReturnTypes: []ast.TypeRef{},
		Fallible:    ft.Fallible,
		Generated:   true,
		Instance:    owner + "." + wrapperName,
	}
	place(decl)

	wrapper := &FuncType{
		Name: wrapperName, Params: ft.Params, ParamNames: names, Returns: ft.Returns, Fallible: ft.Fallible,
		Receiver: PointerType{Element: concrete}, Module: a.module, DeclFile: a.currentFile, Decl: decl,
	}
	a.info.FuncReturns[decl] = ft.Returns
	a.info.VarTypes[decl] = ft.Params
	if !a.phantomBuilding || a.phantoms {
		a.program.Declarations = append(a.program.Declarations, decl)
		a.declFiles[decl] = a.currentFile
	}
	gen := &GenericDecl{Name: wrapperName, Module: a.module, File: a.currentFile, Scope: a.moduleScope}
	a.pending = append(a.pending, pendingBody{decl: decl, ft: wrapper, recv: PointerType{Element: concrete}, scope: a.moduleScope, gen: gen, at: node, file: a.currentFile, depth: a.bodyDepth + 1, phantom: a.phantomBuilding, waive: true})
	return entry
}
