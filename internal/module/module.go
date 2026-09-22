package module

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
	"github.com/Gui97p/wisp/internal/parser"
)

type Module struct {
	Path       string
	Dir        string
	Files      []*ast.Program
	Buffers    [][]byte
	FilePaths  []string
	Imports    []string
	ParseError *SourceError
}

func (m *Module) Merge() (*ast.Program, map[ast.Declaration]string) {
	program := &ast.Program{}
	declFiles := map[ast.Declaration]string{}

	for i, f := range m.Files {
		for _, d := range f.Declarations {
			declFiles[d] = m.FilePaths[i]
		}
		program.Declarations = append(program.Declarations, f.Declarations...)
	}

	return program, declFiles
}

func (m *Module) BufferMap() map[string][]byte {
	buffers := map[string][]byte{}
	for i, p := range m.FilePaths {
		buffers[p] = m.Buffers[i]
	}
	return buffers
}

func stdlibRoot() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exe), "..", "std"), nil
}

func FindProjectRoot(startDir string) string {
	dir := startDir
	for {
		if _, err := os.Stat(filepath.Join(dir, "wisp.toml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return startDir
		}
		dir = parent
	}
}

func resolveImportPath(projectRoot, path string) (dir string, files []string, err error) {
	base := filepath.Join(projectRoot, path)
	if after, ok := strings.CutPrefix(path, "std/"); ok {
		root, err := stdlibRoot()
		if err != nil {
			return "", nil, fmt.Errorf("cannot locate stdlib: %w", err)
		}
		base = filepath.Join(root, after)
	}

	fileExists := false
	if info, err := os.Stat(base + ".wsp"); err == nil && !info.IsDir() {
		fileExists = true
	}
	dirInfo, dirErr := os.Stat(base)
	dirExists := dirErr == nil && dirInfo.IsDir()

	switch {
	case fileExists && dirExists:
		return "", nil, fmt.Errorf("import %q is ambiguous: both %s.wsp and %s/ exist", path, base, base)
	case fileExists:
		return filepath.Dir(base), []string{base + ".wsp"}, nil
	case dirExists:
		entries, err := os.ReadDir(base)
		if err != nil {
			return "", nil, fmt.Errorf("cannot resolve import %q: %w", path, err)
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".wsp") {
				files = append(files, filepath.Join(base, e.Name()))
			}
		}
		if len(files) == 0 {
			return "", nil, fmt.Errorf("import %q (%s) has no .wsp files", path, base)
		}
		return base, files, nil
	default:
		return "", nil, fmt.Errorf("cannot resolve import %q", path)
	}
}

func BuildGraph(entryFile string) ([]*Module, error) {
	root := FindProjectRoot(filepath.Dir(entryFile))

	modules := map[string]*Module{}
	visiting := map[string]bool{}
	visited := map[string]bool{}
	var order []*Module

	var visit func(path, dir string, files []string) error
	visit = func(path, dir string, files []string) error {
		if visited[path] {
			return nil
		}
		if visiting[path] {
			return fmt.Errorf("cyclic import: %q", path)
		}
		visiting[path] = true

		mod := loadModule(path, dir, files)
		modules[path] = mod

		for _, imp := range mod.Imports {
			depDir, depFiles, err := resolveImportPath(root, imp)
			if err != nil {
				return err
			}
			if err := visit(imp, depDir, depFiles); err != nil {
				return err
			}
		}

		visiting[path] = false
		visited[path] = true
		order = append(order, mod)
		return nil
	}

	if err := visit("", filepath.Dir(entryFile), []string{entryFile}); err != nil {
		return nil, err
	}
	return order, nil
}

func loadModule(path, dir string, files []string) *Module {
	mod := &Module{
		Path:      path,
		Dir:       dir,
		FilePaths: files,
	}

	for _, f := range files {
		buffer, err := os.ReadFile(f)
		if err != nil {
			mod.Buffers = append(mod.Buffers, nil)
			mod.Files = append(mod.Files, &ast.Program{})
			continue
		}

		l := lexer.NewLexer(buffer)
		p := parser.NewParser(l)
		program := p.ParseProgram()

		mod.Files = append(mod.Files, program)
		mod.Buffers = append(mod.Buffers, buffer)

		if p.HasErrors() && mod.ParseError == nil {
			mod.ParseError = &SourceError{Path: f, Buffer: buffer, Errors: p.Errors()}
		}

		for _, decl := range program.Declarations {
			if d, ok := decl.(*ast.ImportDecl); ok {
				mod.Imports = append(mod.Imports, d.Path)
			}
		}
	}

	return mod
}
