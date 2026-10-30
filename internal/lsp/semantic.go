package lsp

import (
	"reflect"
	"sort"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var semanticTokenTypes = []string{"property", "enumMember", "type"}

const (
	tokenProperty = iota
	tokenEnumMember
	tokenType
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

	program = declarationsOf(program, declarationFiles(path), path)

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

	walkTypeRefs(reflect.ValueOf(program), func(ref ast.TypeRef) {
		if ref.Name == "" || ref.IsFunc || ref.Line == 0 || ref.EndLine != ref.Line {
			return
		}
		if primitiveType(ref.Name) && ref.Module == "" {
			return
		}
		col := ref.Col
		if ref.Module != "" {
			col += len(ref.Module) + 1
		}
		add(ast.Position{Line: ref.Line, Col: col, EndLine: ref.EndLine, EndCol: ref.EndCol}, tokenType)
	})

	walkProgram(program, func(expr ast.Expression) {
		switch e := expr.(type) {
		case *ast.IdentLiteral:
			if sym := info.Idents[e]; sym != nil && (sym.Kind == analyser.TYPE || sym.Kind == analyser.STRUCT) {
				line, col := e.Position()
				add(ast.Position{Line: line, Col: col, EndLine: line, EndCol: col + len(e.Value) - 1}, tokenType)
			}
		case *ast.MemberExpr:
			if object, ok := e.Object.(*ast.IdentLiteral); ok {
				if owner := info.Idents[object]; owner != nil && owner.Kind == analyser.MODULE {
					if mt, isModule := owner.Type.(*analyser.ModuleType); isModule {
						if sym := mt.Exports[e.Field]; sym != nil && (sym.Kind == analyser.TYPE || sym.Kind == analyser.STRUCT) {
							add(e.FieldAt, tokenType)
						}
					}
				}
			}
			if _, isEnum := info.EnumValues[e]; isEnum {
				add(e.FieldAt, tokenEnumMember)
				switch owner := e.Object.(type) {
				case *ast.MemberExpr:
					add(owner.FieldAt, tokenType)
				case *ast.IdentLiteral:
					line, col := owner.Position()
					add(ast.Position{Line: line, Col: col, EndLine: line, EndCol: col + len(owner.Value) - 1}, tokenType)
				}
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
			line, col := e.Position()
			if e.Module != "" {
				col += len(e.Module) + 1
			}
			add(ast.Position{Line: line, Col: col, EndLine: line, EndCol: col + len(e.Name) - 1}, tokenType)
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

func primitiveType(name string) bool {
	switch name {
	case "int", "uint", "int8", "uint8", "int16", "uint16", "int32", "uint32", "int64", "uint64",
		"float", "float32", "float64", "bool", "char", "string", "void":
		return true
	}
	return false
}

func walkTypeRefs(v reflect.Value, fn func(ast.TypeRef)) {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			walkTypeRefs(v.Elem(), fn)
		}
	case reflect.Struct:
		if ref, ok := reflect.TypeAssert[ast.TypeRef](v); ok {
			fn(ref)
		}
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				walkTypeRefs(v.Field(i), fn)
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			walkTypeRefs(v.Index(i), fn)
		}
	}
}
