package lsp

import (
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
	"github.com/Gui97p/wisp/internal/parser"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func rangeOf(line, col, endLine, endCol int) protocol.Range {
	if endLine == 0 && endCol == 0 {
		endLine, endCol = line, col
	}
	return protocol.Range{
		Start: protocol.Position{Line: uint32(max(line-1, 0)), Character: uint32(max(col-1, 0))},
		End:   protocol.Position{Line: uint32(max(endLine-1, 0)), Character: uint32(endCol)},
	}
}

func symbol(name string, kind protocol.SymbolKind, line, col, endLine, endCol int) protocol.DocumentSymbol {
	r := rangeOf(line, col, endLine, endCol)
	return protocol.DocumentSymbol{Name: name, Kind: kind, Range: r, SelectionRange: protocol.Range{Start: r.Start, End: r.Start}}
}

func documentSymbols(uri protocol.DocumentUri) []protocol.DocumentSymbol {
	src, ok := fileBuffer[uriToPath(uri)]
	if !ok {
		return nil
	}

	program := parser.NewParser(lexer.NewLexer(src)).ParseProgram()

	symbols := []protocol.DocumentSymbol{}
	for _, decl := range program.Declarations {
		line, col := decl.Position()
		endLine, endCol := decl.EndPosition()

		switch d := decl.(type) {
		case *ast.FuncDecl:
			kind := protocol.SymbolKindFunction
			if d.Receiver != nil {
				kind = protocol.SymbolKindMethod
			}
			symbols = append(symbols, symbol(d.Name, kind, line, col, endLine, endCol))
		case *ast.StructDecl:
			s := symbol(d.Name, protocol.SymbolKindStruct, line, col, endLine, endCol)
			for _, m := range d.Members {
				s.Children = append(s.Children, symbol(m.Name, protocol.SymbolKindField, m.Line, m.Col, m.EndLine, m.EndCol))
			}
			symbols = append(symbols, s)
		case *ast.TypeDecl:
			kind := protocol.SymbolKindClass
			if d.IsEnum {
				kind = protocol.SymbolKindEnum
			}
			s := symbol(d.Name, kind, line, col, endLine, endCol)
			for _, m := range d.Members {
				s.Children = append(s.Children, symbol(m.Name, protocol.SymbolKindEnumMember, m.Line, m.Col, m.EndLine, m.EndCol))
			}
			symbols = append(symbols, s)
		case *ast.ConstDecl:
			for _, v := range d.Vars {
				symbols = append(symbols, symbol(v.Name, protocol.SymbolKindConstant, v.Line, v.Col, v.EndLine, v.EndCol))
			}
		case *ast.VarDecl:
			for _, v := range d.Vars {
				symbols = append(symbols, symbol(v.Name, protocol.SymbolKindVariable, v.Line, v.Col, v.EndLine, v.EndCol))
			}
		}
	}
	return symbols
}
