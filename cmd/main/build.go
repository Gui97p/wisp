package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/diag"
	"github.com/Gui97p/wisp/internal/target/x64"
	"github.com/spf13/cobra"
)

var (
	buildOutput  string
	buildOutDir  string
	buildKeepAsm bool
	buildKeepObj bool
)

var buildCmd = &cobra.Command{
	Use:          "build <file.wsp>",
	Short:        "Compile to a native x64 binary",
	Args:         cobra.MaximumNArgs(1),
	RunE:         runBuild,
	SilenceUsage: true,
}

var runCmd = &cobra.Command{
	Use:          "run <file.wsp>",
	Short:        "Compiles to a native x64 and automatically run after",
	Args:         cobra.MaximumNArgs(1),
	RunE:         runRun,
	SilenceUsage: true,
}

func init() {
	f := buildCmd.Flags()

	f.StringVarP(&buildOutput, "output", "o", "", "output binary name")
	f.StringVar(&buildOutDir, "dir", "", "output directory")
	f.BoolVar(&buildKeepAsm, "asm", false, "keep the generated .asm file")
	f.BoolVar(&buildKeepObj, "obj", false, "keep the generated .o file")

	f = runCmd.Flags()
	f.StringVarP(&buildOutput, "output", "o", "", "output binary name")
	f.StringVar(&buildOutDir, "dir", "", "output directory")
}

func runBuild(cmd *cobra.Command, args []string) error {
	inputPath, err := resolveEntry(args)
	if err != nil {
		return err
	}

	modules, ok := resolveModules(inputPath)
	if !ok {
		os.Exit(1)
	}

	exports := map[string]*analyser.ModuleInfo{}
	var entryProgram *ast.Program
	var entryInfo *analyser.Info
	hadErrors := false

	for _, mod := range modules {
		isEntry := mod.Path == ""

		if mod.ParseError != nil {
			diag.Render(os.Stdout, mod.ParseError.Path, mod.ParseError.Buffer, mod.ParseError.Errors)
			hadErrors = true
			continue
		}

		merged, declFiles := mod.Merge()

		a := analyser.NewAnalyser(merged, isEntry, exports, declFiles)
		info := a.Analyze()
		if a.HasErrors() {
			diag.RenderGrouped(os.Stdout, mod.BufferMap(), a.Errors())
			hadErrors = true
			continue
		}

		exports[mod.Path] = &analyser.ModuleInfo{Exports: a.Exports()}

		if isEntry {
			entryProgram, entryInfo = merged, info
		}
	}

	if hadErrors {
		os.Exit(1)
	}

	backend := x64.New(entryProgram, entryInfo)
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

	return nil
}

func runRun(cmd *cobra.Command, args []string) error {
	inputPath, err := resolveEntry(args)
	if err != nil {
		return err
	}
	if _, err := os.Stat(inputPath); err != nil {
		return err
	}

	buildKeepAsm, buildKeepObj = false, false

	outputPath, err := resolveOutput(inputPath, buildOutput, buildOutDir, "")
	if err != nil {
		return err
	}

	if err := runBuild(cmd, args); err != nil {
		return err
	}

	run := exec.Command("./" + outputPath)
	run.Stdout, run.Stderr = os.Stdout, os.Stderr
	if err := run.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	return nil
}
