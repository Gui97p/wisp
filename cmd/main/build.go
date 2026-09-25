package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/diag"
	"github.com/Gui97p/wisp/internal/target/x64"
	"github.com/spf13/cobra"
)

var (
	buildOutput   string
	buildOutDir   string
	buildBuildDir string
	buildKeepAsm  bool
	buildKeepObj  bool
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
	f.StringVar(&buildBuildDir, "build-dir", "", "directory for intermediate .asm/.o files")
	f.BoolVar(&buildKeepAsm, "asm", false, "keep the generated .asm file")
	f.BoolVar(&buildKeepObj, "obj", false, "keep the generated .o file")

	f = runCmd.Flags()
	f.StringVarP(&buildOutput, "output", "o", "", "output binary name")
	f.StringVar(&buildOutDir, "dir", "", "output directory")
	f.StringVar(&buildBuildDir, "build-dir", "", "directory for intermediate .asm/.o files")
	f.BoolVar(&buildKeepAsm, "asm", false, "keep the generated .asm file")
	f.BoolVar(&buildKeepObj, "obj", false, "keep the generated .o file")
}

func runBuild(cmd *cobra.Command, args []string) error {
	inputPath, err := resolveEntry(args)
	if err != nil {
		return err
	}

	if _, err := os.Stat(inputPath); err != nil {
		return err
	}

	modules, ok := resolveModules(inputPath)
	if !ok {
		os.Exit(1)
	}

	exports := map[string]*analyser.ModuleInfo{}
	hadErrors := false

	objPaths := []string{}

	for _, mod := range modules {
		isEntry := mod.Path == ""
		name := mod.Path
		if isEntry {
			name = inputPath
		}

		if mod.ParseError != nil {
			diag.Render(os.Stdout, mod.ParseError.Path, mod.ParseError.Buffer, mod.ParseError.Errors)
			hadErrors = true
			continue
		}

		merged, declFiles := mod.Merge()

		a := analyser.NewAnalyser(merged, isEntry, exports, declFiles)
		info := a.Analyze()
		if a.HasErrors() {
			fmt.Printf("<<  %s  >>\n", name)
			diag.RenderGrouped(os.Stdout, mod.BufferMap(), a.Errors())
			hadErrors = true
			continue
		}

		exports[mod.Path] = &analyser.ModuleInfo{Exports: a.Exports()}

		backend := x64.New(merged, info, isEntry)
		source, err := backend.Compile()
		if err != nil {
			return err
		}

		objFile, err := objPath(buildBuildDir, inputPath, mod.Path, isEntry)
		if err != nil {
			return err
		}
		asmFile, err := asmPath(buildBuildDir, inputPath, mod.Path, isEntry)
		if err != nil {
			return err
		}

		if err := x64.Assemble(source, objFile, asmFile, buildKeepAsm); err != nil {
			return err
		}

		objPaths = append(objPaths, objFile)
	}

	if hadErrors {
		os.Exit(1)
	}

	finalPath, err := resolveOutput(inputPath, buildOutput, buildOutDir, "")
	if err != nil {
		return err
	}

	if err := x64.Link(objPaths, finalPath, buildKeepObj); err != nil {
		return err
	}
	fmt.Printf("[x64] built %s\n", finalPath)

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

	outputPath, err := resolveOutput(inputPath, buildOutput, buildOutDir, "")
	if err != nil {
		return err
	}

	if err := runBuild(cmd, args); err != nil {
		return err
	}

	run := exec.Command(outputPath)
	run.Stdout, run.Stderr = os.Stdout, os.Stderr
	if err := run.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	return nil
}
