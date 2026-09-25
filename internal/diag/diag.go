package diag

import "fmt"

type Diagnostic struct {
	File            string
	Message         string
	Line, Col       int
	EndLine, EndCol int
}

type List []Diagnostic

func (l *List) Add(file string, line, col, endLine, endCol int, format string, args ...any) {
	*l = append(*l, Diagnostic{File: file, Message: fmt.Sprintf(format, args...), Line: line, Col: col, EndLine: endLine, EndCol: endCol})
}
