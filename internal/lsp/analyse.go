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
		if len(mod.Parts) > 0 {
			analyser.LinkPackage(exports, mod.Path, mod.Parts)
		}
	}

	for _, group := range module.Groups(modules) {
		items, ok := analyseGroup(group, exports)
		if !ok {
			continue
		}
		result.modules = append(result.modules, items...)
	}
	return result
}

func analyseGroup(group []*module.Module, exports map[string]*analyser.ModuleInfo) (items []moduleAnalysis, ok bool) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("analyser panic: %v", r)
			items, ok = nil, false
		}
	}()

	paths := make([]string, len(group))
	for i, mod := range group {
		paths[i] = mod.Path
	}
	analyser.PendingExports(exports, paths)

	analysers := make([]*analyser.Analyser, len(group))
	merges := make([]*ast.Program, len(group))
	declMaps := make([]map[ast.Declaration]string, len(group))
	for i, mod := range group {
		merged, declFiles := mod.Merge()
		a := analyser.NewAnalyser(merged, exports, declFiles, mod.Path)
		a.SetImportErrors(mod.ImportErrors)
		a.SetPackageParts(mod.Parts)
		analysers[i], merges[i], declMaps[i] = a, merged, declFiles
	}
	analyser.AnalyzeGroup(analysers)

	for i, mod := range group {
		a := analysers[i]
		merges[i].Declarations = withoutGenerated(merges[i].Declarations)
		item := moduleAnalysis{mod: mod, merged: merges[i], info: a.Info(), declFiles: declMaps[i]}
		if a.HasDiagnostics() {
			item.diags = a.Errors()
		}
		analyser.SetExports(exports, mod.Path, a.Exports())
		items = append(items, item)
	}
	return items, true
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

func declarationFiles(path string) map[ast.Declaration]string {
	pa, err := analyseProject(path)
	if err != nil || pa == nil {
		return nil
	}
	for _, m := range pa.modules {
		if slices.Contains(m.mod.FilePaths, path) {
			return m.declFiles
		}
	}
	return nil
}

func declarationsOf(program *ast.Program, files map[ast.Declaration]string, path string) *ast.Program {
	if files == nil {
		return program
	}
	own := &ast.Program{}
	for _, d := range program.Declarations {
		if file, known := files[d]; !known || file == path {
			own.Declarations = append(own.Declarations, d)
		}
	}
	return own
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
