package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/Gui97p/wisp/internal/module"
	"github.com/Gui97p/wisp/internal/target"
	"github.com/Gui97p/wisp/internal/target/x64"
	"github.com/spf13/cobra"
)

var (
	asmOut          string
	objOut          string
	binOut          string
	luaOut          string
	targetOpt       string
	stdOpt          string
	linkOpt         []string
	libcOpt         bool
	freestandingOpt bool
)

var buildCmd = &cobra.Command{
	Use:          "build",
	Short:        "Compile the project for a target (x86-64 by default)",
	Args:         cobra.NoArgs,
	RunE:         runBuild,
	SilenceUsage: true,
}

var runCmd = &cobra.Command{
	Use:          "run [flags] [-- program arguments]",
	Short:        "Compile the project for a target and run the result",
	Long:         "Compile the project for a target and run the result. Whatever follows -- is passed to the program as its arguments: wisp run -- input.txt",
	Args:         cobra.ArbitraryArgs,
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
	f.StringArrayVar(&linkOpt, "link", nil, "extra argument for the linker, repeatable (object files, archives, -L, -l); x86-64 targets")
	f.BoolVar(&libcOpt, "libc", false, "link the system C library and use its startup code instead of the Wisp start stub (x86-64 targets)")
	addProjectFlags(cmd)
	addOutputFlags(cmd)
}

func addProjectFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	f.StringVar(&stdOpt, "std", "", "folder to use as the standard library (the std/ import prefix)")
	f.BoolVar(&freestandingOpt, "freestanding", false, "build without a standard library: importing std/... is an error")
}

func openProject(cmd *cobra.Command) (string, *module.Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", nil, err
	}
	root := module.FindProjectRoot(cwd)

	cfg, err := module.LoadConfig(root)
	if err != nil {
		return "", nil, err
	}

	if err := configureStdlib(cmd, root, cfg); err != nil {
		return "", nil, err
	}

	return root, cfg, nil
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
	target   string
	product  string
	binPath  string
	luaDir   string
	luaEntry string
}

func build(cmd *cobra.Command, forRun bool) (*buildResult, error) {
	root, cfg, err := openProject(cmd)
	if err != nil {
		return nil, err
	}

	tgt, err := resolveTarget(cfg)
	if err != nil {
		return nil, err
	}

	if verboseOpt {
		x64.Trace = func(name string, args []string) {
			detail("  $ %s %s", name, strings.Join(args, " "))
		}
	}

	var res *buildResult
	switch tgt.Arch {
	case "x64":
		if err := rejectFlags(cmd, tgt, "lua"); err != nil {
			return nil, err
		}
		res, err = buildX64(root, cfg, tgt, forRun)
	case "lua":
		if err := rejectFlags(cmd, tgt, "asm", "obj", "bin", "link", "libc"); err != nil {
			return nil, err
		}
		res, err = buildLua(root, cfg, tgt, forRun)
	default:
		return nil, fmt.Errorf("target %q has no build pipeline", tgt.Name)
	}
	if err != nil {
		return nil, err
	}
	res.target = tgt.Name
	return res, nil
}

func runBuild(cmd *cobra.Command, args []string) error {
	start := time.Now()
	res, err := build(cmd, false)
	if err != nil {
		return err
	}
	status("Finished %s -> %s in %s", res.target, relPath(res.product), elapsed(start))
	return nil
}

func runRun(cmd *cobra.Command, args []string) error {
	start := time.Now()
	res, err := build(cmd, true)
	if err != nil {
		return err
	}
	detail("Finished %s -> %s in %s", res.target, relPath(res.product), elapsed(start))

	programArgs, err := programArguments(cmd, args)
	if err != nil {
		return err
	}

	var run *exec.Cmd
	if res.binPath != "" {
		run = exec.Command(resolveExecutable(res.binPath), programArgs...)
	} else {
		rel, err := filepath.Rel(res.luaDir, res.luaEntry)
		if err != nil {
			return err
		}
		detail("Running %s", relPath(res.luaEntry))
		run = exec.Command("lua", append([]string{rel}, programArgs...)...)
		run.Dir = res.luaDir
	}

	run.Stdin, run.Stdout, run.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := run.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signaled() {
				msg := status.Signal().String()
				fmt.Fprintf(os.Stderr, "%s%s\n", strings.ToUpper(msg[:1]), msg[1:])
				os.Exit(128 + int(status.Signal()))
			}
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

func rooted(root, p string) string {
	if p == "" || filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(root, p)
}

func pick(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func useLibc(cfg *module.Config, tgt target.Target) bool {
	libc := cfg.Libc
	if tgt.Libc != nil {
		libc = *tgt.Libc
	}
	return libc || libcOpt
}

func resolveBuildTargets(root string, cfg *module.Config, tgt target.Target, forceBin bool) buildTargets {
	var t buildTargets

	t.asmDir = pick(asmOut, rooted(root, pick(tgt.Output.Asm, cfg.Output.Asm)))
	t.wantAsm = t.asmDir != ""

	t.objDir = pick(objOut, rooted(root, pick(tgt.Output.Obj, cfg.Output.Obj)))
	t.wantObj = t.objDir != ""

	t.binPath = pick(binOut, rooted(root, pick(tgt.Output.Bin, cfg.Output.Bin)))
	t.wantBin = t.binPath != "" || forceBin

	if !t.wantAsm && !t.wantObj && !t.wantBin {
		t.wantBin = true
	}

	if t.asmDir == "" {
		t.asmDir = filepath.Join(root, "build", tgt.Name, "asm")
	}
	if t.objDir == "" {
		t.objDir = filepath.Join(root, "build", tgt.Name, "obj")
	}
	if t.binPath == "" {
		t.binPath = filepath.Join(root, "bin", tgt.Name, filepath.Base(root))
	}

	return t
}

func buildX64(root string, cfg *module.Config, tgt target.Target, forRun bool) (*buildResult, error) {
	targets := resolveBuildTargets(root, cfg, tgt, forRun)

	analyseStart := time.Now()
	proj, err := loadProject(root, cfg)
	if err != nil {
		return nil, err
	}
	detail("Analysed %s in %s", count(len(proj.compiled), "module"), elapsed(analyseStart))
	codegenStart := time.Now()

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
					return fmt.Errorf("cannot build %s, %w", c.mod.Path, c.reportFailure(err))
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
		backend.SetLibc(useLibc(cfg, tgt))
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

	detail("Generated code and objects in %s", elapsed(codegenStart))
	if targets.wantAsm {
		detail("asm: %s", relPath(targets.asmDir))
	}
	if targets.wantObj {
		detail("obj: %s", relPath(targets.objDir))
	}

	res := &buildResult{}
	switch {
	case targets.wantBin:
		res.product = targets.binPath
	case targets.wantObj:
		res.product = targets.objDir
	default:
		res.product = targets.asmDir
	}

	if targets.wantBin {
		if err := os.MkdirAll(filepath.Dir(targets.binPath), 0755); err != nil {
			return nil, err
		}
		linkStart := time.Now()
		opts := x64.LinkOptions{Extra: append(append([]string{}, cfg.Link...), linkOpt...), Libc: useLibc(cfg, tgt)}
		if err := x64.Link(tgt, root, objPaths, libPaths, targets.binPath, targets.wantObj, opts); err != nil {
			return nil, err
		}
		detail("Linked in %s", elapsed(linkStart))
		detail("bin: %s", relPath(targets.binPath))
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

func programArguments(cmd *cobra.Command, args []string) ([]string, error) {
	dash := cmd.ArgsLenAtDash()
	if dash < 0 {
		if len(args) > 0 {
			return nil, fmt.Errorf("unexpected argument %q: pass the program's arguments after --, as in wisp run -- %s", args[0], strings.Join(args, " "))
		}
		return nil, nil
	}
	if dash > 0 {
		return nil, fmt.Errorf("unexpected argument %q before --", args[0])
	}
	return args[dash:], nil
}
