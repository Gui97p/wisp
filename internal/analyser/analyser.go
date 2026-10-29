package analyser

import (
	"slices"
	"sort"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/diag"
)

type Analyser struct {
	program *ast.Program

	info  *Info
	scope *Scope

	currentReturns  []Type
	currentFallible bool
	yieldFrames     []yieldFrame
	deferredFree    map[*Symbol]*ast.DeferStmt
	loopFrames      []loopFrame
	untilScope      *Scope
	untilFrame      int
	untilSawBody    bool

	mustUse      []*Symbol
	pendingDots  map[*ast.DotIdent]bool
	pendingEmpty map[*ast.ArrayLiteral]bool

	modules map[string]*ModuleInfo
	links   []*importLink
	parts   []string
	exports map[string]*Symbol

	importErrors map[string]string
	module       string

	universe *Scope

	declFiles   map[ast.Declaration]string
	arraySizes  map[ast.Expression]arraySizeResult
	currentFile string

	errors diag.List
}

func (a *Analyser) SetPackageParts(parts []string) {
	a.parts = parts
}

func (a *Analyser) SetImportErrors(errors map[string]string) {
	a.importErrors = errors
}

func NewAnalyser(program *ast.Program, modules map[string]*ModuleInfo, declFiles map[ast.Declaration]string, module string) *Analyser {
	universe := NewScope(nil)
	return &Analyser{
		program:      program,
		info:         NewInfo(),
		universe:     universe,
		scope:        NewScope(universe),
		modules:      modules,
		declFiles:    declFiles,
		module:       module,
		pendingDots:  map[*ast.DotIdent]bool{},
		pendingEmpty: map[*ast.ArrayLiteral]bool{},
	}
}

func (a *Analyser) Analyze() *Info {
	AnalyzeGroup([]*Analyser{a})
	return a.info
}

func AnalyzeGroup(group []*Analyser) {
	for _, a := range group {
		a.info.Module = a.scope
		a.registerImports()
		a.registerStructNames()
	}
	publish(group, false)
	for _, a := range group {
		a.registerTypeAliases()
		a.registerConsts()
		a.generateEnumNames()
	}
	publish(group, false)
	for _, a := range group {
		a.registerMethods()
		a.registerStructFields()
		a.registerBuiltins()
		a.registerFuncSignatures()
		a.registerGlobals()
	}
	publish(group, true)
	for _, a := range group {
		a.checkFuncBodies()
		a.warnUnusedGlobals()
		a.warnUnusedFuncs()
		a.warnUnusedImports()
		a.defaultUntyped()
		a.reportUnresolvedDots()
		a.reportUnresolvedEmpty()
		a.checkMustUse()
		a.exports = a.buildExports(false)
		a.checkPartConflicts()
	}
}

func publish(group []*Analyser, final bool) {
	if len(group) == 1 && !group[0].pendingLinks() {
		return
	}
	for _, a := range group {
		if self, ok := a.modules[a.module]; ok && self.Pending {
			self.Replace(a.buildExports(true))
		}
	}
	if final {
		for _, a := range group {
			if self, ok := a.modules[a.module]; ok {
				self.Pending = false
			}
		}
	}
	for _, a := range group {
		for _, link := range a.links {
			if link.target.Pending || final {
				a.applyLink(link, final && link.pendingAtStart)
			}
		}
	}
}

func (a *Analyser) pendingLinks() bool {
	for _, l := range a.links {
		if l.pendingAtStart {
			return true
		}
	}
	return false
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
	} else if ref.Name == "void" && ref.Module == "" && ref.PointerDepth > 0 && len(ref.Dims) == 0 {
		result = VoidType{}
	} else {
		symbol, ok := a.lookupType(scope, ref.Module, ref.Name)
		if !ok {
			a.suggest(a.errorAtRef(node, ref, "unknown type %s", qualified(ref.Module, ref.Name)), ref.Name, append(scope.names(func(sym *Symbol) bool {
				return sym.Kind == STRUCT || sym.Kind == TYPE
			}), "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float", "float32", "float64", "char", "string", "bool"))
			return nil
		}
		switch symbol.Kind {
		case STRUCT, TYPE:
			result = symbol.Type
		default:
			a.errorAtRef(node, ref, "%s is not a type", qualified(ref.Module, ref.Name))
			return nil
		}
	}

	for _, v := range slices.Backward(ref.Dims) {
		if v.IsSpan {
			result = SpanType{Element: result}
		} else if v.Expr != nil {
			size, ok := a.arraySize(v.Expr)
			if !ok {
				return nil
			}
			result = ArrayType{Element: result, Size: size}
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
		if (sym.Kind == VAR || sym.Kind == CONST) && !sym.Used && sym.Name != "" && !silencesUnused(sym.Name) && sym.Line > 0 {
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
	for _, d := range a.program.Declarations {
		imp, ok := d.(*ast.ImportDecl)
		if !ok {
			continue
		}
		sym, ok := a.scope.symbols[imp.Alias]
		if !ok || sym.Used || sym.Kind != MODULE || silencesUnused(imp.Alias) {
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
	Continued        bool
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
	case *ast.DerefExpr:
		return rootIdentifier(e.Value)
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

func silencesUnused(name string) bool {
	return strings.HasPrefix(name, "_")
}

func (a *Analyser) Info() *Info {
	return a.info
}
