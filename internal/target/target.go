package target

import (
	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

type Target interface {
	Name() string
	Compile(program *ast.Program, info *analyser.Info) (string, error)
}
