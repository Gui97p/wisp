package analyser

import (
	"regexp"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

var nativeConstRef = regexp.MustCompile(`\{([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*)\}`)

func (a *Analyser) markNativeConstants(s *ast.NativeStmt) {
	for _, m := range nativeConstRef.FindAllStringSubmatch(s.Code, -1) {
		first, _, _ := strings.Cut(m[1], ".")
		if sym, ok := a.scope.Resolve(first); ok && (sym.Kind == CONST || sym.Kind == MODULE) {
			sym.Used = true
		}
	}
}

func NativeConstantRef(scope *Scope, parts []string) (*Symbol, []string) {
	sym, ok := scope.Resolve(parts[0])
	if !ok {
		return nil, nil
	}
	if mt, isModule := sym.Type.(*ModuleType); isModule {
		if len(parts) < 2 {
			return nil, nil
		}
		exported, found := mt.Exports[parts[1]]
		if !found {
			return nil, nil
		}
		return exported, parts[2:]
	}
	return sym, parts[1:]
}

func NativeConstant(sym *Symbol) (string, bool) {
	if sym == nil || sym.Kind != CONST || sym.Const == nil {
		return "", false
	}
	switch sym.Const.Kind {
	case ConstInt:
		return sym.Const.Int.String(), true
	case ConstBool:
		if sym.Const.Bool {
			return "1", true
		}
		return "0", true
	}
	return "", false
}
