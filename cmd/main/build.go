package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/diag"
	"github.com/Gui97p/wisp/internal/lexer"
	"github.com/Gui97p/wisp/internal/parser"
	"github.com/Gui97p/wisp/internal/target/x64"
	"github.com/spf13/cobra"
)

var (
	buildOutput  string
	buildOutDir  string
	buildKeepAsm bool
	buildKeepObj bool
	buildRun     bool
)

var buildCmd = &cobra.Command{
	Use:          "build <file.wsp>",
	Short:        "Compile to a native x64 binary",
	Args:         cobra.ExactArgs(1),
	RunE:         runBuild,
	SilenceUsage: true,
}

func init() {
	f := buildCmd.Flags()

	f.StringVarP(&buildOutput, "output", "o", "", "output binary name")
	f.StringVar(&buildOutDir, "dir", "bin", "output directory")
	f.BoolVar(&buildKeepAsm, "asm", false, "keep the generated .asm file")
	f.BoolVar(&buildKeepObj, "obj", false, "keep the generated .o file")
	f.BoolVarP(&buildRun, "run", "r", false, "run the binary after building")
}

func runBuild(cmd *cobra.Command, args []string) error {
	inputPath := args[0]
	buffer, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}

	l := lexer.NewLexer(buffer)
	p := parser.NewParser(l)

	program := p.ParseProgram()
	if p.HasErrors() {
		diag.Render(os.Stdout, inputPath, buffer, p.Errors())
		os.Exit(1)
	}

	a := analyser.NewAnalyser(program)
	info := a.Analyze()
	if a.HasErrors() {
		diag.Render(os.Stdout, inputPath, buffer, a.Errors())
	}

	backend := x64.New(program, info)
	asm, err := backend.Compile()
	if err != nil {
		return err
	}

	outputPath, err := resolveOutput(inputPath, buildOutput, buildOutDir, "")
	if err != nil {
		return err
	}

	if err := x64.Build(asm, outputPath, buildKeepAsm, buildKeepObj); err != nil {
		return err
	}
	fmt.Printf("[%s] built: %s\n", backend.Name(), outputPath)

	if buildRun {
		runCmd := exec.Command("./" + outputPath)
		runCmd.Stdout, runCmd.Stderr = os.Stdout, os.Stderr
		return runCmd.Run()
	}

	return nil
}
