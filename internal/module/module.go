package module

import (
	"fmt"
	"io/fs"
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
	Imports    []*ast.ImportDecl
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

func StdlibRoot() (string, error) {
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
	if abs, err := filepath.Abs(startDir); err == nil {
		startDir = abs
	}

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

func canonicalModulePath(root, file string) (string, error) {
	rel, err := filepath.Rel(root, file)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(strings.TrimSuffix(rel, ".wsp")), nil
}

func resolveImportPath(projectRoot, path string) (key, dir string, files []string, err error) {
	isStd := strings.HasPrefix(path, "std/")
	base := filepath.Join(projectRoot, path)
	if after, ok := strings.CutPrefix(path, "std/"); ok {
		stdRoot, err := StdlibRoot()
		if err != nil {
			return "", "", nil, fmt.Errorf("cannot locate stdlib: %w", err)
		}
		base = filepath.Join(stdRoot, after)
	}

	fileExists := false
	if info, err := os.Stat(base + ".wsp"); err == nil && !info.IsDir() {
		fileExists = true
	}
	dirInfo, dirErr := os.Stat(base)
	dirExists := dirErr == nil && dirInfo.IsDir()

	switch {
	case fileExists && dirExists:
		return "", "", nil, fmt.Errorf("import %q is ambiguous: both %s.wsp and %s/ exist", path, base, base)
	case fileExists:
		file := base + ".wsp"
		key = path
		if !isStd {
			if key, err = canonicalModulePath(projectRoot, file); err != nil {
				return "", "", nil, err
			}
		}
		return key, filepath.Dir(base), []string{file}, nil
	case dirExists:
		modFile := filepath.Join(base, filepath.Base(base)+".wsp")
		if info, statErr := os.Stat(modFile); statErr != nil || info.IsDir() {
			return "", "", nil, fmt.Errorf("directory %q has no module file %s.wsp", path, filepath.Base(base))
		}
		key = path
		if !isStd {
			if key, err = canonicalModulePath(projectRoot, modFile); err != nil {
				return "", "", nil, err
			}
		}
		return key, base, []string{modFile}, nil
	default:
		return "", "", nil, fmt.Errorf("cannot resolve import %q", path)
	}
}

func BuildGraph(entryFile string) ([]*Module, error) {
	return BuildGraphWithOverrides(entryFile, nil)
}

func BuildGraphWithOverrides(entryFile string, overrides map[string][]byte) ([]*Module, error) {
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

		mod := loadModule(path, dir, files, overrides)
		modules[path] = mod

		for _, imp := range mod.Imports {
			key, depDir, depFiles, err := resolveImportPath(root, imp.Path)
			if err != nil {
				return err
			}
			imp.Path = key
			if err := visit(key, depDir, depFiles); err != nil {
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

type moduleRoot struct {
	path  string
	dir   string
	files []string
}

func discoverModuleRoots(root string) ([]moduleRoot, error) {
	var roots []moduleRoot

	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if p != root && (strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".wsp") {
			return nil
		}
		path, err := canonicalModulePath(root, p)
		if err != nil {
			return err
		}
		roots = append(roots, moduleRoot{path: path, dir: filepath.Dir(p), files: []string{p}})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return roots, nil
}

func DiscoverProject(root string) ([]*Module, error) {
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

		mod := loadModule(path, dir, files, nil)
		modules[path] = mod

		for _, imp := range mod.Imports {
			key, depDir, depFiles, err := resolveImportPath(root, imp.Path)
			if err != nil {
				return err
			}
			imp.Path = key
			if err := visit(key, depDir, depFiles); err != nil {
				return err
			}
		}

		visiting[path] = false
		visited[path] = true
		order = append(order, mod)
		return nil
	}

	roots, err := discoverModuleRoots(root)
	if err != nil {
		return nil, err
	}

	for _, r := range roots {
		if err := visit(r.path, r.dir, r.files); err != nil {
			return nil, err
		}
	}

	return order, nil
}

func loadModule(path, dir string, files []string, overrides map[string][]byte) *Module {
	mod := &Module{
		Path:      path,
		Dir:       dir,
		FilePaths: files,
	}

	for _, f := range files {
		buffer, ok := overrides[f]
		if !ok {
			var err error
			buffer, err = os.ReadFile(f)
			if err != nil {
				mod.Buffers = append(mod.Buffers, nil)
				mod.Files = append(mod.Files, &ast.Program{})
				continue
			}
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
				mod.Imports = append(mod.Imports, d)
			}
		}
	}

	return mod
}
