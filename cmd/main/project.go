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
}

func loadProject(root string, cfg *module.Config) (*project, error) {
	modules, err := module.DiscoverProject(root)
	if err != nil {
		return nil, err
	}

	exports := map[string]*analyser.ModuleInfo{}
	hadErrors := false
	var compiled []compiledModule

	for _, mod := range modules {
		if mod.ParseError != nil {
			diag.Render(os.Stderr, mod.ParseError.Path, mod.ParseError.Buffer, mod.ParseError.Errors)
			hadErrors = true
			continue
		}

		merged, declFiles := mod.Merge()

		a := analyser.NewAnalyser(merged, exports, declFiles, mod.Path)
		info := a.Analyze()
		if a.HasDiagnostics() {
			fmt.Fprintf(os.Stderr, "<<  %s  >>\n", mod.Path)
			diag.RenderGrouped(os.Stderr, mod.BufferMap(), a.Errors())
		}
		if a.HasErrors() {
			hadErrors = true
			continue
		}

		modExports := a.Exports()

		exports[mod.Path] = &analyser.ModuleInfo{Exports: modExports}
		compiled = append(compiled, compiledModule{mod: mod, merged: merged, info: info, declFiles: declFiles})
	}

	if hadErrors {
		os.Exit(1)
	}

	mainDecl, mainMod, mainCount := findMain(compiled, root, cfg.Entry)
	if mainDecl != nil {
		mainDecl.Exported = true
	}

	return &project{compiled: compiled, mainMod: mainMod, mainCount: mainCount}, nil
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
