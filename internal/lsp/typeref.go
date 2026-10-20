package lsp

import (
	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func refContains(ref *ast.TypeRef, line, col int) bool {
	if ref.Line == 0 && ref.Col == 0 {
		return false
	}

	endLine, endCol := ref.EndLine, ref.EndCol
	if endLine == 0 && endCol == 0 {
		endLine, endCol = ref.Line, ref.Col
	}
	if line < ref.Line || (line == ref.Line && col < ref.Col) {
		return false
	}
	if line > endLine || (line == endLine && col > endCol) {
		return false
	}
	return true
}

func refIn(ref *ast.TypeRef, line, col int) *ast.TypeRef {
	if ref == nil {
		return nil
	}

	if ref.IsMap {
		if r := refIn(ref.MapKey, line, col); r != nil {
			return r
		}
		if r := refIn(ref.MapValue, line, col); r != nil {
			return r
		}
	}
	if ref.IsFunc {
		for i := range ref.FuncParams {
			if r := refIn(&ref.FuncParams[i], line, col); r != nil {
				return r
			}
		}
		for i := range ref.FuncReturns {
			if r := refIn(&ref.FuncReturns[i], line, col); r != nil {
				return r
			}
		}
	}
	if refContains(ref, line, col) {
		return ref
	}
	return nil
}

func refInParams(params []ast.Param, line, col int) *ast.TypeRef {
	for i := range params {
		if r := refIn(&params[i].Type, line, col); r != nil {
			return r
		}
	}
	return nil
}

func typeRefAt(program *ast.Program, line, col int) *ast.TypeRef {
	for _, decl := range program.Declarations {
		if !contains(decl, line, col) {
			continue
		}

		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Receiver != nil {
				if r := refIn(&d.Receiver.Type, line, col); r != nil {
					return r
				}
			}
			if r := refInParams(d.Params, line, col); r != nil {
				return r
			}
			for i := range d.ReturnTypes {
				if r := refIn(&d.ReturnTypes[i], line, col); r != nil {
					return r
				}
			}
			if d.Body != nil {
				if r := typeRefInStmt(d.Body, line, col); r != nil {
					return r
				}
			}
		case *ast.StructDecl:
			if r := refInParams(d.Members, line, col); r != nil {
				return r
			}
		case *ast.TypeDecl:
			if r := refIn(&d.Underlying, line, col); r != nil {
				return r
			}
		case *ast.ConstDecl:
			if r := refInParams(d.Vars, line, col); r != nil {
				return r
			}
		case *ast.VarDecl:
			if r := refInParams(d.Vars, line, col); r != nil {
				return r
			}
		}
	}
	return nil
}

func typeRefInStmt(stmt ast.Statement, line, col int) *ast.TypeRef {
	if stmt == nil {
		return nil
	}

	switch s := stmt.(type) {
	case *ast.BlockStmt:
		for _, inner := range s.Statements {
			if r := typeRefInStmt(inner, line, col); r != nil {
				return r
			}
		}
	case *ast.GroupStmt:
		for _, inner := range s.Statements {
			if r := typeRefInStmt(inner, line, col); r != nil {
				return r
			}
		}
	case *ast.VarStmt:
		return refInParams(s.Vars, line, col)
	case *ast.ConstStmt:
		return refInParams(s.Vars, line, col)
	case *ast.IfStmt:
		if r := typeRefInStmt(s.Then, line, col); r != nil {
			return r
		}
		return typeRefInStmt(s.Else, line, col)
	case *ast.ForStmt:
		return typeRefInStmt(s.Body, line, col)
	case *ast.LoopStmt:
		return typeRefInStmt(s.Body, line, col)
	case *ast.DeferStmt:
		return typeRefInStmt(s.Body, line, col)
	}
	return nil
}

func innermostScope(info *analyser.Info, line, col int) *analyser.Scope {
	var best *analyser.Scope
	var bestSize int
	for node, scope := range info.Scopes {
		if !contains(node, line, col) {
			continue
		}
		sl, _ := node.Position()
		el, _ := node.EndPosition()
		if best == nil || el-sl < bestSize {
			best, bestSize = scope, el-sl
		}
	}
	return best
}

func scopeFor(info *analyser.Info, line, col int) *analyser.Scope {
	if scope := innermostScope(info, line, col); scope != nil {
		return scope
	}
	if info.Module != nil {
		return info.Module
	}
	return analyser.NewScope(nil)
}

func typeRefSymbol(ref *ast.TypeRef, info *analyser.Info, line, col int) *analyser.Symbol {
	scope := scopeFor(info, line, col)

	if ref.Module != "" {
		mod, ok := scope.Resolve(ref.Module)
		if !ok {
			return nil
		}
		if mt, isModule := mod.Type.(*analyser.ModuleType); isModule {
			return mt.Exports[ref.Name]
		}
		return nil
	}

	sym, ok := scope.Resolve(ref.Name)
	if !ok || (sym.Kind != analyser.STRUCT && sym.Kind != analyser.TYPE) {
		return nil
	}
	return sym
}

func refRange(ref *ast.TypeRef) *protocol.Range {
	return &protocol.Range{
		Start: protocol.Position{Line: uint32(ref.Line - 1), Character: uint32(ref.Col - 1)},
		End:   protocol.Position{Line: uint32(ref.EndLine - 1), Character: uint32(ref.EndCol)},
	}
}
