package lexer

import (
	"strings"
)

const EOF byte = 0

type Lexer struct {
	buf []byte

	pos    int
	line   int
	column int

	ch byte
}

func NewLexer(buffer []byte) *Lexer {
	l := &Lexer{
		buf:    buffer,
		pos:    0,
		line:   1,
		column: 1,
	}

	if len(buffer) > 0 {
		l.ch = buffer[0]
	}

	return l
}

func (l *Lexer) NextToken() Token {
	for isSpace(l.ch) {
		l.advance()
	}

	var t Token

	switch l.ch {
	case EOF:
		t = l.token(TOKEN_EOF, "end of file")
	case '=':
		switch l.peek() {
		case '=':
			l.advance()
			t = l.token(TOKEN_EQUAL, "==")
		case '>':
			l.advance()
			t = l.token(TOKEN_ARROW, "=>")
		default:
			t = l.token(TOKEN_ASSIGN, "=")
		}
	case '+':
		switch l.peek() {
		case '=':
			l.advance()
			t = l.token(TOKEN_PLUS_ASSIGN, "+=")
		case '+':
			l.advance()
			t = l.token(TOKEN_INCREMENT, "++")
		default:
			t = l.token(TOKEN_PLUS, "+")
		}
	case '-':
		switch l.peek() {
		case '=':
			l.advance()
			t = l.token(TOKEN_MINUS_ASSIGN, "-=")
		case '-':
			l.advance()
			t = l.token(TOKEN_DECREMENT, "--")
		default:
			t = l.token(TOKEN_MINUS, "-")
		}
	case '*':
		switch l.peek() {
		case '=':
			l.advance()
			t = l.token(TOKEN_STAR_ASSIGN, "*=")
		default:
			t = l.token(TOKEN_STAR, "*")
		}
	case '/':
		switch l.peek() {
		case '=':
			l.advance()
			t = l.token(TOKEN_SLASH_ASSIGN, "/=")
		case '/':
			for l.peek() != '\n' && l.peek() != EOF {
				l.advance()
			}
			l.advance()
			return l.NextToken()
		case '*':
			for {
				if l.peek() == EOF {
					return l.token(TOKEN_ILLEGAL, "unterminated block comment")
				}

				if l.peek() == '*' && l.peekNext() == '/' {
					l.advance()
					l.advance()
					break
				}

				l.advance()
			}
			l.advance()
			return l.NextToken()
		default:
			t = l.token(TOKEN_SLASH, "/")
		}
	case '%':
		switch l.peek() {
		case '=':
			l.advance()
			t = l.token(TOKEN_PERCENT_ASSIGN, "%=")
		default:
			t = l.token(TOKEN_PERCENT, "%")
		}
	case '!':
		if l.peek() == '=' {
			l.advance()
			t = l.token(TOKEN_NOT_EQUAL, "!=")
		} else {
			t = l.token(TOKEN_NOT, "!")
		}
	case '>':
		if l.peek() == '=' {
			l.advance()
			t = l.token(TOKEN_GTE, ">=")
		} else {
			t = l.token(TOKEN_GT, ">")
		}
	case '<':
		if l.peek() == '=' {
			l.advance()
			t = l.token(TOKEN_LTE, "<=")
		} else {
			t = l.token(TOKEN_LT, "<")
		}
	case '&':
		if l.peek() == '&' {
			l.advance()
			t = l.token(TOKEN_AND, "&&")
		} else {
			t = l.token(TOKEN_ILLEGAL, "&")
		}
	case '|':
		if l.peek() == '|' {
			l.advance()
			t = l.token(TOKEN_OR, "||")
		} else {
			t = l.token(TOKEN_ILLEGAL, "|")
		}
	case ',':
		t = l.token(TOKEN_COMMA, ",")
	case ';':
		t = l.token(TOKEN_SEMICOLON, ";")
	case ':':
		t = l.token(TOKEN_COLON, ":")
	case '.':
		if l.peek() == '.' {
			l.advance()
			t = l.token(TOKEN_RANGE, "..")
		} else if isNumeric(l.peek()) {
			var builder strings.Builder
			builder.WriteString("0.")
			for isNumeric(l.peek()) {
				builder.WriteByte(l.advance())
			}
			t = l.token(TOKEN_FLOAT_LITERAL, builder.String())
		} else {
			t = l.token(TOKEN_DOT, ".")
		}
	case '(':
		t = l.token(TOKEN_LPAREN, "(")
	case ')':
		t = l.token(TOKEN_RPAREN, ")")
	case '{':
		t = l.token(TOKEN_LBRACE, "{")
	case '}':
		t = l.token(TOKEN_RBRACE, "}")
	case '[':
		t = l.token(TOKEN_LBRACKET, "[")
	case ']':
		t = l.token(TOKEN_RBRACKET, "]")

	case '"':
		var str strings.Builder
		for {
			l.advance()

			if l.ch == EOF {
				return l.token(TOKEN_ILLEGAL, "unterminated string")
			}

			if l.ch == '"' {
				break
			}

			str.WriteByte(l.ch)
		}

		t = l.token(TOKEN_STRING_LITERAL, str.String())
	case '\'':
		l.advance()

		if l.ch == EOF || l.ch == '\'' {
			return l.token(TOKEN_ILLEGAL, "empty char literal")
		}

		value := l.ch

		if l.ch != '\'' {
			return l.token(TOKEN_ILLEGAL, "invalid char literal")
		}

		t = l.token(TOKEN_CHAR_LITERAL, string(value))

	default:
		var builder strings.Builder
		if isAlpha(l.ch) {
			builder.WriteByte(l.ch)
			for isAlphaNumeric(l.peek()) {
				builder.WriteByte(l.advance())
			}
			str := builder.String()

			tType, ok := keywords[str]
			if ok {
				t = l.token(tType, str)
			} else {
				t = l.token(TOKEN_IDENT, str)
			}
		} else if isNumeric(l.ch) {
			builder.WriteByte(l.ch)
			for isNumeric(l.peek()) {
				builder.WriteByte(l.advance())
			}

			if l.peek() == '.' && l.peekNext() != '.' {
				builder.WriteByte(l.advance())
				if isNumeric(l.peek()) {
					for isNumeric(l.peek()) {
						builder.WriteByte(l.advance())
					}
					l.advance()
					return l.token(TOKEN_FLOAT_LITERAL, builder.String())
				} else {
					return l.token(TOKEN_ILLEGAL, "invalid float definition")
				}
			}
			t = l.token(TOKEN_INT_LITERAL, builder.String())
		} else {
			t = l.token(TOKEN_ILLEGAL, string(l.ch))
		}
	}

	l.advance()
	return t
}

func (l *Lexer) token(t TokenType, lit string) Token {
	return Token{Type: t, Literal: lit, Line: l.line, Column: l.column}
}

func (l *Lexer) read(pos int) byte {
	if pos >= len(l.buf) {
		return EOF
	}

	return l.buf[pos]
}

func (l *Lexer) advance() byte {
	l.pos++
	l.ch = l.read(l.pos)

	if l.ch == '\n' {
		l.line++
		l.column = 0
	} else {
		l.column++
	}

	return l.ch
}

func (l *Lexer) peek() byte {
	return l.read(l.pos + 1)
}

func (l *Lexer) peekNext() byte {
	return l.read(l.pos + 2)
}

func isSpace(ch byte) bool {
	switch ch {
	case ' ', '\t', '\n', '\r':
		return true
	default:
		return false
	}
}

func isAlpha(ch byte) bool {
	return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || ch == '_'
}

func isNumeric(ch byte) bool {
	return ch >= 48 && ch <= 57
}

func isAlphaNumeric(ch byte) bool {
	return isAlpha(ch) || isNumeric(ch)
}
