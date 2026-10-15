package lsp

import (
	"sort"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var semanticTokenTypes = []string{"property", "enumMember"}

const (
	tokenProperty = iota
	tokenEnumMember
)

type semanticToken struct {
	line, col, length, kind int
}

func semanticTokens(context *glsp.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	path := uriToPath(params.TextDocument.URI)

	program, info, _ := analyseEntryChecked(path)
	if program == nil || info == nil {
		prev, ok := getLastGood(path)
		if !ok {
			return &protocol.SemanticTokens{Data: []uint32{}}, nil
		}
		program, info = prev.program, prev.info
	}

	return &protocol.SemanticTokens{Data: encodeTokens(collectTokens(program, info))}, nil
}

func collectTokens(program *ast.Program, info *analyser.Info) []semanticToken {
	var tokens []semanticToken

	add := func(at ast.Position, kind int) {
		if at.Line == 0 {
			return
		}
		length := at.EndCol - at.Col + 1
		if at.EndLine != at.Line || length <= 0 {
			return
		}
		tokens = append(tokens, semanticToken{at.Line - 1, at.Col - 1, length, kind})
	}

	for _, decl := range program.Declarations {
		switch d := decl.(type) {
		case *ast.StructDecl:
			for _, member := range d.Members {
				add(ast.Position{Line: member.Line, Col: member.Col, EndLine: member.EndLine, EndCol: member.EndCol}, tokenProperty)
			}
		case *ast.TypeDecl:
			for _, member := range d.Members {
				add(ast.Position{Line: member.Line, Col: member.Col, EndLine: member.EndLine, EndCol: member.EndCol}, tokenEnumMember)
			}
		}
	}

	walkProgram(program, func(expr ast.Expression) {
		switch e := expr.(type) {
		case *ast.MemberExpr:
			if _, isEnum := info.EnumValues[e]; isEnum {
				add(e.FieldAt, tokenEnumMember)
				return
			}
			if member := info.Members[e]; member != nil && (member.Kind == analyser.MemberField || member.Kind == analyser.MemberErrorField) {
				add(e.FieldAt, tokenProperty)
			}
		case *ast.DotIdent:
			if _, isEnum := info.EnumValues[e]; isEnum {
				line, col := e.Position()
				add(ast.Position{Line: line, Col: col + 1, EndLine: line, EndCol: col + len(e.Name)}, tokenEnumMember)
			}
		case *ast.StructLiteral:
			for _, at := range e.KeyAt {
				add(at, tokenProperty)
			}
		}
	})

	return tokens
}

func encodeTokens(tokens []semanticToken) []uint32 {
	sort.Slice(tokens, func(i, j int) bool {
		if tokens[i].line != tokens[j].line {
			return tokens[i].line < tokens[j].line
		}
		return tokens[i].col < tokens[j].col
	})

	data := make([]uint32, 0, len(tokens)*5)
	prevLine, prevCol := 0, 0
	for _, t := range tokens {
		deltaLine := t.line - prevLine
		deltaCol := t.col
		if deltaLine == 0 {
			deltaCol = t.col - prevCol
			if deltaCol < 0 {
				continue
			}
		}
		data = append(data, uint32(deltaLine), uint32(deltaCol), uint32(t.length), uint32(t.kind), 0)
		prevLine, prevCol = t.line, t.col
	}
	return data
}
