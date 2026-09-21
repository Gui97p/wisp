package parser

import (
	"github.com/Gui97p/wisp/internal/lexer"
)

type Precedence int

const (
	LOWEST     Precedence = iota
	TERNARY               // x ? a : b
	OR                    // ||
	AND                   // &&
	BIT_OR                // |
	BIT_XOR               // ^
	BIT_AND               // &
	EQUALITY              // == !=
	COMPARISON            // < <= > >=
	SHIFT                 // << >>
	SUM                   // + -
	PRODUCT               // * / %
	PREFIX                // -x !x
	CAST                  // as x
	CALL                  // fn(...)
	INDEX                 // x[0]
	MEMBER                // x.field
)

var precedences = map[lexer.TokenType]Precedence{
	lexer.TOKEN_QUESTION_MARK: TERNARY,
	lexer.TOKEN_COALESCE:      TERNARY,

	lexer.TOKEN_OR:  OR,
	lexer.TOKEN_AND: AND,

	lexer.TOKEN_PIPE: BIT_OR,
	lexer.TOKEN_XOR:  BIT_XOR,
	lexer.TOKEN_AMP:  BIT_AND,

	lexer.TOKEN_EQUAL:     EQUALITY,
	lexer.TOKEN_NOT_EQUAL: EQUALITY,

	lexer.TOKEN_LT:  COMPARISON,
	lexer.TOKEN_LTE: COMPARISON,
	lexer.TOKEN_GT:  COMPARISON,
	lexer.TOKEN_GTE: COMPARISON,

	lexer.TOKEN_SHIFT_LEFT:  SHIFT,
	lexer.TOKEN_SHIFT_RIGHT: SHIFT,

	lexer.TOKEN_PLUS:  SUM,
	lexer.TOKEN_MINUS: SUM,

	lexer.TOKEN_STAR:    PRODUCT,
	lexer.TOKEN_SLASH:   PRODUCT,
	lexer.TOKEN_PERCENT: PRODUCT,

	lexer.TOKEN_AS:       CAST,
	lexer.TOKEN_LPAREN:   CALL,
	lexer.TOKEN_LBRACE:   CALL,
	lexer.TOKEN_LBRACKET: INDEX,
	lexer.TOKEN_DOT:      MEMBER,
}

func (p *Parser) currentPrecedence() Precedence {
	if prec, ok := precedences[p.current.Type]; ok {
		return prec
	}
	return LOWEST
}

func (p *Parser) peekPrecedence() Precedence {
	if prec, ok := precedences[p.peek.Type]; ok {
		return prec
	}
	return LOWEST
}
