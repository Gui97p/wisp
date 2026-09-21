package lexer

func (t TokenType) DisplayName() string {
	switch t {
	case TOKEN_ILLEGAL:
		return "illegal"
	case TOKEN_EOF:
		return "end of file"
	case TOKEN_IDENT:
		return "identifier"

	case TOKEN_INT_LITERAL:
		return "integer literal"
	case TOKEN_FLOAT_LITERAL:
		return "float literal"
	case TOKEN_STRING_LITERAL:
		return "string literal"
	case TOKEN_CHAR_LITERAL:
		return "character literal"

	case TOKEN_INT:
		return "int"
	case TOKEN_UINT:
		return "uint"
	case TOKEN_INT8:
		return "int8"
	case TOKEN_UINT8:
		return "uint8"
	case TOKEN_INT16:
		return "int16"
	case TOKEN_UINT16:
		return "uint16"
	case TOKEN_INT32:
		return "int32"
	case TOKEN_UINT32:
		return "uint32"
	case TOKEN_INT64:
		return "int64"
	case TOKEN_UINT64:
		return "uint64"
	case TOKEN_FLOAT32:
		return "float32"
	case TOKEN_FLOAT64:
		return "float64"
	case TOKEN_BOOL:
		return "bool"
	case TOKEN_CHAR:
		return "char"
	case TOKEN_STRING:
		return "string"
	case TOKEN_MAP:
		return "map"
	case TOKEN_NULL:
		return "null"

	case TOKEN_LET:
		return "let"
	case TOKEN_CONST:
		return "const"
	case TOKEN_FUNC:
		return "func"
	case TOKEN_STRUCT:
		return "struct"
	case TOKEN_ENUM:
		return "enum"
	case TOKEN_SWITCH:
		return "switch"
	case TOKEN_CASE:
		return "case"
	case TOKEN_DEFAULT:
		return "default"
	case TOKEN_TYPE:
		return "type"
	case TOKEN_IF:
		return "if"
	case TOKEN_ELSE:
		return "else"
	case TOKEN_FOR:
		return "for"
	case TOKEN_IN:
		return "in"
	case TOKEN_LOOP:
		return "loop"
	case TOKEN_UNTIL:
		return "until"
	case TOKEN_RETURN:
		return "return"
	case TOKEN_BREAK:
		return "break"
	case TOKEN_CONTINUE:
		return "continue"
	case TOKEN_IMPORT:
		return "import"
	case TOKEN_EXPORT:
		return "export"
	case TOKEN_AS:
		return "as"
	case TOKEN_TRUE:
		return "true"
	case TOKEN_FALSE:
		return "false"

	case TOKEN_ASSIGN:
		return "="
	case TOKEN_PLUS:
		return "+"
	case TOKEN_MINUS:
		return "-"
	case TOKEN_STAR:
		return "*"
	case TOKEN_SLASH:
		return "/"
	case TOKEN_PERCENT:
		return "%"
	case TOKEN_AMP:
		return "&"
	case TOKEN_PIPE:
		return "|"
	case TOKEN_XOR:
		return "^"
	case TOKEN_NXOR:
		return "~"
	case TOKEN_SHIFT_LEFT:
		return "<<"
	case TOKEN_SHIFT_RIGHT:
		return ">>"

	case TOKEN_PLUS_ASSIGN:
		return "+="
	case TOKEN_MINUS_ASSIGN:
		return "-="
	case TOKEN_STAR_ASSIGN:
		return "*="
	case TOKEN_SLASH_ASSIGN:
		return "/="
	case TOKEN_PERCENT_ASSIGN:
		return "%="
	case TOKEN_AMP_ASSIGN:
		return "&="
	case TOKEN_PIPE_ASSIGN:
		return "|="
	case TOKEN_XOR_ASSIGN:
		return "^="
	case TOKEN_SHIFT_LEFT_ASSIGN:
		return "<<="
	case TOKEN_SHIFT_RIGHT_ASSIGN:
		return ">>="

	case TOKEN_INCREMENT:
		return "++"
	case TOKEN_DECREMENT:
		return "--"

	case TOKEN_EQUAL:
		return "=="
	case TOKEN_NOT_EQUAL:
		return "!="
	case TOKEN_LT:
		return "<"
	case TOKEN_LTE:
		return "<="
	case TOKEN_GT:
		return ">"
	case TOKEN_GTE:
		return ">="

	case TOKEN_AND:
		return "&&"
	case TOKEN_OR:
		return "||"
	case TOKEN_NOT:
		return "!"

	case TOKEN_ARROW:
		return "=>"
	case TOKEN_COMMA:
		return ","
	case TOKEN_SEMICOLON:
		return ";"
	case TOKEN_COLON:
		return ":"
	case TOKEN_QUESTION_MARK:
		return "?"
	case TOKEN_COALESCE:
		return "??"

	case TOKEN_DOT:
		return "."
	case TOKEN_RANGE:
		return ".."
	case TOKEN_VARIADIC:
		return "..."

	case TOKEN_LPAREN:
		return "("
	case TOKEN_RPAREN:
		return ")"
	case TOKEN_LBRACE:
		return "{"
	case TOKEN_RBRACE:
		return "}"
	case TOKEN_LBRACKET:
		return "["
	case TOKEN_RBRACKET:
		return "]"

	default:
		return t.String()
	}
}
