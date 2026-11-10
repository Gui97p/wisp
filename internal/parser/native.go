package parser

import (
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

type nativeHeader struct {
	arch string
	at   ast.Position
}

func (p *Parser) parseNativeArch() (string, ast.Position, bool) {
	if !p.expect(lexer.TOKEN_LT) {
		return "", ast.Position{}, false
	}
	if !p.expect(lexer.TOKEN_IDENT) {
		return "", ast.Position{}, false
	}
	arch := p.current.Literal
	endLine, endCol := p.currentEnd()
	at := ast.Position{Line: p.current.Line, Col: p.current.Column, EndLine: endLine, EndCol: endCol}
	if !p.expect(lexer.TOKEN_GT) {
		return "", ast.Position{}, false
	}
	return arch, at, true
}

func (p *Parser) parseNativeDeclaration() *ast.FuncDecl {
	arch, at, ok := p.parseNativeArch()
	if !ok {
		return nil
	}

	p.native = &nativeHeader{arch: arch, at: at}
	defer func() { p.native = nil }()

	decl := p.parseFunc(false)
	if decl == nil {
		return nil
	}
	decl.NativeArch, decl.NativeAt = arch, at
	return decl
}

type nativeResult struct {
	typ  ast.TypeRef
	name string
	at   ast.Position
}

func (p *Parser) parseNativeResult() (nativeResult, bool) {
	ref := p.parseTypeDefinitionPrefix()
	if ref == nil {
		return nativeResult{}, false
	}
	if p.peek.Type != lexer.TOKEN_IDENT {
		p.error("a native function names its results, as in (uint high, uint low)")
		return nativeResult{}, false
	}
	p.advance()
	name := p.current.Literal
	if !p.checkReservedName(name) {
		return nativeResult{}, false
	}
	endLine, endCol := p.currentEnd()
	at := ast.Position{Line: p.current.Line, Col: p.current.Column, EndLine: endLine, EndCol: endCol}
	return nativeResult{typ: *ref, name: name, at: at}, true
}

func (p *Parser) parseNativeResults() ([]nativeResult, bool) {
	var results []nativeResult

	if p.peek.Type == lexer.TOKEN_ARROW {
		return results, true
	}

	p.advance()
	if p.current.Type == lexer.TOKEN_LPAREN {
		for {
			p.advance()
			result, ok := p.parseNativeResult()
			if !ok {
				return nil, false
			}
			results = append(results, result)

			if p.peek.Type == lexer.TOKEN_COMMA {
				p.advance()
				continue
			}
			if !p.expect(lexer.TOKEN_RPAREN) {
				return nil, false
			}
			return results, true
		}
	}

	result, ok := p.parseNativeResult()
	if !ok {
		return nil, false
	}
	return []nativeResult{result}, true
}

func (p *Parser) finishNative(decl *ast.FuncDecl) *ast.FuncDecl {
	header := p.native

	for _, param := range decl.Params {
		if param.Const || param.Variadic || param.Any {
			p.errorf("a native function cannot take %s as a const or variadic parameter", param.Name)
			return nil
		}
	}

	results, ok := p.parseNativeResults()
	if !ok {
		return nil
	}
	if p.peek.Type == lexer.TOKEN_NOT {
		p.error("a native function cannot be fallible")
		return nil
	}

	if !p.expect(lexer.TOKEN_ARROW) {
		return nil
	}
	if !p.expect(lexer.TOKEN_STRING_LITERAL) {
		return nil
	}
	code := p.current.Literal
	raw := p.current.Raw
	codeEndLine, codeEndCol := p.currentEnd()
	codeAt := ast.Position{Line: p.current.Line, Col: p.current.Column, EndLine: codeEndLine, EndCol: codeEndCol}
	if !p.expect(lexer.TOKEN_SEMICOLON) {
		return nil
	}

	decl.ReturnTypes = []ast.TypeRef{}
	for _, result := range results {
		decl.ReturnTypes = append(decl.ReturnTypes, result.typ)
	}

	ident := func(name string, at ast.Position) *ast.IdentLiteral {
		lit := &ast.IdentLiteral{Value: name}
		lit.SetPos(at.Line, at.Col)
		lit.SetEndPos(at.EndLine, at.EndCol)
		return lit
	}
	place := func(stmt ast.Statement, at ast.Position) {
		stmt.SetPos(at.Line, at.Col)
		stmt.SetEndPos(at.EndLine, at.EndCol)
	}

	body := &ast.BlockStmt{}

	var bindings []ast.NativeBinding
	if decl.Receiver != nil && decl.Receiver.Name != "" {
		at := ast.Position{Line: decl.Receiver.Line, Col: decl.Receiver.Col, EndLine: decl.Receiver.EndLine, EndCol: decl.Receiver.EndCol}
		bindings = append(bindings, ast.NativeBinding{In: true, Var: ident(decl.Receiver.Name, at)})
	}
	for _, param := range decl.Params {
		at := ast.Position{Line: param.Line, Col: param.Col, EndLine: param.EndLine, EndCol: param.EndCol}
		bindings = append(bindings, ast.NativeBinding{In: true, Var: ident(param.Name, at)})
	}

	var returned []ast.Expression
	for _, result := range results {
		declared := &ast.VarStmt{Vars: []ast.Param{{
			Name: result.name, Type: result.typ,
			Line: result.at.Line, Col: result.at.Col, EndLine: result.at.EndLine, EndCol: result.at.EndCol,
		}}}
		place(declared, result.at)
		body.Statements = append(body.Statements, declared)

		bindings = append(bindings, ast.NativeBinding{Out: true, Var: ident(result.name, result.at)})
		returned = append(returned, ident(result.name, result.at))
	}

	native := &ast.NativeStmt{
		Backend: header.arch, BackendAt: header.at,
		Bindings: bindings, Code: code, CodeAt: codeAt, Raw: raw,
	}
	place(native, codeAt)
	body.Statements = append(body.Statements, native)

	if len(returned) > 0 {
		ret := &ast.ReturnStmt{Values: returned}
		place(ret, codeAt)
		body.Statements = append(body.Statements, ret)
	}

	decl.Body = body
	return decl
}
