package format

import "github.com/Gui97p/wisp/internal/lexer"

type role int

const (
	roleWord role = iota
	roleBinary
	roleUnary
	rolePostfix
	roleMemberDot
	rolePrefixDot
	roleTernaryQ
	roleOptionalQ
	roleTernaryColon
	roleKeyColon
	roleLabelColon
	roleTightColon
	roleCapOpen
	roleCapClose
	roleSelectorOpen
	roleSelectorClose
	roleGenericOpen
	roleGenericClose
	roleBlockOpen
	roleLiteralOpen
	roleBlockClose
	roleLiteralClose
	roleDeclBang
	roleDeclDots
	roleSpreadDots
	roleTypeBracketClose
	roleParamClose
	roleRange
	roleHash
)

func isTypeKeyword(k lexer.TokenType) bool {
	return k >= lexer.TOKEN_INT && k <= lexer.TOKEN_STRING
}

func isKeyword(k lexer.TokenType) bool {
	switch k {
	case lexer.TOKEN_LET, lexer.TOKEN_CONST, lexer.TOKEN_FUNC, lexer.TOKEN_EXTERN, lexer.TOKEN_STRUCT, lexer.TOKEN_TYPE,
		lexer.TOKEN_ENUM, lexer.TOKEN_SWITCH, lexer.TOKEN_CASE, lexer.TOKEN_DEFAULT, lexer.TOKEN_IF, lexer.TOKEN_ELSE,
		lexer.TOKEN_FOR, lexer.TOKEN_IN, lexer.TOKEN_LOOP, lexer.TOKEN_UNTIL, lexer.TOKEN_RETURN, lexer.TOKEN_BREAK,
		lexer.TOKEN_CONTINUE, lexer.TOKEN_IMPORT, lexer.TOKEN_PUBLIC, lexer.TOKEN_INTERNAL, lexer.TOKEN_PRIVATE,
		lexer.TOKEN_AS, lexer.TOKEN_NATIVE, lexer.TOKEN_DEFER, lexer.TOKEN_YIELD, lexer.TOKEN_INLINE, lexer.TOKEN_VAR,
		lexer.TOKEN_STATIC, lexer.TOKEN_INTERFACE, lexer.TOKEN_IMPLEMENTS:
		return true
	}
	return false
}

func isBinaryOperator(k lexer.TokenType) bool {
	switch k {
	case lexer.TOKEN_ASSIGN, lexer.TOKEN_PLUS, lexer.TOKEN_MINUS, lexer.TOKEN_STAR, lexer.TOKEN_SLASH, lexer.TOKEN_PERCENT,
		lexer.TOKEN_AMP, lexer.TOKEN_PIPE, lexer.TOKEN_XOR, lexer.TOKEN_SHIFT_LEFT, lexer.TOKEN_SHIFT_RIGHT,
		lexer.TOKEN_PLUS_ASSIGN, lexer.TOKEN_MINUS_ASSIGN, lexer.TOKEN_STAR_ASSIGN, lexer.TOKEN_SLASH_ASSIGN,
		lexer.TOKEN_PERCENT_ASSIGN, lexer.TOKEN_AMP_ASSIGN, lexer.TOKEN_PIPE_ASSIGN, lexer.TOKEN_XOR_ASSIGN,
		lexer.TOKEN_SHIFT_LEFT_ASSIGN, lexer.TOKEN_SHIFT_RIGHT_ASSIGN, lexer.TOKEN_EQUAL, lexer.TOKEN_NOT_EQUAL,
		lexer.TOKEN_LT, lexer.TOKEN_LTE, lexer.TOKEN_GT, lexer.TOKEN_GTE, lexer.TOKEN_AND, lexer.TOKEN_OR,
		lexer.TOKEN_COALESCE, lexer.TOKEN_ARROW:
		return true
	}
	return false
}

func operandEnd(t token, r role) bool {
	switch t.kind {
	case lexer.TOKEN_IDENT, lexer.TOKEN_INT_LITERAL, lexer.TOKEN_FLOAT_LITERAL, lexer.TOKEN_STRING_LITERAL,
		lexer.TOKEN_CHAR_LITERAL, lexer.TOKEN_TRUE, lexer.TOKEN_FALSE, lexer.TOKEN_NULL, lexer.TOKEN_AT:
		return true
	case lexer.TOKEN_RPAREN:
		return r != roleParamClose
	case lexer.TOKEN_RBRACKET:
		return r != roleTypeBracketClose
	case lexer.TOKEN_RBRACE:
		return r == roleLiteralClose
	case lexer.TOKEN_NOT, lexer.TOKEN_INCREMENT, lexer.TOKEN_DECREMENT:
		return r == rolePostfix
	case lexer.TOKEN_VARIADIC:
		return r == roleSpreadDots
	}
	return isTypeKeyword(t.kind)
}

type frame struct {
	open      lexer.TokenType
	literal   bool
	ternary   int
	atTop     bool
	opensDecl bool
}

type stmtState struct {
	header bool
	fn     bool
	arrow  bool
}

func headerKeyword(k lexer.TokenType) bool {
	switch k {
	case lexer.TOKEN_IF, lexer.TOKEN_ELSE, lexer.TOKEN_FOR, lexer.TOKEN_LOOP, lexer.TOKEN_UNTIL, lexer.TOKEN_SWITCH,
		lexer.TOKEN_FUNC, lexer.TOKEN_STRUCT, lexer.TOKEN_ENUM, lexer.TOKEN_TYPE, lexer.TOKEN_STATIC, lexer.TOKEN_NATIVE,
		lexer.TOKEN_INTERFACE:
		return true
	}
	return false
}

func selectorKeyword(k lexer.TokenType) bool {
	switch k {
	case lexer.TOKEN_FUNC, lexer.TOKEN_LET, lexer.TOKEN_CONST, lexer.TOKEN_VAR, lexer.TOKEN_NATIVE, lexer.TOKEN_IMPORT,
		lexer.TOKEN_STRUCT, lexer.TOKEN_ENUM, lexer.TOKEN_TYPE, lexer.TOKEN_INTERFACE:
		return true
	}
	return false
}

func classify(tokens []token) []role {
	roles := make([]role, len(tokens))
	stack := []frame{{open: lexer.TOKEN_LBRACE, atTop: true}}
	var stmt stmtState
	inSelector := false
	genericClosers := map[int]bool{}

	top := func() *frame { return &stack[len(stack)-1] }
	atTop := func() bool { return len(stack) == 1 || (top().open == lexer.TOKEN_LBRACE && !top().literal) }
	prevEnd := func(i int) bool {
		if i == 0 {
			return false
		}
		return operandEnd(tokens[i-1], roles[i-1])
	}
	prevKind := func(i int) lexer.TokenType {
		if i == 0 {
			return lexer.TOKEN_ILLEGAL
		}
		return tokens[i-1].kind
	}

	for i, t := range tokens {
		k := t.kind
		r := roleWord

		switch {
		case k == lexer.TOKEN_LBRACE:
			pk := prevKind(i)
			literal := false
			switch {
			case i == 0:
			case pk == lexer.TOKEN_DOT:
				literal = true
			case pk == lexer.TOKEN_IDENT || isTypeKeyword(pk) || (i > 0 && roles[i-1] == roleGenericClose):
				literal = !(atTop() && stmt.header)
				if !literal && !stmt.fn && arrayTypeBefore(tokens, roles, i) {
					literal = true
				}
			}
			if literal {
				r = roleLiteralOpen
			} else {
				r = roleBlockOpen
			}
			stack = append(stack, frame{open: k, literal: literal})
			if !literal {
				stmt = stmtState{}
			}
		case k == lexer.TOKEN_RBRACE:
			if len(stack) > 1 {
				f := *top()
				stack = stack[:len(stack)-1]
				if f.literal {
					r = roleLiteralClose
				} else {
					r = roleBlockClose
					if atTop() {
						stmt = stmtState{}
					}
				}
			} else {
				r = roleBlockClose
			}
		case k == lexer.TOKEN_LPAREN || k == lexer.TOKEN_LBRACKET:
			if k == lexer.TOKEN_LBRACKET && !prevEnd(i) {
				stack = append(stack, frame{open: k, opensDecl: true})
			} else {
				stack = append(stack, frame{open: k})
			}
		case k == lexer.TOKEN_RPAREN || k == lexer.TOKEN_RBRACKET:
			if len(stack) > 1 {
				f := *top()
				stack = stack[:len(stack)-1]
				if k == lexer.TOKEN_RBRACKET && f.opensDecl {
					r = roleTypeBracketClose
				}
				if k == lexer.TOKEN_RPAREN && atTop() && stmt.fn && !stmt.arrow {
					r = roleParamClose
				}
			}
		case k == lexer.TOKEN_SEMICOLON:
			if atTop() {
				stmt = stmtState{}
			}
		case k == lexer.TOKEN_HASH:
			r = roleHash
		case k == lexer.TOKEN_DOT:
			if prevEnd(i) {
				r = roleMemberDot
			} else {
				r = rolePrefixDot
			}
		case k == lexer.TOKEN_RANGE || k == lexer.TOKEN_RANGE_OPEN:
			r = roleRange
		case k == lexer.TOKEN_VARIADIC:
			if i+1 < len(tokens) && tokens[i+1].kind == lexer.TOKEN_IDENT {
				r = roleDeclDots
			} else {
				r = roleSpreadDots
			}
		case k == lexer.TOKEN_AT || k == lexer.TOKEN_INCREMENT || k == lexer.TOKEN_DECREMENT:
			r = rolePostfix
		case k == lexer.TOKEN_NOT:
			switch {
			case i > 0 && roles[i-1] == roleParamClose:
				r = roleDeclBang
			case prevEnd(i):
				r = rolePostfix
			default:
				r = roleUnary
			}
		case k == lexer.TOKEN_NXOR:
			r = roleUnary
		case k == lexer.TOKEN_QUESTION_MARK:
			if prevEnd(i) {
				r = roleTernaryQ
				top().ternary++
			} else {
				r = roleOptionalQ
			}
		case k == lexer.TOKEN_COLON:
			pk := prevKind(i)
			switch {
			case top().ternary > 0:
				r = roleTernaryColon
				top().ternary--
			case top().open == lexer.TOKEN_LBRACE && top().literal && pk == lexer.TOKEN_IDENT:
				r = roleKeyColon
			case !prevEnd(i) && i+1 < len(tokens) && tokens[i+1].kind == lexer.TOKEN_IDENT:
				r = roleLabelColon
			default:
				r = roleTightColon
			}
		case k == lexer.TOKEN_PIPE:
			if isCapture(tokens, roles, i) {
				r = roleCapOpen
			} else if i > 0 && roles[i-1] != roleCapOpen && captureCloses(tokens, roles, i) {
				r = roleCapClose
			} else {
				r = roleBinary
			}
		case (k == lexer.TOKEN_GT || k == lexer.TOKEN_SHIFT_RIGHT) && genericClosers[i]:
			r = roleGenericClose
		case k == lexer.TOKEN_LT && prevKind(i) == lexer.TOKEN_IDENT && genericOpens(tokens, i, genericClosers):
			r = roleGenericOpen
		case k == lexer.TOKEN_LT && selectorKeyword(prevKind(i)):
			r = roleSelectorOpen
			inSelector = true
		case k == lexer.TOKEN_GT && inSelector:
			r = roleSelectorClose
			inSelector = false
		case k == lexer.TOKEN_PLUS || k == lexer.TOKEN_MINUS || k == lexer.TOKEN_STAR || k == lexer.TOKEN_AMP:
			if prevEnd(i) {
				r = roleBinary
			} else {
				r = roleUnary
			}
		case isBinaryOperator(k):
			r = roleBinary
		}

		if atTop() {
			if headerKeyword(k) {
				stmt.header = true
				if k == lexer.TOKEN_FUNC || k == lexer.TOKEN_NATIVE {
					stmt.fn = true
				}
			}
			if k == lexer.TOKEN_ARROW {
				stmt.arrow = true
			}
		}

		roles[i] = r
	}
	return roles
}

func isCapture(tokens []token, roles []role, i int) bool {
	j := i + 1
	if j >= len(tokens) || tokens[j].kind != lexer.TOKEN_IDENT {
		return false
	}
	j++
	for j+1 < len(tokens) && tokens[j].kind == lexer.TOKEN_COMMA && tokens[j+1].kind == lexer.TOKEN_IDENT {
		j += 2
	}
	if j >= len(tokens) || tokens[j].kind != lexer.TOKEN_PIPE {
		return false
	}
	prev := lexer.TOKEN_ILLEGAL
	if i > 0 {
		prev = tokens[i-1].kind
	}
	if prev == lexer.TOKEN_COALESCE {
		return true
	}
	if prev == lexer.TOKEN_NOT && i > 1 && tokens[i-2].kind == lexer.TOKEN_DEFER {
		return true
	}
	if prev == lexer.TOKEN_DEFER {
		return true
	}
	return j+1 < len(tokens) && tokens[j+1].kind == lexer.TOKEN_LBRACE
}

func captureCloses(tokens []token, roles []role, i int) bool {
	j := i - 1
	if j < 0 || tokens[j].kind != lexer.TOKEN_IDENT {
		return false
	}
	for j-2 >= 0 && tokens[j-1].kind == lexer.TOKEN_COMMA && tokens[j-2].kind == lexer.TOKEN_IDENT {
		j -= 2
	}
	return j-1 >= 0 && roles[j-1] == roleCapOpen
}

func arrayTypeBefore(tokens []token, roles []role, brace int) bool {
	j := brace - 1
	for j-2 >= 0 && tokens[j-1].kind == lexer.TOKEN_DOT && tokens[j-2].kind == lexer.TOKEN_IDENT {
		j -= 2
	}
	return j-1 >= 0 && roles[j-1] == roleTypeBracketClose
}

func genericOpens(tokens []token, open int, closers map[int]bool) bool {
	depth := 0
	for j := open; j < len(tokens); j++ {
		switch tokens[j].kind {
		case lexer.TOKEN_LT:
			depth++
		case lexer.TOKEN_GT, lexer.TOKEN_SHIFT_RIGHT:
			if tokens[j].kind == lexer.TOKEN_GT {
				depth--
			} else {
				depth -= 2
			}
			if depth <= 0 {
				if j+1 >= len(tokens) {
					return false
				}
				switch tokens[j+1].kind {
				case lexer.TOKEN_LPAREN, lexer.TOKEN_LBRACE, lexer.TOKEN_IDENT, lexer.TOKEN_LBRACKET, lexer.TOKEN_STAR,
					lexer.TOKEN_QUESTION_MARK, lexer.TOKEN_RPAREN, lexer.TOKEN_COMMA, lexer.TOKEN_GT, lexer.TOKEN_SHIFT_RIGHT,
					lexer.TOKEN_RBRACKET, lexer.TOKEN_SEMICOLON, lexer.TOKEN_ARROW, lexer.TOKEN_IMPLEMENTS:
					closers[j] = true
					return true
				}
				return false
			}
		case lexer.TOKEN_IDENT, lexer.TOKEN_DOT, lexer.TOKEN_COMMA, lexer.TOKEN_STAR, lexer.TOKEN_QUESTION_MARK,
			lexer.TOKEN_LBRACKET, lexer.TOKEN_RBRACKET, lexer.TOKEN_INT_LITERAL, lexer.TOKEN_CHAR_LITERAL,
			lexer.TOKEN_MINUS, lexer.TOKEN_CONST, lexer.TOKEN_LPAREN, lexer.TOKEN_RPAREN, lexer.TOKEN_ARROW,
			lexer.TOKEN_PIPE, lexer.TOKEN_AMP, lexer.TOKEN_NOT:
		default:
			if !isTypeKeyword(tokens[j].kind) {
				return false
			}
		}
	}
	return false
}
