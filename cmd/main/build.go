package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/diag"
	"github.com/Gui97p/wisp/internal/module"
	"github.com/Gui97p/wisp/internal/target/x64"
	"github.com/spf13/cobra"
)

var (
	asmOut    string
	objOut    string
	binOut    string
	targetOpt string
)

var buildCmd = &cobra.Command{
	Use:          "build",
	Short:        "Compile the project to native x64",
	Args:         cobra.NoArgs,
	RunE:         runBuild,
	SilenceUsage: true,
}

var runCmd = &cobra.Command{
	Use:          "run",
	Short:        "Compile the project and run the resulting binary",
	Args:         cobra.NoArgs,
	RunE:         runRun,
	SilenceUsage: true,
}

func init() {
	f := buildCmd.Flags()
	f.StringVar(&asmOut, "asm", "", "output directory for generated .asm files")
	f.StringVar(&objOut, "obj", "", "output directory for generated .o files")
	f.StringVar(&binOut, "bin", "", "output path for the final binary")
	f.StringVar(&targetOpt, "target", "", "native target name (e.g. linux_x64)")

	f = runCmd.Flags()
	f.StringVar(&asmOut, "asm", "", "output directory for generated .asm files")
	f.StringVar(&objOut, "obj", "", "output directory for generated .o files")
	f.StringVar(&binOut, "bin", "", "output path for the final binary")
	f.StringVar(&targetOpt, "target", "", "native target name (e.g. linux_x64)")
}

func resolveTarget(cfg *module.Config) string {
	if targetOpt != "" {
		return targetOpt
	}
	if cfg.Target != "" {
		return cfg.Target
	}
	return runtime.GOOS + "_x64"
}

type buildTargets struct {
	asmDir  string
	objDir  string
	binPath string
	wantAsm bool
	wantObj bool
	wantBin bool
}

func resolveBuildTargets(root string, cfg *module.Config) buildTargets {
	var t buildTargets

	t.asmDir = asmOut
	if t.asmDir == "" {
		t.asmDir = cfg.Output.Asm
	}
	t.wantAsm = t.asmDir != ""

	t.objDir = objOut
	if t.objDir == "" {
		t.objDir = cfg.Output.Obj
	}
	t.wantObj = t.objDir != ""

	t.binPath = binOut
	if t.binPath == "" {
		t.binPath = cfg.Output.Bin
	}
	t.wantBin = t.binPath != ""

	if !t.wantAsm && !t.wantObj && !t.wantBin {
		t.wantBin = true
	}

	if t.asmDir == "" {
		t.asmDir = filepath.Join(root, "build", "asm")
	}
	if t.objDir == "" {
		t.objDir = filepath.Join(root, "build", "obj")
	}
	if t.binPath == "" {
		t.binPath = filepath.Join(root, "bin", filepath.Base(root))
	}

	return t
}

type compiledModule struct {
	mod    *module.Module
	merged *ast.Program
	info   *analyser.Info
}

func findMain(compiled []compiledModule, root string, entry string) (*ast.FuncDecl, *compiledModule, int, error) {
	type found struct {
		decl *ast.FuncDecl
		mod  *compiledModule
	}
	var all []found

	for i := range compiled {
		for _, decl := range compiled[i].merged.Declarations {
			fd, ok := decl.(*ast.FuncDecl)
			if ok && fd.Receiver == nil && fd.Name == "main" {
				all = append(all, found{decl: fd, mod: &compiled[i]})
			}
		}
	}

	if len(all) <= 1 {
		if len(all) == 0 {
			return nil, nil, 0, nil
		}
		return all[0].decl, all[0].mod, 1, nil
	}

	if entry == "" {
		return nil, nil, len(all), nil
	}

	entryPath := filepath.Join(root, entry)
	for _, f := range all {
		if slices.Contains(f.mod.mod.FilePaths, entryPath) {
			return f.decl, f.mod, 1, nil
		}
	}

	return nil, nil, len(all), nil
}

func runBuild(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root := module.FindProjectRoot(cwd)

	cfg, err := module.LoadConfig(root)
	if err != nil {
		return err
	}

	targets := resolveBuildTargets(root, cfg)
	target := resolveTarget(cfg)

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

	if targets.wantBin {
		if mainCount == 0 {
			return fmt.Errorf("--bin requested but no main function was found in the project")
		}
		if mainCount > 1 {
			return fmt.Errorf("multiple main functions found across the project; set 'entry' in wisp.toml to disambiguate")
		}
	}

	if mainDecl != nil {
		mainDecl.Exported = true
	}

	if !targets.wantAsm && !targets.wantObj && !targets.wantBin {
		return nil
	}

	var objPaths []string

	for _, c := range compiled {
		isEntry := mainMod != nil && c.mod == mainMod.mod

		backend := x64.New(c.merged, c.info, isEntry, target, c.mod.Path)
		source, err := backend.Compile()
		if err != nil {
			return err
		}

		name := c.mod.Path

		if targets.wantAsm {
			asmPath := filepath.Join(targets.asmDir, name+".asm")
			if err := os.MkdirAll(filepath.Dir(asmPath), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(asmPath, []byte(source), 0644); err != nil {
				return err
			}
		}

		if targets.wantObj || targets.wantBin {
			objDir := targets.objDir
			asmDir := targets.asmDir
			keepAsm := targets.wantAsm
			if !targets.wantObj {
				objDir = filepath.Join(os.TempDir(), "wisp-build")
			}
			if !keepAsm {
				asmDir = filepath.Join(os.TempDir(), "wisp-build")
			}

			objPath := filepath.Join(objDir, name+".o")
			asmPath := filepath.Join(asmDir, name+".asm")
			if err := os.MkdirAll(filepath.Dir(objPath), 0755); err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(asmPath), 0755); err != nil {
				return err
			}

			if err := x64.Assemble(source, objPath, asmPath, keepAsm); err != nil {
				return err
			}

			if targets.wantObj {
				objPaths = append(objPaths, objPath)
			} else {
				defer os.Remove(objPath)
			}
			if targets.wantBin {
				objPaths = appendUnique(objPaths, objPath)
			}
		}
	}

	if targets.wantAsm {
		fmt.Printf("[x64] asm: %s\n", targets.asmDir)
	}
	if targets.wantObj {
		fmt.Printf("[x64] obj: %s\n", targets.objDir)
	}

	if targets.wantBin {
		if err := os.MkdirAll(filepath.Dir(targets.binPath), 0755); err != nil {
			return err
		}
		if err := x64.Link(objPaths, targets.binPath, targets.wantObj); err != nil {
			return err
		}
		fmt.Printf("[x64] bin: %s\n", targets.binPath)
	}

	return nil
}

func appendUnique(list []string, item string) []string {
	if slices.Contains(list, item) {
		return list
	}
	return append(list, item)
}

func runRun(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root := module.FindProjectRoot(cwd)
	cfg, err := module.LoadConfig(root)
	if err != nil {
		return err
	}

	if binOut == "" {
		binOut = cfg.Output.Bin
	}
	if binOut == "" {
		binOut = filepath.Join(root, "bin", filepath.Base(root))
	}

	if err := runBuild(cmd, args); err != nil {
		return err
	}

	run := exec.Command(binOut)
	run.Stdout, run.Stderr = os.Stdout, os.Stderr
	if err := run.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	return nil
}
