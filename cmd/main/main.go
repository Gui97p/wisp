package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/diag"
	"github.com/Gui97p/wisp/internal/lexer"
	"github.com/Gui97p/wisp/internal/parser"
	"github.com/Gui97p/wisp/internal/target"
	"github.com/Gui97p/wisp/internal/target/lua"
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
		diag.Render(os.Stdout, args[2], buffer, p.Errors())

	case "analyzer":
		program := p.ParseProgram()
		fmt.Println("\nparsing errors:")
		diag.Render(os.Stdout, args[2], buffer, p.Errors())
		a := analyser.NewAnalyser(program)
		a.Analyze()
		fmt.Println("\nanalyzing errors:")
		diag.Render(os.Stdout, args[2], buffer, a.Errors())

	case "build":
		program := p.ParseProgram()
		if p.HasErrors() {
			diag.Render(os.Stdout, args[2], buffer, p.Errors())
			os.Exit(1)
		}

		a := analyser.NewAnalyser(program)
		info := a.Analyze()
		if a.HasErrors() {
			diag.Render(os.Stdout, args[2], buffer, a.Errors())
			os.Exit(1)
		}

		var backend target.Target = x64.New(program, info)
		asm, err := backend.Compile()
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

	case "run":
		program := p.ParseProgram()
		if p.HasErrors() {
			diag.Render(os.Stdout, args[2], buffer, p.Errors())
			os.Exit(1)
		}

		a := analyser.NewAnalyser(program)
		info := a.Analyze()
		if a.HasErrors() {
			diag.Render(os.Stdout, args[2], buffer, a.Errors())
			os.Exit(1)
		}

		var backend target.Target = lua.New(program, info)
		file, err := backend.Compile()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		outputPath := strings.TrimSuffix(args[2], ".wsp")
		f, err := os.Create(outputPath + ".lua")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Fprint(f, file)

		runCmd := exec.Command("lua", outputPath+".lua")
		runCmd.Stdout = os.Stdout
		runCmd.Stderr = os.Stderr
		if err := runCmd.Run(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

	default:
		fmt.Println("invalid module.\nAvaiable: build run lexer parser analyzer")
		os.Exit(1)
	}
}
