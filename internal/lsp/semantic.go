package lsp

import (
	"reflect"
	"sort"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var semanticTokenTypes = []string{"property", "enumMember", "type", "typeParameter", "function", "method", "number", "keyword", "operator", "formatPlain", "formatBrace", "formatSpec"}

const (
	tokenProperty = iota
	tokenEnumMember
	tokenType
	tokenTypeParameter
	tokenFunction
	tokenMethod
	tokenNumber
	tokenKeyword
	tokenOperator
	tokenFormatPlain
	tokenFormatBrace
	tokenFormatSpec
)

type semanticToken struct {
	line, col, length, kind int
}

func semanticTokens(context *glsp.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	path := uriToPath(params.TextDocument.URI)

	program, info, _ := analyseEntryChecked(path)
	if program == nil || info == nil {
		return &protocol.SemanticTokens{Data: []uint32{}}, nil
	}

	program = withoutInstances(declarationsOf(program, declarationFiles(path), path))

	return &protocol.SemanticTokens{Data: encodeTokens(collectTokens(program, info, fileBuffer[path]))}, nil
}

func withoutInstances(program *ast.Program) *ast.Program {
	kept := &ast.Program{}
	*kept = *program
	kept.Declarations = nil
	for _, decl := range program.Declarations {
		if fd, ok := decl.(*ast.FuncDecl); ok && fd.Instance != "" {
			continue
		}
		kept.Declarations = append(kept.Declarations, decl)
	}
	return kept
}

func collectTokens(program *ast.Program, info *analyser.Info, src []byte) []semanticToken {
	var tokens []semanticToken

	params := typeParamUses(program)

	add := func(at ast.Position, kind int) {
		if at.Line == 0 {
			return
		}
		if kind == tokenType && params[[2]int{at.Line, at.Col}] {
			kind = tokenTypeParameter
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

	for _, scope := range typeParamScopes(program) {
		for _, param := range scope.params {
			add(param.At, tokenTypeParameter)
		}
	}

	walkTypeRefs(reflect.ValueOf(program), func(ref ast.TypeRef) {
		if ref.Name == "" || ref.IsFunc || ref.Line == 0 {
			return
		}
		if primitiveType(ref.Name) && ref.Module == "" {
			return
		}
		col := ref.Col
		if ref.Module != "" {
			col += len(ref.Module) + 1
		}
		add(ast.Position{Line: ref.Line, Col: col, EndLine: ref.Line, EndCol: col + len(ref.Name) - 1}, tokenType)
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
		case *ast.AnonLiteral:
			for _, at := range e.KeyAt {
				add(at, tokenProperty)
			}
		case *ast.StructLiteral:
			for _, at := range e.KeyAt {
				add(at, tokenProperty)
			}
			line, col := e.Position()
			span := func(from, length int) ast.Position {
				return ast.Position{Line: line, Col: from, EndLine: line, EndCol: from + length - 1}
			}
			if _, isVariant := info.Variants[e]; isVariant {
				switch {
				case e.Dot:
					add(span(col+1, len(e.Name)), tokenEnumMember)
				case e.Variant != "":
					typeAt := col + len(e.Module) + 1
					add(span(typeAt, len(e.Name)), tokenType)
					add(span(typeAt+len(e.Name)+1, len(e.Variant)), tokenEnumMember)
				default:
					add(span(col, len(e.Module)), tokenType)
					add(span(col+len(e.Module)+1, len(e.Name)), tokenEnumMember)
				}
				return
			}
			if e.Module != "" {
				col += len(e.Module) + 1
			}
			add(span(col, len(e.Name)), tokenType)
		}
	})

	lines := splitLines(src)
	for _, site := range info.FormatSites {
		for _, at := range site.Opens {
			add(at, tokenFormatBrace)
		}
		for _, at := range site.Closes {
			add(at, tokenFormatBrace)
		}
		for _, at := range site.Specs {
			add(at, tokenFormatSpec)
		}
		for _, expr := range site.Exprs {
			formatExprTokens(expr, info, add)
		}
		formatTextTokens(lines, site, add)
	}

	return tokens
}

func formatExprTokens(expr ast.Expression, info *analyser.Info, add func(ast.Position, int)) {
	word := func(line, col, length int, kind int) {
		add(ast.Position{Line: line, Col: col, EndLine: line, EndCol: col + length - 1}, kind)
	}
	ast.Inspect(expr, func(n ast.Node) bool {
		switch e := n.(type) {
		case *ast.IdentLiteral:
			if e.Synthetic {
				return true
			}
			line, col := e.Position()
			if sym := info.Idents[e]; sym != nil {
				switch sym.Kind {
				case analyser.FUNC:
					word(line, col, len(e.Value), tokenFunction)
				case analyser.TYPE, analyser.STRUCT:
				default:
					word(line, col, len(e.Value), tokenFormatPlain)
				}
			} else {
				word(line, col, len(e.Value), tokenFormatPlain)
			}
		case *ast.IntLiteral, *ast.FloatLiteral:
			line, col := e.(ast.Expression).Position()
			endLine, endCol := e.(ast.Expression).EndPosition()
			if line == endLine {
				word(line, col, endCol-col+1, tokenNumber)
			}
		case *ast.BoolLiteral, *ast.NullLiteral:
			line, col := e.(ast.Expression).Position()
			endLine, endCol := e.(ast.Expression).EndPosition()
			if line == endLine {
				word(line, col, endCol-col+1, tokenKeyword)
			}
		case *ast.CallExpr:
			if member, ok := e.Name.(*ast.MemberExpr); ok && info.MethodCalls[e] != nil {
				add(member.FieldAt, tokenMethod)
			}
		case *ast.MemberExpr:
			if object, ok := e.Object.(*ast.IdentLiteral); ok {
				if owner := info.Idents[object]; owner != nil && owner.Kind == analyser.MODULE {
					if mt, isModule := owner.Type.(*analyser.ModuleType); isModule {
						if sym := mt.Exports[e.Field]; sym != nil {
							switch sym.Kind {
							case analyser.FUNC:
								add(e.FieldAt, tokenFunction)
							case analyser.VAR, analyser.CONST:
								add(e.FieldAt, tokenProperty)
							}
						}
					}
				}
			}
		}
		return true
	})
}

func formatTextTokens(lines []string, site analyser.FormatSite, add func(ast.Position, int)) {
	const operators = "+-*/%&|^~!<>=?:@"
	for k, open := range site.Opens {
		closing := site.Closes[k]
		if open.Line != closing.Line || open.Line < 1 || open.Line > len(lines) {
			continue
		}
		end := closing.Col - 1
		for _, spec := range site.Specs {
			if spec.Line == open.Line && spec.Col > open.Col && spec.EndCol < closing.Col {
				end = spec.Col - 1
			}
		}

		runes := []rune(lines[open.Line-1])
		offsets := make([]int, len(runes)+1)
		for i, r := range runes {
			width := 1
			if r > 0xFFFF {
				width = 2
			}
			offsets[i+1] = offsets[i] + width
		}
		emit := func(from, to, kind int) {
			add(ast.Position{Line: open.Line, Col: offsets[from] + 1, EndLine: open.Line, EndCol: offsets[to]}, kind)
		}

		i := 0
		for i < len(runes) && offsets[i] < open.Col {
			i++
		}
		for i < len(runes) && offsets[i] < end {
			c := runes[i]
			switch {
			case c == '"' || c == '\'':
				i++
				for i < len(runes) && offsets[i] < end && runes[i] != c {
					if runes[i] == '\\' {
						i++
					}
					i++
				}
				i++
			case c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z':
				float := c >= '0' && c <= '9'
				for i < len(runes) && (runes[i] == '_' || runes[i] >= '0' && runes[i] <= '9' || runes[i] >= 'a' && runes[i] <= 'z' || runes[i] >= 'A' && runes[i] <= 'Z') {
					i++
				}
				if float && i+1 < len(runes) && runes[i] == '.' && runes[i+1] >= '0' && runes[i+1] <= '9' {
					i++
					for i < len(runes) && (runes[i] == '_' || runes[i] >= '0' && runes[i] <= '9') {
						i++
					}
				}
			case c == '.':
				from := i
				switch {
				case i+1 < len(runes) && runes[i+1] == '.':
					for i < len(runes) && runes[i] == '.' {
						i++
					}
					if i < len(runes) && runes[i] == '=' {
						i++
					}
					emit(from, i, tokenOperator)
				case i+1 < len(runes) && runes[i+1] == '?':
					i += 2
					emit(from, i, tokenOperator)
				default:
					i++
					emit(from, i, tokenFormatPlain)
				}
			case strings.ContainsRune(operators, c):
				from := i
				for i < len(runes) && offsets[i] < end && strings.ContainsRune(operators, runes[i]) {
					i++
				}
				emit(from, i, tokenOperator)
			case strings.ContainsRune(",()[]{}", c):
				emit(i, i+1, tokenFormatPlain)
				i++
			default:
				i++
			}
		}
	}
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
			if deltaCol <= 0 && len(data) > 0 {
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
		"float", "float32", "float64", "bool", "byte", "string", "void":
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
