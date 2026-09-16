package analyser

import "github.com/Gui97p/wisp/internal/ast"

func (a *Analyser) registerFuncSignatures() {
	for _, d := range a.program.Declarations {
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
			if t == nil {
				continue
			}
			returns = append(returns, t)
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
		fd, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		a.checkFuncBody(fd)
	}
}

func (a *Analyser) checkFuncBody(fd *ast.FuncDecl) {
	symbol, _ := a.scope.Resolve(fd.Name)
	ft := symbol.Type.(*FuncType)

	a.enterScope()
	defer a.exitScope()

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
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Name != "main" {
			continue
		}
		count++
		mainDecl = fd
	}

	if count == 0 {
		a.errors.Add(0, 0, "program has no main function")
		return
	}
	if count > 1 {
		a.errors.Add(0, 0, "program has more than one main function")
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
