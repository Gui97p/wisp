package lsp

import (
	"fmt"
	"log"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/diag"
	"github.com/Gui97p/wisp/internal/module"
)

type moduleAnalysis struct {
	mod       *module.Module
	merged    *ast.Program
	info      *analyser.Info
	diags     diag.List
	declFiles map[ast.Declaration]string
}

type projectAnalysis struct {
	modules []moduleAnalysis
}

type analysisSlot struct {
	gen   uint64
	value *projectAnalysis
	err   error
}

var (
	generation   atomic.Uint64
	analysisMu   sync.Mutex
	analysisHold = map[string]analysisSlot{}
)

func invalidateAnalysis() {
	generation.Add(1)
}

func cachedAnalysis(key string, gen uint64, run func() (*projectAnalysis, error)) (*projectAnalysis, error) {
	if slot, ok := analysisHold[key]; ok && slot.gen == gen {
		return slot.value, slot.err
	}
	value, err := run()
	analysisHold[key] = analysisSlot{gen, value, err}
	return value, err
}

func analyseProject(path string) (*projectAnalysis, error) {
	gen := generation.Load()
	root := module.FindProjectRoot(filepath.Dir(path))

	analysisMu.Lock()
	defer analysisMu.Unlock()

	whole, err := cachedAnalysis("project:"+root, gen, func() (*projectAnalysis, error) {
		return runProjectAnalysis(path, fileBuffer)
	})
	if err == nil && whole != nil && analysisHas(whole, path) {
		return whole, nil
	}

	entry := resolveProjectEntry(path)
	return cachedAnalysis("entry:"+entry, gen, func() (*projectAnalysis, error) {
		return runAnalysis(entry)
	})
}

func analysisHas(pa *projectAnalysis, path string) bool {
	for _, m := range pa.modules {
		if slices.Contains(m.mod.FilePaths, path) {
			return true
		}
	}
	return false
}

func runProjectAnalysis(path string, buffers map[string][]byte) (result *projectAnalysis, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("analyser panic: %v", r)
			result, err = nil, fmt.Errorf("the analysis failed")
		}
	}()

	modules, err := discoverProjectGraph(path, buffers)
	if err != nil {
		return nil, err
	}
	return analyseModules(modules), nil
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

	return analyseModules(modules), nil
}

func analyseModules(modules []*module.Module) *projectAnalysis {
	result := &projectAnalysis{}
	exports := map[string]*analyser.ModuleInfo{}

	for _, mod := range modules {
		item, ok := analyseModule(mod, exports)
		if !ok {
			continue
		}
		result.modules = append(result.modules, item)
	}
	return result
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
	merged.Declarations = withoutGenerated(merged.Declarations)

	item = moduleAnalysis{mod: mod, merged: merged, info: info, declFiles: declFiles}
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

func withoutGenerated(decls []ast.Declaration) []ast.Declaration {
	kept := make([]ast.Declaration, 0, len(decls))
	for _, d := range decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Generated {
			continue
		}
		kept = append(kept, d)
	}
	return kept
}
