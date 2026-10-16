package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/module"
	"github.com/spf13/cobra"
)

var docCmd = &cobra.Command{
	Use:   "doc [module [symbol]]",
	Short: "Show the exported API of a module with its doc comments",
	Long: `Show the exported API of a module with its doc comments.

Without arguments it lists the modules that can be imported. With a module
path, as written in an import, it lists what the module exports. With a
symbol too, it shows just that one.`,
	Args:         cobra.MaximumNArgs(2),
	RunE:         runDoc,
	SilenceUsage: true,
}

func init() {
	addProjectFlags(docCmd)
}

func runDoc(cmd *cobra.Command, args []string) error {
	root, _, err := openProject(cmd)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		for _, c := range module.ImportablePaths(root) {
			fmt.Println(c.Path)
		}
		return nil
	}

	files, err := module.ResolveImport(root, args[0])
	if err != nil {
		return err
	}

	modules, err := module.BuildGraph(files[0])
	if err != nil {
		return err
	}

	var program *ast.Program
	var exports map[string]*analyser.Symbol
	known := map[string]*analyser.ModuleInfo{}

	for _, mod := range modules {
		if mod.ParseError != nil {
			return fmt.Errorf("%s has syntax errors: run `wisp check`", mod.ParseError.Path)
		}

		merged, declFiles := mod.Merge()
		a := analyser.NewAnalyser(merged, known, declFiles, mod.Path)
		a.SetImportErrors(mod.ImportErrors)
		a.Analyze()
		known[mod.Path] = &analyser.ModuleInfo{Exports: a.Exports()}

		for _, f := range mod.FilePaths {
			if f == files[0] {
				program, exports = merged, a.Exports()
			}
		}
	}

	if exports == nil {
		return fmt.Errorf("module %q was not analysed", args[0])
	}

	names := make([]string, 0, len(exports))
	for name := range exports {
		names = append(names, name)
	}
	sort.Strings(names)

	if len(args) == 2 {
		sym, ok := exports[args[1]]
		if !ok {
			return fmt.Errorf("module %q does not export %q", args[0], args[1])
		}
		printSymbol(args[1], sym, program)
		return nil
	}

	for i, name := range names {
		if i > 0 {
			fmt.Println()
		}
		printSymbol(name, exports[name], program)
	}
	return nil
}

func printSymbol(name string, sym *analyser.Symbol, program *ast.Program) {
	fmt.Println(symbolLine(name, sym, program))
	if sym.Doc == "" {
		return
	}
	for _, line := range strings.Split(sym.Doc, "\n") {
		fmt.Println("    " + line)
	}
}

func symbolLine(name string, sym *analyser.Symbol, program *ast.Program) string {
	switch sym.Kind {
	case analyser.FUNC:
		if ft, ok := sym.Type.(*analyser.FuncType); ok {
			var decl *ast.FuncDecl
			for _, d := range program.Declarations {
				if fd, ok := d.(*ast.FuncDecl); ok && fd.Receiver == nil && fd.Name == name {
					decl = fd
				}
			}
			return analyser.FuncSignature(name, ft, decl)
		}
	case analyser.STRUCT:
		if st, ok := sym.Type.(*analyser.StructType); ok {
			return analyser.StructText(st)
		}
	case analyser.TYPE:
		if nt, ok := sym.Type.(analyser.NamedType); ok {
			if nt.Enum {
				return analyser.EnumText(nt)
			}
			return fmt.Sprintf("type %s %s", name, analyser.TypeText(nt.Underlying))
		}
	case analyser.CONST:
		if sym.Const != nil {
			value := sym.Const.String()
			if sym.Const.Kind == analyser.ConstString {
				value = strconv.Quote(sym.Const.Str)
			}
			return fmt.Sprintf("const %s = %s", name, value)
		}
	case analyser.MODULE:
		return "module " + name
	}
	if sym.Type == nil {
		return name
	}
	return fmt.Sprintf("%s %s", analyser.TypeText(sym.Type), name)
}
