package diag

import "fmt"

type Diagnostic struct {
	Message   string
	Line, Col int
}

type List []Diagnostic

func (l *List) Add(line, col int, format string, args ...any) {
	*l = append(*l, Diagnostic{Message: fmt.Sprintf(format, args...), Line: line, Col: col})
}
