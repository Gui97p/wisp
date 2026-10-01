package x64

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Gui97p/wisp/internal/target"
)

func Link(t target.Target, root string, objPaths, libPaths []string, outputPath string, keepObj bool) error {
	tmp, err := os.MkdirTemp("", "wisp-link-")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmp)

	outputPath, err = filepath.Abs(outputPath)
	if err != nil {
		return err
	}

	args := append([]string{}, t.LinkArgs...)
	if t.Entry != "" && t.Entry != "_start" {
		args = append(args, "-e", t.Entry)
	}
	args = append(args, "-o", outputPath)

	if t.StartSource != "" {
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

	ldCmd := exec.Command("ld", args...)
	ldCmd.Dir = root
	ldCmd.Stderr = os.Stderr
	if err := ldCmd.Run(); err != nil {
		return fmt.Errorf("ld failed: %w", err)
	}

	if !keepObj {
		for _, path := range objPaths {
			os.Remove(path)
			removeEmptyDirs(filepath.Dir(path))
		}
	}

	return nil
}
