package lsp

import (
	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func definition(uri protocol.DocumentUri, pos protocol.Position) *protocol.Location {
	path := uriToPath(uri)
	line := int(pos.Line) + 1
	col := int(pos.Character) + 1

	program, info := analyseEntry(path)
	if program == nil || info == nil {
		return nil
	}

	expr := exprInProgram(program, line, col)
	if expr == nil {
		return nil
	}

	sym := resolveSymbol(expr, info)
	if sym == nil || sym.File == "" {
		return nil
	}

	target := protocol.Position{
		Line:      uint32(sym.Line - 1),
		Character: uint32(sym.Col - 1),
	}

	return &protocol.Location{
		URI: pathToURI(sym.File),
		Range: protocol.Range{
			Start: target,
			End:   target,
		},
	}
}

func resolveSymbol(expr ast.Expression, info *analyser.Info) *analyser.Symbol {
	switch e := expr.(type) {
	case *ast.IdentLiteral:
		return info.Idents[e]
	case *ast.MemberExpr:
		if mt, ok := info.Types[e.Object].(*analyser.ModuleType); ok {
			return mt.Exports[e.Field]
		}
	}
	return nil
}
