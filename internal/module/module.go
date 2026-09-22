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
	Path      string
	Dir       string
	Files     []*ast.Program
	Buffers   [][]byte
	FilePaths []string
	Imports   []string
}

var stdlibRoot = "std"

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
		base = filepath.Join(stdlibRoot, after)
	}

	if strings.HasSuffix(path, ".wsp") {
		return filepath.Dir(base), []string{base}, nil
	}

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

		mod, err := loadModule(path, dir, files)
		if err != nil {
			return err
		}
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

func loadModule(path, dir string, files []string) (*Module, error) {
	mod := &Module{
		Path:      path,
		Dir:       dir,
		FilePaths: files,
	}

	for _, f := range files {
		buffer, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}

		l := lexer.NewLexer(buffer)
		p := parser.NewParser(l)
		program := p.ParseProgram()

		if p.HasErrors() {
			return nil, &SourceError{Path: f, Buffer: buffer, Errors: p.Errors()}
		}

		mod.Files = append(mod.Files, program)
		mod.Buffers = append(mod.Buffers, buffer)

		for _, decl := range program.Declarations {
			if d, ok := decl.(*ast.ImportDecl); ok {
				mod.Imports = append(mod.Imports, d.Path)
			}
		}
	}

	return mod, nil
}
