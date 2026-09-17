package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/diag"
	"github.com/Gui97p/wisp/internal/lexer"
	"github.com/Gui97p/wisp/internal/parser"
	"github.com/Gui97p/wisp/internal/target/lua"
	"github.com/spf13/cobra"
)

var (
	luaOutput string
	luaOutDir string
	luaRun    bool
)

var luaCmd = &cobra.Command{
	Use:          "lua <file.wsp>",
	Short:        "Transpile to lua source code",
	Args:         cobra.ExactArgs(1),
	RunE:         runLua,
	SilenceUsage: true,
}

func init() {
	f := luaCmd.Flags()

	f.StringVarP(&luaOutput, "output", "o", "", "output lua file name")
	f.StringVar(&luaOutDir, "dir", "dist", "output directory")
	f.BoolVarP(&luaRun, "run", "r", false, "run lua file after building (requires lua installed)")
}

func runLua(cmd *cobra.Command, args []string) error {
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
		os.Exit(1)
	}

	backend := lua.New(program, info)
	source, err := backend.Compile()
	if err != nil {
		return err
	}

	outputPath, err := resolveOutput(inputPath, luaOutput, luaOutDir, ".lua")
	if err != nil {
		return err
	}

	if err := lua.Build(source, outputPath); err != nil {
		return err
	}
	fmt.Printf("[%s] built: %s\n", backend.Name(), outputPath)

	if luaRun {
		runCmd := exec.Command("lua", outputPath)
		runCmd.Stdout, runCmd.Stderr = os.Stdout, os.Stderr
		return runCmd.Run()
	}

	return nil
}
