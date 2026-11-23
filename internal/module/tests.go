package module

import (
	"path/filepath"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

const TestMainPath = ".wisp-test-main"

const testSuffix = ".test.wsp"

var testsEnabled bool

func ConfigureTests(on bool) {
	testsEnabled = on
}

func TestsEnabled() bool {
	return testsEnabled
}

func skipFile(name string) bool {
	return !testsEnabled && strings.HasSuffix(name, testSuffix)
}

type TestFunc struct {
	Module   string
	File     string
	Line     int
	Name     string
	Display  string
	Fallible bool

	ExpectAbort bool
	AbortText   string
}

func (t TestFunc) ID() string {
	return t.Module + ":" + t.Name
}

func exposeTests(program *ast.Program) {
	for _, decl := range program.Declarations {
		if fd, ok := decl.(*ast.FuncDecl); ok && fd.Receiver == nil && fd.TagNamed("test") != nil {
			fd.Exported = true
		}
	}
}

func dropTests(program *ast.Program) {
	kept := program.Declarations[:0]
	for _, decl := range program.Declarations {
		if fd, ok := decl.(*ast.FuncDecl); ok && fd.Receiver == nil && fd.TagNamed("test") != nil {
			continue
		}
		kept = append(kept, decl)
	}
	program.Declarations = kept
}

func testFunctions(mods []*Module) []TestFunc {
	var tests []TestFunc
	for _, mod := range mods {
		if strings.HasPrefix(mod.Path, "std/") || mod.Path == TestMainPath {
			continue
		}
		for i, file := range mod.Files {
			for _, decl := range file.Declarations {
				fd, ok := decl.(*ast.FuncDecl)
				if !ok || fd.Receiver != nil {
					continue
				}
				tag := fd.TagNamed("test")
				if tag == nil {
					continue
				}
				line, _ := fd.Position()
				test := TestFunc{Module: mod.Path, File: mod.FilePaths[i], Line: line, Name: fd.Name, Display: fd.Name, Fallible: fd.Fallible}
				if len(tag.Args) == 1 {
					if text, isText := tag.Args[0].(*ast.StringLiteral); isText && strings.TrimSpace(text.Value) != "" {
						test.Display = text.Value
					}
				}
				if abort := fd.TagNamed("expect_abort"); abort != nil {
					test.ExpectAbort = true
					if len(abort.Args) == 1 {
						if text, isText := abort.Args[0].(*ast.StringLiteral); isText {
							test.AbortText = text.Value
						}
					}
				}
				tests = append(tests, test)
			}
		}
	}
	return tests
}

func DiscoverProjectWithTests(root string, keep func(TestFunc) bool, generate func([]TestFunc) string) ([]*Module, []TestFunc, int, error) {
	roots, err := discoverModuleRoots(root)
	if err != nil {
		return nil, nil, 0, err
	}

	mods := buildOrder(root, roots, nil)
	all := testFunctions(mods)
	var tests []TestFunc
	for _, test := range all {
		if keep == nil || keep(test) {
			tests = append(tests, test)
		}
	}
	if len(tests) == 0 {
		return nil, nil, len(all), nil
	}

	virtual := filepath.Join(root, TestMainPath+".wsp")
	roots = []moduleRoot{{path: TestMainPath, dir: root, files: []string{virtual}}}
	mods = buildOrder(root, roots, map[string][]byte{virtual: []byte(generate(tests))})
	return mods, tests, len(all), nil
}
