package x64

import (
	"fmt"
	"github.com/Gui97p/wisp/internal/diag"
	"os"
	"os/exec"
	"path/filepath"

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
