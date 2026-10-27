package analyser

import "github.com/Gui97p/wisp/internal/ast"

func (a *Analyser) registerFuncSignatures() {
	for _, d := range a.program.Declarations {
		a.currentFile = a.declFiles[d]
		fd, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}

		if fd.Name == "main" {
			a.checkMainFunc(fd)
			continue
		}

		params := make([]Type, 0, len(fd.Params))
		names := make([]string, 0, len(fd.Params))
		variadic := false

		for _, p := range fd.Params {
			var t Type = BoxedType{}
			if !p.Any {
				t = a.resolveTypeRef(fd, a.scope, p.Type)
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

		returns := make([]Type, 0, len(fd.ReturnTypes))
		for _, r := range fd.ReturnTypes {
			t := a.resolveTypeRef(fd, a.scope, r)
			if t == nil {
				t = InvalidType{}
			}
			returns = append(returns, t)
		}

		ft := &FuncType{Name: fd.Name, Params: params, ParamNames: names, Returns: returns, Variadic: variadic, Fallible: fd.Fallible}
		a.info.VarTypes[fd] = params
		a.info.FuncReturns[fd] = returns
		if fd.Receiver != nil {
			continue
		}
		line, col := fd.Position()
		symbol := &Symbol{Name: fd.Name, Kind: FUNC, Type: ft, Line: line, Col: col, File: a.currentFile, Module: a.module, Doc: fd.Doc}
		ft.CVariadic = fd.CVariadic
		if fd.Extern {
			a.checkCSignature(fd, "an extern function", params, returns, variadic)
			symbol.CName = fd.ExternName
			symbol.Extern = true
		}
		if fd.CExport != "" {
			a.checkCExport(fd, params, returns, variadic)
			symbol.CName = fd.CExport
		}

		if !a.scope.Define(symbol) {
			a.errorAlreadyDeclared(fd, FUNC, fd.Name)
		}
	}
}

func (a *Analyser) checkFuncBodies() {
	for _, d := range a.program.Declarations {
		a.currentFile = a.declFiles[d]
		fd, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fd.Body != nil {
			a.checkFuncBody(fd)
		}
	}
}

func (a *Analyser) checkFuncBody(fd *ast.FuncDecl) {
	a.enterScope()
	defer a.exitScope()

	line, col := fd.Position()

	var ft *FuncType

	if fd.Receiver != nil {
		recvType := a.resolveTypeRef(fd, a.scope, fd.Receiver.Type)
		if recvType == nil {
			return
		}
		methods := MethodsOf(recvType)
		if methods == nil {
			return
		}
		ft = methods[fd.Name]
		if ft == nil {
			return
		}

		recvLine, recvCol := identPos(*fd.Receiver, line, col)
		recvSym := &Symbol{Name: fd.Receiver.Name, Kind: PARAM, Type: recvType, Line: recvLine, Col: recvCol, File: a.currentFile}
		if fd.Receiver.Name != "" {
			if !a.scope.Define(recvSym) {
				a.errorf(fd, "duplicated receiver %s", fd.Receiver.Name)
			}
		}
		a.info.VarSymbols[fd] = append(a.info.VarSymbols[fd], recvSym)
	} else {
		symbol, _ := a.scope.Resolve(fd.Name)
		ft = symbol.Type.(*FuncType)
	}

	var declared []*Symbol
	for i, p := range fd.Params {
		if i >= len(ft.Params) {
			break
		}
		paramType := ft.Params[i]
		if p.Variadic {
			paramType = SpanType{Element: paramType}
		}
		paramLine, paramCol := identPos(p, line, col)
		paramSym := &Symbol{Name: p.Name, Kind: PARAM, Type: paramType, Line: paramLine, Col: paramCol, File: a.currentFile}
		if !a.scope.Define(paramSym) {
			a.errorf(fd, "duplicated %s parameter", p.Name)
		}
		a.info.VarSymbols[fd] = append(a.info.VarSymbols[fd], paramSym)
		declared = append(declared, paramSym)
	}

	prevReturns, prevFallible := a.currentReturns, a.currentFallible
	a.currentReturns, a.currentFallible = ft.Returns, ft.Fallible
	defer func() { a.currentReturns, a.currentFallible = prevReturns, prevFallible }()

	a.checkBlock(fd.Body)
	a.warnUnusedParams(fd, declared)
	a.warnInfiniteRecursion(fd)

	if len(ft.Returns) > 0 && !blockTerminates(fd.Body) {
		a.errorAtEnd(fd.Body, "missing return at end of function %s", fd.Name)
	}
}

func (a *Analyser) warnUnusedParams(fd *ast.FuncDecl, params []*Symbol) {
	if fd.Generated {
		return
	}
	for _, sym := range params {
		if sym.Used || sym.Line == 0 || silencesUnused(sym.Name) {
			continue
		}
		a.errors.Warn(sym.File, sym.Line, sym.Col, sym.Line, sym.Col+len(sym.Name)-1, "parameter %s is never used", sym.Name).
			WithHelp("remove it, or rename it to _%s to keep it", sym.Name)
	}
}

func (a *Analyser) checkMainFunc(fd *ast.FuncDecl) {
	params := []Type{}
	switch {
	case len(fd.Params) == 0:
	case len(fd.Params) == 1 && !fd.Params[0].Variadic && !fd.Params[0].Any:
		t := a.resolveTypeRef(fd, a.scope, fd.Params[0].Type)
		want := SpanType{Element: PrimitiveType{Name: "string"}}
		if t != nil && !t.Equals(want) {
			a.errorf(fd, "main takes the arguments as string[], got %s", t.String())
		}
		params = append(params, want)
	default:
		a.error(fd, "main takes no parameters, or a single string[] with the arguments")
	}

	returns := []Type{}
	switch len(fd.ReturnTypes) {
	case 0:
		// implicit exit code 0
	case 1:
		t := a.resolveTypeRef(fd, a.scope, fd.ReturnTypes[0])
		if t != nil && !t.Equals(PrimitiveType{Name: "int"}) {
			a.errorf(fd, "main function must return int, got %s", t.String())
		}
		if t != nil {
			returns = append(returns, t)
		}
	default:
		a.errorf(fd, "main function must return nothing or a single int, got %d values", len(fd.ReturnTypes))
	}

	ft := &FuncType{Name: fd.Name, Params: params, Returns: returns, Variadic: false, Fallible: fd.Fallible}
	a.info.VarTypes[fd] = params
	a.info.FuncReturns[fd] = returns
	line, col := fd.Position()
	symbol := &Symbol{Name: fd.Name, Kind: FUNC, Type: ft, Line: line, Col: col, File: a.currentFile, Module: a.module, Doc: fd.Doc}

	if !a.scope.Define(symbol) {
		a.errorAlreadyDeclared(fd, FUNC, fd.Name)
	}
}

func (a *Analyser) checkMustUse() {
	for _, sym := range a.mustUse {
		if sym.Used {
			continue
		}
		a.errors.Add(sym.File, sym.Line, sym.Col, sym.Line, sym.Col+len(sym.Name), "error %s is never used: handle it, or bind it to _", sym.Name)
	}
}

func cCompatible(t Type) bool {
	switch u := underlyingOf(t).(type) {
	case PrimitiveType:
		return u.Name != "string"
	case PointerType:
		return true
	}
	return false
}

func (a *Analyser) checkCExport(fd *ast.FuncDecl, params, returns []Type, variadic bool) {
	if !fd.Exported {
		a.errorf(fd, "as %s gives a C name to a public function: write public func %s", fd.CExport, fd.Name)
	}
	if fd.Receiver != nil {
		a.error(fd, "a method cannot have a C name")
	}
	a.checkCSignature(fd, "a function with a C name", params, returns, variadic)
}

func (a *Analyser) checkCSignature(fd *ast.FuncDecl, what string, params, returns []Type, variadic bool) {
	if fd.Fallible {
		a.errorf(fd, "%s cannot be fallible: C has no Wisp error values", what)
	}
	if variadic {
		a.errorf(fd, "%s cannot be variadic yet", what)
	}
	if len(fd.ReturnTypes) > 1 {
		a.errorf(fd, "%s returns at most one value", what)
	}

	for i, t := range params {
		if !cCompatible(t) && i < len(fd.Params) {
			a.errorf(fd, "parameter %s has type %s, which C cannot take: use integers, bool, char, floats or pointers", fd.Params[i].Name, t.String())
		}
	}
	for _, t := range returns {
		if !cCompatible(t) {
			a.errorf(fd, "function %s returns %s, which C cannot return: use an integer, bool, char, float or pointer", fd.Name, t.String())
		}
	}
}
