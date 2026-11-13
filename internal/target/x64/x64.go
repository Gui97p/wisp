package x64

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/diag"
	"github.com/Gui97p/wisp/internal/target"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
	"github.com/Gui97p/wisp/internal/toolchain"
)

type coalesceFrame struct {
	slots      []Mem
	end        string
	deferDepth int
	lastYield  ast.Statement
}

type callSite struct {
	label string
	text  string
}

type savedReg struct {
	reg    x64context.Register
	offset int
}

type candidate struct {
	sym    *analyser.Symbol
	weight int
	class  x64context.Class
}

type inlineFrame struct {
	call       *analyser.InlineCall
	lastReturn ast.Statement
	results    []Operand
	label      string
	deferDepth int
}

type loopState struct {
	p, n, i, count *analyser.Symbol
}

type X64Target struct {
	coalesceFrames []coalesceFrame
	loopSyms       map[*ast.ForStmt]*loopState

	program *ast.Program
	info    *analyser.Info
	isEntry bool
	module  string
	target  string
	format  string
	abi     string

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

	file            string
	line, col       int
	endLine, endCol int

	labels       map[string]int
	funcReturns  []analyser.Type
	fallible     bool
	endsInReturn bool
	frameless    bool
	lastReturn   ast.Statement
	hidden       Mem
	inlineFrames []inlineFrame
	candidates   []candidate
	savedRegs    []savedReg
	savedXmms    []savedReg

	deferScopes []*deferScope
	lambdas     []pendingLambda
	lambdaCount int

	sources  map[string][]byte
	sites    []callSite
	funcName string

	inPlace x64context.Register

	libc, trace, snippets, optimize, checks bool
}

func New(program *ast.Program, info *analyser.Info, isEntry bool, t, format, abi string, module string) (*X64Target, error) {
	prof := target.ActiveProfile()

	data, err := NewData(format)
	if err != nil {
		return nil, err
	}
	rodata, err := NewRodata(format)
	if err != nil {
		return nil, err
	}
	text, err := NewText(format)
	if err != nil {
		return nil, err
	}

	return &X64Target{
		program: program,
		info:    info,
		isEntry: isEntry,
		module:  module,
		target:  t,
		format:  format,
		abi:     abi,

		prelude:  NewPrelude(),
		data:     data,
		rodata:   rodata,
		text:     text,
		postlude: NewPostlude(),

		dataLabel: map[any]string{},

		labels: map[string]int{},

		trace:    prof.Trace,
		snippets: prof.Snippets,
		optimize: prof.Optimize,
		checks:   prof.Checks,
	}, nil
}

func (t *X64Target) SetLibc(libc bool) {
	t.libc = libc
}

func (t *X64Target) SetSources(sources map[string][]byte) {
	t.sources = sources
}

func (t *X64Target) SetFiles(files map[ast.Declaration]string) {
	t.files = files
}

func (t *X64Target) enter(node ast.Node) (line, col, endLine, endCol int) {
	line, col, endLine, endCol = t.line, t.col, t.endLine, t.endCol
	t.line, t.col = node.Position()
	t.endLine, t.endCol = node.EndPosition()
	return
}

func (t *X64Target) leave(line, col, endLine, endCol int) {
	t.line, t.col = line, col
	t.endLine, t.endCol = endLine, endCol
}

func displayFile(file string) string {
	return diag.DisplayPath(file)
}

func (t *X64Target) where(label string) string {
	if t.line == 0 {
		return ""
	}
	if src, ok := t.sources[t.file]; ok && t.snippets {
		block := diag.RuntimeBlock("", t.file, src, t.line, t.col, t.endLine, t.endCol, label)
		if block != "" {
			return strings.TrimSuffix(block, "\n")
		}
	}
	file := displayFile(t.file)
	if file == "" {
		return fmt.Sprintf("\n--> line %d:%d", t.line, t.col)
	}
	return fmt.Sprintf("\n--> %s:%d:%d", file, t.line, t.col)
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

func Assemble(asmSource, objPath, asmPath string, keepAsm bool, format string, defines ...string) error {
	if err := os.WriteFile(asmPath, []byte(asmSource), 0644); err != nil {
		return fmt.Errorf("failed to write the assembly file (%s)", diag.Reason(err))
	}

	args := []string{"-f", format, "-w+error=number-overflow", asmPath, "-o", objPath}
	for _, def := range defines {
		args = append(args, "-d"+def)
	}
	nasm, err := toolchain.Find("nasm")
	if err != nil {
		return err
	}
	trace(nasm, args)
	nasmCmd := exec.Command(nasm, args...)
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
