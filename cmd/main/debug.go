package main

import (
	"fmt"
	"os"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/diag"
	"github.com/Gui97p/wisp/internal/lexer"
	"github.com/Gui97p/wisp/internal/parser"
	"github.com/spf13/cobra"
)

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Show the compiler intermediate steps",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
		}
		return cmd.Help()
	},
	SilenceUsage: true,
}

var debugLexerCmd = &cobra.Command{
	Use:          "lexer [file.wsp]",
	Short:        "Print the tokens of a file",
	Args:         cobra.MaximumNArgs(1),
	RunE:         func(cmd *cobra.Command, args []string) error { return runDebug("lexer", args) },
	SilenceUsage: true,
}

var debugParserCmd = &cobra.Command{
	Use:          "parser [file.wsp]",
	Short:        "Print the syntax tree of a file",
	Args:         cobra.MaximumNArgs(1),
	RunE:         func(cmd *cobra.Command, args []string) error { return runDebug("parser", args) },
	SilenceUsage: true,
}

var debugAnalyserCmd = &cobra.Command{
	Use:          "analyser [file.wsp]",
	Short:        "Run the analyser over a file and its imports and print the diagnostics",
	Args:         cobra.MaximumNArgs(1),
	RunE:         func(cmd *cobra.Command, args []string) error { return runDebug("analyser", args) },
	SilenceUsage: true,
}

func init() {
	debugCmd.AddCommand(debugLexerCmd, debugParserCmd, debugAnalyserCmd)
}

func runDebug(step string, args []string) error {
	inputPath, err := resolveEntry(args)
	if err != nil {
		return err
	}

	buffer, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}

	switch step {
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
		modules, ok := resolveModules(inputPath)
		if !ok {
			return nil
		}

		exports := map[string]*analyser.ModuleInfo{}
		anyErrors := false

		for _, mod := range modules {
			isEntry := mod.Path == ""
			name := mod.Path
			if isEntry {
				name = inputPath
			}

			if mod.ParseError != nil {
				anyErrors = true
				fmt.Printf("<<Parsing Errors: %s>>\n", name)
				diag.Render(os.Stdout, mod.ParseError.Path, mod.ParseError.Buffer, mod.ParseError.Errors)
				continue
			}

			merged, declFiles := mod.Merge()

			a := analyser.NewAnalyser(merged, exports, declFiles, mod.Path)

			a.SetImportErrors(mod.ImportErrors)
			a.Analyze()
			if a.HasDiagnostics() {
				fmt.Printf("<<Analyser Diagnostics: %s>>\n", name)
				diag.RenderGrouped(os.Stdout, mod.BufferMap(), a.Errors())
			}
			if a.HasErrors() {
				anyErrors = true
				continue
			}
			modExports := a.Exports()
			exports[mod.Path] = &analyser.ModuleInfo{Exports: modExports}
		}

		if !anyErrors {
			fmt.Println("<<No error found on analysis>>")
		}
	}

	return nil
}
