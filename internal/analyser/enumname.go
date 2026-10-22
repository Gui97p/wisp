package analyser

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
	"github.com/Gui97p/wisp/internal/parser"
)

func (a *Analyser) generateEnumNames() {
	declared := map[string]bool{}
	for _, d := range a.program.Declarations {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Receiver != nil && fd.Name == "name" {
			declared[fd.Receiver.Type.Name] = true
		}
	}

	var generated []ast.Declaration
	for _, d := range a.program.Declarations {
		td, ok := d.(*ast.TypeDecl)
		if !ok || !td.IsEnum || declared[td.Name] {
			continue
		}

		fd := enumNameMethod(td)
		if fd == nil {
			continue
		}
		a.declFiles[fd] = a.declFiles[d]
		generated = append(generated, fd)
	}

	a.program.Declarations = append(a.program.Declarations, generated...)
}

func enumNameMethod(td *ast.TypeDecl) *ast.FuncDecl {
	var src strings.Builder
	if td.Exported {
		src.WriteString("export ")
	}
	fmt.Fprintf(&src, "func (%s v) name() string {\n", td.Name)
	for _, m := range td.Members {
		fmt.Fprintf(&src, "if v == %s.%s { return %q; }\n", td.Name, m.Name, m.Name)
	}
	src.WriteString("return \"?\";\n}\n")

	program := parser.NewParser(lexer.NewLexer([]byte(src.String()))).ParseProgram()
	if len(program.Declarations) != 1 {
		return nil
	}
	fd, ok := program.Declarations[0].(*ast.FuncDecl)
	if !ok {
		return nil
	}

	fd.Generated = true
	fd.Doc = "the name of the member this value holds, or ? when it holds none"
	line, col := td.Position()
	endLine, endCol := td.EndPosition()
	fd.SetPos(line, col)
	fd.SetEndPos(endLine, endCol)
	return fd
}
