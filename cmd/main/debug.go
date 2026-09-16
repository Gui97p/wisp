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
	Use:          "debug <mod> <file.wsp>",
	Short:        "Debug command that shows the compiler intermediate steps\nAvaiable modules: lexer, parser, analyser",
	Args:         cobra.ExactArgs(2),
	RunE:         runDebug,
	SilenceUsage: true,
}

func runDebug(cmd *cobra.Command, args []string) error {
	inputPath := args[1]
	buffer, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}

	l := lexer.NewLexer(buffer)
	p := parser.NewParser(l)

	switch args[0] {
	case "lexer":
		token := l.NextToken()
		for token.Type != lexer.TOKEN_EOF {
			fmt.Printf("%s(%s)\n", token.Type.String(), token.Literal)
			token = l.NextToken()
		}

	case "parser":
		program := p.ParseProgram()
		fmt.Println(program.Tree(""))
		if p.HasErrors() {
			fmt.Println("\n<<Parsing Errors>>")
			diag.Render(os.Stdout, args[1], buffer, p.Errors())
		} else {
			fmt.Println("<<No error found on parsing>>")
		}

	case "analyser":
		program := p.ParseProgram()
		a := analyser.NewAnalyser(program)
		a.Analyze()
		if a.HasErrors() {
			fmt.Println("\n<<Analyser Errors>>")
			diag.Render(os.Stdout, args[2], buffer, a.Errors())
		} else {
			fmt.Println("<<No error found on analysis>>")
		}
	}

	return nil
}
