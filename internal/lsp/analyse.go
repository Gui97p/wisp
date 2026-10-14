package lsp

import (
	"log"
	"slices"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

func analyseEntry(path string) (*ast.Program, *analyser.Info) {
	program, info, _ := analyseEntryChecked(path)
	return program, info
}

func analyseEntryChecked(path string) (program *ast.Program, info *analyser.Info, clean bool) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("analyser panic: %v", r)
			program, info, clean = nil, nil, false
		}
	}()

	modules, err := buildProjectGraph(resolveProjectEntry(path), fileBuffer)
	if err != nil {
		return nil, nil, false
	}

	exports := map[string]*analyser.ModuleInfo{}

	for _, mod := range modules {
		merged, declFiles := mod.Merge()
		a := analyser.NewAnalyser(merged, exports, declFiles, mod.Path)
		info := a.Analyze()

		if slices.Contains(mod.FilePaths, path) {
			return merged, info, mod.ParseError == nil
		}

		exports[mod.Path] = &analyser.ModuleInfo{Exports: a.Exports()}
	}

	return nil, nil, false
}
