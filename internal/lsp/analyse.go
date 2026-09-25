package lsp

import (
	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/module"
)

func analyseEntry(path string) (*ast.Program, *analyser.Info) {
	modules, err := module.BuildGraphWithOverrides(path, fileBuffer)
	if err != nil {
		return nil, nil
	}

	exports := map[string]*analyser.ModuleInfo{}

	for _, mod := range modules {
		isEntry := mod.Path == ""

		merged, declFiles := mod.Merge()
		a := analyser.NewAnalyser(merged, isEntry, exports, declFiles)
		info := a.Analyze()

		if isEntry {
			return merged, info
		}

		exports[mod.Path] = &analyser.ModuleInfo{Exports: a.Exports()}
	}

	return nil, nil
}
