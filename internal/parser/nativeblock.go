package parser

import (
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

var nativePrefixes = map[string]bool{
	"rep": true, "repe": true, "repne": true, "repz": true, "repnz": true, "lock": true,
}

var nativeSizes = map[string]bool{
	"byte": true, "word": true, "dword": true, "qword": true, "tword": true,
	"oword": true, "yword": true, "zword": true,
}

func isWordToken(t lexer.Token) bool {
	if t.Type == lexer.TOKEN_IDENT {
		return true
	}
	if t.Literal == "" {
		return false
	}
	c := t.Literal[0]
	if !(c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
		return false
	}
	switch t.Type {
	case lexer.TOKEN_STRING_LITERAL, lexer.TOKEN_CHAR_LITERAL, lexer.TOKEN_INT_LITERAL, lexer.TOKEN_FLOAT_LITERAL:
		return false
	}
	return true
}

func (p *Parser) nativeEnd(at *ast.Position) {
	at.EndLine, at.EndCol = p.currentEnd()
}

func (p *Parser) skipNativeBlock() {
	depth := 1
	for p.current.Type != lexer.TOKEN_EOF {
		switch p.current.Type {
		case lexer.TOKEN_LBRACE:
			depth++
		case lexer.TOKEN_RBRACE:
			depth--
			if depth == 0 {
				return
			}
		}
		p.advance()
	}
}

func (p *Parser) parseNativeBlock() ([]ast.NativeInstr, bool) {
	var instrs []ast.NativeInstr
	if !p.expect(lexer.TOKEN_LBRACE) {
		return nil, false
	}
	for {
		if p.peek.Type == lexer.TOKEN_RBRACE {
			p.advance()
			return instrs, true
		}
		if p.peek.Type == lexer.TOKEN_EOF {
			p.error("a native block is never closed: expected }")
			return nil, false
		}
		p.advance()
		parsed, ok := p.parseNativeInstr()
		if !ok {
			p.errorSkip()
			return nil, false
		}
		instrs = append(instrs, parsed...)
	}
}

func (p *Parser) errorSkip() {
	p.skipNativeBlock()
}

func (p *Parser) parseNativeInstr() ([]ast.NativeInstr, bool) {
	var out []ast.NativeInstr
	if !isWordToken(p.current) {
		p.errorAt(p.currentPosition(), "expected an instruction or a label, got %s", p.current.Type.DisplayName())
		return nil, false
	}

	if p.peek.Type == lexer.TOKEN_COLON {
		label := ast.NativeInstr{Label: p.current.Literal, At: p.currentPosition()}
		p.advance()
		p.nativeEnd(&label.At)
		if p.peek.Type == lexer.TOKEN_SEMICOLON {
			p.advance()
			return []ast.NativeInstr{label}, true
		}
		if p.peek.Type == lexer.TOKEN_RBRACE {
			return []ast.NativeInstr{label}, true
		}
		p.advance()
		if !isWordToken(p.current) {
			p.errorAt(p.currentPosition(), "expected an instruction after the label, got %s", p.current.Type.DisplayName())
			return nil, false
		}
		rest, ok := p.parseNativeInstr()
		if !ok {
			return nil, false
		}
		return append([]ast.NativeInstr{label}, rest...), true
	}

	instr := ast.NativeInstr{At: p.currentPosition()}
	for nativePrefixes[p.current.Literal] && isWordToken(p.peek) {
		instr.Prefixes = append(instr.Prefixes, p.current.Literal)
		p.advance()
	}
	instr.Mnemonic = p.current.Literal
	p.nativeEnd(&instr.At)

	if p.peek.Type == lexer.TOKEN_SEMICOLON {
		p.advance()
		return append(out, instr), true
	}

	for {
		p.advance()
		operand, ok := p.parseNativeOperand()
		if !ok {
			return nil, false
		}
		instr.Operands = append(instr.Operands, operand)
		if p.peek.Type == lexer.TOKEN_COMMA {
			p.advance()
			continue
		}
		break
	}
	if p.peek.Type != lexer.TOKEN_SEMICOLON {
		p.advance()
		p.errorAt(p.currentPosition(), "expected ',' or ';' after the operand, got %s", p.current.Type.DisplayName())
		return nil, false
	}
	p.advance()
	p.nativeEnd(&instr.At)
	return append(out, instr), true
}

func (p *Parser) parseNativePath() ([]string, bool) {
	path := []string{p.current.Literal}
	for p.peek.Type == lexer.TOKEN_DOT {
		p.advance()
		p.advance()
		if !isWordToken(p.current) {
			p.errorAt(p.currentPosition(), "expected a field name after '.', got %s", p.current.Type.DisplayName())
			return nil, false
		}
		path = append(path, p.current.Literal)
	}
	return path, true
}

func (p *Parser) parseNativeOperand() (ast.NativeOperand, bool) {
	at := p.currentPosition()
	op := ast.NativeOperand{At: at}

	switch {
	case p.current.Type == lexer.TOKEN_MINUS || p.current.Type == lexer.TOKEN_PLUS:
		sign := ""
		if p.current.Type == lexer.TOKEN_MINUS {
			sign = "-"
		}
		if p.peek.Type != lexer.TOKEN_INT_LITERAL {
			p.advance()
			p.errorAt(p.currentPosition(), "expected a number after the sign, got %s", p.current.Type.DisplayName())
			return op, false
		}
		p.advance()
		op.Kind, op.Number = ast.NativeNumber, sign+p.current.Literal
	case p.current.Type == lexer.TOKEN_INT_LITERAL:
		op.Kind, op.Number = ast.NativeNumber, p.current.Literal
	case p.current.Type == lexer.TOKEN_CHAR_LITERAL:
		op.Kind, op.Number = ast.NativeNumber, "'"+p.current.Literal+"'"
	case p.current.Type == lexer.TOKEN_AMP:
		p.advance()
		if !isWordToken(p.current) {
			p.errorAt(p.currentPosition(), "expected a variable name after &, got %s", p.current.Type.DisplayName())
			return op, false
		}
		path, ok := p.parseNativePath()
		if !ok {
			return op, false
		}
		op.Kind, op.Path = ast.NativeAddr, path
	case p.current.Type == lexer.TOKEN_LBRACKET:
		return p.parseNativeMem("", at)
	case isWordToken(p.current):
		if nativeSizes[p.current.Literal] && p.peek.Type == lexer.TOKEN_LBRACKET {
			size := p.current.Literal
			p.advance()
			return p.parseNativeMem(size, at)
		}
		path, ok := p.parseNativePath()
		if !ok {
			return op, false
		}
		op.Kind, op.Path = ast.NativeWord, path
	default:
		p.errorAt(p.currentPosition(), "expected an operand, got %s", p.current.Type.DisplayName())
		return op, false
	}
	p.nativeEnd(&op.At)
	return op, true
}

func (p *Parser) parseNativeMem(size string, at ast.Position) (ast.NativeOperand, bool) {
	op := ast.NativeOperand{Kind: ast.NativeMem, Size: size, At: at}

	if isWordToken(p.peek) && p.peek.Literal == "rel" {
		p.advance()
		op.Rel = true
	}

	negative := false
	for first := true; ; first = false {
		p.advance()
		if p.current.Type == lexer.TOKEN_PLUS || p.current.Type == lexer.TOKEN_MINUS {
			negative = p.current.Type == lexer.TOKEN_MINUS
			p.advance()
		} else if !first {
			p.errorAt(p.currentPosition(), "expected '+', '-' or ']' in the address, got %s", p.current.Type.DisplayName())
			return op, false
		}

		term := ast.NativeTerm{Negative: negative, At: p.currentPosition()}
		negative = false
		if p.current.Type == lexer.TOKEN_INT_LITERAL && p.peek.Type == lexer.TOKEN_STAR {
			term.Scale = p.current.Literal
			p.advance()
			p.advance()
		}
		switch {
		case p.current.Type == lexer.TOKEN_INT_LITERAL:
			term.Number = p.current.Literal
		case isWordToken(p.current):
			path, ok := p.parseNativePath()
			if !ok {
				return op, false
			}
			term.Path = path
			if term.Scale == "" && p.peek.Type == lexer.TOKEN_STAR {
				p.advance()
				p.advance()
				if p.current.Type != lexer.TOKEN_INT_LITERAL {
					p.errorAt(p.currentPosition(), "expected a scale (1, 2, 4 or 8) after '*', got %s", p.current.Type.DisplayName())
					return op, false
				}
				term.Scale = p.current.Literal
			}
		default:
			p.errorAt(p.currentPosition(), "expected a register, a number or a name in the address, got %s", p.current.Type.DisplayName())
			return op, false
		}
		p.nativeEnd(&term.At)
		op.Terms = append(op.Terms, term)

		if p.peek.Type == lexer.TOKEN_RBRACKET {
			p.advance()
			break
		}
		if p.peek.Type != lexer.TOKEN_PLUS && p.peek.Type != lexer.TOKEN_MINUS {
			p.advance()
			p.errorAt(p.currentPosition(), "expected '+', '-' or ']' in the address, got %s", p.current.Type.DisplayName())
			return op, false
		}
	}
	p.nativeEnd(&op.At)
	return op, true
}

func (p *Parser) peekPosition() ast.Position {
	at := ast.Position{Line: p.peek.Line, Col: p.peek.Column, EndLine: p.peek.Line, EndCol: p.peek.Column}
	if p.peek.EndLine != 0 {
		at.EndLine, at.EndCol = p.peek.EndLine, p.peek.EndColumn
	}
	return at
}
