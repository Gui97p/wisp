package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Gui97p/wisp/internal/module"
	"github.com/Gui97p/wisp/internal/native"
	"github.com/Gui97p/wisp/internal/toolchain"
	"github.com/Gui97p/wisp/internal/version"
	"github.com/spf13/cobra"
)

const releasesURL = "https://github.com/Gui97p/wisp/releases"

var doctorCmd = &cobra.Command{
	Use:          "doctor",
	Short:        "Check that the compiler, the standard library and the toolchain are in place",
	Args:         cobra.NoArgs,
	RunE:         runDoctor,
	SilenceUsage: true,
}

type check struct {
	label    string
	names    []string
	required bool
	note     string
}

func runDoctor(cmd *cobra.Command, args []string) error {
	fmt.Printf("wisp %s (%s) on %s/%s\n", version.Version(), version.Codename(), runtime.GOOS, runtime.GOARCH)
	if exe, err := os.Executable(); err == nil {
		fmt.Printf("  %-12s %s\n", "executable", exe)
	}

	problems := 0
	report := func(label, state, detail string) {
		fmt.Printf("  %-12s %-8s %s\n", label, state, detail)
	}

	fmt.Println()
	if root, err := module.StdlibRoot(); err != nil {
		report("std", "MISSING", err.Error())
		problems++
	} else if info, err := os.Stat(root); err != nil || !info.IsDir() {
		report("std", "MISSING", root+" does not exist: the installation is incomplete")
		problems++
	} else {
		report("std", "ok", root)
	}

	packaged := false
	if dir := toolchain.PackageDir(); dir != "" {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			packaged = true
		}
	}
	if packaged {
		report("toolchain", "ok", "bundled with this installation ("+toolchain.PackageDir()+")")
	} else {
		report("toolchain", "none", "this installation has no bundled toolchain, so the tools below come from PATH")
	}

	checks := []check{
		{label: "nasm", names: []string{"nasm"}, required: true},
		{label: "linker", names: linkerNames(false), required: true},
		{label: "lld", names: []string{"ld.lld"}, required: runtime.GOOS == "windows", note: "needed to build for Windows"},
		{label: "archiver", names: []string{"llvm-ar", "ar"}, note: "needed for libraries"},
		{label: "cc", names: []string{"cc"}, note: "only for --libc"},
	}
	for _, c := range checks {
		tool, err := toolchain.Locate(c.names...)
		if err != nil {
			state := "missing"
			if c.required {
				state = "MISSING"
				problems++
			}
			detail := "not found"
			if c.note != "" {
				detail += " (" + c.note + ")"
			}
			report(c.label, state, detail)
			continue
		}
		detail := tool.Path
		if v := toolchain.Version(tool); v != "" {
			detail += "  [" + v + "]"
		}
		if tool.Source == toolchain.SourcePath && packaged {
			detail += "  (from PATH, not from the bundle)"
		}
		report(c.label, "ok", detail)
	}

	if dir, ok := toolchain.FindLibDir("windows-x64"); ok {
		report("win libs", "ok", dir)
	} else {
		state := "missing"
		if runtime.GOOS == "windows" {
			state = "MISSING"
			problems++
		}
		report("win libs", state, "import libraries not found (needed to build for Windows)")
	}

	problems += reportNatives(report)

	if problems == 0 {
		fmt.Println("\neverything needed is in place")
		return nil
	}

	fmt.Println()
	if !packaged {
		fmt.Printf("fix: install a release package from %s. It includes nasm, lld and the Windows import libraries, and wisp finds them by itself.\n", releasesURL)
		fmt.Println("     Putting your own nasm and lld in PATH works for Linux builds, but not for Windows ones, which also need the import libraries from the package.")
	} else {
		fmt.Printf("fix: this installation is incomplete. Reinstall it from %s.\n", releasesURL)
	}
	return errors.New("the environment is not ready")
}

func reportNatives(report func(label, state, detail string)) int {
	cwd, err := os.Getwd()
	if err != nil {
		return 0
	}
	root := module.FindProjectRoot(cwd)
	if _, err := os.Stat(filepath.Join(root, "wisp.toml")); err != nil {
		return 0
	}
	cfg, err := module.LoadConfig(root)
	if err != nil {
		return 0
	}

	format := "elf64"
	if runtime.GOOS == "windows" {
		format = "win64"
	}
	spec := cfg.NativeSpec(runtime.GOOS)
	if len(spec.Libs) == 0 {
		return 0
	}

	fmt.Println()
	problems := 0
	for _, lib := range spec.Libs {
		single := spec
		single.Libs = []string{lib}
		res, err := native.Resolve(format, single, root, false)
		if err != nil {
			report("native "+lib, "MISSING", err.Error())
			problems++
			continue
		}
		detail := "found"
		if res.Interpreter != "" {
			detail += ", loader " + res.Interpreter
		}
		report("native "+lib, "ok", detail)
	}
	return problems
}
