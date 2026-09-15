package x64

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

type X64Target struct{}

func New() *X64Target {
	return &X64Target{}
}

func (*X64Target) Name() string {
	return "x64"
}

func (*X64Target) Compile(program *ast.Program, info *analyser.Info) (string, error) {
	var b strings.Builder

	b.WriteString("global main\n\n")

	for _, decl := range program.Declarations {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if err := compileFunc(&b, fd); err != nil {
			return "", err
		}
	}

	return b.String(), nil
}

func Build(asmSource, outputPath string) error {
	asmPath := outputPath + ".asm"
	objPath := outputPath + ".o"

	if err := os.WriteFile(asmPath, []byte(asmSource), 0644); err != nil {
		return fmt.Errorf("failed to write asm file: %w", err)
	}

	nasmCmd := exec.Command("nasm", "-f", "elf64", asmPath, "-o", objPath)
	nasmCmd.Stderr = os.Stderr
	if err := nasmCmd.Run(); err != nil {
		return fmt.Errorf("nasm failed: %w", err)
	}

	gccCmd := exec.Command("gcc", objPath, "-o", outputPath)
	gccCmd.Stderr = os.Stderr
	if err := gccCmd.Run(); err != nil {
		return fmt.Errorf("gcc failed: %w", err)
	}

	return nil
}

func compileFunc(b *strings.Builder, fd *ast.FuncDecl) error {
	fmt.Fprintf(b, "%s:\n", fd.Name)
	b.WriteString("\tpush rbp\n")
	b.WriteString("\tmov rbp, rsp\n\n")

	for _, stmt := range fd.Body.Statements {
		if err := compileStmt(b, stmt); err != nil {
			return err
		}
	}

	b.WriteString("\n\tmov rsp, rbp\n")
	b.WriteString("\tpop rbp\n")
	b.WriteString("\tret\n\n")

	return nil
}
