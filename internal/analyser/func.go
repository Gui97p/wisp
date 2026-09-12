package analyser

import "github.com/Gui97p/wisp/internal/ast"

func (a *Analyser) registerFuncSignatures(program *ast.Program) {
	for _, d := range program.Declarations {
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
