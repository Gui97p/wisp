package parser

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) atCapture() bool {
	cp := p.checkpoint()
	defer p.restore(cp)

	p.advance()
	if p.peek.Type != lexer.TOKEN_IDENT {
		return false
	}
	p.advance()
	if p.peek.Type != lexer.TOKEN_PIPE {
		return false
	}
	p.advance()
	return p.peek.Type == lexer.TOKEN_LBRACE
}

func (p *Parser) parseConditionWithCapture() (ast.Expression, string, ast.Position, bool) {
	prevStruct, prevCapture := p.allowStructLiteral, p.captureAllowed
	p.allowStructLiteral = false
	p.captureAllowed = true
	cond := p.parseExpression()
	p.allowStructLiteral, p.captureAllowed = prevStruct, prevCapture
	if cond == nil {
		return nil, "", ast.Position{}, false
	}

	if p.peek.Type != lexer.TOKEN_PIPE || !p.atCapture() {
		return cond, "", ast.Position{}, true
	}

	p.advance()
	if !p.expect(lexer.TOKEN_IDENT) {
		return nil, "", ast.Position{}, false
	}
	name := p.current.Literal
	at := p.currentPosition()
	if !p.checkReservedName(name) {
		return nil, "", ast.Position{}, false
	}
	if !p.expect(lexer.TOKEN_PIPE) {
		return nil, "", ast.Position{}, false
	}
	return cond, name, at, true
}

func (p *Parser) captureSource(value ast.Expression) (*ast.VarStmt, *ast.IdentLiteral) {
	name := fmt.Sprintf("__wisp_capture_%d", p.captureCount)
	p.captureCount++

	line, col := value.Position()
	endLine, endCol := value.EndPosition()

	decl := &ast.VarStmt{}
	decl.Vars = append(decl.Vars, ast.Param{Name: name})
	decl.Values = append(decl.Values, value)
	decl.SetPos(line, col)
	decl.SetEndPos(endLine, endCol)

	ident := &ast.IdentLiteral{Value: name, Synthetic: true}
	ident.SetPos(line, col)
	ident.SetEndPos(endLine, endCol)
	return decl, ident
}

func (p *Parser) wrapCapture(stmt *ast.IfStmt, value ast.Expression, name string, at ast.Position) ast.Statement {
	decl, ident := p.captureSource(value)
	stmt.Condition = ident
	stmt.Capture = name
	stmt.CaptureAt = at
	stmt.CaptureSrc = ident

	line, col := value.Position()
	block := &ast.BlockStmt{Statements: []ast.Statement{decl, stmt}}
	block.SetPos(line, col)
	block.SetEndPos(p.currentEnd())
	return block
}
