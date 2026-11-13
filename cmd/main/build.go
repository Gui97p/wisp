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
	asmOut     string
	objOut     string
	binOut     string
	osOpt      string
	archOpt    string
	profileOpt string
	releaseOpt bool
	stdOpt     string
	linkOpt    []string
	libcOpt    bool
)

var buildCmd = &cobra.Command{
	Use:          "build [program...]",
	Short:        "Compile the programs of the project for a system (the machine's by default)",
	Long:         "Compile the programs of the project. Without names it builds every [program.<name>] of wisp.toml, or the one program of a project that declares none.",
	Args:         cobra.ArbitraryArgs,
	RunE:         runBuild,
	SilenceUsage: true,
}

var runCmd = &cobra.Command{
	Use:          "run [program] [flags] [-- program arguments]",
	Short:        "Compile a program of the project and run it",
	Long:         "Compile a program of the project and run it. The name is only needed when wisp.toml declares several [program.<name>]. Whatever follows -- is passed to the program as its arguments: wisp run tools -- input.txt",
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
	f.StringVar(&asmOut, "asm", "", "output directory for generated .asm files (x64)")
	f.StringVar(&objOut, "obj", "", "output directory for the object files (x64), or for the modules other than the entry (lua)")
	f.StringVar(&binOut, "bin", "", "folder of the final product: the binary named after the program (x64), or its entry script (lua)")
	f.StringVar(&osOpt, "os", "", "system to build for (linux, cc, or one declared in wisp.toml)")
	f.StringVar(&archOpt, "arch", "", "architecture to build for (x64, lua)")
	addProfileFlags(cmd)
	f.StringArrayVar(&linkOpt, "link", nil, "extra argument for the linker, repeatable (object files, archives, -L, -l); x86-64 targets")
	f.BoolVar(&libcOpt, "libc", false, "link the system C library and use its startup code instead of the Wisp start stub (x86-64 targets)")
	addProjectFlags(cmd)
	addOutputFlags(cmd)
}

func addProfileFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	f.StringVar(&profileOpt, "profile", "", "profile to build with: debug (the default), release or fast")
	f.BoolVar(&releaseOpt, "release", false, "build with the release profile, the same as --profile release")
}

func addProjectFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	f.StringVar(&stdOpt, "std", "", "standard library for this build: a folder to replace the bundled one, or none to build without one")
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
	osName, arch := runtime.GOOS, cfg.Arch
	if cfg.OS != "" {
		osName = cfg.OS
	}
	if osOpt != "" {
		osName = osOpt
		arch = ""
	}
	if archOpt != "" {
		arch = archOpt
	}

	tgt, err := target.Resolve(osName, arch)
	if err != nil {
		return tgt, err
	}
	profile, err := resolveProfile(cfg)
	if err != nil {
		return tgt, err
	}
	tgt.Profile = profile
	return tgt, nil
}

func resolveProfile(cfg *module.Config) (string, error) {
	name := cfg.Profile
	if name == "" {
		name = target.DefaultProfile
	}
	if releaseOpt {
		if profileOpt != "" && profileOpt != "release" {
			return "", fmt.Errorf("--release and --profile %s ask for different profiles", profileOpt)
		}
		name = "release"
	} else if profileOpt != "" {
		name = profileOpt
	}
	if err := target.ValidateProfile(name); err != nil {
		return "", err
	}
	return name, nil
}

func profileDir(tgt target.Target) string {
	profile := tgt.Profile
	if profile == "" {
		profile = target.DefaultProfile
	}
	return filepath.Join(profile, tgt.OS+"-"+tgt.Arch)
}

func renderErrorBlocks(c compiledModule) {
	if target.ActiveProfile().Snippets {
		c.info.RenderErrorBlocks(c.mod.BufferMap())
	}
}

func settle(cfg *module.Config) (*module.Config, target.Target, error) {
	tgt, err := resolveTarget(cfg)
	if err != nil {
		return nil, tgt, err
	}
	return cfg.ForSystem(tgt.OS, tgt.Arch), tgt, nil
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
	name     string
	target   string
	product  string
	binPath  string
	luaDir   string
	luaEntry string
	env      []module.EnvVar
}

func projectName(root string) string {
	return filepath.Base(root)
}

func programNameOf(root string, cfg *module.Config) string {
	if cfg.ProgramName != "" {
		return cfg.ProgramName
	}
	return projectName(root)
}

func binaryNameOf(root string, cfg *module.Config) string {
	if cfg.BinaryName != "" {
		return cfg.BinaryName
	}
	return programNameOf(root, cfg)
}

func expandPath(path string, tgt target.Target, name string) string {
	profile := tgt.Profile
	if profile == "" {
		profile = target.DefaultProfile
	}
	return strings.NewReplacer("{profile}", profile, "{target}", tgt.OS+"-"+tgt.Arch, "{os}", tgt.OS, "{arch}", tgt.Arch, "{name}", name).Replace(path)
}

func pickPath(root string, tgt target.Target, name, flag string, layers ...string) string {
	value := flag
	if value == "" {
		for _, l := range layers {
			if l != "" {
				value = rooted(root, l)
				break
			}
		}
	}
	return expandPath(value, tgt, name)
}

type buildJob struct {
	name    string
	library bool
}

func jobConfig(base *module.Config, job buildJob) *module.Config {
	if job.library {
		return base.ForLibrary(job.name)
	}
	return base.ForProgram(job.name)
}

func selectJobs(root string, cfg *module.Config, names []string) ([]buildJob, error) {
	project := projectName(root)
	var jobs []buildJob
	if len(names) == 0 {
		for _, name := range cfg.ProgramNames(project) {
			jobs = append(jobs, buildJob{name: name})
		}
		for _, name := range cfg.LibraryNames() {
			jobs = append(jobs, buildJob{name: name, library: true})
		}
		return jobs, nil
	}

	seen := map[string]bool{}
	for _, name := range names {
		var job buildJob
		switch {
		case cfg.HasProgram(name, project):
			job = buildJob{name: name}
		case cfg.HasLibrary(name):
			job = buildJob{name: name, library: true}
		default:
			known := append(cfg.ProgramNames(project), cfg.LibraryNames()...)
			return nil, fmt.Errorf("unknown program or library %q (there are %s)", name, strings.Join(known, ", "))
		}
		if !seen[name] {
			seen[name] = true
			jobs = append(jobs, job)
		}
	}
	return jobs, nil
}

func archivePath(binPath string) string {
	return binPath + ".a"
}

func build(cmd *cobra.Command, names []string, forRun bool) ([]*buildResult, error) {
	root, base, err := openProject(cmd)
	if err != nil {
		return nil, err
	}

	jobs, err := selectJobs(root, base, names)
	if err != nil {
		return nil, err
	}
	if len(jobs) == 0 {
		return nil, fmt.Errorf("the project declares nothing to build")
	}
	if len(jobs) > 1 {
		for _, flag := range []string{"bin", "obj", "asm"} {
			if cmd.Flags().Changed(flag) {
				return nil, fmt.Errorf("--%s names one place, and %d programs and libraries are being built: name one", flag, len(jobs))
			}
		}
	}

	if len(jobs) > 1 {
		type owner struct{ program, target string }
		folders := map[string]owner{}
		products := map[string]string{}
		for _, job := range jobs {
			name := job.name
			eff, tgt, err := settle(jobConfig(base, job))
			if err != nil {
				return nil, err
			}
			var dirs []string
			product := ""
			switch tgt.Arch {
			case "x64":
				t := resolveBuildTargets(root, eff, tgt, forRun)
				dirs = []string{t.objDir, t.asmDir}
				product = t.binPath
				if job.library {
					product = archivePath(product)
				}
			case "lua":
				entry, objDir := resolveLuaPaths(root, eff, tgt)
				dirs = []string{objDir}
				product = entry
				if job.library {
					product = objDir + "|" + name
				}
			}
			if other, taken := products[product]; taken {
				return nil, fmt.Errorf("the programs %s and %s would both write %s: give one of them another name", other, name, relPath(product))
			}
			products[product] = name
			for _, dir := range dirs {
				key := tgt.Arch + "|" + filepath.Clean(dir)
				if other, taken := folders[key]; taken && other.target != tgt.Name {
					return nil, fmt.Errorf("the programs %s (%s) and %s (%s) would write their modules into the same folder %s: put {os} and {arch} in the path", other.program, other.target, name, tgt.Name, relPath(dir))
				}
				folders[key] = owner{program: name, target: tgt.Name}
			}
		}
	}

	if verboseOpt {
		x64.Trace = func(name string, args []string) {
			detail("  $ %s %s", name, strings.Join(args, " "))
		}
	}

	for _, job := range jobs {
		if job.library {
			continue
		}
		if err := validateEntry(root, base.ForProgram(job.name)); err != nil {
			return nil, err
		}
	}

	var results []*buildResult
	for _, job := range jobs {
		name := job.name
		jobStart := time.Now()

		cfg, tgt, err := settle(jobConfig(base, job))
		if err != nil {
			return nil, err
		}

		var res *buildResult
		switch tgt.Arch {
		case "x64":
			res, err = buildX64(root, cfg, tgt, forRun, job)
		case "lua":
			if err := rejectFlags(cmd, tgt, "asm", "link", "libc"); err != nil {
				return nil, err
			}
			res, err = buildLua(root, cfg, tgt, forRun, job)
		default:
			return nil, fmt.Errorf("target %q has no build pipeline", tgt.Name)
		}
		if err != nil {
			return nil, err
		}
		res.name = name
		res.target = tgt.Name
		res.env = cfg.RunEnv()
		if !forRun {
			status("Finished %s %s -> %s in %s", res.name, res.target, relPath(res.product), elapsed(jobStart))
		}
		results = append(results, res)
	}
	return results, nil
}

func runBuild(cmd *cobra.Command, args []string) error {
	_, err := build(cmd, args, false)
	return err
}

func runRun(cmd *cobra.Command, args []string) error {
	start := time.Now()
	names, programArgs, err := splitRunArguments(cmd, args)
	if err != nil {
		return err
	}

	root, cfg, err := openProject(cmd)
	if err != nil {
		return err
	}
	if len(names) == 0 {
		if def := cfg.DefaultProgram(projectName(root)); def != "" {
			names = []string{def}
		}
	}
	jobs, err := selectJobs(root, cfg, names)
	if err != nil {
		return err
	}
	var bins []string
	for _, job := range jobs {
		if job.library {
			return fmt.Errorf("%s is a library, and a library is not run", job.name)
		}
		bins = append(bins, job.name)
	}
	if len(bins) == 0 {
		return fmt.Errorf("the project has no program to run")
	}
	if len(bins) > 1 {
		return fmt.Errorf("the project has several programs (%s): say which one to run, as in wisp run %s, or choose one with default in wisp.toml", strings.Join(bins, ", "), bins[0])
	}

	results, err := build(cmd, bins, true)
	if err != nil {
		return err
	}
	res := results[0]
	detail("Finished %s -> %s in %s", res.target, relPath(res.product), elapsed(start))

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

	run.Env = os.Environ()
	for _, v := range res.env {
		if _, set := os.LookupEnv(v.Name); !set {
			run.Env = append(run.Env, v.Name+"="+v.Value)
			detail("Setting %s from wisp.toml", v.Name)
		}
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
	if cfg.ProgramLibc != nil {
		libc = *cfg.ProgramLibc
	}
	return libc || libcOpt
}

func resolveBuildTargets(root string, cfg *module.Config, tgt target.Target, forceBin bool) buildTargets {
	var t buildTargets
	program := programNameOf(root, cfg)
	name := binaryNameOf(root, cfg)

	t.asmDir = pickPath(root, tgt, name, asmOut, cfg.ProgramOutput.Asm, cfg.Output.Asm)
	t.wantAsm = t.asmDir != ""

	t.objDir = pickPath(root, tgt, name, objOut, cfg.ProgramOutput.Obj, cfg.Output.Obj)
	t.wantObj = t.objDir != ""

	binDir := pickPath(root, tgt, name, binOut, cfg.ProgramOutput.Bin, cfg.Output.Bin)
	t.wantBin = binDir != "" || forceBin

	if !t.wantAsm && !t.wantObj && !t.wantBin {
		t.wantBin = true
	}

	if t.asmDir == "" {
		t.asmDir = filepath.Join(root, "build", profileDir(tgt), program, "asm")
	}
	if t.objDir == "" {
		t.objDir = filepath.Join(root, "build", profileDir(tgt), program, "obj")
	}
	if binDir == "" {
		binDir = filepath.Join(root, "bin", profileDir(tgt))
	}
	t.binPath = filepath.Join(binDir, name)

	return t
}

func buildX64(root string, cfg *module.Config, tgt target.Target, forRun bool, job buildJob) (*buildResult, error) {
	targets := resolveBuildTargets(root, cfg, tgt, forRun)
	if job.library {
		targets.binPath = archivePath(targets.binPath)
	}

	analyseStart := time.Now()
	proj, err := loadProject(root, cfg)
	if err != nil {
		return nil, err
	}
	detail("Analysed %s in %s", count(len(proj.compiled), "module"), elapsed(analyseStart))
	codegenStart := time.Now()

	if job.library {
		roots, err := libraryRoots(root, job.name, cfg.Libraries[job.name].Modules)
		if err != nil {
			return nil, err
		}
		if err := proj.restrict(roots); err != nil {
			return nil, fmt.Errorf("the library %s: %w", job.name, err)
		}
	} else {
		if targets.wantBin {
			if err := proj.requireMain(); err != nil {
				return nil, err
			}
		}
		if proj.mainMod != nil {
			if err := proj.restrict([]string{proj.mainMod.mod.Path}); err != nil {
				return nil, err
			}
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
				renderErrorBlocks(c)
				std := x64.New(c.merged, c.info, false, tgt.Name, c.mod.Path)
				std.SetFiles(c.declFiles)
				std.SetSources(c.mod.BufferMap())
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

		renderErrorBlocks(c)
		backend := x64.New(c.merged, c.info, isEntry, tgt.Name, c.mod.Path)
		backend.SetFiles(c.declFiles)
		backend.SetSources(c.mod.BufferMap())
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
		if err := makeOutputDir(filepath.Dir(targets.binPath)); err != nil {
			return nil, err
		}
		if job.library {
			prof := cfg.ProfileFacts(tgt.Profile)
			archiveStart := time.Now()
			if err := x64.Archive(tgt, append(append([]string{}, objPaths...), libPaths...), targets.binPath, prof.Trace); err != nil {
				return nil, err
			}
			detail("Archived in %s", elapsed(archiveStart))
			detail("lib: %s", relPath(targets.binPath))
			res.product = targets.binPath
			return res, nil
		}
		linkStart := time.Now()
		prof := cfg.ProfileFacts(tgt.Profile)
		opts := x64.LinkOptions{Extra: append(append(append([]string{}, cfg.Link...), cfg.ProgramLink...), linkOpt...), Libc: useLibc(cfg, tgt), Strip: prof.Strip, Trace: prof.Trace}
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

func splitRunArguments(cmd *cobra.Command, args []string) ([]string, []string, error) {
	dash := cmd.ArgsLenAtDash()
	if dash < 0 {
		if len(args) > 1 {
			return nil, nil, fmt.Errorf("unexpected argument %q: pass the program's arguments after --, as in wisp run -- %s", args[1], strings.Join(args[1:], " "))
		}
		return args, nil, nil
	}
	if dash > 1 {
		return nil, nil, fmt.Errorf("unexpected argument %q before --: wisp run takes at most the name of a program", args[1])
	}
	return args[:dash], args[dash:], nil
}

func makeOutputDir(dir string) error {
	err := os.MkdirAll(dir, 0755)
	if err == nil {
		return nil
	}
	for p := dir; p != filepath.Dir(p); p = filepath.Dir(p) {
		if info, statErr := os.Stat(p); statErr == nil {
			if !info.IsDir() {
				return fmt.Errorf("cannot create the output folder %s: %s is a file (maybe a binary from an older build); remove it or change the bin folder", relPath(dir), relPath(p))
			}
			break
		}
	}
	return err
}
