package lexer

//go:generate stringer -type=TokenType
type TokenType int

const (
	// Special
	TOKEN_ILLEGAL TokenType = iota
	TOKEN_EOF

	// Literals
	TOKEN_IDENT
	TOKEN_INT_LITERAL
	TOKEN_FLOAT_LITERAL
	TOKEN_STRING_LITERAL
	TOKEN_CHAR_LITERAL

	// Types
	TOKEN_INT
	TOKEN_UINT
	TOKEN_INT8
	TOKEN_UINT8
	TOKEN_INT16
	TOKEN_UINT16
	TOKEN_INT32
	TOKEN_UINT32
	TOKEN_INT64
	TOKEN_UINT64

	TOKEN_FLOAT32
	TOKEN_FLOAT64

	TOKEN_BOOL
	TOKEN_CHAR
	TOKEN_STRING

	// Keywords
	TOKEN_LET
	TOKEN_FUNC
	TOKEN_STRUCT

	TOKEN_IF
	TOKEN_ELSE

	TOKEN_FOR
	TOKEN_LOOP
	TOKEN_UNTIL

	TOKEN_RETURN
	TOKEN_BREAK
	TOKEN_CONTINUE

	TOKEN_TRUE
	TOKEN_FALSE

	// Operators
	TOKEN_ASSIGN  // =
	TOKEN_PLUS    // +
	TOKEN_MINUS   // -
	TOKEN_STAR    // *
	TOKEN_SLASH   // /
	TOKEN_PERCENT // %

	TOKEN_PLUS_ASSIGN    // +=
	TOKEN_MINUS_ASSIGN   // -=
	TOKEN_STAR_ASSIGN    // *=
	TOKEN_SLASH_ASSIGN   // /=
	TOKEN_PERCENT_ASSIGN // %=

	TOKEN_INCREMENT // ++
	TOKEN_DECREMENT // --

	TOKEN_EQUAL     // ==
	TOKEN_NOT_EQUAL // !=

	TOKEN_LT  // <
	TOKEN_LTE // <=
	TOKEN_GT  // >
	TOKEN_GTE // >=

	TOKEN_AND // &&
	TOKEN_OR  // ||
	TOKEN_NOT // !

	TOKEN_ARROW // =>

	// Delimiters
	TOKEN_COMMA     // ,
	TOKEN_SEMICOLON // ;
	TOKEN_COLON     // :

	TOKEN_DOT   // .
	TOKEN_RANGE // ..

	TOKEN_LPAREN // (
	TOKEN_RPAREN // )

	TOKEN_LBRACE // {
	TOKEN_RBRACE // }

	TOKEN_LBRACKET // [
	TOKEN_RBRACKET // ]
)

var keywords = map[string]TokenType{
	"let":      TOKEN_LET,
	"func":     TOKEN_FUNC,
	"struct":   TOKEN_STRUCT,
	"if":       TOKEN_IF,
	"else":     TOKEN_ELSE,
	"for":      TOKEN_FOR,
	"loop":     TOKEN_LOOP,
	"until":    TOKEN_UNTIL,
	"return":   TOKEN_RETURN,
	"break":    TOKEN_BREAK,
	"continue": TOKEN_CONTINUE,

	"int":    TOKEN_INT,
	"uint":   TOKEN_UINT,
	"int8":   TOKEN_INT8,
	"uint8":  TOKEN_UINT8,
	"int16":  TOKEN_INT16,
	"uint16": TOKEN_UINT16,
	"int32":  TOKEN_INT32,
	"uint32": TOKEN_UINT32,
	"int64":  TOKEN_INT64,
	"uint64": TOKEN_UINT64,

	"float32": TOKEN_FLOAT32,
	"float64": TOKEN_FLOAT64,
	"bool":    TOKEN_BOOL,
	"char":    TOKEN_CHAR,
	"string":  TOKEN_STRING,

	"true":  TOKEN_TRUE,
	"false": TOKEN_FALSE,
}

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}
