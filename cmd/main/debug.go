package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/diag"
	"github.com/Gui97p/wisp/internal/lexer"
	"github.com/Gui97p/wisp/internal/module"
	"github.com/Gui97p/wisp/internal/parser"
	"github.com/spf13/cobra"
)

var debugCmd = &cobra.Command{
	Use:          "debug <mod> <file.wsp>",
	Short:        "Debug command that shows the compiler intermediate steps",
	Long:         "Debug command that shows the compiler intermediate steps.\nAvailable modules: lexer, parser, analyser",
	Args:         cobra.RangeArgs(1, 2),
	RunE:         runDebug,
	SilenceUsage: true,
}

func runDebug(cmd *cobra.Command, args []string) error {
	inputPath, err := resolveEntry(args[1:])
	if err != nil {
		return err
	}

	buffer, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}

	switch args[0] {
	case "lexer":
		l := lexer.NewLexer(buffer)
		token := l.NextToken()
		for token.Type != lexer.TOKEN_EOF {
			fmt.Printf("%s(%s)\n", token.Type.String(), token.Literal)
			token = l.NextToken()
		}

	case "parser":
		l := lexer.NewLexer(buffer)
		p := parser.NewParser(l)
		program := p.ParseProgram()
		fmt.Println(program.Tree(""))
		if p.HasErrors() {
			fmt.Println("\n<<Parsing Errors>>")
			diag.Render(os.Stdout, inputPath, buffer, p.Errors())
		} else {
			fmt.Println("<<No error found on parsing>>")
		}

	case "analyser":
		modules, err := module.BuildGraph(inputPath)
		if err != nil {
			if srcErr, ok := errors.AsType[*module.SourceError](err); ok {
				diag.Render(os.Stdout, srcErr.Path, srcErr.Buffer, srcErr.Errors)
			} else {
				fmt.Fprintln(os.Stderr, err)
			}
			return nil
		}

		exports := map[string]*analyser.ModuleInfo{}
		anyErrors := false

		for _, mod := range modules {
			merged := &ast.Program{}
			for _, f := range mod.Files {
				merged.Declarations = append(merged.Declarations, f.Declarations...)
			}

			isEntry := mod.Path == ""
			a := analyser.NewAnalyser(merged, isEntry, exports)
			a.Analyze()
			if a.HasErrors() {
				anyErrors = true
				name := mod.Path
				if isEntry {
					name = inputPath
				}
				fmt.Printf("<<Analyser Errors: %s>>\n", name)
				diag.Render(os.Stdout, mod.FilePaths[0], mod.Buffers[0], a.Errors())
			}
			exports[mod.Path] = &analyser.ModuleInfo{Exports: a.Exports()}
		}

		if !anyErrors {
			fmt.Println("<<No error found on analysis>>")
		}
	}

	return nil
}
