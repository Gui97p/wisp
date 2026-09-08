package parser

import (
	"github.com/Gui97p/wisp/internal/lexer"
)

type Precedence int

const (
	LOWEST     Precedence = iota
	ASSIGN                // = += -= *= /= %=
	OR                    // ||
	AND                   // &&
	EQUALITY              // == !=
	COMPARISON            // < <= > >=
	RANGE                 // ..
	SUM                   // + -
	PRODUCT               // * / %
	PREFIX                // -x !x
	POSTFIX               // x++ x--
	CALL                  // fn(...)
	INDEX                 // x[0]
	MEMBER                // x.field
)

var precedences = map[lexer.TokenType]Precedence{
	lexer.TOKEN_ASSIGN:         ASSIGN,
	lexer.TOKEN_PLUS_ASSIGN:    ASSIGN,
	lexer.TOKEN_MINUS_ASSIGN:   ASSIGN,
	lexer.TOKEN_STAR_ASSIGN:    ASSIGN,
	lexer.TOKEN_SLASH_ASSIGN:   ASSIGN,
	lexer.TOKEN_PERCENT_ASSIGN: ASSIGN,

	lexer.TOKEN_OR:  OR,
	lexer.TOKEN_AND: AND,

	lexer.TOKEN_EQUAL:     EQUALITY,
	lexer.TOKEN_NOT_EQUAL: EQUALITY,

	lexer.TOKEN_LT:  COMPARISON,
	lexer.TOKEN_LTE: COMPARISON,
	lexer.TOKEN_GT:  COMPARISON,
	lexer.TOKEN_GTE: COMPARISON,

	lexer.TOKEN_RANGE: RANGE,

	lexer.TOKEN_PLUS:  SUM,
	lexer.TOKEN_MINUS: SUM,

	lexer.TOKEN_STAR:    PRODUCT,
	lexer.TOKEN_SLASH:   PRODUCT,
	lexer.TOKEN_PERCENT: PRODUCT,

	lexer.TOKEN_INCREMENT: POSTFIX,
	lexer.TOKEN_DECREMENT: POSTFIX,

	lexer.TOKEN_LPAREN:   CALL,
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
