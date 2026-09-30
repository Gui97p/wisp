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
	target  string

	ctx *x64context.Context

	data   *dataSection
	rodata *rodataSection
	text   *textSection

	dataIdent  map[any]string
	identCount int

	labels map[string]int
}

func New(program *ast.Program, info *analyser.Info, isEntry bool, target string) *X64Target {
	return &X64Target{
		program: program,
		info:    info,
		isEntry: isEntry,
		target:  target,

		data:   NewData(),
		rodata: NewRodata(),
		text:   NewText(),

		dataIdent: map[any]string{},

		labels: map[string]int{},
	}
}

func (t *X64Target) Compile() (string, error) {
	for _, decl := range t.program.Declarations {
		if err := t.compileDeclaration(decl); err != nil {
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

var sizeLabels = map[int]string{
	1: "byte",
	2: "word",
	4: "dword",
	8: "qword",
}

func (t *X64Target) sizeOf(at analyser.Type) int {
	switch tp := at.(type) {
	case analyser.PrimitiveType:
		switch tp.Name {
		case "int8", "uint8", "bool", "char":
			return 1
		case "int16", "uint16":
			return 2
		case "int32", "uint32", "float32":
			return 4
		case "int", "int64", "uint", "uint64", "float64":
			return 8
		case "string":
			return 16
		}
	case analyser.UntypedIntType, analyser.UntypedFloatType:
		return 8
	case analyser.PointerType:
		return 8
	case analyser.ArrayType:
		return int(tp.Size) * t.sizeOf(tp.Element)
	case *analyser.StructType:
		size := 0
		for _, field := range tp.Fields {
			size += t.sizeOf(field)
		}
		return size
	case analyser.SpanType:
		return 16
	}

	return 0
}

func (t *X64Target) newLabel(key string) string {
	if _, ok := t.labels[key]; !ok {
		t.labels[key] = 0
	}

	label := fmt.Sprintf(".%s%d", key, t.labels[key])
	t.labels[key]++
	return label
}

func (t *X64Target) createData(key string, value any) string {
	if ident, ok := t.dataIdent[value]; ok {
		return ident
	}

	ident := fmt.Sprintf("%s%d", key, t.identCount)
	t.dataIdent[value] = ident
	t.identCount++
	return ident
}
