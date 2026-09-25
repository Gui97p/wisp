package lsp

import (
	"slices"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/module"
)

func analyseEntry(path string) (*ast.Program, *analyser.Info) {
	modules, err := module.BuildGraphWithOverrides(resolveProjectEntry(path), fileBuffer)
	if err != nil {
		return nil, nil
	}

	exports := map[string]*analyser.ModuleInfo{}

	for _, mod := range modules {
		isEntry := false

		merged, declFiles := mod.Merge()
		a := analyser.NewAnalyser(merged, isEntry, exports, declFiles)
		info := a.Analyze()

		if slices.Contains(mod.FilePaths, path) {
			return merged, info
		}

		exports[mod.Path] = &analyser.ModuleInfo{Exports: a.Exports()}
	}

	return nil, nil
}
