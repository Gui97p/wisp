package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/diag"
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
	Args:         cobra.MaximumNArgs(1),
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
	inputPath, err := resolveEntry(args)
	if err != nil {
		return err
	}

	modules, ok := resolveModules(inputPath)
	if !ok {
		os.Exit(1)
	}

	exports := map[string]*analyser.ModuleInfo{}
	var entryOutputPath string
	hadErrors := false

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

		outputPath, err := modulePath(luaOutDir, inputPath, luaOutput, mod.Path, isEntry)
		if err != nil {
			return err
		}

		backend := lua.New(merged, info, isEntry, mod.Path)
		source, err := backend.Compile()
		if err != nil {
			return err
		}

		if err := lua.Build(source, outputPath); err != nil {
			return err
		}

		fmt.Printf("[%s] built: %s\n", backend.Name(), outputPath)

		if isEntry {
			entryOutputPath = outputPath
		}
	}

	if hadErrors {
		os.Exit(1)
	}

	if luaRun {
		run := exec.Command("lua", filepath.Base(entryOutputPath))
		run.Dir = filepath.Dir(entryOutputPath)
		run.Stdout, run.Stderr = os.Stdout, os.Stderr
		if err := run.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			return err
		}
	}

	return nil
}
