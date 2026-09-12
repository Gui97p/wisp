package analyser

import "github.com/Gui97p/wisp/internal/ast"

func (a *Analyser) registerFuncSignatures() {
	for _, d := range a.program.Declarations {
		fd, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}

		params := make([]Type, 0, len(fd.Params))
		for _, p := range fd.Params {
			t := a.resolveTypeRef(a.scope, p.Type)
			if t == nil {
				continue
			}
			params = append(params, t)
		}

		returns := make([]Type, 0, len(fd.ReturnTypes))
		for _, r := range fd.ReturnTypes {
			t := a.resolveTypeRef(a.scope, r)
			if t == nil {
				continue
			}
			returns = append(returns, t)
		}

		ft := &FuncType{Name: fd.Name, Params: params, Returns: returns}
		symbol := &Symbol{Name: fd.Name, Kind: FUNC, Type: ft}

		if !a.scope.Define(symbol) {
			a.errorf("%s func already defined in this scope", fd.Name)
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
		if !a.scope.Define(&Symbol{Name: p.Name, Kind: PARAM, Type: ft.Params[i]}) {
			a.errorf("duplicated %s parameter", p.Name)
		}
	}

	prevReturns := a.currentReturns
	a.currentReturns = ft.Returns
	defer func() { a.currentReturns = prevReturns }()

	a.checkBlock(fd.Body)
}
