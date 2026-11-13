package x64

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Gui97p/wisp/internal/diag"

	"github.com/Gui97p/wisp/internal/target"
	"github.com/Gui97p/wisp/internal/toolchain"
)

var Trace func(name string, args []string)

func trace(name string, args []string) {
	if Trace != nil {
		Trace(name, args)
	}
}

type LinkOptions struct {
	Extra []string
	Libc  bool
	Trace bool
	Strip bool
}

const windowsStack = "8388608,8388608"

var windowsLibraries = []string{"kernel32", "ntdll", "advapi32", "bcrypt"}

func Link(t target.Target, root string, objPaths, libPaths []string, outputPath string, keepObj bool, opts LinkOptions) error {
	pe := t.Format == "win64"
	if t.Format != "elf64" && !pe {
		return fmt.Errorf("linking the %s format is not implemented yet", t.Format)
	}
	if pe && opts.Libc {
		return fmt.Errorf("--libc is not available on %s: the Wisp runtime calls the system libraries directly", t.OS)
	}
	tmp, err := os.MkdirTemp("", "wisp-link-")
	if err != nil {
		return fmt.Errorf("failed to create a temporary folder (%s)", diag.Reason(err))
	}
	defer os.RemoveAll(tmp)

	outputPath, err = filepath.Abs(outputPath)
	if err != nil {
		return err
	}

	var linker string
	args := []string{}
	if opts.Libc {
		cc, err := toolchain.Find("cc")
		if err != nil {
			return fmt.Errorf("--libc links through a C compiler driver, but %w", err)
		}
		linker = cc
		args = append(args, "-no-pie")
		args = append(args, t.LinkArgs...)
	} else if pe {
		ld, err := toolchain.Find("ld.lld")
		if err != nil {
			return fmt.Errorf("linking for %s needs lld: %w", t.OS, err)
		}
		linker = ld
		args = append(args, "-m", "i386pep", "--subsystem", "console", "-e", t.Entry, "--stack", windowsStack)
		args = append(args, t.LinkArgs...)
	} else {
		ld, err := toolchain.Find("ld.lld", "ld")
		if err != nil {
			return err
		}
		linker = ld
		args = append(args, t.LinkArgs...)
		if t.Entry != "" && t.Entry != "_start" {
			args = append(args, "-e", t.Entry)
		}
	}
	if opts.Libc {
		args = append(args, "-Wl,--gc-sections")
	} else {
		args = append(args, "--gc-sections")
	}
	args = append(args, "-o", outputPath)

	wisp_trace := "WISP_TRACE="
	if opts.Trace {
		wisp_trace += "1"
	} else {
		wisp_trace += "0"
	}

	if t.StartSource != "" && !opts.Libc {
		startObj := filepath.Join(tmp, "start.o")
		if err := Assemble(t.StartSource, startObj, filepath.Join(tmp, "start.asm"), false, t.Format, wisp_trace); err != nil {
			return err
		}
		args = append(args, startObj)
	}

	if t.RuntimeSource != "" {
		runtimeObj := filepath.Join(tmp, "runtime.o")
		if err := Assemble(t.RuntimeSource, runtimeObj, filepath.Join(tmp, "runtime.asm"), false, t.Format, wisp_trace); err != nil {
			return err
		}
		args = append(args, runtimeObj)
	}

	for _, p := range append(append([]string{}, objPaths...), libPaths...) {
		abs, err := filepath.Abs(p)
		if err != nil {
			return err
		}
		args = append(args, abs)
	}

	if pe {
		libDir, err := toolchain.LibDir(t.OS + "-" + t.Arch)
		if err != nil {
			return err
		}
		args = append(args, "-L"+libDir)
		for _, lib := range windowsLibraries {
			args = append(args, "-l"+lib)
		}
	}

	if opts.Strip {
		args = append(args, "-s")
	}
	args = append(args, opts.Extra...)

	trace(linker, args)
	ldCmd := exec.Command(linker, args...)
	ldCmd.Dir = root
	ldCmd.Stderr = os.Stderr
	if err := ldCmd.Run(); err != nil {
		return fmt.Errorf("%s failed (%s)", filepath.Base(linker), diag.Reason(err))
	}

	if !keepObj {
		for _, path := range objPaths {
			os.Remove(path)
			removeEmptyDirs(filepath.Dir(path))
		}
	}

	return nil
}

func Archive(t target.Target, objPaths []string, outputPath string, traceOpt bool) error {
	ar, err := toolchain.Find("llvm-ar", "ar")
	if err != nil {
		return fmt.Errorf("building a library needs an archiver, but %w", err)
	}

	tmp, err := os.MkdirTemp("", "wisp-archive-")
	if err != nil {
		return fmt.Errorf("failed to create a temporary folder (%s)", diag.Reason(err))
	}
	defer os.RemoveAll(tmp)

	outputPath, err = filepath.Abs(outputPath)
	if err != nil {
		return err
	}

	var members []string
	stage := func(name, from string) error {
		data, err := os.ReadFile(from)
		if err != nil {
			return err
		}
		member := filepath.Join(tmp, name)
		if err := os.WriteFile(member, data, 0644); err != nil {
			return err
		}
		members = append(members, name)
		return nil
	}

	wisp_trace := "WISP_TRACE="
	if traceOpt {
		wisp_trace += "1"
	} else {
		wisp_trace += "0"
	}

	if t.RuntimeSource != "" {
		runtimeObj := filepath.Join(tmp, "runtime-build.o")
		if err := Assemble(t.RuntimeSource, runtimeObj, filepath.Join(tmp, "runtime.asm"), false, t.Format, wisp_trace); err != nil {
			return err
		}
		if err := stage("wisp_runtime.o", runtimeObj); err != nil {
			return err
		}
	}

	for i, p := range objPaths {
		abs, err := filepath.Abs(p)
		if err != nil {
			return err
		}
		name := fmt.Sprintf("%03d_%s", i, strings.ReplaceAll(strings.TrimSuffix(filepath.Base(abs), ".o"), string(filepath.Separator), "_")) + ".o"
		if err := stage(name, abs); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}
	os.Remove(outputPath)

	args := append([]string{"rcs", outputPath}, members...)
	trace(ar, args)
	cmd := exec.Command(ar, args...)
	cmd.Dir = tmp
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ar failed (%s)", diag.Reason(err))
	}
	return nil
}
