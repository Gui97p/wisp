package lsp

import (
	"fmt"
	"log"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/diag"
	"github.com/Gui97p/wisp/internal/module"
)

type moduleAnalysis struct {
	mod    *module.Module
	merged *ast.Program
	info   *analyser.Info
	diags  diag.List
}

type projectAnalysis struct {
	modules []moduleAnalysis
}

var (
	generation   atomic.Uint64
	analysisMu   sync.Mutex
	analysisHold struct {
		entry string
		gen   uint64
		value *projectAnalysis
		err   error
		valid bool
	}
)

func invalidateAnalysis() {
	generation.Add(1)
}

func analyseProject(path string) (*projectAnalysis, error) {
	entry := resolveProjectEntry(path)
	gen := generation.Load()

	analysisMu.Lock()
	defer analysisMu.Unlock()

	h := &analysisHold
	if h.valid && h.entry == entry && h.gen == gen {
		return h.value, h.err
	}

	value, err := runAnalysis(entry)
	h.entry, h.gen, h.value, h.err, h.valid = entry, gen, value, err, true
	return value, err
}

func runAnalysis(entry string) (result *projectAnalysis, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("analyser panic: %v", r)
			result, err = nil, fmt.Errorf("the analysis failed")
		}
	}()

	modules, err := buildProjectGraph(entry, fileBuffer)
	if err != nil {
		return nil, err
	}

	result = &projectAnalysis{}
	exports := map[string]*analyser.ModuleInfo{}

	for _, mod := range modules {
		item, ok := analyseModule(mod, exports)
		if !ok {
			break
		}
		result.modules = append(result.modules, item)
	}
	return result, nil
}

func analyseModule(mod *module.Module, exports map[string]*analyser.ModuleInfo) (item moduleAnalysis, ok bool) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("analyser panic: %v", r)
			ok = false
		}
	}()

	merged, declFiles := mod.Merge()
	a := analyser.NewAnalyser(merged, exports, declFiles, mod.Path)
	a.SetImportErrors(mod.ImportErrors)
	info := a.Analyze()

	item = moduleAnalysis{mod: mod, merged: merged, info: info}
	if a.HasDiagnostics() {
		item.diags = a.Errors()
	}
	exports[mod.Path] = &analyser.ModuleInfo{Exports: a.Exports()}
	return item, true
}

func analyseEntry(path string) (*ast.Program, *analyser.Info) {
	program, info, _ := analyseEntryChecked(path)
	return program, info
}

func analyseEntryChecked(path string) (program *ast.Program, info *analyser.Info, clean bool) {
	pa, err := analyseProject(path)
	if err != nil || pa == nil {
		return nil, nil, false
	}

	for _, m := range pa.modules {
		if slices.Contains(m.mod.FilePaths, path) {
			return m.merged, m.info, m.mod.ParseError == nil
		}
	}
	return nil, nil, false
}
