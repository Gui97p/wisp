package lexer

type Comment struct {
	Line       int
	EndLine    int
	Text       string
	Standalone bool
}

func (l *Lexer) Comments() []Comment {
	return l.comments
}

func (l *Lexer) addComment(start, startLine int) {
	l.comments = append(l.comments, Comment{
		Line:       startLine,
		EndLine:    l.line,
		Text:       string(l.buf[start : l.pos+1]),
		Standalone: l.lastTokenEnd < startLine,
	})
}
