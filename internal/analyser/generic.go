package analyser

import (
	"fmt"
	"hash/fnv"
	"math/big"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/diag"
)

const maxInstanceDepth = 64

type GenericDecl struct {
	Name      string
	Module    string
	File      string
	Scope     *Scope
	Params    []ast.TypeParam
	Struct    *ast.StructDecl
	Func      *ast.FuncDecl
	Alias     *ast.TypeDecl
	Sum       *ast.TypeDecl
	Interface *ast.InterfaceDecl
	Methods   []*ast.FuncDecl
	Exported  bool
	Symbol    *Symbol
}

type TypeValue struct {
	Type  Type
	Value *ConstValue
	Label string
}

func (v TypeValue) String() string {
	if v.Label != "" {
		return v.Label
	}
	if v.Value != nil {
		return v.Value.String()
	}
	if v.Type == nil {
		return "?"
	}
	return v.Type.String()
}

type InstanceInfo struct {
	Decl *GenericDecl
	Args []TypeValue
}

type GenericType struct {
	Decl *GenericDecl
}

func (g GenericType) String() string {
	names := make([]string, len(g.Decl.Params))
	for i, p := range g.Decl.Params {
		names[i] = p.Name
	}
	return g.Decl.Name + "<" + strings.Join(names, ", ") + ">"
}

func (g GenericType) Equals(other Type) bool {
	o, ok := other.(GenericType)
	return ok && o.Decl == g.Decl
}

type MethodTemplate struct {
	Decl   *ast.FuncDecl
	Scope  *Scope
	Owner  string
	Struct *StructType
	Gen    *GenericDecl
}

type pendingBody struct {
	decl  *ast.FuncDecl
	ft    *FuncType
	recv  Type
	scope *Scope
	gen   *GenericDecl
	at    ast.Node
	file  string
	depth int
	args  string

	phantom   bool
	checkNote string
	argValues []TypeValue
	waive     bool
}

func typeKey(t Type) string {
	switch v := t.(type) {
	case *StructType:
		if v.key != "" {
			return v.key
		}
		return "struct:" + qualifiedName(v.Module, v.Name)
	case NamedType:
		return "named:" + qualifiedName(v.Module, v.Name)
	case PointerType:
		return "*" + typeKey(v.Element)
	case SpanType:
		return "[]" + typeKey(v.Element)
	case ArrayType:
		return fmt.Sprintf("[%d]%s", v.Size, typeKey(v.Element))
	case *FuncType:
		parts := make([]string, 0, len(v.Params)+len(v.Returns))
		for _, p := range v.Params {
			parts = append(parts, typeKey(p))
		}
		parts = append(parts, "=>")
		for _, r := range v.Returns {
			parts = append(parts, typeKey(r))
		}
		return "(" + strings.Join(parts, ",") + ")"
	}
	if t == nil {
		return "?"
	}
	if opt, ok := optionalElement(t); ok {
		return "?" + typeKey(opt)
	}
	return t.String()
}

func (g *GenericDecl) key(args []TypeValue) string {
	parts := make([]string, len(args))
	for i, v := range args {
		if v.Label != "" {
			parts[i] = "phantom:" + v.Label
		} else if v.Value != nil {
			parts[i] = "const:" + v.Value.String()
		} else {
			parts[i] = typeKey(v.Type)
		}
	}
	return qualifiedName(g.Module, g.Name) + "<" + strings.Join(parts, ",") + ">"
}

func (g *GenericDecl) display(args []TypeValue) string {
	parts := make([]string, len(args))
	for i, v := range args {
		parts[i] = v.String()
	}
	return g.Name + "<" + strings.Join(parts, ", ") + ">"
}

func mangle(name, key string) string {
	var b strings.Builder
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	h := fnv.New32a()
	h.Write([]byte(key))
	return fmt.Sprintf("%s_%08x", strings.Trim(b.String(), "_"), h.Sum32())
}

func isGenericMethod(fd *ast.FuncDecl) bool {
	return fd.Receiver != nil && len(fd.Receiver.Type.Args) > 0
}

func isGenericDecl(fd *ast.FuncDecl) bool {
	return len(fd.TypeParams) > 0 || isGenericMethod(fd)
}

func (a *Analyser) registerGenericStruct(sd *ast.StructDecl) {
	g := &GenericDecl{Name: sd.Name, Module: a.module, File: a.currentFile, Scope: a.moduleScope, Params: sd.TypeParams, Struct: sd, Exported: sd.Exported}
	line, col := sd.Position()
	symbol := &Symbol{Name: sd.Name, Kind: STRUCT, Type: GenericType{Decl: g}, Line: line, Col: col, File: a.currentFile, Module: a.module, Doc: sd.Doc, Generic: g}
	g.Symbol = symbol
	if !a.scope.Define(symbol) {
		a.errorAlreadyDeclared(sd, STRUCT, sd.Name)
	}
}

func (a *Analyser) registerGenericAlias(td *ast.TypeDecl) {
	g := &GenericDecl{Name: td.Name, Module: a.module, File: a.currentFile, Scope: a.moduleScope, Params: td.TypeParams, Alias: td, Exported: td.Exported}
	line, col := td.Position()
	symbol := &Symbol{Name: td.Name, Kind: TYPE, Type: GenericType{Decl: g}, Line: line, Col: col, File: a.currentFile, Module: a.module, Doc: td.Doc, Generic: g}
	g.Symbol = symbol
	if !a.scope.Define(symbol) {
		a.errorAlreadyDeclared(td, TYPE, td.Name)
	}
}

func (a *Analyser) registerGenericSum(td *ast.TypeDecl) {
	g := &GenericDecl{Name: td.Name, Module: a.module, File: a.currentFile, Scope: a.moduleScope, Params: td.TypeParams, Sum: td, Exported: td.Exported}
	line, col := td.Position()
	symbol := &Symbol{Name: td.Name, Kind: STRUCT, Type: GenericType{Decl: g}, Line: line, Col: col, File: a.currentFile, Module: a.module, Doc: td.Doc, Generic: g}
	g.Symbol = symbol
	if !a.scope.Define(symbol) {
		a.errorAlreadyDeclared(td, STRUCT, td.Name)
	}
}

func (a *Analyser) registerGenericFunc(fd *ast.FuncDecl) {
	g := &GenericDecl{Name: fd.Name, Module: a.module, File: a.currentFile, Scope: a.moduleScope, Params: fd.TypeParams, Func: fd, Exported: fd.Exported}
	line, col := fd.Position()
	symbol := &Symbol{Name: fd.Name, Kind: FUNC, Type: GenericType{Decl: g}, Line: line, Col: col, File: a.currentFile, Module: a.module, Doc: fd.Doc, Generic: g}
	g.Symbol = symbol
	if !a.scope.Define(symbol) {
		a.errorAlreadyDeclared(fd, FUNC, fd.Name)
	}
}

func (a *Analyser) attachGenericMethods() {
	for _, d := range a.program.Declarations {
		a.attachGenericMethod(d)
	}
}

func (a *Analyser) attachGenericMethod(d ast.Declaration) {
	fd, ok := d.(*ast.FuncDecl)
	if !ok || !isGenericMethod(fd) {
		return
	}
	a.currentFile = a.declFiles[d]
	ref := fd.Receiver.Type
	symbol, found := a.scope.Resolve(ref.Name)
	if ref.Module != "" || !found || symbol.Generic == nil || (symbol.Generic.Struct == nil && symbol.Generic.Sum == nil) {
		a.errorAtRef(fd, ref, "%s is not a generic struct of this module", ref.String())
		return
	}
	g := symbol.Generic
	if len(ref.Args) != len(g.Params) {
		a.errorAtRef(fd, ref, "%s takes %d type arguments, and the receiver gives %d", g.Name, len(g.Params), len(ref.Args))
		return
	}
	for _, arg := range ref.Args {
		if arg.Type == nil || arg.Type.Module != "" || len(arg.Type.Args) > 0 || len(arg.Type.Layers) > 0 {
			a.errorAtRef(fd, ref, "the names between <> in a receiver declare the type parameters: write func (*%s<T> value) ...", g.Name)
			return
		}
	}
	g.Methods = append(g.Methods, fd)
}

func (a *Analyser) bindScope(parent *Scope, names []string, params []ast.TypeParam, args []TypeValue, line int) *Scope {
	scope := NewScope(parent)
	for i, name := range names {
		if i >= len(args) {
			break
		}
		sym := &Symbol{Name: name, Used: true, Line: 0}
		if i < len(params) && params[i].Const {
			sym.Kind = CONST
			sym.Type = args[i].Type
			sym.Const = args[i].Value
			sym.Phantom = args[i].Label
		} else {
			sym.Kind = TYPE
			sym.Type = args[i].Type
		}
		scope.Define(sym)
	}
	return scope
}

func paramNames(params []ast.TypeParam) []string {
	names := make([]string, len(params))
	for i, p := range params {
		names[i] = p.Name
	}
	return names
}

func (a *Analyser) evalTypeArgs(node ast.Node, g *GenericDecl, given []ast.TypeArg, scope *Scope) ([]TypeValue, bool) {
	if len(given) != len(g.Params) {
		a.errorf(node, "%s takes %d type arguments, got %d", g.Name, len(g.Params), len(given))
		return nil, false
	}
	return a.evalArgList(node, g, given, scope)
}

func (a *Analyser) evalArgList(node ast.Node, g *GenericDecl, given []ast.TypeArg, scope *Scope) ([]TypeValue, bool) {
	values := make([]TypeValue, len(given))
	ok := true
	for i, arg := range given {
		param := g.Params[i]
		if param.Const {
			value, good := a.evalValueArg(node, g, param, arg, scope)
			if !good {
				ok = false
				continue
			}
			values[i] = value
			continue
		}
		if arg.Type == nil {
			a.errorf(arg.Expr, "%s of %s is a type parameter: give a type", param.Name, g.Name)
			ok = false
			continue
		}
		t := a.resolveTypeRef(node, scope, *arg.Type)
		if t == nil {
			ok = false
			continue
		}
		if invalid, isInvalid := t.(InvalidType); isInvalid && invalid.Name == "" {
			ok = false
			continue
		}
		values[i] = TypeValue{Type: t}
	}
	return values, ok
}

func (a *Analyser) evalValueArg(node ast.Node, g *GenericDecl, param ast.TypeParam, arg ast.TypeArg, scope *Scope) (TypeValue, bool) {
	paramType := a.resolveTypeRef(node, g.Scope, param.Type)
	if paramType == nil {
		return TypeValue{}, false
	}
	expr := arg.Expr
	if expr == nil {
		ref := arg.Type
		if ref.Module != "" || len(ref.Args) > 0 || len(ref.Layers) > 0 {
			a.errorf(node, "%s of %s is a value parameter: give a constant", param.Name, g.Name)
			return TypeValue{}, false
		}
		expr = &ast.IdentLiteral{Value: ref.Name}
		expr.(*ast.IdentLiteral).SetPos(ref.Line, ref.Col)
		expr.(*ast.IdentLiteral).SetEndPos(ref.EndLine, ref.EndCol)
	}
	saved := a.scope
	a.scope = scope
	t := a.checkExpr(expr)
	value, ok := a.evalConst(expr)
	label := ""
	if id, isIdent := expr.(*ast.IdentLiteral); isIdent {
		if sym, found := scope.Resolve(id.Value); found {
			label = sym.Phantom
		}
	}
	a.scope = saved
	if _, invalid := t.(InvalidType); invalid {
		return TypeValue{}, false
	}
	if !ok {
		a.errorf(expr, "%s of %s is a value parameter: give a constant", param.Name, g.Name)
		return TypeValue{}, false
	}
	converted, fits := a.convertConst(value, paramType)
	if !fits {
		a.errorf(expr, "the value %s does not fit %s, the type of %s", value.String(), paramType.String(), param.Name)
		return TypeValue{}, false
	}
	return TypeValue{Type: paramType, Value: &converted, Label: label}, true
}

func (a *Analyser) instantiateStructRef(node ast.Node, g *GenericDecl, args []ast.TypeArg, scope *Scope) Type {
	values, ok := a.evalTypeArgs(node, g, args, scope)
	if !ok {
		return InvalidType{}
	}
	switch {
	case g.Interface != nil:
		return a.instantiateInterface(node, g, values)
	case g.Alias != nil:
		return a.instantiateAlias(node, g, values)
	case g.Sum != nil:
		return a.instantiateSum(node, g, values)
	}
	return a.instantiateStruct(node, g, values)
}

func (a *Analyser) instantiateSum(node ast.Node, g *GenericDecl, args []TypeValue) Type {
	key := g.key(args)
	if st, ok := a.instances[key]; ok {
		return st
	}
	if a.instanceDepth >= maxInstanceDepth {
		a.errorf(node, "the instantiation of %s is nested too deep", shortName(g.display(args))).
			WithHelp("a generic type that contains itself with a bigger type never ends")
		return InvalidType{}
	}

	if !a.checkBounds(node, g, args) {
		return InvalidType{}
	}

	st := a.buildSum(g.Sum, g.display(args), g.Module, filepath.Dir(g.File))
	st.key = key
	st.Instance = &InstanceInfo{Decl: g, Args: args}
	a.instances[key] = st

	scope := a.bindScope(g.Scope, paramNames(g.Params), g.Params, args, 0)
	savedImplFile := a.currentFile
	a.currentFile = g.File
	st.Impl = a.newImplSet(g.Sum.Implements, st, g.Sum, scope, args)
	a.currentFile = savedImplFile
	if st.Impl != nil {
		a.pendingImpls = append(a.pendingImpls, st.Impl)
	}
	a.instanceDepth++
	savedFile := a.currentFile
	a.currentFile = g.File
	a.fillSum(g.Sum, st, scope)
	a.currentFile = savedFile
	a.instanceDepth--

	st.methodInit = func() { a.registerInstanceMethods(g, st, args) }
	return st
}

func (a *Analyser) instantiateAlias(node ast.Node, g *GenericDecl, args []TypeValue) Type {
	key := g.key(args)
	if t, ok := a.aliases[key]; ok {
		return t
	}
	if a.instanceDepth >= maxInstanceDepth {
		a.errorf(node, "the instantiation of %s is nested too deep", shortName(g.display(args))).
			WithHelp("a generic type that contains itself with a bigger type never ends")
		return InvalidType{}
	}

	if !a.checkBounds(node, g, args) {
		return InvalidType{}
	}

	scope := a.bindScope(g.Scope, paramNames(g.Params), g.Params, args, 0)
	a.instanceDepth++
	savedFile := a.currentFile
	a.currentFile = g.File
	underlying := a.resolveTypeRef(g.Alias, scope, g.Alias.Underlying)
	a.currentFile = savedFile
	a.instanceDepth--
	if underlying == nil {
		return InvalidType{}
	}
	if _, invalid := underlying.(InvalidType); invalid {
		return InvalidType{}
	}

	named := NamedType{
		Name:       g.display(args),
		Underlying: underlying,
		Methods:    make(map[string]*FuncType),
		Module:     g.Module,
		Dir:        filepath.Dir(g.File),
		Exported:   g.Exported,
		Instance:   &InstanceInfo{Decl: g, Args: args},
	}
	savedImplFile := a.currentFile
	a.currentFile = g.File
	named.Impl = a.newImplSet(g.Alias.Implements, named, g.Alias, scope, args)
	a.currentFile = savedImplFile
	if named.Impl != nil {
		a.pendingImpls = append(a.pendingImpls, named.Impl)
	}
	a.aliases[key] = named
	return named
}

func (a *Analyser) instantiateStruct(node ast.Node, g *GenericDecl, args []TypeValue) Type {
	key := g.key(args)
	if st, ok := a.instances[key]; ok {
		return st
	}
	if a.instanceDepth >= maxInstanceDepth {
		a.errorf(node, "the instantiation of %s is nested too deep", shortName(g.display(args))).
			WithHelp("a generic type that contains itself with a bigger type never ends")
		return InvalidType{}
	}

	if !a.checkBounds(node, g, args) {
		return InvalidType{}
	}

	st := &StructType{
		Name:      g.display(args),
		Fields:    make(map[string]Type),
		Order:     make([]string, 0, len(g.Struct.Members)),
		Methods:   make(map[string]*FuncType),
		FieldDocs: make(map[string]string),
		FieldDirs: make(map[string]string),
		Module:    g.Module,
		Dir:       filepath.Dir(g.File),
		Exported:  g.Exported,
		Packed:    g.Struct.TagNamed("packed") != nil,
		key:       key,
		Instance:  &InstanceInfo{Decl: g, Args: args},
	}
	a.instances[key] = st

	scope := a.bindScope(g.Scope, paramNames(g.Params), g.Params, args, 0)

	savedImplFile := a.currentFile
	a.currentFile = g.File
	st.Impl = a.newImplSet(g.Struct.Implements, st, g.Struct, scope, args)
	a.currentFile = savedImplFile
	if st.Impl != nil {
		a.pendingImpls = append(a.pendingImpls, st.Impl)
	}

	a.instanceDepth++
	savedFile := a.currentFile
	a.currentFile = g.File
	for _, member := range g.Struct.Members {
		fieldType := a.resolveTypeRef(g.Struct, scope, member.Type)
		if fieldType == nil {
			continue
		}
		if _, ok := st.Fields[member.Name]; ok {
			a.errorf(g.Struct, "duplicated field %s in struct %s", member.Name, g.Name)
			continue
		}
		st.Fields[member.Name] = fieldType
		st.FieldDocs[member.Name] = member.Doc
		if dir := reachDir(member.Reach, g.File); dir != "" {
			st.FieldDirs[member.Name] = dir
		}
		st.Order = append(st.Order, member.Name)
	}
	a.currentFile = savedFile
	a.instanceDepth--
	st.complete = true

	st.methodInit = func() { a.registerInstanceMethods(g, st, args) }
	return st
}

func (a *Analyser) resolveSignature(fd *ast.FuncDecl, scope *Scope) (params []Type, names []string, variadic bool, returns []Type) {
	for _, p := range fd.Params {
		var t Type = BoxedType{}
		if !p.Any {
			t = a.resolveTypeRef(fd, scope, p.Type)
		}
		if t == nil {
			continue
		}
		params = append(params, t)
		names = append(names, p.Name)
		if p.Variadic {
			variadic = true
		}
	}
	for _, r := range fd.ReturnTypes {
		t := a.resolveTypeRef(fd, scope, r)
		if t == nil {
			t = InvalidType{}
		}
		returns = append(returns, t)
	}
	return
}

func (a *Analyser) registerInstanceMethods(g *GenericDecl, st *StructType, args []TypeValue) {
	savedFile := a.currentFile
	defer func() { a.currentFile = savedFile }()
	a.currentFile = g.File

	owner := mangle(st.Name, st.key)
	for _, fd := range g.Methods {
		names := make([]string, len(fd.Receiver.Type.Args))
		for i, arg := range fd.Receiver.Type.Args {
			names[i] = arg.Type.Name
		}
		scope := a.bindScope(g.Scope, names, g.Params, args, 0)

		recvType := a.resolveTypeRef(fd, scope, fd.Receiver.Type)
		if recvType == nil {
			continue
		}
		if _, exists := st.Methods[fd.Name]; exists {
			a.errorAtName(fd, "method %s already declared for %s", fd.Name, st.Name)
			continue
		}

		params, paramNames, variadic, returns := a.resolveSignature(fd, scope)
		line, _ := fd.Position()
		receiverName := ""
		if fd.Receiver != nil {
			receiverName = fd.Receiver.Name
		}
		st.Methods[fd.Name] = &FuncType{
			Name: fd.Name, Params: params, ParamNames: paramNames, ReceiverName: receiverName, Returns: returns,
			Variadic: variadic, Fallible: fd.Fallible, Receiver: recvType, Exported: fd.Exported,
			InternalDir: reachDir(fd.Reach, g.File), Module: g.Module, DeclFile: g.File, DeclLine: line, Decl: fd,
			Scope: scope, Inline: fd.Inline,
			Template: &MethodTemplate{Decl: fd, Scope: scope, Owner: owner, Struct: st, Gen: g},
		}
	}
}

func (a *Analyser) instanceOfMethod(node ast.Node, tmpl *FuncType) *FuncType {
	if tmpl.Template == nil || tmpl.Inline {
		return tmpl
	}
	key := tmpl.Template.Owner + "." + tmpl.Name
	cache := a.methodCache()
	if ft, ok := cache[key]; ok {
		return ft
	}

	t := tmpl.Template
	clone := ast.Clone(t.Decl, ast.CloneOptions{})
	clone.Receiver.Type.Name = t.Owner
	clone.Receiver.Type.Args = nil
	clone.Exported = false
	clone.Instance = t.Struct.Name + "." + tmpl.Name

	ft := *tmpl
	ft.Module = a.module
	ft.Decl = clone
	ft.Template = t
	cache[key] = &ft

	a.info.FuncReturns[clone] = ft.Returns
	a.info.VarTypes[clone] = ft.Params
	if !a.phantomBuilding || a.phantoms {
		a.program.Declarations = append(a.program.Declarations, clone)
		a.declFiles[clone] = t.Gen.File
	}
	a.pending = append(a.pending, pendingBody{decl: clone, ft: &ft, recv: tmpl.Receiver, scope: t.Scope, gen: t.Gen, at: node, file: a.currentFile, depth: a.bodyDepth + 1, args: instanceArgs(t.Struct), phantom: a.phantomBuilding, checkNote: a.phantomNote, argValues: structArgs(t.Struct)})
	return &ft
}

func structArgs(st *StructType) []TypeValue {
	if st.Instance == nil {
		return nil
	}
	return st.Instance.Args
}

func instanceArgs(st *StructType) string {
	if st.Instance == nil {
		return ""
	}
	names := paramNames(st.Instance.Decl.Params)
	parts := make([]string, len(names))
	for i, n := range names {
		parts[i] = n + " = " + st.Instance.Args[i].String()
	}
	return strings.Join(parts, ", ")
}

func (a *Analyser) drainInstances() {
	for {
		for len(a.pending) > 0 {
			next := a.pending[0]
			a.pending = a.pending[1:]
			a.checkInstanceBody(next)
		}
		if !a.verifyPendingImpls() && len(a.pending) == 0 {
			return
		}
	}
}

func (a *Analyser) checkInstanceBody(p pendingBody) {
	if p.depth > maxInstanceDepth {
		savedFile := a.currentFile
		a.currentFile = p.file
		a.errorf(p.at, "the instantiation of %s is nested too deep", shortName(p.decl.Instance)).
			WithHelp("a generic function that calls itself with a bigger type never ends")
		a.currentFile = savedFile
		return
	}

	savedScope, savedModuleScope, savedModule, savedFile := a.scope, a.moduleScope, a.module, a.currentFile
	savedDepth := a.bodyDepth
	savedBuilding, savedNote, savedContext, savedWaive := a.phantomBuilding, a.phantomNote, a.instanceContext, a.waiveAccess
	a.waiveAccess = p.waive
	a.instanceContext = &instanceContext{gen: p.gen, args: p.argValues}
	a.scope, a.moduleScope, a.module, a.currentFile = p.scope, p.gen.Scope, p.gen.Module, p.gen.File
	a.bodyDepth = p.depth
	a.phantomBuilding, a.phantomNote = p.phantom, p.checkNote
	start := len(a.errors)

	a.checkFuncBodyOf(p.decl, p.ft, p.recv)

	for i := start; i < len(a.errors); i++ {
		callLine, callCol := p.at.Position()
		endLine, endCol := p.at.EndPosition()
		switch {
		case !p.phantom:
			a.errors[i].Note(p.file, callLine, callCol, endLine, endCol, "with %s, instantiated here", p.args)
		case p.checkNote != "":
			a.errors[i].Note(p.file, callLine, callCol, endLine, endCol, "checking the body with %s", p.checkNote)
		}
	}
	a.scope, a.moduleScope, a.module, a.currentFile = savedScope, savedModuleScope, savedModule, savedFile
	a.bodyDepth = savedDepth
	a.phantomBuilding, a.phantomNote, a.instanceContext, a.waiveAccess = savedBuilding, savedNote, savedContext, savedWaive
}

func (g *GenericDecl) paramIndex(name string) int {
	for i, p := range g.Params {
		if p.Name == name {
			return i
		}
	}
	return -1
}

func (a *Analyser) callGeneric(expr *ast.CallExpr, calleeType Type, explicit []ast.TypeArg) []Type {
	gt, ok := calleeType.(GenericType)
	if !ok {
		a.evalArgTypes(expr)
		if _, invalid := calleeType.(InvalidType); !invalid {
			a.errorf(expr, "%s is not a generic function", calleeType.String())
		}
		return []Type{InvalidType{}}
	}
	g := gt.Decl
	if g.Func == nil {
		a.evalArgTypes(expr)
		a.errorf(expr, "%s is a generic struct, not a function: write %s<...>{...} to build one", g.Name, g.Name)
		return []Type{InvalidType{}}
	}

	argTypes := a.evalArgTypes(expr)
	bound := make([]*TypeValue, len(g.Params))

	if len(explicit) > len(g.Params) {
		a.errorf(expr, "%s takes %d type arguments, got %d", g.Name, len(g.Params), len(explicit))
		return []Type{InvalidType{}}
	}
	if len(explicit) > 0 {
		values, ok := a.evalArgList(expr, g, explicit, a.scope)
		if !ok {
			return []Type{InvalidType{}}
		}
		for i := range values {
			v := values[i]
			bound[i] = &v
		}
	}

	if hint := a.rangeHint; hint != nil && expr.Range && bound[0] == nil && allUntyped(argTypes) {
		bound[0] = &TypeValue{Type: hint}
	}
	a.rangeHint = nil

	if !a.inferTypeArgs(expr, g, argTypes, bound) {
		return []Type{InvalidType{}}
	}

	values := make([]TypeValue, len(bound))
	for i, b := range bound {
		if b == nil {
			a.errorf(expr, "cannot infer %s from the arguments of %s", g.Params[i].Name, g.Name).
				WithHelp("name it: %s<%s>(...)", g.Name, strings.Join(paramNames(g.Params), ", "))
			return []Type{InvalidType{}}
		}
		values[i] = *b
	}

	sym, ft := a.instantiateFunc(expr, g, values)
	if ft == nil {
		return []Type{InvalidType{}}
	}

	syn := &ast.IdentLiteral{Value: sym.Name, Synthetic: false}
	line, col := expr.Name.Position()
	endLine, endCol := expr.Name.EndPosition()
	syn.SetPos(line, col)
	syn.SetEndPos(endLine, endCol)
	a.info.Idents[syn] = sym
	a.info.Types[syn] = ft
	if expr.Generic == nil {
		expr.Generic = expr.Name
	}
	expr.Name = syn

	a.checkCallArgsWith(expr, ft, argTypes)
	a.expandInline(expr, ft)
	return callResults(ft)
}

func (a *Analyser) inferTypeArgs(expr *ast.CallExpr, g *GenericDecl, argTypes []Type, bound []*TypeValue) bool {
	fd := g.Func
	for i, arg := range argTypes {
		var param *ast.Param
		switch {
		case i < len(fd.Params):
			param = &fd.Params[i]
		case len(fd.Params) > 0 && fd.Params[len(fd.Params)-1].Variadic:
			param = &fd.Params[len(fd.Params)-1]
		default:
			continue
		}
		if param.Any {
			continue
		}
		if expr.Spread && i == len(argTypes)-1 {
			if span, ok := arg.(SpanType); ok {
				arg = span.Element
			}
		}
		if msg := a.unify(g, param.Type, arg, bound); msg != "" {
			a.errorf(expr.Args[min(i, len(expr.Args)-1)], "%s", msg)
			return false
		}
	}
	return true
}

func (a *Analyser) unify(g *GenericDecl, ref ast.TypeRef, t Type, bound []*TypeValue) string {
	switch t.(type) {
	case InvalidType, NullType, AnonType, nil:
		return ""
	}

	for _, layer := range ref.Layers {
		switch {
		case layer.Pointer:
			pt, ok := t.(PointerType)
			if !ok {
				return ""
			}
			t = pt.Element
		case layer.Optional:
			element, ok := optionalElement(t)
			if !ok {
				return ""
			}
			t = element
		case layer.IsSpan:
			switch v := t.(type) {
			case SpanType:
				t = v.Element
			case ArrayType:
				t = v.Element
			default:
				return ""
			}
		default:
			at, ok := t.(ArrayType)
			if !ok {
				return ""
			}
			if id, isIdent := layer.Expr.(*ast.IdentLiteral); isIdent {
				if idx := g.paramIndex(id.Value); idx >= 0 && g.Params[idx].Const {
					value := ConstValue{Kind: ConstInt, Int: bigInt(at.Size)}
					if msg := bindValue(g, bound, idx, value, a.resolveTypeRef(g.Func, g.Scope, g.Params[idx].Type)); msg != "" {
						return msg
					}
				}
			}
			t = at.Element
		}
	}

	if ref.IsFunc {
		ft, ok := t.(*FuncType)
		if !ok {
			return ""
		}
		for i := range ref.FuncParams {
			if i < len(ft.Params) {
				if msg := a.unify(g, ref.FuncParams[i], ft.Params[i], bound); msg != "" {
					return msg
				}
			}
		}
		for i := range ref.FuncReturns {
			if i < len(ft.Returns) {
				if msg := a.unify(g, ref.FuncReturns[i], ft.Returns[i], bound); msg != "" {
					return msg
				}
			}
		}
		return ""
	}

	if ref.Module == "" && len(ref.Args) == 0 {
		if idx := g.paramIndex(ref.Name); idx >= 0 && !g.Params[idx].Const {
			untyped := isUntyped(t)
			t = defaultType(t)
			if bound[idx] != nil {
				if untyped && bound[idx].Type != nil && fitsUntyped(bound[idx].Type, t) {
					return ""
				}
				if !bound[idx].Type.Equals(t) {
					return fmt.Sprintf("%s is both %s and %s in this call", ref.Name, bound[idx].Type.String(), t.String())
				}
				return ""
			}
			bound[idx] = &TypeValue{Type: t}
		}
		return ""
	}

	if len(ref.Args) > 0 {
		instance := instanceOf(t)
		if instance == nil {
			return ""
		}
		symbol, found := a.lookupType(g.Scope, ref.Module, ref.Name)
		if !found || symbol.Generic != instance.Decl {
			return ""
		}
		for i, arg := range ref.Args {
			if i >= len(instance.Args) {
				break
			}
			given := instance.Args[i]
			switch {
			case arg.Type != nil && given.Value == nil:
				if msg := a.unify(g, *arg.Type, given.Type, bound); msg != "" {
					return msg
				}
			case arg.Type != nil && given.Value != nil:
				if idx := g.paramIndex(arg.Type.Name); idx >= 0 && g.Params[idx].Const && arg.Type.Module == "" {
					if msg := bindValue(g, bound, idx, *given.Value, given.Type); msg != "" {
						return msg
					}
				}
			case arg.Expr != nil && given.Value != nil:
				if id, isIdent := arg.Expr.(*ast.IdentLiteral); isIdent {
					if idx := g.paramIndex(id.Value); idx >= 0 && g.Params[idx].Const {
						if msg := bindValue(g, bound, idx, *given.Value, given.Type); msg != "" {
							return msg
						}
					}
				}
			}
		}
	}
	return ""
}

func instanceOf(t Type) *InstanceInfo {
	switch v := t.(type) {
	case *StructType:
		return v.Instance
	case NamedType:
		return v.Instance
	}
	return nil
}

func bindValue(g *GenericDecl, bound []*TypeValue, idx int, value ConstValue, tp Type) string {
	if bound[idx] != nil {
		if bound[idx].Value == nil || bound[idx].Value.String() != value.String() {
			return fmt.Sprintf("%s is both %s and %s in this call", g.Params[idx].Name, bound[idx].String(), value.String())
		}
		return ""
	}
	v := value
	bound[idx] = &TypeValue{Type: tp, Value: &v}
	return ""
}

func (a *Analyser) instantiateFunc(node ast.Node, g *GenericDecl, args []TypeValue) (*Symbol, *FuncType) {
	key := g.key(args)
	cache := a.funcCache()
	if sym, ok := cache[key]; ok {
		return sym, sym.Type.(*FuncType)
	}
	if a.bodyDepth >= maxInstanceDepth {
		a.errorf(node, "the instantiation of %s is nested too deep", shortName(g.display(args))).
			WithHelp("a generic function that calls itself with a bigger type never ends")
		return nil, nil
	}

	if !a.checkBounds(node, g, args) {
		return nil, nil
	}

	fd := g.Func
	name := mangle(g.Name, key)
	scope := a.bindScope(g.Scope, paramNames(g.Params), g.Params, args, 0)

	savedFile := a.currentFile
	a.currentFile = g.File
	params, paramNames, variadic, returns := a.resolveSignature(fd, scope)
	a.currentFile = savedFile

	for _, r := range returns {
		if invalid, ok := r.(InvalidType); ok && invalid.Name == "" {
			return nil, nil
		}
	}

	clone := ast.Clone(fd, ast.CloneOptions{})
	clone.Name = name
	clone.TypeParams = nil
	clone.Exported = false
	clone.Instance = g.display(args)

	module := a.home
	if fd.Inline {
		module = g.Module
	}
	ft := &FuncType{
		Name: name, Params: params, ParamNames: paramNames, Returns: returns, Variadic: variadic, Fallible: fd.Fallible,
		Module: module, DeclFile: g.File, Decl: clone, Scope: scope, Inline: fd.Inline,
	}
	if fd.Inline {
		ft.Generic = &InstanceInfo{Decl: g, Args: args}
	}
	line, col := fd.Position()
	sym := &Symbol{Name: name, Kind: FUNC, Type: ft, Line: line, Col: col, File: g.File, Module: module, Doc: fd.Doc, Used: true}
	cache[key] = sym

	a.info.FuncReturns[clone] = returns
	a.info.VarTypes[clone] = params
	if !fd.Inline {
		if !a.phantomBuilding || a.phantoms {
			a.program.Declarations = append(a.program.Declarations, clone)
			a.declFiles[clone] = g.File
		}
		argText := make([]string, len(g.Params))
		for i, p := range g.Params {
			argText[i] = p.Name + " = " + args[i].String()
		}
		a.pending = append(a.pending, pendingBody{decl: clone, ft: ft, scope: scope, gen: g, at: node, file: a.currentFile, depth: a.bodyDepth + 1, args: strings.Join(argText, ", "), phantom: a.phantomBuilding, checkNote: a.phantomNote, argValues: args})
	}
	return sym, ft
}

func bigInt(n int64) *big.Int {
	return big.NewInt(n)
}

var typeRefType = reflect.TypeOf(ast.TypeRef{})

func walkTypeRefs(v reflect.Value, fn func(ast.TypeRef)) {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			walkTypeRefs(v.Elem(), fn)
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			walkTypeRefs(v.Index(i), fn)
		}
	case reflect.Struct:
		if v.Type() == typeRefType {
			fn(v.Interface().(ast.TypeRef))
		}
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				walkTypeRefs(v.Field(i), fn)
			}
		}
	}
}

func (a *Analyser) markGenericUses() {
	mark := func(name string) {
		if name == "" {
			return
		}
		if sym, ok := a.scope.Resolve(name); ok {
			sym.Used = true
		}
	}
	for _, d := range a.program.Declarations {
		generic := false
		switch decl := d.(type) {
		case *ast.FuncDecl:
			generic = isGenericDecl(decl)
		case *ast.StructDecl:
			generic = len(decl.TypeParams) > 0
		}
		if !generic {
			continue
		}
		ast.Inspect(d, func(n ast.Node) bool {
			if id, ok := n.(*ast.IdentLiteral); ok {
				mark(id.Value)
			}
			return true
		})
		walkTypeRefs(reflect.ValueOf(d), func(ref ast.TypeRef) {
			mark(ref.Module)
			if ref.Module == "" {
				mark(ref.Name)
			}
		})
	}
}

func (a *Analyser) dedupeDiagnostics() {
	if len(a.methodInstances) == 0 && len(a.funcInstances) == 0 && len(a.checkFuncInstances) == 0 && len(a.checkMethodInstances) == 0 && len(a.instances) == 0 {
		return
	}
	type key struct {
		file      string
		line, col int
		message   string
		severity  diag.Severity
	}
	seen := map[key]int{}
	kept := a.errors[:0]
	for _, d := range a.errors {
		k := key{d.File, d.Line, d.Col, d.Message, d.Severity}
		if _, dup := seen[k]; dup {
			seen[k]++
			continue
		}
		seen[k] = 1
		kept = append(kept, d)
	}
	for i := range kept {
		d := &kept[i]
		if n := seen[key{d.File, d.Line, d.Col, d.Message, d.Severity}]; n > 1 {
			d.Notes = append(d.Notes[:min(len(d.Notes), 1)], diag.Note{File: d.File, Line: d.Line, Col: d.Col, EndLine: d.EndLine, EndCol: d.EndCol, Message: fmt.Sprintf("the same happens in %d other instantiations", n-1)})
		}
	}
	a.errors = kept
}

func typeTests(cond ast.Expression) ([]*ast.BinaryExpr, bool) {
	b, ok := cond.(*ast.BinaryExpr)
	if !ok {
		return nil, false
	}
	if b.TypeTest {
		return []*ast.BinaryExpr{b}, true
	}
	if b.Operator == "||" && b.Synthetic {
		left, lok := typeTests(b.Left)
		right, rok := typeTests(b.Right)
		if lok && rok {
			return append(left, right...), true
		}
	}
	return nil, false
}

func (a *Analyser) staticTypeMatch(test *ast.BinaryExpr, subject Type) bool {
	var ref ast.TypeRef
	switch right := test.Right.(type) {
	case *ast.TypeLabel:
		ref = right.Type
	case *ast.IdentLiteral:
		ref = ast.TypeRef{Name: right.Value}
	case *ast.MemberExpr:
		module, isIdent := right.Object.(*ast.IdentLiteral)
		if !isIdent {
			return false
		}
		ref = ast.TypeRef{Module: module.Value, Name: right.Field}
	default:
		return false
	}

	if ref.Module == "" && len(ref.Layers) == 0 && len(ref.Args) == 0 && (ref.Name == "signed" || ref.Name == "unsigned") {
		if _, own := a.scope.Resolve(ref.Name); !own {
			if ref.Name == "signed" {
				return isSignedInteger(subject)
			}
			return isUnsignedInteger(subject)
		}
	}

	a.constraintAsTest = true
	resolved := a.resolveTypeRef(test, a.scope, ref)
	a.constraintAsTest = false
	if resolved == nil {
		return false
	}
	if _, invalid := resolved.(InvalidType); invalid {
		return false
	}
	if it, isInterface := resolved.(*InterfaceType); isInterface {
		return a.satisfiesInterface(subject, it) == ""
	}
	return resolved.Equals(subject)
}

func isSignedInteger(t Type) bool {
	p, ok := t.(PrimitiveType)
	if !ok {
		return false
	}
	_, signed, isInt := integerRange(p)
	return isInt && signed
}

func isUnsignedInteger(t Type) bool {
	p, ok := t.(PrimitiveType)
	if !ok {
		return false
	}
	_, signed, isInt := integerRange(p)
	return isInt && !signed
}

func (a *Analyser) pruneTypeSwitch(stmt *ast.IfStmt) bool {
	first, ok := typeTests(stmt.Condition)
	if !ok {
		return false
	}
	subjectExpr, ok := first[0].Left.(*ast.IdentLiteral)
	if !ok {
		return false
	}
	subject := a.checkExpr(subjectExpr)
	switch subject.(type) {
	case BoxedType, InvalidType, *InterfaceType:
		return false
	}
	subject = defaultType(subject)

	var chosen *ast.BlockStmt
	var bind *ast.TypeBind
	arm := stmt
	for arm != nil && chosen == nil {
		tests, isTest := typeTests(arm.Condition)
		if !isTest {
			break
		}
		for _, test := range tests {
			if a.staticTypeMatch(test, subject) {
				chosen, bind = arm.Then, test.Bind
				break
			}
		}
		if chosen != nil {
			break
		}
		next, linked := arm.Else.(*ast.BlockStmt)
		switch {
		case arm.Else == nil:
			arm = nil
		case linked && len(next.Statements) == 1 && isSwitchLink(next.Statements[0]):
			arm = next.Statements[0].(*ast.IfStmt)
		default:
			if linked {
				chosen = next
			}
			arm = nil
		}
	}

	line, col := stmt.Position()
	endLine, endCol := stmt.EndPosition()
	cond := &ast.BoolLiteral{Value: chosen != nil}
	cond.SetPos(line, col)
	cond.SetEndPos(endLine, endCol)

	body := &ast.BlockStmt{}
	if chosen != nil {
		body.Statements = chosen.Statements
		body.SetPos(chosen.Position())
		if bind != nil {
			value := ast.Clone(subjectExpr, ast.CloneOptions{})
			decl := &ast.VarStmt{Vars: []ast.Param{{Name: bind.Name, Line: bind.Line, Col: bind.Col, EndLine: bind.EndLine, EndCol: bind.EndCol}}, Values: []ast.Expression{value}}
			decl.SetPos(bind.Line, bind.Col)
			decl.SetEndPos(bind.EndLine, bind.EndCol)
			body.Statements = append([]ast.Statement{decl}, body.Statements...)
		}
	}
	stmt.Condition = cond
	stmt.Then = body
	stmt.Else = nil
	stmt.SwitchLink, stmt.SwitchDefault = false, false
	return true
}

func (a *Analyser) pruneTypeSwitchExpr(expr *ast.SwitchExpr) Type {
	if len(expr.Arms) == 0 {
		return nil
	}
	first, ok := typeTests(expr.Arms[0].Condition)
	if !ok {
		return nil
	}
	temp, ok := first[0].Left.(*ast.IdentLiteral)
	if !ok {
		return nil
	}
	source := ast.Expression(temp)
	tempAt := -1
	for i, stmt := range expr.Prelude {
		if decl, isVar := stmt.(*ast.VarStmt); isVar && len(decl.Vars) == 1 && len(decl.Values) == 1 && decl.Vars[0].Name == temp.Value {
			source, tempAt = decl.Values[0], i
		}
	}
	subject := a.checkExpr(source)
	switch subject.(type) {
	case BoxedType, InvalidType, *InterfaceType:
		return nil
	}
	subject = defaultType(subject)

	var chosen *ast.SwitchArm
	var bind *ast.TypeBind
	for i := range expr.Arms {
		tests, isTest := typeTests(expr.Arms[i].Condition)
		if !isTest {
			return nil
		}
		for _, test := range tests {
			if a.staticTypeMatch(test, subject) {
				chosen, bind = &expr.Arms[i], test.Bind
				break
			}
		}
		if chosen != nil {
			break
		}
	}
	if chosen == nil {
		chosen = expr.Default
	}
	if chosen == nil {
		a.errorf(expr, "no case of this switch matches %s, and there is no default", subject.String())
		return InvalidType{}
	}

	arm := *chosen
	arm.Condition = nil
	if _, plain := source.(*ast.IdentLiteral); plain && tempAt >= 0 {
		expr.Prelude = append(expr.Prelude[:tempAt:tempAt], expr.Prelude[tempAt+1:]...)
	}
	if bind != nil {
		value := ast.Clone(source, ast.CloneOptions{})
		decl := &ast.VarStmt{Vars: []ast.Param{{Name: bind.Name, Line: bind.Line, Col: bind.Col, EndLine: bind.EndLine, EndCol: bind.EndCol}}, Values: []ast.Expression{value}}
		decl.SetPos(bind.Line, bind.Col)
		decl.SetEndPos(bind.EndLine, bind.EndCol)
		expr.Prelude = append(expr.Prelude, decl)
	}
	expr.Arms = nil
	expr.Default = &arm
	return nil
}

func isSwitchLink(s ast.Statement) bool {
	link, ok := s.(*ast.IfStmt)
	return ok && link.SwitchLink
}

func shortName(s string) string {
	if len(s) > 48 {
		return s[:45] + "..."
	}
	return s
}

func (a *Analyser) genericOf(name string, node ast.Node) *GenericDecl {
	symbol, ok := a.moduleScope.Resolve(name)
	if !ok || symbol.Generic == nil {
		return nil
	}
	if g := symbol.Generic; g.node() == node {
		return g
	}
	return nil
}

func (g *GenericDecl) node() ast.Node {
	switch {
	case g.Func != nil:
		return g.Func
	case g.Alias != nil:
		return g.Alias
	case g.Sum != nil:
		return g.Sum
	case g.Interface != nil:
		return g.Interface
	}
	return g.Struct
}

func (a *Analyser) genericFuncValue(e *ast.IdentLiteral, symbol *Symbol) Type {
	if symbol.Generic == nil {
		a.errorf(e, "%s is not a generic function", e.Value)
		return InvalidType{}
	}
	sym, ft := a.instantiateValue(e, symbol.Generic, e.TypeArgs)
	if ft == nil {
		return InvalidType{}
	}
	a.info.Idents[e] = sym
	return ft
}

func (a *Analyser) genericExprValue(e *ast.GenericExpr) Type {
	savedCallee := a.callee
	a.callee = e.Base
	baseType := a.checkExpr(e.Base)
	a.callee = savedCallee
	switch bt := baseType.(type) {
	case InvalidType:
		return InvalidType{}
	case GenericType:
		sym, ft := a.instantiateValue(e, bt.Decl, e.Args)
		if ft == nil {
			return InvalidType{}
		}
		a.info.GenericFuncs[e] = sym
		return ft
	}
	a.errorf(e, "%s is not generic", ast.ExprText(e.Base))
	return InvalidType{}
}

func (a *Analyser) instantiateValue(node ast.Node, g *GenericDecl, args []ast.TypeArg) (*Symbol, *FuncType) {
	if g.Func == nil {
		a.errorf(node, "%s is not a generic function", g.Name)
		return nil, nil
	}
	if len(args) != len(g.Params) {
		a.errorf(node, "a generic function used as a value needs all its type arguments: %s takes %d, got %d", g.Name, len(g.Params), len(args)).
			WithHelp("there are no arguments to infer them from: write %s<%s>", g.Name, strings.Join(paramNames(g.Params), ", "))
		return nil, nil
	}
	values, ok := a.evalArgList(node, g, args, a.scope)
	if !ok {
		return nil, nil
	}
	sym, ft := a.instantiateFunc(node, g, values)
	if ft == nil {
		return nil, nil
	}
	if ft.Inline {
		a.errorf(node, "%s is inline, so it has no address to use as a value", g.Name)
		return nil, nil
	}
	return sym, ft
}
