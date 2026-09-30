package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/diag"
	"github.com/Gui97p/wisp/internal/module"
	"github.com/Gui97p/wisp/internal/target/lua"
	"github.com/spf13/cobra"
)

var (
	luaOutDir string
	luaRun    bool
)

var luaCmd = &cobra.Command{
	Use:          "lua",
	Short:        "Transpile the project to lua source code",
	Args:         cobra.NoArgs,
	RunE:         runLua,
	SilenceUsage: true,
}

func init() {
	f := luaCmd.Flags()
	f.StringVar(&luaOutDir, "dir", "", "output directory")
	f.BoolVarP(&luaRun, "run", "r", false, "run the compiled entry point after building (requires lua installed)")
}

func resolveLuaOutDir(root string, cfg *module.Config) string {
	dir := luaOutDir
	if dir == "" {
		dir = cfg.Output.Lua
	}
	if dir == "" {
		return filepath.Join(root, "dist")
	}
	if filepath.IsAbs(dir) {
		return dir
	}
	return filepath.Join(root, dir)
}

func runLua(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root := module.FindProjectRoot(cwd)

	cfg, err := module.LoadConfig(root)
	if err != nil {
		return err
	}

	outDir := resolveLuaOutDir(root, cfg)

	modules, err := module.DiscoverProject(root)
	if err != nil {
		return err
	}

	exports := map[string]*analyser.ModuleInfo{}
	hadErrors := false
	var compiled []compiledModule

	for _, mod := range modules {
		if mod.ParseError != nil {
			diag.Render(os.Stdout, mod.ParseError.Path, mod.ParseError.Buffer, mod.ParseError.Errors)
			hadErrors = true
			continue
		}

		merged, declFiles := mod.Merge()

		a := analyser.NewAnalyser(merged, false, exports, declFiles, mod.Path)
		info := a.Analyze()
		if a.HasErrors() {
			fmt.Printf("<<  %s  >>\n", mod.Path)
			diag.RenderGrouped(os.Stdout, mod.BufferMap(), a.Errors())
			hadErrors = true
			continue
		}

		modExports := a.Exports()
		if a.HasErrors() {
			fmt.Printf("<<  %s  >>\n", mod.Path)
			diag.RenderGrouped(os.Stdout, mod.BufferMap(), a.Errors())
			hadErrors = true
			continue
		}

		exports[mod.Path] = &analyser.ModuleInfo{Exports: modExports}
		compiled = append(compiled, compiledModule{mod: mod, merged: merged, info: info})
	}

	if hadErrors {
		os.Exit(1)
	}

	mainDecl, mainMod, mainCount, err := findMain(compiled, root, cfg.Entry)
	if err != nil {
		return err
	}

	if luaRun && mainCount == 0 {
		return fmt.Errorf("--run requested but no main function was found in the project")
	}
	if mainCount > 1 {
		return fmt.Errorf("multiple main functions found across the project; set 'entry' in wisp.toml to disambiguate")
	}

	if mainDecl != nil {
		mainDecl.Exported = true
	}

	var entryOutputPath string

	for _, c := range compiled {
		isEntry := mainMod != nil && c.mod == mainMod.mod

		backend := lua.New(c.merged, c.info, isEntry)
		source, err := backend.Compile()
		if err != nil {
			return err
		}

		id := lua.ModuleID(c.mod.Path)
		outPath := filepath.Join(outDir, strings.ReplaceAll(id, ".", string(filepath.Separator))+".lua")

		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return err
		}
		if err := lua.Build(source, outPath); err != nil {
			return err
		}

		if isEntry {
			entryOutputPath = outPath
		}
	}

	fmt.Printf("[lua] built %s\n", outDir)

	if luaRun {
		rel, err := filepath.Rel(outDir, entryOutputPath)
		if err != nil {
			return err
		}
		fmt.Printf("[lua] running %s\n", entryOutputPath)
		run := exec.Command("lua", rel)
		run.Dir = outDir
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
