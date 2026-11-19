package format

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/lexer"
)

type token struct {
	kind    lexer.TokenType
	text    string
	start   int
	end     int
	line    int
	endLine int
}

type comment struct {
	text     string
	nlBefore int
}

type gap struct {
	comments []comment
	nlAfter  int
}

func lineStarts(src []byte) []int {
	starts := []int{0}
	for i, b := range src {
		if b == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

func byteAt(src []byte, starts []int, line, col int) int {
	if line < 1 || line > len(starts) {
		return -1
	}
	counter := 0
	for i := starts[line-1]; i < len(src); i++ {
		b := src[i]
		if b == '\n' {
			break
		}
		switch {
		case b&0xC0 == 0x80:
		case b >= 0xF0:
			counter += 2
		default:
			counter++
		}
		if counter == col {
			return i
		}
	}
	return -1
}

func scan(src []byte) ([]token, error) {
	starts := lineStarts(src)
	l := lexer.NewLexer(src)
	var tokens []token
	for {
		t := l.NextToken()
		if t.Type == lexer.TOKEN_EOF {
			break
		}
		if t.Type == lexer.TOKEN_ILLEGAL {
			return nil, fmt.Errorf("%d:%d: %s", t.Line, t.Column, t.Literal)
		}
		start := byteAt(src, starts, t.Line, t.Column)
		last := byteAt(src, starts, t.EndLine, t.EndColumn)
		if start < 0 || last < start {
			return nil, fmt.Errorf("%d:%d: cannot place the token %q", t.Line, t.Column, t.Literal)
		}
		tokens = append(tokens, token{kind: t.Type, text: string(src[start : last+1]), start: start, end: last + 1, line: t.Line, endLine: t.EndLine})
	}
	return tokens, nil
}

func readGap(text string) (gap, error) {
	var g gap
	nl := 0
	i := 0
	for i < len(text) {
		switch c := text[i]; {
		case c == '\n':
			nl++
			i++
		case c == ' ' || c == '\t' || c == '\r':
			i++
		case c == '/' && i+1 < len(text) && text[i+1] == '/':
			j := i
			for j < len(text) && text[j] != '\n' {
				j++
			}
			g.comments = append(g.comments, comment{text: strings.TrimRight(text[i:j], " \t\r"), nlBefore: nl})
			nl = 0
			i = j
		case c == '/' && i+1 < len(text) && text[i+1] == '*':
			j := strings.Index(text[i+2:], "*/")
			if j < 0 {
				return g, fmt.Errorf("unterminated block comment")
			}
			end := i + 2 + j + 2
			g.comments = append(g.comments, comment{text: text[i:end], nlBefore: nl})
			nl = 0
			i = end
		default:
			return g, fmt.Errorf("unexpected text between tokens: %q", text[i:min(len(text), i+10)])
		}
	}
	g.nlAfter = nl
	return g, nil
}

func readGaps(src []byte, tokens []token) ([]gap, error) {
	gaps := make([]gap, len(tokens)+1)
	prev := 0
	for i := 0; i <= len(tokens); i++ {
		end := len(src)
		if i < len(tokens) {
			end = tokens[i].start
		}
		g, err := readGap(string(src[prev:end]))
		if err != nil {
			return nil, err
		}
		gaps[i] = g
		if i < len(tokens) {
			prev = tokens[i].end
		}
	}
	return gaps, nil
}
