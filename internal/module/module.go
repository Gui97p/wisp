package module

import (
	"fmt"
	"github.com/Gui97p/wisp/internal/diag"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/folders"
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

	Parts  []string
	PartOf string
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
		return key, filepath.Dir(base), withVariants([]string{file}), nil
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
		return key, base, withVariants([]string{modFile}), nil
	default:
		return "", "", nil, fmt.Errorf("cannot resolve import %q", path)
	}
}

func BuildGraph(entryFile string) ([]*Module, error) {
	return BuildGraphWithOverrides(entryFile, nil)
}

func BuildGraphWithOverrides(entryFile string, overrides map[string][]byte) ([]*Module, error) {
	root := FindProjectRoot(filepath.Dir(entryFile))

	return buildOrder(root, []moduleRoot{{path: "", dir: filepath.Dir(entryFile), files: withVariants([]string{entryFile})}}, overrides), nil
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
			if len(depFiles) == 1 && len(mod.FilePaths) > 0 {
				if folder := packageFolder(depFiles[0]); folder != "" && !InsideFolder(mod.FilePaths[0], folder) {
					pkg := importKey(root, folder)
					mod.FailImport(imp.Path, fmt.Sprintf("module %q is internal to %q\nimport %q instead", importKey(root, strings.TrimSuffix(depFiles[0], ".wsp")), pkg, pkg))
					continue
				}
			}
			if key == path {
				mod.FailImport(imp.Path, fmt.Sprintf("module %q imports itself", key))
				continue
			}
			imp.Path = key
			if visiting[key] {
				continue
			}
			visit(key, depDir, depFiles)
		}

		for _, sibling := range methodSiblings(root, mod, overrides) {
			visit(sibling.path, sibling.dir, sibling.files)
		}

		for _, part := range packageParts(root, mod) {
			mod.Parts = append(mod.Parts, part.path)
			visit(part.path, part.dir, part.files)
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

func packageFolder(file string) string {
	dir := filepath.Dir(file)
	entry := filepath.Join(dir, filepath.Base(dir)+".wsp")
	if file == entry {
		return ""
	}
	if info, err := os.Stat(entry); err != nil || info.IsDir() {
		return ""
	}
	return dir
}

func CanImport(importer, candidate string) bool {
	folder := packageFolder(candidate)
	if folder == "" {
		return true
	}
	return importer != "" && InsideFolder(importer, folder)
}

func InsideFolder(file, folder string) bool {
	return folders.Inside(file, folder)
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
			if p != root {
				if _, err := os.Stat(filepath.Join(p, "wisp.toml")); err == nil {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(p, ".wsp") || !FileApplies(p) || isVariantFile(p) {
			return nil
		}
		path, err := canonicalModulePath(root, p)
		if err != nil {
			return err
		}
		roots = append(roots, moduleRoot{path: path, dir: filepath.Dir(p), files: withVariants([]string{p})})
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
		if !strings.HasSuffix(p, ".wsp") || !FileApplies(p) || isVariantFile(p) {
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

func (m *Module) methodTargets() []string {
	aliases := map[string]string{}
	for _, imp := range m.Imports {
		aliases[imp.Alias] = imp.Path
	}

	var targets []string
	for _, f := range m.Files {
		for _, d := range f.Declarations {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Receiver == nil || fd.Receiver.Type.Module == "" {
				continue
			}
			if path, ok := aliases[fd.Receiver.Type.Module]; ok && !slices.Contains(targets, path) {
				targets = append(targets, path)
			}
		}
	}
	return targets
}

func packageParts(root string, mod *Module) []moduleRoot {
	if len(mod.FilePaths) == 0 || mod.Dir == "" {
		return nil
	}
	entryFile := filepath.Join(mod.Dir, filepath.Base(mod.Dir)+".wsp")
	if mod.FilePaths[0] != entryFile {
		return nil
	}
	entries, err := os.ReadDir(mod.Dir)
	if err != nil {
		return nil
	}

	var parts []moduleRoot
	for _, entry := range entries {
		file := filepath.Join(mod.Dir, entry.Name())
		if entry.IsDir() || !strings.HasSuffix(file, ".wsp") || file == entryFile || !FileApplies(file) || isVariantFile(file) {
			continue
		}
		key := importKey(root, strings.TrimSuffix(file, ".wsp"))
		parts = append(parts, moduleRoot{path: key, dir: mod.Dir, files: withVariants([]string{file})})
	}
	return parts
}

func methodSiblings(root string, mod *Module, overrides map[string][]byte) []moduleRoot {
	if len(mod.FilePaths) == 0 || mod.Dir == "" {
		return nil
	}
	entries, err := os.ReadDir(mod.Dir)
	if err != nil {
		return nil
	}

	var found []moduleRoot
	for _, entry := range entries {
		file := filepath.Join(mod.Dir, entry.Name())
		if entry.IsDir() || !strings.HasSuffix(file, ".wsp") || slices.Contains(mod.FilePaths, file) || !FileApplies(file) || isVariantFile(file) {
			continue
		}

		sibling := loadModule("", mod.Dir, withVariants([]string{file}), overrides)
		resolved := map[string]string{}
		for _, imp := range sibling.Imports {
			if key, _, _, err := resolveImportPath(root, imp.Path); err == nil {
				resolved[imp.Alias] = key
			}
		}
		contributes := false
		for _, f := range sibling.Files {
			for _, d := range f.Declarations {
				fd, ok := d.(*ast.FuncDecl)
				if ok && fd.Receiver != nil && fd.Receiver.Type.Module != "" && resolved[fd.Receiver.Type.Module] == mod.Path {
					contributes = true
				}
			}
		}
		if !contributes {
			continue
		}

		key := importKey(root, strings.TrimSuffix(file, ".wsp"))
		if base := filepath.Base(mod.Dir); entry.Name() == base+".wsp" {
			key = importKey(root, mod.Dir)
		}
		found = append(found, moduleRoot{path: key, dir: mod.Dir, files: withVariants([]string{file})})
	}
	return found
}

func Groups(modules []*Module) [][]*Module {
	byPath := make(map[string]*Module, len(modules))
	for _, m := range modules {
		byPath[m.Path] = m
	}

	for _, m := range modules {
		for _, part := range m.Parts {
			if p, ok := byPath[part]; ok {
				p.PartOf = m.Path
			}
		}
	}

	contributors := map[string][]string{}
	for _, m := range modules {
		for _, target := range m.methodTargets() {
			contributors[target] = append(contributors[target], m.Path)
		}
	}

	index := map[string]int{}
	low := map[string]int{}
	onStack := map[string]bool{}
	var stack []*Module
	var groups [][]*Module
	next := 0

	var connect func(m *Module)
	connect = func(m *Module) {
		index[m.Path], low[m.Path] = next, next
		next++
		stack = append(stack, m)
		onStack[m.Path] = true

		deps := make([]string, 0, len(m.Imports)+len(contributors[m.Path])+len(m.Parts))
		for _, imp := range m.Imports {
			deps = append(deps, imp.Path)
		}
		deps = append(deps, contributors[m.Path]...)
		deps = append(deps, m.Parts...)

		for _, depPath := range deps {
			dep, ok := byPath[depPath]
			if !ok {
				continue
			}
			if _, seen := index[dep.Path]; !seen {
				connect(dep)
				low[m.Path] = min(low[m.Path], low[dep.Path])
			} else if onStack[dep.Path] {
				low[m.Path] = min(low[m.Path], index[dep.Path])
			}
		}

		if low[m.Path] == index[m.Path] {
			var group []*Module
			for {
				top := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[top.Path] = false
				group = append(group, top)
				if top == m {
					break
				}
			}
			slices.Reverse(group)
			groups = append(groups, group)
		}
	}

	for _, m := range modules {
		if _, seen := index[m.Path]; !seen {
			connect(m)
		}
	}
	return groups
}
