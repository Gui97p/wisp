package diag

import (
	"errors"
	"fmt"
	"strings"
)

type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
)

func (s Severity) String() string {
	if s == SeverityWarning {
		return "warning"
	}
	return "error"
}

type Note struct {
	File            string
	Message         string
	Line, Col       int
	EndLine, EndCol int
}

type Diagnostic struct {
	File            string
	Message         string
	Line, Col       int
	EndLine, EndCol int
	Severity        Severity
	Origin          string
	Notes           []Note
	Help            string
}

func (d *Diagnostic) Note(file string, line, col, endLine, endCol int, format string, args ...any) *Diagnostic {
	d.Notes = append(d.Notes, Note{File: file, Message: fmt.Sprintf(format, args...), Line: line, Col: col, EndLine: endLine, EndCol: endCol})
	return d
}

func (d *Diagnostic) WithHelp(format string, args ...any) *Diagnostic {
	d.Help = fmt.Sprintf(format, args...)
	return d
}

type List []Diagnostic

func (l *List) add(severity Severity, file string, line, col, endLine, endCol int, format string, args ...any) *Diagnostic {
	*l = append(*l, Diagnostic{
		File: file, Message: fmt.Sprintf(format, args...),
		Line: line, Col: col, EndLine: endLine, EndCol: endCol,
		Severity: severity,
	})
	return &(*l)[len(*l)-1]
}

func (l *List) Add(file string, line, col, endLine, endCol int, format string, args ...any) *Diagnostic {
	return l.add(SeverityError, file, line, col, endLine, endCol, format, args...)
}

func (l *List) Warn(file string, line, col, endLine, endCol int, format string, args ...any) *Diagnostic {
	return l.add(SeverityWarning, file, line, col, endLine, endCol, format, args...)
}

func (l List) HasErrors() bool {
	for _, d := range l {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}

type Error struct {
	Diagnostic
}

func (e *Error) Error() string {
	return e.Message
}

func Locate(origin string, err error, line, col, endLine, endCol int) error {
	if err == nil {
		return nil
	}
	var located *Error
	if errors.As(err, &located) {
		return err
	}
	message := strings.TrimPrefix(err.Error(), origin+": ")
	return &Error{Diagnostic{Message: message, Origin: origin, Line: line, Col: col, EndLine: endLine, EndCol: endCol}}
}

func InFile(err error, file string) error {
	var located *Error
	if errors.As(err, &located) && located.File == "" {
		located.File = file
	}
	return err
}
