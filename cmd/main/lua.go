package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/diag"
	"github.com/Gui97p/wisp/internal/module"
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

	modules, err := module.BuildGraph(inputPath)
	if err != nil {
		if srcErr, ok := errors.AsType[*module.SourceError](err); ok {
			diag.Render(os.Stdout, srcErr.Path, srcErr.Buffer, srcErr.Errors)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}

	exports := map[string]*analyser.ModuleInfo{}
	var entryOutputPath string

	for _, mod := range modules {
		merged, declFiles := mod.Merge()

		isEntry := mod.Path == ""
		a := analyser.NewAnalyser(merged, isEntry, exports, declFiles)
		info := a.Analyze()
		if a.HasErrors() {
			name := mod.Path
			if isEntry {
				name = inputPath
			}
			fmt.Printf("<<  %s  >>\n", name)
			diag.RenderGrouped(os.Stdout, mod.BufferMap(), a.Errors())
			os.Exit(1)
		}

		exports[mod.Path] = &analyser.ModuleInfo{Exports: a.Exports()}

		backend := lua.New(merged, info, isEntry)
		source, err := backend.Compile()
		if err != nil {
			return err
		}

		outputPath, err := modulePath(luaOutDir, inputPath, luaOutput, mod.Path, isEntry)
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
