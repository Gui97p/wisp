package parser

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
	"github.com/Gui97p/wisp/internal/target"
)

type Directive struct {
	Name string
	Doc  string
}

var Directives = []Directive{
	{Name: "target", Doc: "Restricts the file to the targets that match the expression that follows, such as #target x64 & linux."},
}

func directiveNames() string {
	names := make([]string, len(Directives))
	for i, d := range Directives {
		names[i] = "#" + d.Name
	}
	return strings.Join(names, ", ")
}

type nativeHeader struct {
	arch  string
	at    ast.Position
	naked bool
}

func (p *Parser) selectorWord() (string, bool) {
	switch p.current.Type {
	case lexer.TOKEN_IDENT:
		return p.current.Literal, true
	case lexer.TOKEN_AMP:
		return "&", true
	case lexer.TOKEN_PIPE:
		return "|", true
	case lexer.TOKEN_NOT:
		return "!", true
	case lexer.TOKEN_LPAREN:
		return "(", true
	case lexer.TOKEN_RPAREN:
		return ")", true
	}
	return "", false
}

func (p *Parser) unexpectedInSelector() {
	d := p.errorAt(p.currentAt(), "unexpected %s in the target expression", p.current.Type.DisplayName())
	if d != nil && (p.current.Type == lexer.TOKEN_AND || p.current.Type == lexer.TOKEN_OR) {
		d.WithHelp("a target expression joins names with & and |, as in x64 & linux or release | fast")
	}
}

func (p *Parser) finishSelector(parts []string, at ast.Position) (string, bool) {
	if len(parts) == 0 {
		p.errorAt(at, "expected an architecture, a system or a target name here")
		return "", false
	}
	text := strings.Join(parts, " ")
	if err := target.ValidateSelectorText(text); err != nil {
		d := p.errorAt(at, "%s", selectorMessage(err))
		if unknown, ok := errors.AsType[*target.UnknownError](err); ok && d != nil {
			d.WithHelp("%s", unknown.Help)
		}
		return "", false
	}
	return text, true
}

func selectorMessage(err error) string {
	if unknown, ok := errors.AsType[*target.UnknownError](err); ok {
		return fmt.Sprintf("unknown name %q in the target expression", unknown.Name)
	}
	return err.Error()
}

func (p *Parser) currentAt() ast.Position {
	endLine, endCol := p.currentEnd()
	return ast.Position{Line: p.current.Line, Col: p.current.Column, EndLine: endLine, EndCol: endCol}
}

func (p *Parser) parseSelector() (string, ast.Position, bool) {
	if !p.expect(lexer.TOKEN_LT) {
		return "", ast.Position{}, false
	}
	open := p.currentAt()

	var parts []string
	at := ast.Position{Line: open.EndLine, Col: open.EndCol, EndLine: open.EndLine, EndCol: open.EndCol}
	for p.peek.Type != lexer.TOKEN_GT {
		if p.peek.Type == lexer.TOKEN_EOF || p.peek.Type == lexer.TOKEN_SEMICOLON {
			p.error("a native selector is never closed: expected >")
			return "", ast.Position{}, false
		}
		p.advance()
		word, ok := p.selectorWord()
		if !ok {
			p.unexpectedInSelector()
			return "", ast.Position{}, false
		}
		here := p.currentAt()
		if len(parts) == 0 {
			at = here
		}
		at.EndLine, at.EndCol = here.EndLine, here.EndCol
		parts = append(parts, word)
	}
	p.advance()

	text, ok := p.finishSelector(parts, at)
	return text, at, ok
}

func (p *Parser) optionalSelector() (string, ast.Position, bool) {
	if p.peek.Type != lexer.TOKEN_LT {
		return "", ast.Position{}, true
	}
	selector, at, ok := p.parseSelector()
	if !ok {
		p.skipDeclaration()
	}
	return selector, at, ok
}

func (p *Parser) skipDeclaration() {
	depth := 0
	for p.current.Type != lexer.TOKEN_EOF {
		switch p.current.Type {
		case lexer.TOKEN_LBRACE:
			depth++
		case lexer.TOKEN_RBRACE:
			depth--
			if depth <= 0 {
				return
			}
		case lexer.TOKEN_SEMICOLON:
			if depth == 0 {
				return
			}
		}
		p.advance()
	}
}

func (p *Parser) parseDirectives(program *ast.Program) {
	for p.current.Type == lexer.TOKEN_HASH {
		line := p.current.Line
		cp := p.checkpoint()
		if p.peek.Type == lexer.TOKEN_IDENT && p.peek.Line == line && !isDirective(p.peek.Literal) {
			p.restore(cp)
			return
		}
		if p.peek.Type != lexer.TOKEN_IDENT || p.peek.Line != line {
			p.errorAt(p.currentAt(), "expected a directive name after #, such as #target")
			p.skipLine(line)
			p.advance()
			continue
		}
		p.advance()
		name := p.current.Literal
		if !isDirective(name) {
			p.errorAt(p.currentAt(), "unknown directive #%s", name).WithHelp("the known directives are %s", directiveNames())
			p.skipLine(line)
			p.advance()
			continue
		}

		if program.Target != "" {
			p.errorAt(p.currentAt(), "a file has at most one #target")
		}
		var parts []string
		var at ast.Position
		for p.peek.Line == line && p.peek.Type != lexer.TOKEN_EOF {
			p.advance()
			word, ok := p.selectorWord()
			if !ok {
				p.unexpectedInSelector()
				p.skipLine(line)
				parts = nil
				break
			}
			here := p.currentAt()
			if len(parts) == 0 {
				at = here
			}
			at.EndLine, at.EndCol = here.EndLine, here.EndCol
			parts = append(parts, word)
		}
		if len(parts) == 0 && at.Line == 0 {
			at = p.currentAt()
		}
		if len(parts) > 0 {
			if text, ok := p.finishSelector(parts, at); ok {
				program.Target, program.TargetAt = text, at
			}
		}
		p.advance()
	}
}

func isDirective(name string) bool {
	for _, d := range Directives {
		if d.Name == name {
			return true
		}
	}
	return false
}

func (p *Parser) skipLine(line int) {
	for p.peek.Line == line && p.peek.Type != lexer.TOKEN_EOF {
		p.advance()
	}
}

func (p *Parser) parseNativeDeclaration() *ast.FuncDecl {
	var arch string
	var at ast.Position
	if p.peek.Type == lexer.TOKEN_LT {
		var ok bool
		arch, at, ok = p.parseSelector()
		if !ok {
			p.skipDeclaration()
			return nil
		}
	}

	p.native = &nativeHeader{arch: arch, at: at, naked: p.nakedTag}
	defer func() { p.native = nil }()

	decl := p.parseFunc(false)
	if decl == nil {
		return nil
	}
	decl.Selector, decl.SelectorAt = arch, at
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
		if p.native != nil && p.native.naked {
			return nativeResult{typ: *ref}, true
		}
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

	if p.peek.Type == lexer.TOKEN_LBRACE {
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

	if p.peek.Type != lexer.TOKEN_LBRACE {
		p.advance()
		p.errorAt(p.currentPosition(), "a native function has a block body: native<arch> name(params) result { mov rax, x; }")
		return nil
	}
	bodyAt := p.peekPosition()
	instrs, ok := p.parseNativeBlock()
	if !ok {
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

	var returned []ast.Expression
	for _, result := range results {
		if result.name == "" {
			continue
		}
		declared := &ast.VarStmt{Vars: []ast.Param{{
			Name: result.name, Type: result.typ,
			Line: result.at.Line, Col: result.at.Col, EndLine: result.at.EndLine, EndCol: result.at.EndCol,
		}}, Result: true}
		place(declared, result.at)
		body.Statements = append(body.Statements, declared)

		returned = append(returned, ident(result.name, result.at))
	}

	native := &ast.NativeStmt{Backend: header.arch, BackendAt: header.at, Instrs: instrs, BodyAt: bodyAt}
	place(native, bodyAt)
	body.Statements = append(body.Statements, native)

	if len(returned) > 0 {
		ret := &ast.ReturnStmt{Values: returned}
		place(ret, bodyAt)
		body.Statements = append(body.Statements, ret)
	}

	decl.Body = body
	decl.Native = true
	return decl
}
