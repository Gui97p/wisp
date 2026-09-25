package x64

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

type X64Target struct {
	program *ast.Program
	info    *analyser.Info
	isEntry bool

	data   *dataSection
	rodata *rodataSection
	text   *textSection
}

func New(program *ast.Program, info *analyser.Info, isEntry bool) *X64Target {
	return &X64Target{
		program: program,
		info:    info,
		isEntry: isEntry,

		data:   NewData(),
		rodata: NewRodata(),
		text:   NewText(),
	}
}

func (t *X64Target) Compile() (string, error) {
	if t.isEntry {
		t.text.println("global main\n")
	}

	for _, decl := range t.program.Declarations {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if err := compileFunc(&t.text.b, fd); err != nil {
			return "", err
		}
	}

	code := fmt.Sprintf("%s\n%s\n%s", t.rodata.b.String(), t.data.b.String(), t.text.b.String())

	return code, nil
}

func Assemble(asmSource, objPath, asmPath string, keepAsm bool) error {
	if err := os.WriteFile(asmPath, []byte(asmSource), 0644); err != nil {
		return fmt.Errorf("failed to write asm file: %w", err)
	}

	nasmCmd := exec.Command("nasm", "-f", "elf64", asmPath, "-o", objPath)
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

func Link(objPaths []string, outputPath string, keepObj bool) error {
	args := append(objPaths, "-o", outputPath)
	gccCmd := exec.Command("gcc", args...)
	gccCmd.Stderr = os.Stderr
	if err := gccCmd.Run(); err != nil {
		return fmt.Errorf("gcc failed: %w", err)
	}

	if !keepObj {
		for _, path := range objPaths {
			os.Remove(path)
			removeEmptyDirs(filepath.Dir(path))
		}
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
