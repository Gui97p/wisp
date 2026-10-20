package x64

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/diag"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

type X64Target struct {
	program *ast.Program
	info    *analyser.Info
	isEntry bool
	module  string
	target  string

	ctx *x64context.Context

	prelude  *prelude
	data     *dataSection
	rodata   *rodataSection
	text     *textSection
	postlude *postlude

	dataLabel  map[any]string
	labelCount int

	err   error
	files map[ast.Declaration]string

	file      string
	line, col int

	labels      map[string]int
	funcReturns []analyser.Type
	fallible    bool
	hidden      Mem

	deferScopes []*deferScope

	libc bool
}

func New(program *ast.Program, info *analyser.Info, isEntry bool, target string, module string) *X64Target {
	return &X64Target{
		program: program,
		info:    info,
		isEntry: isEntry,
		module:  module,
		target:  target,

		prelude:  NewPrelude(),
		data:     NewData(),
		rodata:   NewRodata(),
		text:     NewText(),
		postlude: NewPostlude(),

		dataLabel: map[any]string{},

		labels: map[string]int{},
	}
}

func (t *X64Target) SetLibc(libc bool) {
	t.libc = libc
}

func (t *X64Target) SetFiles(files map[ast.Declaration]string) {
	t.files = files
}

func (t *X64Target) enter(node ast.Node) (line, col int) {
	line, col = t.line, t.col
	t.line, t.col = node.Position()
	return line, col
}

func (t *X64Target) leave(line, col int) {
	t.line, t.col = line, col
}

func displayFile(file string) string {
	if cwd, err := os.Getwd(); err == nil && file != "" {
		if rel, err := filepath.Rel(cwd, file); err == nil {
			return rel
		}
	}
	return file
}

func (t *X64Target) where() string {
	if t.line == 0 {
		return ""
	}
	file := displayFile(t.file)
	if file == "" {
		return fmt.Sprintf("\n  at line %d:%d", t.line, t.col)
	}
	return fmt.Sprintf("\n  at %s:%d:%d", file, t.line, t.col)
}

func (t *X64Target) locate(node ast.Node, err error) error {
	if err == nil {
		return nil
	}
	line, col := node.Position()
	endLine, endCol := node.EndPosition()
	return diag.Locate("x86-64", err, line, col, endLine, endCol)
}

func (t *X64Target) fail(message string, args ...any) {
	if t.err == nil {
		t.err = fmt.Errorf(message, args...)
	}
}

func (t *X64Target) Compile() (string, error) {
	for _, decl := range t.program.Declarations {
		if err := t.compileDeclaration(decl); err != nil {
			return "", diag.InFile(err, t.files[decl])
		}
	}

	code := fmt.Sprintf("%s\n%s\n%s\n%s\n%s", t.prelude.b.String(), t.rodata.b.String(), t.data.b.String(), t.text.b.String(), t.postlude.b.String())

	return code, nil
}

func Assemble(asmSource, objPath, asmPath string, keepAsm bool, format string) error {
	if format == "" {
		format = "elf64"
	}

	if err := os.WriteFile(asmPath, []byte(asmSource), 0644); err != nil {
		return fmt.Errorf("failed to write the assembly file (%s)", diag.Reason(err))
	}

	trace("nasm", []string{"-f", format, "-w+error=number-overflow", asmPath, "-o", objPath})
	nasmCmd := exec.Command("nasm", "-f", format, "-w+error=number-overflow", asmPath, "-o", objPath)
	nasmCmd.Stderr = os.Stderr
	if err := nasmCmd.Run(); err != nil {
		return fmt.Errorf("nasm failed (%s)", diag.Reason(err))
	}

	if !keepAsm {
		os.Remove(asmPath)
		removeEmptyDirs(filepath.Dir(asmPath))
	}

	return nil
}

func removeEmptyDirs(dir string) {
	for {
		if err := os.Remove(dir); err != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}
