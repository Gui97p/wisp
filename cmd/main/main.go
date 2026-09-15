package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/lexer"
	"github.com/Gui97p/wisp/internal/parser"
	"github.com/Gui97p/wisp/internal/target"
	"github.com/Gui97p/wisp/internal/target/x64"
)

func main() {
	args := os.Args
	if len(args) < 3 {
		fmt.Println("usage: wisp <module> <filename>.wsp")
		os.Exit(1)
	}

	buffer, err := os.ReadFile(args[2])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	l := lexer.NewLexer(buffer)
	p := parser.NewParser(l)

	switch args[1] {
	case "lexer":
		token := l.NextToken()
		for token.Type != lexer.TOKEN_EOF {
			fmt.Printf("%s(%s)\n", token.Type.String(), token.Literal)
			token = l.NextToken()
		}

	case "parser":
		program := p.ParseProgram()
		fmt.Println(program.Tree(""))
		fmt.Println("\nparsing errors:")
		p.ShowErrors()

	case "analyzer":
		program := p.ParseProgram()
		fmt.Println("\nparsing errors:")
		p.ShowErrors()
		a := analyser.NewAnalyser(program)
		a.Analyze()
		fmt.Println("\nanalyzing errors:")
		a.ShowErrors()

	case "build":
		program := p.ParseProgram()
		if p.HasErrors() {
			p.ShowErrors()
			os.Exit(1)
		}

		a := analyser.NewAnalyser(program)
		info := a.Analyze()
		if a.HasErrors() {
			a.ShowErrors()
			os.Exit(1)
		}

		var backend target.Target = x64.New()
		asm, err := backend.Compile(program, info)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		outputPath := strings.TrimSuffix(args[2], ".wsp")
		if err := x64.Build(asm, outputPath); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Printf("built (%s): %s\n", backend.Name(), outputPath)

	default:
		fmt.Println("invalid module.\nAvaiable: build lexer parser analyzer")
		os.Exit(1)
	}
}
