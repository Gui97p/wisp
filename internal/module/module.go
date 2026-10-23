package module

import (
	"fmt"
	"github.com/Gui97p/wisp/internal/diag"
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

	ImportErrors map[string]string
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

var (
	stdlibOverride string
	stdlibDisabled bool
)

func ConfigureStdlib(path string, disabled bool) {
	stdlibOverride = path
	stdlibDisabled = disabled
}

func StdlibRoot() (string, error) {
	if stdlibOverride != "" {
		return stdlibOverride, nil
	}

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
	if isStd && stdlibDisabled {
		return "", "", nil, fmt.Errorf("import %q: this build does not use the standard library (--freestanding); provide one with --std <path>", path)
	}
	base := filepath.Join(projectRoot, path)
	if after, ok := strings.CutPrefix(path, "std/"); ok {
		stdRoot, err := StdlibRoot()
		if err != nil {
			return "", "", nil, fmt.Errorf("cannot locate the standard library (%s)", diag.Reason(err))
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

	return buildOrder(root, []moduleRoot{{path: "", dir: filepath.Dir(entryFile), files: []string{entryFile}}}, overrides), nil
}

func buildOrder(root string, roots []moduleRoot, overrides map[string][]byte) []*Module {
	visiting := map[string]bool{}
	visited := map[string]bool{}
	var order []*Module

	var visit func(path, dir string, files []string)
	visit = func(path, dir string, files []string) {
		if visited[path] {
			return
		}
		visiting[path] = true

		mod := loadModule(path, dir, files, overrides)

		for _, imp := range mod.Imports {
			key, depDir, depFiles, err := resolveImportPath(root, imp.Path)
			if err != nil {
				message := err.Error()
				if hint := siblingImportHint(root, mod.Dir, imp.Path); hint != "" {
					message += "\n" + hint
				}
				mod.FailImport(imp.Path, message)
				continue
			}
			if visiting[key] {
				mod.FailImport(imp.Path, fmt.Sprintf("cyclic import: %q", key))
				continue
			}
			imp.Path = key
			visit(key, depDir, depFiles)
		}

		visiting[path] = false
		visited[path] = true
		order = append(order, mod)
	}

	for _, r := range roots {
		visit(r.path, r.dir, r.files)
	}

	return order
}

func siblingImportHint(root, dir, path string) string {
	if dir == "" || strings.HasPrefix(path, "std/") {
		return ""
	}
	candidate := filepath.Join(dir, path)
	if _, _, _, err := resolveImportPath(root, importKey(root, candidate)); err != nil {
		return ""
	}
	return fmt.Sprintf("an import path starts at the project root, so this one is written %q", importKey(root, candidate))
}

func importKey(root, target string) string {
	if stdRoot, err := StdlibRoot(); err == nil {
		if rel, err := filepath.Rel(stdRoot, target); err == nil && !strings.HasPrefix(rel, "..") {
			return "std/" + filepath.ToSlash(rel)
		}
	}
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return target
	}
	return filepath.ToSlash(rel)
}

func (m *Module) FailImport(path, message string) {
	if m.ImportErrors == nil {
		m.ImportErrors = map[string]string{}
	}
	m.ImportErrors[path] = message
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
	return DiscoverProjectWithOverrides(root, nil)
}

func DiscoverProjectWithOverrides(root string, overrides map[string][]byte) ([]*Module, error) {
	roots, err := discoverModuleRoots(root)
	if err != nil {
		return nil, err
	}

	return buildOrder(root, roots, overrides), nil
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

type ImportCandidate struct {
	Path string
	File string
}

func ImportablePaths(projectRoot string) []ImportCandidate {
	candidates := importableUnder(projectRoot, "", true)

	if !stdlibDisabled {
		if stdRoot, err := StdlibRoot(); err == nil {
			candidates = append(candidates, importableUnder(stdRoot, "std/", false)...)
		}
	}

	return candidates
}

func importableUnder(root, prefix string, skipNested bool) []ImportCandidate {
	var out []ImportCandidate

	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if p != root && (strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")) {
				return filepath.SkipDir
			}
			if skipNested && p != root {
				if _, err := os.Stat(filepath.Join(p, "wisp.toml")); err == nil {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(p, ".wsp") {
			return nil
		}

		path, err := canonicalModulePath(root, p)
		if err != nil {
			return nil
		}
		if dir, base := filepath.Split(path); strings.HasSuffix(dir, base+"/") {
			path = strings.TrimSuffix(dir, "/")
		}
		out = append(out, ImportCandidate{Path: prefix + path, File: p})
		return nil
	})

	return out
}

func ResolveImport(projectRoot, path string) ([]string, error) {
	_, _, files, err := resolveImportPath(projectRoot, path)
	return files, err
}
