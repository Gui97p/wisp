package module

import (
	"fmt"
	"github.com/Gui97p/wisp/internal/diag"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/folders"
	"github.com/Gui97p/wisp/internal/lexer"
	"github.com/Gui97p/wisp/internal/parser"
	"github.com/Gui97p/wisp/internal/target"
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

	Dependency string
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

func StdlibDisabled() bool {
	return stdlibDisabled
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

func stdModulePath(file string) (string, bool) {
	stdRoot, err := StdlibRoot()
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(stdRoot, file)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", false
	}
	rel = filepath.ToSlash(strings.TrimSuffix(rel, ".wsp"))
	dir, name := filepath.Split(rel)
	if dir != "" && strings.TrimSuffix(dir, "/") == name {
		return "std/" + name, true
	}
	return "std/" + rel, true
}

func canonicalModulePath(root, file string) (string, error) {
	rel, err := filepath.Rel(root, file)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(strings.TrimSuffix(rel, ".wsp")), nil
}

type projectScope struct {
	root       string
	prefix     string
	dependency string
}

func resolveImportPath(projectRoot, path string) (key, dir string, files []string, err error) {
	key, dir, files, _, err = resolveImport(projectScope{root: projectRoot}, path, nil)
	return key, dir, files, err
}

func resolveImport(scope projectScope, path string, registry map[string]string) (key, dir string, files []string, next projectScope, err error) {
	name, rest, hasRest := strings.Cut(path, "/")
	if name != "std" {
		if depRoot, ok := dependenciesOf(scope.root)[name]; ok {
			next = projectScope{root: depRoot, prefix: name + "/", dependency: name}
			key, dir, files, err = resolveDependency(scope, next, name, rest, hasRest, registry)
			return key, dir, files, next, err
		}
	}
	key, dir, files, err = resolveIn(scope.root, scope.prefix, path)
	return key, dir, files, scope, err
}

func resolveDependency(scope, next projectScope, name, rest string, hasRest bool, registry map[string]string) (key, dir string, files []string, err error) {
	if info, statErr := os.Stat(next.root); statErr != nil || !info.IsDir() {
		return "", "", nil, fmt.Errorf("the dependency %s points to %s, which is not a folder", name, next.root)
	}
	local := filepath.Join(scope.root, name)
	if _, statErr := os.Stat(local + ".wsp"); statErr == nil {
		return "", "", nil, fmt.Errorf("import %q is ambiguous: it names the dependency %s and the file %s.wsp", name, name, local)
	}
	if info, statErr := os.Stat(local); statErr == nil && info.IsDir() {
		return "", "", nil, fmt.Errorf("import %q is ambiguous: it names the dependency %s and the folder %s", name, name, local)
	}
	if registry != nil {
		if other, seen := registry[name]; seen && other != next.root {
			return "", "", nil, fmt.Errorf("the dependency %s is %s here and %s somewhere else: a build uses one project per name, so declare the same path in both", name, next.root, other)
		}
		registry[name] = next.root
	}

	if !hasRest {
		moduleFiles, _ := folderFiles(next.root, nil)
		if len(moduleFiles) == 0 {
			return "", "", nil, fmt.Errorf("the dependency %s has no Wisp files in its root folder: import one of its folders, such as %q", name, name+"/…")
		}
		return name, next.root, moduleFiles, nil
	}
	return resolveIn(next.root, next.prefix, rest)
}

func scopedKey(scope projectScope, target string) string {
	if stdRoot, err := StdlibRoot(); err == nil {
		if rel, err := filepath.Rel(stdRoot, target); err == nil && !strings.HasPrefix(rel, "..") {
			return "std/" + filepath.ToSlash(rel)
		}
	}
	rel, err := filepath.Rel(scope.root, target)
	if err != nil {
		return target
	}
	return scope.prefix + filepath.ToSlash(rel)
}

func resolveIn(projectRoot, prefix, path string) (key, dir string, files []string, err error) {
	isStd := strings.HasPrefix(path, "std/")
	if isStd && stdlibDisabled {
		return "", "", nil, fmt.Errorf("import %q: this build does not use the standard library (std = \"none\"); provide one with --std <path>", path)
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
		if !isStandalone(file, nil) {
			return "", "", nil, fmt.Errorf("%s.wsp is one of the files of the module of its folder: import the folder, or write #standalone at the top of the file to make it a module of its own", filepath.Base(base))
		}
		key = path
		if !isStd {
			key = prefix + filepath.ToSlash(strings.TrimSuffix(mustRel(projectRoot, file), ".wsp"))
		}
		return key, filepath.Dir(base), []string{file}, nil
	case dirExists:
		moduleFiles, _ := folderFiles(base, nil)
		if len(moduleFiles) == 0 {
			return "", "", nil, fmt.Errorf("the folder %q has no Wisp files of its own to make a module", path)
		}
		key = path
		if !isStd {
			key = prefix + filepath.ToSlash(mustRel(projectRoot, base))
		}
		return key, base, moduleFiles, nil
	default:
		return "", "", nil, fmt.Errorf("cannot resolve import %q", path)
	}
}

func mustRel(root, target string) string {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return target
	}
	return rel
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
	registry := map[string]string{}
	var order []*Module

	var visit func(path, dir string, files []string, scope projectScope)
	visit = func(path, dir string, files []string, scope projectScope) {
		if visited[path] {
			return
		}
		visiting[path] = true

		mod := loadModule(path, dir, files, overrides)
		mod.Dependency = scope.dependency

		for _, imp := range mod.Imports {
			key, depDir, depFiles, depScope, err := resolveImport(scope, imp.Path, registry)
			if err != nil {
				message := err.Error()
				if hint := siblingImportHint(scope.root, mod.Dir, imp.Path); hint != "" {
					message += "\n" + hint
				}
				mod.FailImport(imp.Path, message)
				continue
			}
			if key == path {
				mod.FailImport(imp.Path, fmt.Sprintf("module %q imports itself", key))
				continue
			}
			imp.Path = key
			if visiting[key] {
				continue
			}
			visit(key, depDir, depFiles, depScope)
		}

		visiting[path] = false
		visited[path] = true
		order = append(order, mod)
	}

	for _, r := range roots {
		scope := r.scope
		if scope.root == "" {
			scope.root = root
		}
		visit(r.path, r.dir, r.files, scope)
	}

	return order
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
	scope projectScope
}

func discoverModuleRoots(root string, overrides map[string][]byte) ([]moduleRoot, error) {
	var roots []moduleRoot

	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if p != root && (strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")) {
			return filepath.SkipDir
		}
		if p != root {
			if _, err := os.Stat(filepath.Join(p, "wisp.toml")); err == nil {
				return filepath.SkipDir
			}
		}

		files, standalone := folderFiles(p, overrides)
		if len(files) > 0 {
			roots = append(roots, moduleRoot{path: folderKey(root, p), dir: p, files: files})
		}
		for _, file := range standalone {
			roots = append(roots, moduleRoot{path: fileKey(root, file), dir: p, files: []string{file}})
		}
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
	roots, err := discoverModuleRoots(root, overrides)
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
		if program.Target != "" && !target.MatchesText(currentTarget(), program.Target) {
			program.Declarations = nil
		}

		if testsEnabled {
			exposeTests(program)
		} else {
			dropTests(program)
		}

		mod.Files = append(mod.Files, program)
		mod.Buffers = append(mod.Buffers, buffer)

		if p.HasErrors() && mod.ParseError == nil {
			mod.ParseError = &SourceError{Path: f, Buffer: buffer, Errors: p.Errors()}
		}

		for _, decl := range program.Declarations {
			if d, ok := decl.(*ast.ImportDecl); ok {
				if d.Selector != "" && !target.MatchesText(currentTarget(), d.Selector) {
					continue
				}
				mod.Imports = append(mod.Imports, d)
			}
		}
	}

	return mod
}

type ImportCandidate struct {
	Path  string
	File  string
	Files []string
}

func (c ImportCandidate) Contains(file string) bool {
	return slices.Contains(c.Files, file)
}

func ImportablePaths(projectRoot string) []ImportCandidate {
	candidates := importableUnder(projectRoot, "", true, "")

	if !stdlibDisabled {
		if stdRoot, err := StdlibRoot(); err == nil {
			candidates = append(candidates, importableUnder(stdRoot, "std/", false, "")...)
		}
	}

	deps := dependenciesOf(projectRoot)
	names := make([]string, 0, len(deps))
	for name := range deps {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		candidates = append(candidates, importableUnder(deps[name], name+"/", false, name)...)
	}
	return candidates
}

func importableUnder(root, prefix string, skipNested bool, rootName string) []ImportCandidate {
	var out []ImportCandidate

	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if p != root && (strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")) {
			return filepath.SkipDir
		}
		if skipNested && p != root {
			if _, err := os.Stat(filepath.Join(p, "wisp.toml")); err == nil {
				return filepath.SkipDir
			}
		}

		files, standalone := folderFiles(p, nil)
		if len(files) > 0 {
			switch {
			case p != root:
				out = append(out, ImportCandidate{Path: prefix + filepath.ToSlash(mustRel(root, p)), File: files[0], Files: files})
			case rootName != "":
				out = append(out, ImportCandidate{Path: rootName, File: files[0], Files: files})
			}
		}
		for _, file := range standalone {
			rel := strings.TrimSuffix(filepath.ToSlash(mustRel(root, file)), ".wsp")
			out = append(out, ImportCandidate{Path: prefix + rel, File: file, Files: []string{file}})
		}
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

func Closure(modules []*Module, roots []string) map[string]bool {
	byPath := make(map[string]*Module, len(modules))
	contributors := map[string][]string{}
	for _, m := range modules {
		byPath[m.Path] = m
		for _, target := range m.methodTargets() {
			contributors[target] = append(contributors[target], m.Path)
		}
	}

	seen := map[string]bool{}
	var visit func(path string)
	visit = func(path string) {
		m, ok := byPath[path]
		if !ok || seen[path] {
			return
		}
		seen[path] = true
		for _, imp := range m.Imports {
			visit(imp.Path)
		}
		for _, part := range m.Parts {
			visit(part)
		}
		for _, contributor := range contributors[path] {
			visit(contributor)
		}
	}
	for _, root := range roots {
		visit(root)
	}
	return seen
}

func ImportKey(projectRoot, path string) (string, error) {
	key, _, _, err := resolveImportPath(projectRoot, path)
	return key, err
}
