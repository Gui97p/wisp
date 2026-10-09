package analyser

import (
	"slices"
	"sort"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/diag"
)

type Analyser struct {
	program *ast.Program

	info  *Info
	scope *Scope

	currentReturns  []Type
	currentFallible bool
	loopFrames      []loopFrame

	mustUse     []*Symbol
	handlerOK   map[ast.Expression]bool
	pendingDots map[*ast.DotIdent]bool

	modules map[string]*ModuleInfo
	module  string

	universe *Scope

	declFiles   map[ast.Declaration]string
	currentFile string

	errors diag.List
}

func NewAnalyser(program *ast.Program, modules map[string]*ModuleInfo, declFiles map[ast.Declaration]string, module string) *Analyser {
	universe := NewScope(nil)
	return &Analyser{
		program:     program,
		info:        NewInfo(),
		universe:    universe,
		scope:       NewScope(universe),
		modules:     modules,
		declFiles:   declFiles,
		module:      module,
		handlerOK:   map[ast.Expression]bool{},
		pendingDots: map[*ast.DotIdent]bool{},
	}
}

func (a *Analyser) Analyze() *Info {
	a.registerImports()
	a.registerStructNames()
	a.registerTypeAliases()
	a.registerMethods()
	a.registerStructFields()
	a.registerBuiltins()
	a.registerFuncSignatures()
	a.registerConsts()
	a.checkFuncBodies()
	a.warnUnusedImports()
	a.defaultUntyped()
	a.reportUnresolvedDots()
	a.checkMustUse()

	return a.info
}

func (a *Analyser) resolveTypeRef(node ast.Node, scope *Scope, ref ast.TypeRef) Type {
	if ref.IsFunc {
		params := make([]Type, len(ref.FuncParams))
		for i, p := range ref.FuncParams {
			params[i] = a.resolveTypeRef(node, scope, p)
		}
		returns := make([]Type, len(ref.FuncReturns))
		for i, r := range ref.FuncReturns {
			returns[i] = a.resolveTypeRef(node, scope, r)
		}
		return &FuncType{Params: params, Returns: returns, Fallible: ref.FuncFallible}
	}

	if ref.IsMap {
		keyType := a.resolveTypeRef(node, scope, *ref.MapKey)
		valueType := a.resolveTypeRef(node, scope, *ref.MapValue)
		if keyType == nil || valueType == nil {
			return nil
		}

		var result Type = &MapType{Key: keyType, Value: valueType}
		for i := 0; i < ref.PointerDepth; i++ {
			result = PointerType{Element: result}
		}
		return result
	}

	var result Type

	if primitives[ref.Name] {
		result = PrimitiveType{Name: ref.Name}
	} else {
		symbol, ok := a.lookupType(scope, ref.Module, ref.Name)
		if !ok {
			a.suggest(a.errorf(node, "unknown type %s", qualified(ref.Module, ref.Name)), ref.Name, append(scope.names(func(sym *Symbol) bool {
				return sym.Kind == STRUCT || sym.Kind == TYPE
			}), "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64", "char", "string", "bool"))
			return nil
		}
		switch symbol.Kind {
		case STRUCT, TYPE:
			result = symbol.Type
		default:
			a.errorf(node, "%s is not a type", qualified(ref.Module, ref.Name))
			return nil
		}
	}

	for _, v := range slices.Backward(ref.Dims) {
		if v.IsSpan {
			result = SpanType{Element: result}
		} else {
			result = ArrayType{Element: result, Size: v.Size}
		}
	}

	for i := 0; i < ref.PointerDepth; i++ {
		result = PointerType{Element: result}
	}

	return result
}

func (a *Analyser) enterScope() *Scope {
	a.scope = NewScope(a.scope)
	return a.scope
}

func (a *Analyser) exitScope() {
	a.warnUnused(a.scope)
	a.scope = a.scope.parent
}

func (a *Analyser) warnUnused(scope *Scope) {
	var unused []*Symbol
	for _, sym := range scope.symbols {
		if (sym.Kind == VAR || sym.Kind == CONST) && !sym.Used && sym.Name != "_" && sym.Line > 0 {
			unused = append(unused, sym)
		}
	}
	sort.Slice(unused, func(i, j int) bool {
		if unused[i].Line != unused[j].Line {
			return unused[i].Line < unused[j].Line
		}
		return unused[i].Col < unused[j].Col
	})
	for _, sym := range unused {
		a.errors.Warn(sym.File, sym.Line, sym.Col, sym.Line, sym.Col+len(sym.Name)-1, "%s %s is never used", sym.Kind, sym.Name)
	}
}

func (a *Analyser) warnUnusedImports() {
	reexported := map[string]bool{}
	for _, d := range a.program.Declarations {
		if re, ok := d.(*ast.ReexportDecl); ok {
			reexported[re.Name] = true
		}
	}

	for _, d := range a.program.Declarations {
		imp, ok := d.(*ast.ImportDecl)
		if !ok || reexported[imp.Alias] {
			continue
		}
		sym, ok := a.scope.symbols[imp.Alias]
		if !ok || sym.Used || sym.Kind != MODULE || imp.Alias == "_" {
			continue
		}
		a.currentFile = a.declFiles[d]
		a.warnf(imp, "import %s is never used", imp.Alias)
	}
}

func (a *Analyser) requireBool(node ast.Node, t Type, context string) {
	if _, ok := t.(InvalidType); ok {
		return
	}
	if !t.Equals(PrimitiveType{Name: "bool"}) {
		a.errorf(node, "%s must be bool, got %s", context, t.String())
	}
}

func (a *Analyser) requireNumeric(node ast.Node, t Type, context string) {
	if _, ok := t.(InvalidType); ok {
		return
	}
	if !isNumeric(t) {
		a.errorf(node, "%s must be numeric, got %s", context, t.String())
	}
}

type loopFrame struct {
	Label            string
	SupportsContinue bool
}

func (a *Analyser) pushLoop(label string) {
	a.loopFrames = append(a.loopFrames, loopFrame{Label: label, SupportsContinue: true})
}

func (a *Analyser) pushSwitch() {
	a.loopFrames = append(a.loopFrames, loopFrame{SupportsContinue: false})
}

func (a *Analyser) popLoop() {
	a.loopFrames = a.loopFrames[:len(a.loopFrames)-1]
}

func (a *Analyser) hasContinuableLoop() bool {
	for _, f := range a.loopFrames {
		if f.SupportsContinue {
			return true
		}
	}
	return false
}

func rootIdentifier(expr ast.Expression) *ast.IdentLiteral {
	switch e := expr.(type) {
	case *ast.IdentLiteral:
		return e
	case *ast.MemberExpr:
		return rootIdentifier(e.Object)
	case *ast.IndexExpr:
		return rootIdentifier(e.Array)
	case *ast.UnaryExpr:
		if e.Operator == "*" {
			return rootIdentifier(e.Value)
		}
	}
	return nil
}

func qualified(module, name string) string {
	if module == "" {
		return name
	}
	return module + "." + name
}

func (a *Analyser) lookupType(scope *Scope, module, name string) (*Symbol, bool) {
	if module == "" {
		return scope.Resolve(name)
	}
	modSym, ok := scope.Resolve(module)
	if !ok || modSym.Kind != MODULE {
		return nil, false
	}
	modSym.Used = true
	mt, ok := modSym.Type.(*ModuleType)
	if !ok {
		return nil, false
	}
	sym, ok := mt.Exports[name]
	return sym, ok
}
