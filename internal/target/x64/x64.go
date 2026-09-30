package x64

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

type X64Target struct {
	program *ast.Program
	info    *analyser.Info
	isEntry bool
	module  string
	target  string

	ctx *x64context.Context

	prelude *prelude
	data    *dataSection
	rodata  *rodataSection
	text    *textSection

	dataLabel  map[any]string
	labelCount int

	labels map[string]int
}

func New(program *ast.Program, info *analyser.Info, isEntry bool, target string, module string) *X64Target {
	return &X64Target{
		program: program,
		info:    info,
		isEntry: isEntry,
		module:  module,
		target:  target,

		prelude: NewPrelude(),
		data:    NewData(),
		rodata:  NewRodata(),
		text:    NewText(),

		dataLabel: map[any]string{},

		labels: map[string]int{},
	}
}

func (t *X64Target) Compile() (string, error) {
	for _, decl := range t.program.Declarations {
		if err := t.compileDeclaration(decl); err != nil {
			return "", err
		}
	}

	code := fmt.Sprintf("%s\n%s\n%s\n%s", t.prelude.b.String(), t.rodata.b.String(), t.data.b.String(), t.text.b.String())

	return code, nil
}

func Assemble(asmSource, objPath, asmPath string, keepAsm bool) error {
	if err := os.WriteFile(asmPath, []byte(asmSource), 0644); err != nil {
		return fmt.Errorf("failed to write asm file: %w", err)
	}

	nasmCmd := exec.Command("nasm", "-f", "elf64", "-w+error=number-overflow", asmPath, "-o", objPath)
	nasmCmd.Stderr = os.Stderr
	if err := nasmCmd.Run(); err != nil {
		return fmt.Errorf("nasm failed: %w", err)
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
