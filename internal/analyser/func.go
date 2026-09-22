package analyser

import "github.com/Gui97p/wisp/internal/ast"

func (a *Analyser) registerFuncSignatures() {
	for _, d := range a.program.Declarations {
		a.currentFile = a.declFiles[d]
		fd, ok := d.(*ast.FuncDecl)
		if !ok {
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

		fallibleCount := 0
		for _, t := range returns {
			if _, ok := t.(ErrorUnionType); ok {
				fallibleCount++
			}
		}
		if fallibleCount > 1 {
			a.errorf(fd, "function can only have one fallible (!T) return value")
		}

		ft := &FuncType{Name: fd.Name, Params: params, Returns: returns, Variadic: variadic}
		symbol := &Symbol{Name: fd.Name, Kind: FUNC, Type: ft}

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
			if !a.scope.Define(&Symbol{Name: fd.Receiver.Name, Kind: PARAM, Type: recvType}) {
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
		if !a.scope.Define(&Symbol{Name: p.Name, Kind: PARAM, Type: paramType}) {
			a.errorf(fd, "duplicated %s parameter", p.Name)
		}
	}

	prevReturns := a.currentReturns
	a.currentReturns = ft.Returns
	defer func() { a.currentReturns = prevReturns }()

	a.checkBlock(fd.Body)

	if len(ft.Returns) > 0 && !blockTerminates(fd.Body) {
		a.errorf(fd, "missing return at end of function %s", fd.Name)
	}
}

func (a *Analyser) checkMainFunc() {
	var mainDecl *ast.FuncDecl
	count := 0
	for _, d := range a.program.Declarations {
		a.currentFile = a.declFiles[d]
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Name != "main" {
			continue
		}
		count++
		mainDecl = fd
	}

	if count == 0 {
		a.errors.Add(a.currentFile, 0, 0, "program has no main function")
		return
	}
	if count > 1 {
		a.errors.Add(a.currentFile, 0, 0, "program has more than one main function")
		return
	}

	if len(mainDecl.Params) > 0 {
		a.error(mainDecl, "main function cannot have parameters")
	}

	switch len(mainDecl.ReturnTypes) {
	case 0:
		// implicit exit code 0
	case 1:
		t := a.resolveTypeRef(mainDecl, a.scope, mainDecl.ReturnTypes[0])
		if t != nil && !t.Equals(PrimitiveType{Name: "int"}) {
			a.errorf(mainDecl, "main function must return int, got %s", t.String())
		}
	default:
		a.errorf(mainDecl, "main function must return nothing or a single int, got %d values", len(mainDecl.ReturnTypes))
	}
}
