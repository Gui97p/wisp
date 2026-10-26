package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/diag"
	"github.com/Gui97p/wisp/internal/module"
)

type compiledModule struct {
	mod       *module.Module
	merged    *ast.Program
	info      *analyser.Info
	declFiles map[ast.Declaration]string
}

func (c compiledModule) reportFailure(err error) error {
	var located *diag.Error
	if errors.As(err, &located) {
		fmt.Fprintf(os.Stderr, "<<  %s  >>\n", c.mod.Path)
		diag.RenderError(os.Stderr, c.mod.BufferMap(), located)
		os.Exit(1)
	}
	return err
}

type project struct {
	compiled  []compiledModule
	mainMod   *compiledModule
	mainCount int
	cycles    [][]string
}

func loadProject(root string, cfg *module.Config) (*project, error) {
	modules, err := module.DiscoverProject(root)
	if err != nil {
		return nil, err
	}

	exports := map[string]*analyser.ModuleInfo{}
	hadErrors := false
	var compiled []compiledModule

	failed := map[string]bool{}
	totalErrors, totalWarnings := 0, 0

	for _, mod := range modules {
		if len(mod.Parts) > 0 {
			analyser.LinkPackage(exports, mod.Path, mod.Parts)
		}
	}

	var cycles [][]string
	for _, group := range module.Groups(modules) {
		var members []*module.Module
		groupFailed := false
		if len(group) > 1 {
			paths := make([]string, len(group))
			for i, mod := range group {
				paths[i] = mod.Path
			}
			cycles = append(cycles, paths)
		}

		for _, mod := range group {
			for _, imp := range mod.Imports {
				if failed[imp.Path] {
					mod.FailImport(imp.Path, fmt.Sprintf("module %q has errors", imp.Path))
				}
			}

			if mod.ParseError != nil {
				diag.Render(os.Stderr, mod.ParseError.Path, mod.ParseError.Buffer, mod.ParseError.Errors)
				e, w := diag.Count(mod.ParseError.Errors)
				totalErrors, totalWarnings = totalErrors+e, totalWarnings+w
				hadErrors = true
				groupFailed = true
				continue
			}
			members = append(members, mod)
		}

		if groupFailed {
			for _, mod := range group {
				failed[mod.Path] = true
			}
			continue
		}

		paths := make([]string, len(members))
		for i, mod := range members {
			paths[i] = mod.Path
		}
		analyser.PendingExports(exports, paths)

		analysers := make([]*analyser.Analyser, len(members))
		merges := make([]*ast.Program, len(members))
		declMaps := make([]map[ast.Declaration]string, len(members))
		for i, mod := range members {
			merged, declFiles := mod.Merge()
			a := analyser.NewAnalyser(merged, exports, declFiles, mod.Path)
			a.SetImportErrors(mod.ImportErrors)
			a.SetPackageParts(mod.Parts)
			analysers[i], merges[i], declMaps[i] = a, merged, declFiles
		}
		analyser.AnalyzeGroup(analysers)

		for i, mod := range members {
			a := analysers[i]
			if a.HasDiagnostics() {
				fmt.Fprintf(os.Stderr, "<<  %s  >>\n", mod.Path)
				diag.RenderGrouped(os.Stderr, mod.BufferMap(), a.Errors())
				e, w := diag.Count(a.Errors())
				totalErrors, totalWarnings = totalErrors+e, totalWarnings+w
			}
			if a.HasErrors() {
				hadErrors = true
				failed[mod.Path] = true
				continue
			}

			analyser.SetExports(exports, mod.Path, a.Exports())
			compiled = append(compiled, compiledModule{mod: mod, merged: merges[i], info: a.Info(), declFiles: declMaps[i]})
		}
	}

	if hadErrors {
		diag.Summary(os.Stderr, totalErrors, totalWarnings)
		os.Exit(1)
	}
	if totalWarnings > 0 && !quietOpt {
		diag.Summary(os.Stderr, 0, totalWarnings)
	}

	mainDecl, mainMod, mainCount := findMain(compiled, root, cfg.Entry)
	if mainDecl != nil {
		mainDecl.Exported = true
	}

	return &project{compiled: compiled, mainMod: mainMod, mainCount: mainCount, cycles: cycles}, nil
}

func (p *project) requireMain() error {
	if p.mainCount == 0 {
		return fmt.Errorf("no main function was found in the project")
	}
	return p.rejectManyMains()
}

func (p *project) rejectManyMains() error {
	if p.mainCount > 1 {
		return fmt.Errorf("multiple main functions found across the project; set 'entry' in wisp.toml to disambiguate")
	}
	return nil
}

func findMain(compiled []compiledModule, root string, entry string) (*ast.FuncDecl, *compiledModule, int) {
	type found struct {
		decl *ast.FuncDecl
		mod  *compiledModule
	}
	var all []found

	for i := range compiled {
		for _, decl := range compiled[i].merged.Declarations {
			fd, ok := decl.(*ast.FuncDecl)
			if ok && fd.Receiver == nil && fd.Name == "main" {
				all = append(all, found{decl: fd, mod: &compiled[i]})
			}
		}
	}

	if len(all) <= 1 {
		if len(all) == 0 {
			return nil, nil, 0
		}
		return all[0].decl, all[0].mod, 1
	}

	if entry == "" {
		return nil, nil, len(all)
	}

	entryPath := filepath.Join(root, entry)
	for _, f := range all {
		if slices.Contains(f.mod.mod.FilePaths, entryPath) {
			return f.decl, f.mod, 1
		}
	}

	return nil, nil, len(all)
}
