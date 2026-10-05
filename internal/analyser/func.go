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
		variadic := false

		for _, p := range fd.Params {
			t := a.resolveTypeRef(fd, a.scope, p.Type)
			if t == nil {
				continue
			}
			params = append(params, t)
			if p.Variadic {
				variadic = true
			}
		}

		returns := make([]Type, 0, len(fd.ReturnTypes))
		for _, r := range fd.ReturnTypes {
			t := a.resolveTypeRef(fd, a.scope, r)
			if t != nil {
				returns = append(returns, t)
			}
		}

		ft := &FuncType{Name: fd.Name, Params: params, Returns: returns, Variadic: variadic, Fallible: fd.Fallible}
		a.info.VarTypes[fd] = params
		a.info.FuncReturns[fd] = returns
		line, col := fd.Position()
		symbol := &Symbol{Name: fd.Name, Kind: FUNC, Type: ft, Line: line, Col: col, File: a.currentFile, Module: a.module}

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

		if fd.Receiver.Name != "" {
			if !a.scope.Define(&Symbol{Name: fd.Receiver.Name, Kind: PARAM, Type: recvType, Line: line, Col: col, File: a.currentFile}) {
				a.errorf(fd, "duplicated receiver %s", fd.Receiver.Name)
			}
		}
	} else {
		symbol, _ := a.scope.Resolve(fd.Name)
		ft = symbol.Type.(*FuncType)
	}

	for i, p := range fd.Params {
		paramType := ft.Params[i]
		if p.Variadic {
			paramType = SpanType{Element: paramType}
		}
		paramSym := &Symbol{Name: p.Name, Kind: PARAM, Type: paramType, Line: line, Col: col, File: a.currentFile}
		if !a.scope.Define(paramSym) {
			a.errorf(fd, "duplicated %s parameter", p.Name)
		}
		a.info.VarSymbols[fd] = append(a.info.VarSymbols[fd], paramSym)
	}

	prevReturns, prevFallible := a.currentReturns, a.currentFallible
	a.currentReturns, a.currentFallible = ft.Returns, ft.Fallible
	defer func() { a.currentReturns, a.currentFallible = prevReturns, prevFallible }()

	a.checkBlock(fd.Body)

	if len(ft.Returns) > 0 && !blockTerminates(fd.Body) {
		a.errorf(fd, "missing return at end of function %s", fd.Name)
	}
}

func (a *Analyser) checkMainFunc(fd *ast.FuncDecl) {
	if len(fd.Params) > 0 {
		a.error(fd, "main function cannot have parameters")
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
		returns = append(returns, t)
	default:
		a.errorf(fd, "main function must return nothing or a single int, got %d values", len(fd.ReturnTypes))
	}

	ft := &FuncType{Name: fd.Name, Params: []Type{}, Returns: returns, Variadic: false, Fallible: fd.Fallible}
	a.info.FuncReturns[fd] = returns
	line, col := fd.Position()
	symbol := &Symbol{Name: fd.Name, Kind: FUNC, Type: ft, Line: line, Col: col, File: a.currentFile, Module: a.module}

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
