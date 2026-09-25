package lsp

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func hover(uri protocol.DocumentUri, pos protocol.Position) *protocol.Hover {
	path := uriToPath(uri)
	line := int(pos.Line) + 1
	col := int(pos.Character) + 1

	program, info := analyseEntry(path)
	if program == nil || info == nil {
		return nil
	}

	if param, owner, idx := paramAt(program, line, col); param != nil {
		text := paramHoverText(param, owner, idx, info)
		if text == "" {
			return nil
		}
		return &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: text,
			},
			Range: paramRange(param),
		}
	}

	expr := exprInProgram(program, line, col)
	if expr == nil {
		return nil
	}

	text := hoverText(expr, info)
	if text == "" {
		return nil
	}

	sl, sc := expr.Position()
	el, ec := expr.EndPosition()

	rng := protocol.Range{
		Start: protocol.Position{Line: uint32(sl - 1), Character: uint32(sc - 1)},
		End:   protocol.Position{Line: uint32(el - 1), Character: uint32(ec)},
	}

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: text,
		},
		Range: &rng,
	}
}

func paramRange(param *ast.Param) *protocol.Range {
	return &protocol.Range{
		Start: protocol.Position{Line: uint32(param.Line - 1), Character: uint32(param.Col - 1)},
		End:   protocol.Position{Line: uint32(param.EndLine - 1), Character: uint32(param.EndCol)},
	}
}

func paramHoverText(param *ast.Param, owner ast.Node, idx int, info *analyser.Info) string {
	kind := "variable"
	switch owner.(type) {
	case *ast.FuncDecl:
		kind = "parameter"
	case *ast.ConstDecl, *ast.ConstStmt:
		kind = "constant"
	}

	typeStr := param.Type.Name
	if typeStr == "" {
		if types, ok := info.VarTypes[owner]; ok && idx >= 0 && idx < len(types) {
			typeStr = types[idx].String()
		}
	}
	if typeStr == "" {
		return ""
	}

	return fmt.Sprintf("**%s** `%s`: `%s`", kind, param.Name, typeStr)
}

func hoverText(expr ast.Expression, info *analyser.Info) string {
	t, ok := info.Types[expr]
	if !ok {
		return ""
	}

	if ident, ok := expr.(*ast.IdentLiteral); ok {
		if sym, ok := info.Idents[ident]; ok {
			return fmt.Sprintf("**%s** `%s`: `%s`", sym.Kind, sym.Name, t)
		}
	}

	return fmt.Sprintf("`%s`", t)
}
