package x64

import (
	"fmt"
	"github.com/Gui97p/wisp/internal/diag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Gui97p/wisp/internal/target"
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
	Strip bool
}

func Link(t target.Target, root string, objPaths, libPaths []string, outputPath string, keepObj bool, opts LinkOptions) error {
	tmp, err := os.MkdirTemp("", "wisp-link-")
	if err != nil {
		return fmt.Errorf("failed to create a temporary folder (%s)", diag.Reason(err))
	}
	defer os.RemoveAll(tmp)

	outputPath, err = filepath.Abs(outputPath)
	if err != nil {
		return err
	}

	linker := "ld"
	args := []string{}
	if opts.Libc {
		cc, err := exec.LookPath("cc")
		if err != nil {
			return fmt.Errorf("--libc links through a C compiler driver, but cc was not found in PATH")
		}
		linker = cc
		args = append(args, "-no-pie")
		args = append(args, t.LinkArgs...)
	} else {
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

	if t.StartSource != "" && !opts.Libc {
		startObj := filepath.Join(tmp, "start.o")
		if err := Assemble(t.StartSource, startObj, filepath.Join(tmp, "start.asm"), false, t.Format); err != nil {
			return err
		}
		args = append(args, startObj)
	}

	if t.RuntimeSource != "" {
		runtimeObj := filepath.Join(tmp, "runtime.o")
		if err := Assemble(t.RuntimeSource, runtimeObj, filepath.Join(tmp, "runtime.asm"), false, t.Format); err != nil {
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

func Archive(t target.Target, objPaths []string, outputPath string) error {
	ar, err := exec.LookPath("ar")
	if err != nil {
		return fmt.Errorf("building a library needs ar, which was not found in PATH")
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

	if t.RuntimeSource != "" {
		runtimeObj := filepath.Join(tmp, "runtime-build.o")
		if err := Assemble(t.RuntimeSource, runtimeObj, filepath.Join(tmp, "runtime.asm"), false, t.Format); err != nil {
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
