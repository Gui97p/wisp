package analyser

import (
	"math"

	"github.com/Gui97p/wisp/internal/ast"
)

type arraySizeResult struct {
	size int64
	ok   bool
}

func (a *Analyser) arraySize(expr ast.Expression) (int64, bool) {
	if cached, seen := a.arraySizes[expr]; seen {
		return cached.size, cached.ok
	}

	size, ok := a.evalArraySize(expr)
	if a.arraySizes == nil {
		a.arraySizes = map[ast.Expression]arraySizeResult{}
	}
	a.arraySizes[expr] = arraySizeResult{size, ok}
	return size, ok
}

func (a *Analyser) evalArraySize(expr ast.Expression) (int64, bool) {
	if a.namesGlobalVar(expr) {
		a.error(expr, "the size of an array must be a constant integer").
			WithHelp("a global variable is not a constant, declare it with `const`")
		return 0, false
	}
	if _, bad := a.checkExpr(expr).(InvalidType); bad {
		return 0, false
	}

	value, isConst := a.evalConst(expr)
	if !isConst || value.Kind != ConstInt {
		a.error(expr, "the size of an array must be a constant integer").
			WithHelp("write a number or a const, such as `const Size = 64;`")
		return 0, false
	}
	if value.Int.Sign() < 0 || !value.Int.IsInt64() || value.Int.Int64() > math.MaxInt32 {
		a.errorf(expr, "%s is not a valid array size", value.Int).
			WithHelp("a size is between 0 and %d", math.MaxInt32)
		return 0, false
	}
	return value.Int.Int64(), true
}

func (a *Analyser) namesGlobalVar(expr ast.Expression) bool {
	switch e := expr.(type) {
	case *ast.IdentLiteral:
		if _, found := a.scope.Resolve(e.Value); found {
			return false
		}
		for _, d := range a.program.Declarations {
			if decl, ok := d.(*ast.VarDecl); ok {
				for _, v := range decl.Vars {
					if v.Name == e.Value {
						return true
					}
				}
			}
		}
	case *ast.BinaryExpr:
		return a.namesGlobalVar(e.Left) || a.namesGlobalVar(e.Right)
	case *ast.UnaryExpr:
		return a.namesGlobalVar(e.Value)
	}
	return false
}
