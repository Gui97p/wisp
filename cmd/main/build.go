package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/Gui97p/wisp/internal/module"
	"github.com/Gui97p/wisp/internal/target"
	"github.com/Gui97p/wisp/internal/target/x64"
	"github.com/spf13/cobra"
)

var (
	asmOut    string
	objOut    string
	binOut    string
	luaOut    string
	targetOpt string
)

var buildCmd = &cobra.Command{
	Use:          "build",
	Short:        "Compile the project for a target (x86-64 by default)",
	Args:         cobra.NoArgs,
	RunE:         runBuild,
	SilenceUsage: true,
}

var runCmd = &cobra.Command{
	Use:          "run",
	Short:        "Compile the project for a target and run the result",
	Args:         cobra.NoArgs,
	RunE:         runRun,
	SilenceUsage: true,
}

func init() {
	addBuildFlags(buildCmd)
	addBuildFlags(runCmd)
}

func addBuildFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	f.StringVar(&asmOut, "asm", "", "output directory for generated .asm files (x86-64 targets)")
	f.StringVar(&objOut, "obj", "", "output directory for generated .o files (x86-64 targets)")
	f.StringVar(&binOut, "bin", "", "output path for the final binary (x86-64 targets)")
	f.StringVar(&luaOut, "lua", "", "output directory for generated .lua files (lua target)")
	f.StringVar(&targetOpt, "target", "", "target name (e.g. linux_x64, lua)")
}

func resolveTarget(cfg *module.Config) (target.Target, error) {
	name := runtime.GOOS + "_x64"
	if cfg.Target != "" {
		name = cfg.Target
	}
	if targetOpt != "" {
		name = targetOpt
	}

	return target.Lookup(name)
}

func resolveExecutable(name string) string {
	if !strings.ContainsRune(name, os.PathSeparator) {
		return "." + string(os.PathSeparator) + name
	}

	return name
}

func rejectFlags(cmd *cobra.Command, tgt target.Target, names ...string) error {
	for _, name := range names {
		if cmd.Flags().Changed(name) {
			return fmt.Errorf("--%s does not apply to target %q", name, tgt.Name)
		}
	}

	return nil
}

type buildResult struct {
	binPath  string
	luaDir   string
	luaEntry string
}

func build(cmd *cobra.Command, forRun bool) (*buildResult, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	root := module.FindProjectRoot(cwd)

	cfg, err := module.LoadConfig(root)
	if err != nil {
		return nil, err
	}

	tgt, err := resolveTarget(cfg)
	if err != nil {
		return nil, err
	}

	switch tgt.Arch {
	case "x64":
		if err := rejectFlags(cmd, tgt, "lua"); err != nil {
			return nil, err
		}
		return buildX64(root, cfg, tgt, forRun)
	case "lua":
		if err := rejectFlags(cmd, tgt, "asm", "obj", "bin"); err != nil {
			return nil, err
		}
		return buildLua(root, cfg, forRun)
	default:
		return nil, fmt.Errorf("target %q has no build pipeline", tgt.Name)
	}
}

func runBuild(cmd *cobra.Command, args []string) error {
	_, err := build(cmd, false)
	return err
}

func runRun(cmd *cobra.Command, args []string) error {
	res, err := build(cmd, true)
	if err != nil {
		return err
	}

	var run *exec.Cmd
	if res.binPath != "" {
		run = exec.Command(resolveExecutable(res.binPath))
	} else {
		rel, err := filepath.Rel(res.luaDir, res.luaEntry)
		if err != nil {
			return err
		}
		fmt.Printf("[lua] running %s\n", res.luaEntry)
		run = exec.Command("lua", rel)
		run.Dir = res.luaDir
	}

	run.Stdout, run.Stderr = os.Stdout, os.Stderr
	if err := run.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	return nil
}

type buildTargets struct {
	asmDir  string
	objDir  string
	binPath string
	wantAsm bool
	wantObj bool
	wantBin bool
}

func resolveBuildTargets(root string, cfg *module.Config, forceBin bool) buildTargets {
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
	t.wantBin = t.binPath != "" || forceBin

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

func buildX64(root string, cfg *module.Config, tgt target.Target, forRun bool) (*buildResult, error) {
	targets := resolveBuildTargets(root, cfg, forRun)

	proj, err := loadProject(root, cfg)
	if err != nil {
		return nil, err
	}

	if targets.wantBin {
		if err := proj.requireMain(); err != nil {
			return nil, fmt.Errorf("a binary was requested but %w", err)
		}
	}

	var objPaths, libPaths []string

	var stdc *stdCache

	for _, c := range proj.compiled {
		if strings.HasPrefix(c.mod.Path, "std/") {
			if !targets.wantBin {
				continue
			}
			if stdc == nil {
				stdc, err = openStdCache(tgt)
				if err != nil {
					return nil, err
				}
			}
			c := c
			obj, err := stdc.object(c.mod.Path, func(objPath string) error {
				std := x64.New(c.merged, c.info, false, tgt.Name, c.mod.Path)
				std.SetFiles(c.declFiles)
				source, err := std.Compile()
				if err != nil {
					return fmt.Errorf("%s: %w", c.mod.Path, c.reportFailure(err))
				}
				tmp, err := os.MkdirTemp("", "wisp-std-")
				if err != nil {
					return err
				}
				defer os.RemoveAll(tmp)
				return x64.Assemble(source, objPath, filepath.Join(tmp, "std.asm"), false, tgt.Format)
			})
			if err != nil {
				return nil, err
			}
			libPaths = append(libPaths, obj)
			continue
		}

		isEntry := proj.mainMod != nil && c.mod == proj.mainMod.mod

		backend := x64.New(c.merged, c.info, isEntry, tgt.Name, c.mod.Path)
		backend.SetFiles(c.declFiles)
		source, err := backend.Compile()
		if err != nil {
			return nil, c.reportFailure(err)
		}

		name := c.mod.Path

		if targets.wantAsm {
			asmPath := filepath.Join(targets.asmDir, name+".asm")
			if err := os.MkdirAll(filepath.Dir(asmPath), 0755); err != nil {
				return nil, err
			}
			if err := os.WriteFile(asmPath, []byte(source), 0644); err != nil {
				return nil, err
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
				return nil, err
			}
			if err := os.MkdirAll(filepath.Dir(asmPath), 0755); err != nil {
				return nil, err
			}

			if err := x64.Assemble(source, objPath, asmPath, keepAsm, tgt.Format); err != nil {
				return nil, err
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

	res := &buildResult{}

	if targets.wantBin {
		if err := os.MkdirAll(filepath.Dir(targets.binPath), 0755); err != nil {
			return nil, err
		}
		if err := x64.Link(tgt, root, objPaths, libPaths, targets.binPath, targets.wantObj); err != nil {
			return nil, err
		}
		fmt.Printf("[x64] bin: %s\n", targets.binPath)
		res.binPath = targets.binPath
	}

	return res, nil
}

func appendUnique(list []string, item string) []string {
	if slices.Contains(list, item) {
		return list
	}
	return append(list, item)
}
